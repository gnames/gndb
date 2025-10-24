package iooptimize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gnlib"
)

// removeOrphans orchestrates the removal of orphaned records from the database.
// This is Step 3 of the optimization workflow from gnidump.
//
// Orphans are removed in this order:
//  1. name_strings not referenced by name_string_indices
//  2. canonicals not referenced by name_strings
//  3. canonical_fulls not referenced by name_strings
//  4. canonical_stems not referenced by name_strings
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphans(ctx context.Context, o *OptimizerImpl, _ *config.Config) error {
	var err error
	var totalDeleted int64

	// Step 1: Remove orphan name_strings
	count, err := removeOrphanNameStrings(ctx, o)
	if err != nil {
		return err
	}
	totalDeleted += count

	// Step 2: Remove orphan canonicals
	count, err = removeOrphanCanonicals(ctx, o)
	if err != nil {
		return err
	}
	totalDeleted += count

	// Step 3: Remove orphan canonical_fulls
	count, err = removeOrphanCanonicalFulls(ctx, o)
	if err != nil {
		return err
	}
	totalDeleted += count

	// Step 4: Remove orphan canonical_stems
	count, err = removeOrphanCanonicalStems(ctx, o)
	if err != nil {
		return err
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
	fmt.Println(gnlib.FormatMessage(msg, nil))

	return nil
}

// removeOrphanNameStrings deletes name_strings not referenced by name_string_indices.
// Uses LEFT OUTER JOIN pattern from gnidump for performance.
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphanNameStrings(ctx context.Context, o *OptimizerImpl) (int64, error) {
	slog.Debug("Removing orphan name-strings")

	query := `DELETE FROM name_strings
  WHERE id IN (
    SELECT ns.id
      FROM name_strings ns
        LEFT OUTER JOIN name_string_indices nsi
          ON ns.id = nsi.name_string_id
      WHERE nsi.name_string_id IS NULL
    )`

	cmdTag, err := o.operator.Pool().Exec(ctx, query)
	if err != nil {
		return 0, NewOrphanRemovalError("name_strings", err)
	}

	deletedCount := cmdTag.RowsAffected()
	slog.Debug("Removed orphan name-strings", "count", deletedCount)

	return deletedCount, nil
}

// removeOrphanCanonicals deletes canonicals not referenced by name_strings.
// Uses LEFT OUTER JOIN pattern from gnidump for performance.
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphanCanonicals(ctx context.Context, o *OptimizerImpl) (int64, error) {
	slog.Debug("Removing orphan canonicals")

	query := `DELETE FROM canonicals
  WHERE id IN (
    SELECT c.id
      FROM canonicals c
        LEFT OUTER JOIN name_strings ns
          ON c.id = ns.canonical_id
      WHERE ns.id IS NULL
    )`

	cmdTag, err := o.operator.Pool().Exec(ctx, query)
	if err != nil {
		return 0, NewOrphanRemovalError("canonicals", err)
	}

	deletedCount := cmdTag.RowsAffected()
	slog.Debug("Removed orphan canonicals", "count", deletedCount)

	return deletedCount, nil
}

// removeOrphanCanonicalFulls deletes canonical_fulls not referenced by name_strings.
// Uses LEFT OUTER JOIN pattern from gnidump for performance.
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphanCanonicalFulls(ctx context.Context, o *OptimizerImpl) (int64, error) {
	slog.Debug("Removing orphan canonical_fulls")

	query := `DELETE FROM canonical_fulls
  WHERE id IN (
    SELECT cf.id
      FROM canonical_fulls cf
        LEFT OUTER JOIN name_strings ns
          ON cf.id = ns.canonical_full_id
      WHERE ns.id IS NULL
    )`

	cmdTag, err := o.operator.Pool().Exec(ctx, query)
	if err != nil {
		return 0, NewOrphanRemovalError("canonical_fulls", err)
	}

	deletedCount := cmdTag.RowsAffected()
	slog.Debug("Removed orphan canonical_fulls", "count", deletedCount)

	return deletedCount, nil
}

// removeOrphanCanonicalStems deletes canonical_stems not referenced by name_strings.
// Uses LEFT OUTER JOIN pattern from gnidump for performance.
//
// Reference: gnidump removeOrphans() in db_views.go
func removeOrphanCanonicalStems(ctx context.Context, o *OptimizerImpl) (int64, error) {
	slog.Debug("Removing orphan canonical_stems")

	query := `DELETE FROM canonical_stems
    WHERE id IN (
      SELECT cs.id
        FROM canonical_stems cs
          LEFT OUTER JOIN name_strings ns
            ON cs.id = ns.canonical_stem_id
        WHERE ns.id IS NULL
      )`

	cmdTag, err := o.operator.Pool().Exec(ctx, query)
	if err != nil {
		return 0, NewOrphanRemovalError("canonical_stems", err)
	}

	deletedCount := cmdTag.RowsAffected()
	slog.Debug("Removed orphan canonical_stems", "count", deletedCount)

	return deletedCount, nil
}
