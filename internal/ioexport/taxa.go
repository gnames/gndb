package ioexport

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gnfmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfborg/sflib/pkg/coldp"
	"github.com/sfborg/sflib/pkg/sfga"
)

// exportTaxa reads accepted name_string_indices for a data source in batches
// and writes them to the SFGA archive. Returns the count of taxa written.
func exportTaxa(
	ctx context.Context,
	pool *pgxpool.Pool,
	arc sfga.Archive,
	sourceID int,
	batchSize int,
) (int, error) {
	t := time.Now()
	gn.Info("(3/5) exporting taxa...")

	totalCount, err := countTaxa(ctx, pool, sourceID)
	if err != nil {
		return 0, err
	}

	bar := pb.Full.Start(totalCount)
	bar.Set("prefix", "Exporting taxa: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	total := 0
	cursor := ""
	for {
		batch, err := queryTaxaBatch(ctx, pool, sourceID, batchSize, cursor)
		if err != nil {
			return total, fmt.Errorf("taxa batch after cursor %q: %w", cursor, err)
		}
		if len(batch) == 0 {
			break
		}
		if err = arc.InsertTaxa(batch); err != nil {
			return total, SFGAWriteError(sourceID, "taxa", err)
		}
		total += len(batch)
		cursor = batch[len(batch)-1].ID
		bar.Add(len(batch))

		slog.Debug("taxa batch written",
			"source_id", sourceID,
			"cursor", cursor,
			"batch", len(batch),
			"total", total,
		)
	}

	gn.Message("<em>Exported %s taxa</em> %s",
		humanize.Comma(int64(total)),
		gnfmt.TimeString(time.Since(t).Seconds()),
	)
	return total, nil
}

func countTaxa(ctx context.Context, pool *pgxpool.Pool, sourceID int) (int, error) {
	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM name_string_indices
		  WHERE data_source_id = $1
		    AND (taxonomic_status NOT IN ('synonym', 'ambiguous synonym', 'misapplied')
		         OR taxonomic_status IS NULL)`, sourceID).Scan(&count)
	return count, err
}

const taxaQuery = `
SELECT
    record_id, name_string_id,
    global_id, local_id, code_id,
    classification, classification_ids, classification_ranks
FROM name_string_indices
WHERE data_source_id = $1
  AND record_id > $2
  AND (taxonomic_status NOT IN ('synonym', 'ambiguous synonym', 'misapplied')
       OR taxonomic_status IS NULL)
ORDER BY record_id
LIMIT $3
`

func queryTaxaBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	sourceID int,
	limit int,
	cursor string,
) ([]coldp.Taxon, error) {
	rows, err := pool.Query(ctx, taxaQuery, sourceID, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []coldp.Taxon
	for rows.Next() {
		var (
			recordID            string
			nameStringID        string
			globalID            *string
			localID             *string
			codeID              int
			classification      *string
			classificationIDs   *string
			classificationRanks *string
		)
		if err := rows.Scan(
			&recordID, &nameStringID,
			&globalID, &localID, &codeID,
			&classification, &classificationIDs, &classificationRanks,
		); err != nil {
			return nil, err
		}

		t := coldp.Taxon{
			ID:     recordID,
			NameID: nameStringID,
		}
		if globalID != nil {
			t.GlobalID = *globalID
		}
		if localID != nil {
			t.LocalID = *localID
		}

		clsf := strVal(classification)
		ids := strVal(classificationIDs)
		ranks := strVal(classificationRanks)

		applyClassification(&t, clsf, ranks, ids, codeID)

		batch = append(batch, t)
	}
	return batch, rows.Err()
}

// applyClassification populates the flat classification fields and ParentID
// on t from the pipe-delimited classification strings stored in PostgreSQL.
//
//   - Case 1 (ranks + IDs): fills flat fields and sets ParentID from
//     the penultimate element of classification_ids.
//   - Case 2 (ranks, no IDs): fills flat fields only.
//   - Case 3 (no ranks): attempts rank inference per node via inferRank;
//     nodes whose rank cannot be resolved are silently skipped.
func applyClassification(t *coldp.Taxon, clsf, ranks, ids string, codeID int) {
	if clsf == "" {
		return
	}

	names := strings.Split(clsf, "|")

	var rnks []string
	if ranks != "" {
		rnks = strings.Split(ranks, "|")
	}

	var idArr []string
	if ids != "" {
		idArr = strings.Split(ids, "|")
	}

	// When ranks are absent, infer them.
	if len(rnks) == 0 {
		// Use breadcrumb majority vote for code when code_id is unknown.
		effectiveCode := codeID
		if effectiveCode == 0 {
			effectiveCode = inferCodeFromBreadcrumb(clsf)
		}
		for _, name := range names {
			rnks = append(rnks, inferRank(name, effectiveCode))
		}
	}

	for i, rank := range rnks {
		if i >= len(names) {
			break
		}
		rank = strings.ToLower(rank)
		if !isTaxonRank(rank) {
			continue
		}
		id := ""
		if i < len(idArr) {
			id = idArr[i]
		}
		setFlatField(t, rank, names[i], id)
	}

	// ParentID = penultimate element of classification_ids (valid only when
	// IDs are real record_ids from this source).
	if len(idArr) >= 2 && idArr[len(idArr)-2] != "" {
		t.ParentID = idArr[len(idArr)-2]
	}
}

// setFlatField assigns name and id to the appropriate coldp.Taxon flat field.
func setFlatField(t *coldp.Taxon, rank, name, id string) {
	switch rank {
	case "kingdom":
		t.Kingdom = name
		t.KingdomID = id
	case "phylum":
		t.Phylum = name
		t.PhylumID = id
	case "subphylum":
		t.Subphylum = name
		t.SubphylumID = id
	case "class":
		t.Class = name
		t.ClassID = id
	case "subclass":
		t.Subclass = name
		t.SubclassID = id
	case "order":
		t.Order = name
		t.OrderID = id
	case "suborder":
		t.Suborder = name
		t.SuborderID = id
	case "superfamily":
		t.Superfamily = name
		t.SuperfamilyID = id
	case "family":
		t.Family = name
		t.FamilyID = id
	case "subfamily":
		t.Subfamily = name
		t.SubfamilyID = id
	case "tribe":
		t.Tribe = name
		t.TribeID = id
	case "subtribe":
		t.Subtribe = name
		t.SubtribeID = id
	case "genus":
		t.Genus = name
		t.GenusID = id
	case "subgenus":
		t.Subgenus = name
		t.SubgenusID = id
	case "species":
		t.Species = name
		t.SpeciesID = id
	}
}

// strVal safely dereferences a *string, returning "" for nil.
func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
