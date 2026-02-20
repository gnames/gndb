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

// removeOrphans orchestrates the removal of orphaned records
// from the database. This is Step 3 of the optimization
// workflow from gnidump.
//
// Orphans are removed in this order:
//  1. name_strings not referenced by name_string_indices
//  2. canonicals not referenced by name_strings
//  3. canonical_fulls not referenced by name_strings
//  4. canonical_stems not referenced by name_strings
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphans(
	ctx context.Context,
	opt *optimizer,
	_ *config.Config,
) (string, error) {
	pool := opt.operator.Pool()
	if pool == nil {
		return "", &gn.Error{
			Code: errcode.OptimizerOrphanRemovalError,
			Msg:  "Database connection lost",
			Err:  fmt.Errorf("pool is nil"),
		}
	}

	var err error
	var totalDeleted int64

	// Step 1: Remove orphan name_strings
	count, err := removeOrphanNameStrings(ctx, pool)
	if err != nil {
		return "", err
	}
	totalDeleted += count

	// Step 2: Remove orphan canonicals
	count, err = removeOrphanCanonicals(ctx, pool)
	if err != nil {
		return "", err
	}
	totalDeleted += count

	// Step 3: Remove orphan canonical_fulls
	count, err = removeOrphanCanonicalFulls(ctx, pool)
	if err != nil {
		return "", err
	}
	totalDeleted += count

	// Step 4: Remove orphan canonical_stems
	count, err = removeOrphanCanonicalStems(ctx, pool)
	if err != nil {
		return "", err
	}
	totalDeleted += count

	// Report stats
	msg := "<em>No orphaned records found</em>"
	if totalDeleted > 0 {
		msg = fmt.Sprintf(
			"<em>Removed %s orphaned records</em>",
			humanize.Comma(totalDeleted),
		)
	}

	return msg, nil
}

// removeOrphanNameStrings deletes name_strings not referenced
// by name_string_indices. Uses LEFT OUTER JOIN pattern from
// gnidump for performance.
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphanNameStrings(
	ctx context.Context,
	pool *pgxpool.Pool,
) (int64, error) {
	slog.Info("Removing orphan name-strings")

	query := `
DELETE FROM name_strings
WHERE id IN (
	SELECT ns.id
	FROM name_strings ns
	LEFT OUTER JOIN name_string_indices nsi
		ON ns.id = nsi.name_string_id
	WHERE nsi.name_string_id IS NULL
)`

	cmdTag, err := pool.Exec(ctx, query)
	if err != nil {
		return 0, &gn.Error{
			Code: errcode.OptimizerOrphanRemovalError,
			Msg:  "Failed to remove orphan name strings",
			Err:  fmt.Errorf("delete name_strings: %w", err),
		}
	}

	deletedCount := cmdTag.RowsAffected()
	slog.Info(
		"Removed orphan name-strings",
		"count",
		deletedCount,
	)

	return deletedCount, nil
}

// removeOrphanCanonicals deletes canonicals not referenced by
// name_strings. Uses LEFT OUTER JOIN pattern from gnidump for
// performance.
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphanCanonicals(
	ctx context.Context,
	pool *pgxpool.Pool,
) (int64, error) {
	slog.Info("Removing orphan canonicals")

	query := `
DELETE FROM canonicals
WHERE id IN (
	SELECT c.id
	FROM canonicals c
	LEFT OUTER JOIN name_strings ns
		ON c.id = ns.canonical_id
	WHERE ns.id IS NULL
)`

	cmdTag, err := pool.Exec(ctx, query)
	if err != nil {
		return 0, &gn.Error{
			Code: errcode.OptimizerOrphanRemovalError,
			Msg:  "Failed to remove orphan canonicals",
			Err:  fmt.Errorf("delete canonicals: %w", err),
		}
	}

	deletedCount := cmdTag.RowsAffected()
	slog.Info("Removed orphan canonicals", "count", deletedCount)

	return deletedCount, nil
}

// removeOrphanCanonicalFulls deletes canonical_fulls not
// referenced by name_strings. Uses LEFT OUTER JOIN pattern from
// gnidump for performance.
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphanCanonicalFulls(
	ctx context.Context,
	pool *pgxpool.Pool,
) (int64, error) {
	slog.Info("Removing orphan canonical_fulls")

	query := `
DELETE FROM canonical_fulls
WHERE id IN (
	SELECT cf.id
	FROM canonical_fulls cf
	LEFT OUTER JOIN name_strings ns
		ON cf.id = ns.canonical_full_id
	WHERE ns.id IS NULL
)`

	cmdTag, err := pool.Exec(ctx, query)
	if err != nil {
		return 0, &gn.Error{
			Code: errcode.OptimizerOrphanRemovalError,
			Msg:  "Failed to remove orphan canonical fulls",
			Err:  fmt.Errorf("delete canonical_fulls: %w", err),
		}
	}

	deletedCount := cmdTag.RowsAffected()
	slog.Info(
		"Removed orphan canonical_fulls",
		"count",
		deletedCount,
	)

	return deletedCount, nil
}

// removeOrphanCanonicalStems deletes canonical_stems not
// referenced by name_strings. Uses LEFT OUTER JOIN pattern from
// gnidump for performance.
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphanCanonicalStems(
	ctx context.Context,
	pool *pgxpool.Pool,
) (int64, error) {
	slog.Info("Removing orphan canonical_stems")

	query := `
DELETE FROM canonical_stems
WHERE id IN (
	SELECT cs.id
	FROM canonical_stems cs
	LEFT OUTER JOIN name_strings ns
		ON cs.id = ns.canonical_stem_id
	WHERE ns.id IS NULL
)`

	cmdTag, err := pool.Exec(ctx, query)
	if err != nil {
		return 0, &gn.Error{
			Code: errcode.OptimizerOrphanRemovalError,
			Msg:  "Failed to remove orphan canonical stems",
			Err:  fmt.Errorf("delete canonical_stems: %w", err),
		}
	}

	deletedCount := cmdTag.RowsAffected()
	slog.Info(
		"Removed orphan canonical_stems",
		"count",
		deletedCount,
	)

	return deletedCount, nil
}
