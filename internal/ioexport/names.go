package ioexport

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gnfmt"
	"github.com/gnames/gnlib/ent/nomcode"
	"github.com/gnames/gnparser"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfborg/sflib/pkg/coldp"
	"github.com/sfborg/sflib/pkg/sfga"
)

// exportNames reads name_strings for a data source in batches, re-parses
// each name with gnparser using the appropriate nomenclatural code, and
// writes them to the SFGA archive. Returns the count of names written.
//
// Uses keyset pagination (WHERE ns.id > $cursor) instead of OFFSET for
// stable performance regardless of dataset size.
func exportNames(
	ctx context.Context,
	pool *pgxpool.Pool,
	arc sfga.Archive,
	parsers map[nomcode.Code]gnparser.GNparser,
	sourceID int,
	batchSize int,
) (int, error) {
	t := time.Now()
	gn.Info("(2/5) exporting names...")

	totalCount, err := countNames(ctx, pool, sourceID)
	if err != nil {
		return 0, err
	}

	bar := pb.Full.Start(totalCount)
	bar.Set("prefix", "Exporting names: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	// Use nil UUID as initial cursor — it sorts before all real UUID5 values,
	// so the first batch returns the earliest rows.
	total := 0
	cursor := "00000000-0000-0000-0000-000000000000"
	for {
		batch, err := queryNamesBatch(ctx, pool, parsers, sourceID, batchSize, cursor)
		if err != nil {
			return total, fmt.Errorf("names batch after cursor %q: %w", cursor, err)
		}
		if len(batch) == 0 {
			break
		}
		if err = arc.InsertNames(batch); err != nil {
			return total, SFGAWriteError(sourceID, "names", err)
		}
		total += len(batch)
		cursor = batch[len(batch)-1].ID
		bar.Add(len(batch))

		slog.Debug("names batch written",
			"source_id", sourceID,
			"cursor", cursor,
			"batch", len(batch),
			"total", total,
		)
	}

	gn.Message("<em>Exported %s names</em> %s",
		humanize.Comma(int64(total)),
		gnfmt.TimeString(time.Since(t).Seconds()),
	)
	return total, nil
}

func countNames(ctx context.Context, pool *pgxpool.Pool, sourceID int) (int, error) {
	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT name_string_id)
		   FROM name_string_indices
		  WHERE data_source_id = $1`, sourceID).Scan(&count)
	return count, err
}

// namesQuery fetches one page of distinct name_strings for a data source
// using keyset pagination on name_string_id.
//
// The DISTINCT ON is pushed into a subquery so PostgreSQL can use the
// name_string_indices index to seek directly to the cursor position and
// limit early, avoiding a full sort of the remaining rows.
const namesQuery = `
SELECT ns.id, ns.name,
       sub.code_id, sub.classification, sub.outlink_id
FROM (
    SELECT DISTINCT ON (name_string_id)
        name_string_id, code_id, classification, outlink_id
    FROM name_string_indices
    WHERE data_source_id = $1
      AND name_string_id > $2
    ORDER BY name_string_id, outlink_id NULLS LAST
    LIMIT $3
) sub
JOIN name_strings ns ON ns.id = sub.name_string_id
ORDER BY ns.id
`

func queryNamesBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	parsers map[nomcode.Code]gnparser.GNparser,
	sourceID int,
	limit int,
	cursor string,
) ([]coldp.Name, error) {
	rows, err := pool.Query(ctx, namesQuery, sourceID, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []coldp.Name
	for rows.Next() {
		var (
			id             string
			name           string
			codeID         int
			classification *string
			outlinkID      *string
		)
		if err := rows.Scan(&id, &name, &codeID, &classification, &outlinkID); err != nil {
			return nil, err
		}

		clsf := ""
		if classification != nil {
			clsf = *classification
		}

		code := dbCodeToNomCode(codeID, clsf)
		parser := parsers[code]

		n := coldp.Name{
			ID:                   id,
			ScientificNameString: name,
		}

		// Re-parse to populate canonical forms, cardinality, virus, surrogate,
		// parse_quality, GnID and all other gnparser-derived fields.
		n.Amend(parser)

		if outlinkID != nil && *outlinkID != "" {
			n.AlternativeID = "gnoutlink:" + *outlinkID
		}

		batch = append(batch, n)
	}
	return batch, rows.Err()
}

// dbCodeToNomCode converts a DB code_id integer to a nomcode.Code.
// DB encoding: 0=unknown, 1=ICZN, 2=ICN, 3=bacterial, 4=viral.
// When code_id is 0, scans the classification against rankDict and suffix
// rules for a majority-vote. Falls back to Botanical as the final default:
// it treats parentheses like "(Bus)" as genus authorship rather than a
// subgenus bracket, which is the lesser parsing evil for ambiguous names.
func dbCodeToNomCode(codeID int, classification string) nomcode.Code {
	switch codeID {
	case 1:
		return nomcode.Zoological
	case 2:
		return nomcode.Botanical
	case 3:
		return nomcode.Bacterial
	case 4:
		return nomcode.Virus
	}

	switch inferCodeFromBreadcrumb(classification) {
	case 1:
		return nomcode.Zoological
	case 2:
		return nomcode.Botanical
	}
	return nomcode.Botanical
}
