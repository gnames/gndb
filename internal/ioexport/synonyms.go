package ioexport

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/gnames/gnfmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfborg/sflib/pkg/coldp"
	"github.com/sfborg/sflib/pkg/sfga"
)

// exportSynonyms reads synonym name_string_indices for a data source in
// batches and writes them to the SFGA archive. Returns the count written.
func exportSynonyms(
	ctx context.Context,
	pool *pgxpool.Pool,
	arc sfga.Archive,
	ds schema.DataSource,
	batchSize int,
) (int, error) {
	t := time.Now()
	gn.Info("(4/5) exporting synonyms...")

	total := 0
	for offset := 0; ; offset += batchSize {
		batch, err := querySynonymsBatch(ctx, pool, ds.ID, batchSize, offset)
		if err != nil {
			return total, fmt.Errorf("synonyms batch at offset %d: %w", offset, err)
		}
		if len(batch) == 0 {
			break
		}
		if err = arc.InsertSynonyms(batch); err != nil {
			return total, SFGAWriteError(ds.ID, "synonyms", err)
		}
		total += len(batch)
		slog.Debug("synonyms batch written",
			"source_id", ds.ID,
			"offset", offset,
			"batch", len(batch),
			"total", total,
		)
	}

	gn.Message("<em>Exported %s synonyms</em> %s",
		humanize.Comma(int64(total)),
		gnfmt.TimeString(time.Since(t).Seconds()),
	)
	return total, nil
}

const synonymsQuery = `
SELECT
    record_id, accepted_record_id, name_string_id, taxonomic_status
FROM name_string_indices
WHERE data_source_id = $1
  AND taxonomic_status IN ('synonym', 'ambiguous synonym', 'misapplied')
ORDER BY record_id
LIMIT $2 OFFSET $3
`

func querySynonymsBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	sourceID int,
	limit, offset int,
) ([]coldp.Synonym, error) {
	rows, err := pool.Query(ctx, synonymsQuery, sourceID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []coldp.Synonym
	for rows.Next() {
		var (
			recordID         string
			acceptedRecordID *string
			nameStringID     string
			taxonomicStatus  *string
		)
		if err := rows.Scan(
			&recordID, &acceptedRecordID, &nameStringID, &taxonomicStatus,
		); err != nil {
			return nil, err
		}

		s := coldp.Synonym{
			ID:     recordID,
			NameID: nameStringID,
		}
		if acceptedRecordID != nil {
			s.TaxonID = *acceptedRecordID
		}
		if taxonomicStatus != nil {
			s.Status = coldp.NewTaxonomicStatus(*taxonomicStatus)
		}

		batch = append(batch, s)
	}
	return batch, rows.Err()
}
