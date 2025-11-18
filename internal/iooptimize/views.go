package iooptimize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
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

	// Drop existing views and create new ones using operator
	slog.Info("Building verification view")
	if err := opt.operator.DropMaterializedViews(ctx); err != nil {
		return err
	}

	if err := opt.operator.CreateMaterializedViews(ctx); err != nil {
		return err
	}

	slog.Info("Verification view created successfully")

	// Count records in verification view and report stats
	var count int64
	q := "SELECT COUNT(*) FROM verification"
	err := pool.QueryRow(ctx, q).Scan(&count)
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
