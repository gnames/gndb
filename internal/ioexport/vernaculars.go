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
	sourceID int,
	batchSize int,
) (int, error) {
	t := time.Now()
	gn.Info("(5/5) exporting vernaculars...")

	totalCount, err := countVernaculars(ctx, pool, sourceID)
	if err != nil {
		return 0, err
	}

	bar := pb.Full.Start(totalCount)
	bar.Set("prefix", "Exporting vernaculars: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	total := 0
	cursor := vernCursor{
		recordID:           "",
		vernacularStringID: "00000000-0000-0000-0000-000000000000",
	}
	for {
		batch, lastCursor, err := queryVernacularsBatch(ctx, pool, sourceID, batchSize, cursor)
		if err != nil {
			return total, fmt.Errorf("vernaculars batch after cursor %q: %w", cursor.recordID, err)
		}
		if len(batch) == 0 {
			break
		}
		if err = arc.InsertVernaculars(batch); err != nil {
			return total, SFGAWriteError(sourceID, "vernaculars", err)
		}
		total += len(batch)
		cursor = lastCursor
		bar.SetCurrent(int64(total))

		slog.Debug("vernaculars batch written",
			"source_id", sourceID,
			"cursor", cursor.recordID,
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

func countVernaculars(ctx context.Context, pool *pgxpool.Pool, sourceID int) (int, error) {
	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM vernacular_string_indices
		  WHERE data_source_id = $1`, sourceID).Scan(&count)
	return count, err
}

const vernacularsQuery = `
SELECT
    vsi.record_id,
    vs.name,
    vsi.language,
    vsi.lang_code,
    vsi.country_code,
    vsi.locality,
    vsi.preferred,
    vsi.vernacular_string_id
FROM vernacular_string_indices vsi
JOIN vernacular_strings vs ON vs.id = vsi.vernacular_string_id
WHERE vsi.data_source_id = $1
  AND (vsi.record_id, vsi.vernacular_string_id) > ($2, $3)
ORDER BY vsi.record_id, vsi.vernacular_string_id
LIMIT $4
`

// vernCursor tracks the composite cursor for vernacular keyset pagination.
type vernCursor struct {
	recordID           string
	vernacularStringID string
}

func queryVernacularsBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	sourceID int,
	limit int,
	cursor vernCursor,
) ([]coldp.Vernacular, vernCursor, error) {
	rows, err := pool.Query(ctx, vernacularsQuery, sourceID, cursor.recordID, cursor.vernacularStringID, limit)
	if err != nil {
		return nil, vernCursor{}, err
	}
	defer rows.Close()

	var (
		batch   []coldp.Vernacular
		lastRec string
		lastVSI string
	)
	for rows.Next() {
		var (
			recordID           string
			name               string
			language           *string
			langCode           *string
			countryCode        *string
			locality           *string
			preferred          *bool
			vernacularStringID string
		)
		if err := rows.Scan(
			&recordID, &name,
			&language, &langCode, &countryCode, &locality, &preferred,
			&vernacularStringID,
		); err != nil {
			return nil, vernCursor{}, err
		}

		lastRec = recordID
		lastVSI = vernacularStringID

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
	last := vernCursor{recordID: lastRec, vernacularStringID: lastVSI}
	return batch, last, rows.Err()
}
