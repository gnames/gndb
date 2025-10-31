package iooptimize

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// createReparseTempTable creates an UNLOGGED temporary table to
// hold parsed results.
func createReparseTempTable(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	query := `
CREATE UNLOGGED TABLE IF NOT EXISTS temp_reparse_names (
	name_string_id UUID PRIMARY KEY,
	canonical_id UUID,
	canonical_full_id UUID,
	canonical_stem_id UUID,
	canonical TEXT,
	canonical_full TEXT,
	canonical_stem TEXT,
	bacteria BOOLEAN,
	virus BOOLEAN,
	surrogate BOOLEAN,
	parse_quality INTEGER,
	cardinality INTEGER,
	year SMALLINT
)`

	_, err := pool.Exec(ctx, query)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerTempTableError,
			Msg:  "Failed to create temporary table",
			Err:  fmt.Errorf("create table: %w", err),
		}
	}
	return nil
}

// bulkInsertToTempTable inserts a batch using CopyFrom.
func bulkInsertToTempTable(
	ctx context.Context,
	pool *pgxpool.Pool,
	batch []reparsed,
) error {
	if len(batch) == 0 {
		return nil
	}

	columns := []string{
		"name_string_id",
		"canonical_id",
		"canonical_full_id",
		"canonical_stem_id",
		"canonical",
		"canonical_full",
		"canonical_stem",
		"bacteria",
		"virus",
		"surrogate",
		"parse_quality",
		"cardinality",
		"year",
	}

	copySource := pgx.CopyFromSlice(
		len(batch),
		func(i int) ([]any, error) {
			r := batch[i]
			return []any{
				r.nameStringID,
				nullStringToInterface(r.canonicalID),
				nullStringToInterface(r.canonicalFullID),
				nullStringToInterface(r.canonicalStemID),
				r.canonical,
				r.canonicalFull,
				r.canonicalStem,
				r.bacteria,
				nullBoolToInterface(r.virus),
				nullBoolToInterface(r.surrogate),
				r.parseQuality,
				nullInt32ToInterface(r.cardinality),
				nullInt16ToInterface(r.year),
			}, nil
		},
	)

	rowsInserted, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"temp_reparse_names"},
		columns,
		copySource,
	)

	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerTempTableError,
			Msg:  "Failed to bulk insert to temporary table",
			Err:  fmt.Errorf("copy from: %w", err),
		}
	}

	if int(rowsInserted) != len(batch) {
		return &gn.Error{
			Code: errcode.OptimizerTempTableError,
			Msg:  "Bulk insert row count mismatch",
			Err: fmt.Errorf(
				"expected %d rows, inserted %d",
				len(batch),
				rowsInserted,
			),
		}
	}

	return nil
}

// Helper functions for NULL handling
func nullStringToInterface(ns sql.NullString) any {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func nullBoolToInterface(nb sql.NullBool) any {
	if nb.Valid {
		return nb.Bool
	}
	return nil
}

func nullInt32ToInterface(ni sql.NullInt32) any {
	if ni.Valid {
		return ni.Int32
	}
	return nil
}

func nullInt16ToInterface(ni sql.NullInt16) any {
	if ni.Valid {
		return ni.Int16
	}
	return nil
}

// batchUpdateNameStrings updates name_strings from temp table.
func batchUpdateNameStrings(
	ctx context.Context,
	pool *pgxpool.Pool,
) (int64, error) {
	query := `
UPDATE name_strings ns
SET
	canonical_id = t.canonical_id,
	canonical_full_id = t.canonical_full_id,
	canonical_stem_id = t.canonical_stem_id,
	bacteria = t.bacteria,
	virus = t.virus,
	surrogate = t.surrogate,
	parse_quality = t.parse_quality,
	cardinality = t.cardinality,
	year = t.year
FROM temp_reparse_names t
WHERE ns.id = t.name_string_id`

	result, err := pool.Exec(ctx, query)
	if err != nil {
		return 0, &gn.Error{
			Code: errcode.OptimizerReparseError,
			Msg:  "Failed to update name strings",
			Err:  fmt.Errorf("batch update: %w", err),
		}
	}

	return result.RowsAffected(), nil
}

// batchInsertCanonicals inserts unique canonicals into canonical
// tables.
func batchInsertCanonicals(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	// Insert canonicals
	_, err := pool.Exec(ctx, `
INSERT INTO canonicals (id, name)
SELECT DISTINCT canonical_id, canonical
FROM temp_reparse_names
WHERE canonical_id IS NOT NULL AND canonical != ''
ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerCanonicalInsertError,
			Msg:  "Failed to insert canonicals",
			Err:  fmt.Errorf("insert canonicals: %w", err),
		}
	}

	// Insert canonical_stems
	_, err = pool.Exec(ctx, `
INSERT INTO canonical_stems (id, name)
SELECT DISTINCT canonical_stem_id, canonical_stem
FROM temp_reparse_names
WHERE canonical_stem_id IS NOT NULL AND canonical_stem != ''
ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerCanonicalInsertError,
			Msg:  "Failed to insert canonical stems",
			Err:  fmt.Errorf("insert stems: %w", err),
		}
	}

	// Insert canonical_fulls
	_, err = pool.Exec(ctx, `
INSERT INTO canonical_fulls (id, name)
SELECT DISTINCT canonical_full_id, canonical_full
FROM temp_reparse_names
WHERE canonical_full_id IS NOT NULL AND canonical_full != ''
ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerCanonicalInsertError,
			Msg:  "Failed to insert canonical fulls",
			Err:  fmt.Errorf("insert fulls: %w", err),
		}
	}

	return nil
}
