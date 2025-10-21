// Package optimize implements Optimizer interface for database performance optimization.
// This is an impure I/O package that creates indexes, materialized views, and statistics.
package iooptimize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/lifecycle"
)

// OptimizerImpl implements the Optimizer interface.
type OptimizerImpl struct {
	operator db.Operator
}

// NewOptimizer creates a new Optimizer.
func NewOptimizer(op db.Operator) lifecycle.Optimizer {
	return &OptimizerImpl{
		operator: op,
	}
}

// Optimize applies performance optimizations by executing 6 sequential steps:
//  1. Reparse all name_strings with latest gnparser algorithms (Step 1)
//  2. Normalize vernacular language codes (Step 2)
//  3. Remove orphaned records (Step 3)
//  4. Extract and link words for fuzzy matching (Step 4)
//  5. Create verification materialized view with indexes (Step 5)
//  6. Run VACUUM ANALYZE to update statistics (Step 6)
//
// Errors are returned to the CLI layer for user-friendly display via gnlib.PrintUserMessage().
// Progress messages are logged via slog.Info() for developer visibility on STDERR.
//
// Reference: gnidump Build() workflow in buildio.go
func (o *OptimizerImpl) Optimize(ctx context.Context, cfg *config.Config) error {
	pool := o.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	slog.Info("Starting database optimization workflow")

	fmt.Println("STEP1")
	// Step 1: Reparse all name_strings with latest gnparser algorithms
	slog.Info("Step 1/6: Reparsing name strings")
	if err := reparseNames(ctx, o, cfg); err != nil {
		return NewStep1Error(err)
	}
	slog.Info("Step 1/6: Complete - Name strings reparsed")

	fmt.Println("STEP2")
	// Step 2: Normalize vernacular language codes
	slog.Info("Step 2/6: Normalizing vernacular languages")
	if err := fixVernacularLanguages(ctx, o, cfg); err != nil {
		return NewStep2Error(err)
	}
	slog.Info("Step 2/6: Complete - Vernacular languages normalized")

	fmt.Println("STEP3")
	// Step 3: Remove orphaned records
	slog.Info("Step 3/6: Removing orphaned records")
	if err := removeOrphans(ctx, o, cfg); err != nil {
		return NewStep3Error(err)
	}
	slog.Info("Step 3/6: Complete - Orphaned records removed")

	// Step 4: Extract and link words for fuzzy matching
	slog.Info("Step 4/6: Creating words tables")
	if err := createWords(ctx, o, cfg); err != nil {
		return NewStep4Error(err)
	}
	slog.Info("Step 4/6: Complete - Words tables created")

	// Step 5: Create verification materialized view with indexes
	slog.Info("Step 5/6: Creating verification view")
	if err := createVerificationView(ctx, o, cfg); err != nil {
		return NewStep5Error(err)
	}
	slog.Info("Step 5/6: Complete - Verification view created")

	// Step 6: Run VACUUM ANALYZE to update statistics
	slog.Info("Step 6/6: Running VACUUM ANALYZE")
	if err := vacuumAnalyze(ctx, o, cfg); err != nil {
		return NewStep6Error(err)
	}
	slog.Info("Step 6/6: Complete - VACUUM ANALYZE finished")

	slog.Info("Database optimization completed successfully")
	return nil
}
