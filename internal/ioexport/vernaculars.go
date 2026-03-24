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

// exportVernaculars reads vernacular_string_indices for a data source in
// batches and writes them to the SFGA archive. Returns the count written.
func exportVernaculars(
	ctx context.Context,
	pool *pgxpool.Pool,
	arc sfga.Archive,
	ds schema.DataSource,
	batchSize int,
) (int, error) {
	t := time.Now()
	gn.Info("(5/5) exporting vernaculars...")

	total := 0
	for offset := 0; ; offset += batchSize {
		batch, err := queryVernacularsBatch(ctx, pool, ds.ID, batchSize, offset)
		if err != nil {
			return total, fmt.Errorf("vernaculars batch at offset %d: %w", offset, err)
		}
		if len(batch) == 0 {
			break
		}
		if err = arc.InsertVernaculars(batch); err != nil {
			return total, SFGAWriteError(ds.ID, "vernaculars", err)
		}
		total += len(batch)
		slog.Debug("vernaculars batch written",
			"source_id", ds.ID,
			"offset", offset,
			"batch", len(batch),
			"total", total,
		)
	}

	gn.Message("<em>Exported %s vernaculars</em> %s",
		humanize.Comma(int64(total)),
		gnfmt.TimeString(time.Since(t).Seconds()),
	)
	return total, nil
}

const vernacularsQuery = `
SELECT
    vsi.record_id,
    vs.name,
    vsi.language,
    vsi.lang_code,
    vsi.country_code,
    vsi.locality,
    vsi.preferred
FROM vernacular_string_indices vsi
JOIN vernacular_strings vs ON vs.id = vsi.vernacular_string_id
WHERE vsi.data_source_id = $1
ORDER BY vsi.record_id
LIMIT $2 OFFSET $3
`

func queryVernacularsBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	sourceID int,
	limit, offset int,
) ([]coldp.Vernacular, error) {
	rows, err := pool.Query(ctx, vernacularsQuery, sourceID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []coldp.Vernacular
	for rows.Next() {
		var (
			recordID    string
			name        string
			language    *string
			langCode    *string
			countryCode *string
			locality    *string
			preferred   *bool
		)
		if err := rows.Scan(
			&recordID, &name,
			&language, &langCode, &countryCode, &locality, &preferred,
		); err != nil {
			return nil, err
		}

		v := coldp.Vernacular{
			TaxonID: recordID,
			Name:    name,
		}

		// Use lang_code (ISO 639-3) if available, otherwise language string.
		if langCode != nil && *langCode != "" {
			v.Language = *langCode
		} else if language != nil {
			v.Language = *language
		}

		if countryCode != nil {
			v.Country = *countryCode
		}
		if locality != nil {
			v.Area = *locality
		}
		if preferred != nil {
			v.Preferred = coldp.ToBool(fmt.Sprintf("%v", *preferred))
		}

		batch = append(batch, v)
	}
	return batch, rows.Err()
}
