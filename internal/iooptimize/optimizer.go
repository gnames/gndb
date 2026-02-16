// Package iooptimize implements Optimizer interface for database
// performance optimization. This is an impure I/O package that
// creates indexes, materialized views, and statistics.
package iooptimize

import (
	"context"
	"log/slog"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/gnames/gndb/pkg/gndb"
)

// optimizer implements the Optimizer interface.
type optimizer struct {
	operator db.Operator
}

// NewOptimizer creates a new Optimizer.
func NewOptimizer(op db.Operator) gndb.Optimizer {
	return &optimizer{
		operator: op,
	}
}

// Optimize applies performance optimizations by executing 6
// sequential steps:
//  1. Reparse all name_strings with latest gnparser algorithms
//  2. Normalize vernacular language codes
//  3. Remove orphaned records
//  4. Extract and link words for fuzzy matching
//  5. Create verification materialized view with indexes
//  6. Run VACUUM ANALYZE to update statistics
//
// Errors are returned to the CLI layer for user-friendly display
// via gn.PrintErrorMessage(). Progress messages are logged via
// slog.Info() for developer visibility.
//
// Reference: gnidump Build() workflow in buildio.go
func (o *optimizer) Optimize(
	ctx context.Context,
	cfg *config.Config,
) error {
	var msg string
	pool := o.operator.Pool()
	if pool == nil {
		return &gn.Error{
			Code: errcode.DBConnectionError,
			Msg:  "Database not connected",
			Err:  nil,
		}
	}

	slog.Info("Starting database optimization")
	gn.Info(
		"Optimization in progress, " +
			"<em>it might take a while</em>...",
	)

	// Step 1: Reparse all name_strings with latest gnparser
	// algorithms
	msg = "Step 1/6: Reparsing name strings"
	gn.Info(msg)
	slog.Info(msg)
	if err := reparseNames(ctx, o); err != nil {
		return err
	}
	slog.Info("Step 1/6: Complete - Name strings reparsed")

	// Step 2: Normalize vernacular language codes
	msg = "Step 2/6: Normalizing vernacular languages"
	gn.Info(msg)
	slog.Info(msg)
	if err := normalizeVernaculars(ctx, o, cfg); err != nil {
		return err
	}
	slog.Info(
		"Step 2/6: Complete - " +
			"Vernacular languages normalized",
	)

	// Step 3: Remove orphaned records
	msg = "Step 3/6: Removing orphaned records"
	gn.Info(msg)
	slog.Info(msg)
	if err := removeOrphans(ctx, o, cfg); err != nil {
		return err
	}
	slog.Info("Step 3/6: Complete - Orphaned records removed")

	// Step 4: Extract and link words for advanced matching
	msg = "Step 4/6: Extracting words for advanced matching"
	gn.Info(msg)
	slog.Info(msg)
	if err := extractWords(ctx, o, cfg); err != nil {
		return err
	}
	slog.Info("Step 4/6: Complete - Words extracted and linked")

	// Step 5: Create verification materialized view
	msg = "Step 5/6: Creating verification view"
	gn.Info(msg)
	slog.Info(msg)
	if err := createVerificationView(ctx, o, cfg); err != nil {
		return err
	}
	slog.Info("Step 5/6: Complete - Verification view created")

	// Step 6: Run VACUUM ANALYZE
	msg = "Step 6/6: Running VACUUM ANALYZE"
	gn.Info(msg)
	slog.Info(msg)
	if err := vacuumAnalyze(ctx, o, cfg); err != nil {
		return err
	}
	slog.Info("Step 6/6: Complete - VACUUM ANALYZE finished")
	return nil
}
