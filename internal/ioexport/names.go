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
	"github.com/gnames/gnlib/ent/nomcode"
	"github.com/gnames/gnparser"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfborg/sflib/pkg/coldp"
	"github.com/sfborg/sflib/pkg/sfga"
)

// exportNames reads name_strings for a data source in batches, re-parses
// each name with gnparser using the appropriate nomenclatural code, and
// writes them to the SFGA archive. Returns the count of names written.
func exportNames(
	ctx context.Context,
	pool *pgxpool.Pool,
	arc sfga.Archive,
	parsers map[nomcode.Code]gnparser.GNparser,
	ds schema.DataSource,
	batchSize int,
) (int, error) {
	t := time.Now()
	gn.Info("(2/5) exporting names...")

	total := 0
	for offset := 0; ; offset += batchSize {
		batch, err := queryNamesBatch(ctx, pool, parsers, int(ds.ID), batchSize, offset)
		if err != nil {
			return total, fmt.Errorf("names batch at offset %d: %w", offset, err)
		}
		if len(batch) == 0 {
			break
		}
		if err = arc.InsertNames(batch); err != nil {
			return total, SFGAWriteError(ds.ID, "names", err)
		}
		total += len(batch)
		slog.Debug("names batch written",
			"source_id", ds.ID,
			"offset", offset,
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

// namesQuery fetches one page of distinct name_strings for a data source.
// code_id and classification are used to select the appropriate parser.
// Rows with a non-null outlink_id are preferred (NULLS LAST ordering).
const namesQuery = `
SELECT DISTINCT ON (ns.id)
    ns.id,
    ns.name,
    nsi.code_id,
    nsi.classification,
    nsi.outlink_id
FROM name_strings ns
JOIN name_string_indices nsi ON nsi.name_string_id = ns.id
WHERE nsi.data_source_id = $1
ORDER BY ns.id, nsi.outlink_id NULLS LAST
LIMIT $2 OFFSET $3
`

func queryNamesBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	parsers map[nomcode.Code]gnparser.GNparser,
	sourceID int,
	limit, offset int,
) ([]coldp.Name, error) {
	rows, err := pool.Query(ctx, namesQuery, sourceID, limit, offset)
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
