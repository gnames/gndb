package iooptimize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/jackc/pgx/v5/pgxpool"
)

// createVerificationView orchestrates the creation of the
// verification materialized view. This is Step 5 of the
// optimization workflow from gnidump.
//
// Workflow:
//  1. Drop existing verification view if exists
//  2. Create verification materialized view with proper SQL
//  3. Create indexes on: canonical_id, name_string_id, year
//
// Reference: gnidump createVerification() in db_views.go
func createVerificationView(
	ctx context.Context,
	opt *optimizer,
	_ *config.Config,
) error {
	pool := opt.operator.Pool()
	if pool == nil {
		return &gn.Error{
			Code: errcode.OptimizerViewCreationError,
			Msg:  "Database connection lost",
			Err:  fmt.Errorf("pool is nil"),
		}
	}

	// Drop existing view
	err := dropVerificationView(ctx, pool)
	if err != nil {
		return err
	}

	// Create materialized view
	slog.Info("Building verification view")
	viewSQL := buildVerificationViewSQL()
	_, err = pool.Exec(ctx, viewSQL)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerViewCreationError,
			Msg:  "Failed to create verification view",
			Err:  fmt.Errorf("create view: %w", err),
		}
	}

	// Create indexes
	err = createVerificationIndexes(ctx, pool)
	if err != nil {
		return err
	}

	slog.Info("Verification view created successfully")

	// Count records in verification view and report stats
	var count int64
	q := "SELECT COUNT(*) FROM verification"
	err = pool.QueryRow(ctx, q).Scan(&count)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerViewCreationError,
			Msg:  "Failed to count verification records",
			Err:  fmt.Errorf("count query: %w", err),
		}
	}

	msg := fmt.Sprintf(
		"<em>Created verification view with %s records</em>",
		humanize.Comma(count),
	)
	gn.Info(msg)

	return nil
}

// dropVerificationView drops the existing verification
// materialized view if it exists.
//
// Reference: gnidump createVerification() in db_views.go
func dropVerificationView(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	q := "DROP MATERIALIZED VIEW IF EXISTS verification"
	_, err := pool.Exec(ctx, q)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerViewCreationError,
			Msg:  "Failed to drop existing verification view",
			Err:  fmt.Errorf("drop view: %w", err),
		}
	}
	return nil
}

// buildVerificationViewSQL returns the SQL statement to create
// the verification materialized view. The view denormalizes data
// from name_string_indices and name_strings for fast verification
// queries.
//
// Reference: gnidump createVerification() in db_views.go and
// data-model.md
func buildVerificationViewSQL() string {
	return `CREATE MATERIALIZED VIEW verification AS
WITH taxon_names AS (
	SELECT nsi.data_source_id, nsi.record_id,
		nsi.name_string_id, ns.name
	FROM name_string_indices nsi
	JOIN name_strings ns
		ON nsi.name_string_id = ns.id
)
SELECT nsi.data_source_id, nsi.record_id, nsi.name_string_id,
	ns.name, nsi.name_id, nsi.code_id, ns.year, ns.cardinality,
	ns.canonical_id, ns.virus, ns.bacteria, ns.parse_quality,
	nsi.local_id, nsi.outlink_id, nsi.taxonomic_status,
	nsi.accepted_record_id, tn.name_string_id as accepted_name_id,
	tn.name as accepted_name, nsi.classification,
	nsi.classification_ranks, nsi.classification_ids
FROM name_string_indices nsi
JOIN name_strings ns ON ns.id = nsi.name_string_id
LEFT JOIN taxon_names tn
	ON nsi.data_source_id = tn.data_source_id AND
	   nsi.accepted_record_id = tn.record_id
WHERE
	(
		ns.canonical_id is not NULL AND
		surrogate != TRUE AND
		(bacteria != TRUE OR parse_quality < 3)
	) OR ns.virus = TRUE`
}

// createVerificationIndexes creates 3 indexes on the verification
// materialized view to optimize common query patterns:
//  1. canonical_id - for canonical name lookups
//  2. name_string_id - for name string lookups
//  3. year - for year-based filtering
//
// Reference: gnidump createVerification() in db_views.go
func createVerificationIndexes(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	slog.Info("Building indexes for verification view")

	// Index 1: canonical_id
	_, err := pool.Exec(
		ctx,
		"CREATE INDEX ON verification (canonical_id)",
	)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerViewCreationError,
			Msg:  "Failed to create canonical_id index",
			Err:  fmt.Errorf("create index: %w", err),
		}
	}

	// Index 2: name_string_id
	_, err = pool.Exec(
		ctx,
		"CREATE INDEX ON verification (name_string_id)",
	)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerViewCreationError,
			Msg:  "Failed to create name_string_id index",
			Err:  fmt.Errorf("create index: %w", err),
		}
	}

	// Index 3: year
	_, err = pool.Exec(ctx, "CREATE INDEX ON verification (year)")
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerViewCreationError,
			Msg:  "Failed to create year index",
			Err:  fmt.Errorf("create index: %w", err),
		}
	}

	slog.Info("Verification indexes created successfully")
	return nil
}
