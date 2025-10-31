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
	"github.com/gnames/gndb/pkg/lifecycle"
)

// optimizer implements the Optimizer interface.
type optimizer struct {
	operator db.Operator
}

// NewOptimizer creates a new Optimizer.
func NewOptimizer(op db.Operator) lifecycle.Optimizer {
	return &optimizer{
		operator: op,
	}
}

// Optimize applies performance optimizations by executing 6
// sequential steps:
//  1. Reparse all name_strings with latest gnparser algorithms
//  2. Normalize vernacular language codes - TODO
//  3. Remove orphaned records - TODO
//  4. Extract and link words for fuzzy matching - TODO
//  5. Create verification materialized view with indexes - TODO
//  6. Run VACUUM ANALYZE to update statistics - TODO
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
	slog.Info("Step 1/6: Reparsing name strings")
	if err := reparseNames(ctx, o); err != nil {
		return err
	}
	slog.Info("Step 1/6: Complete - Name strings reparsed")

	// Step 2: Normalize vernacular language codes
	slog.Info("Step 2/6: Normalizing vernacular languages")
	if err := normalizeVernaculars(ctx, o, cfg); err != nil {
		return err
	}
	slog.Info(
		"Step 2/6: Complete - " +
			"Vernacular languages normalized",
	)

	// TODO: Remaining steps will be implemented in subsequent
	// phases

	slog.Info("Database optimization completed successfully")
	return nil
}
