// Package iooptimize implements Optimizer interface for database
// performance optimization. This is an impure I/O package that
// creates indexes, materialized views, and statistics.
package iooptimize

import (
	"context"
	"log/slog"
	"time"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/gnames/gndb/pkg/gndb"
	"github.com/gnames/gnfmt"
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

// Optimize applies performance optimizations by executing 8
// sequential steps:
//  1. Reparse all name_strings with latest gnparser algorithms
//  2. Normalize vernacular language codes
//  3. Remove orphaned records
//  4. Flag possibly misspelled ION canonicals
//  5. Extract and link words for fuzzy matching
//  6. Create verification materialized view with indexes
//  7. Build abbreviation lookup table
//  8. Run VACUUM ANALYZE to update statistics
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
	var err error
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

	startTime := time.Now()
	var stepStart time.Time

	// Step 1: Reparse all name_strings with latest gnparser
	// algorithms
	msg = "Step 1/8: Reparsing name strings"
	gn.Info(msg)
	slog.Info(msg)
	stepStart = time.Now()
	if msg, err = reparseNames(ctx, o); err != nil {
		return err
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(stepStart).Seconds()))
	slog.Info("Step 1/8: Complete - Name strings reparsed")

	// Step 2: Normalize vernacular language codes
	msg = "Step 2/8: Normalizing vernacular languages"
	gn.Info(msg)
	slog.Info(msg)
	stepStart = time.Now()
	if msg, err = normalizeVernaculars(ctx, o, cfg); err != nil {
		return err
	}
	gn.Message(
		"%s %s", msg, gnfmt.TimeString(time.Since(stepStart).Seconds()),
	)
	slog.Info(
		"Step 2/8: Complete - " +
			"Vernacular languages normalized",
	)

	// Step 3: Remove orphaned records
	msg = "Step 3/8: Removing orphaned records"
	gn.Info(msg)
	slog.Info(msg)
	stepStart = time.Now()
	if msg, err = removeOrphans(ctx, o, cfg); err != nil {
		return err
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(stepStart).Seconds()))
	slog.Info("Step 3/8: Complete - Orphaned records removed")

	// Step 4: Flag possibly misspelled ION canonicals
	msg = "Step 4/8: Flagging ION misspellings"
	gn.Info(msg)
	slog.Info(msg)
	stepStart = time.Now()
	if msg, err = flagMisspellings(ctx, o, cfg); err != nil {
		return err
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(stepStart).Seconds()))
	slog.Info("Step 4/8: Complete - ION misspellings flagged")

	// Step 5: Extract and link words for advanced matching
	msg = "Step 5/8: Extracting words for advanced matching"
	gn.Info(msg)
	slog.Info(msg)
	stepStart = time.Now()
	if msg, err = extractWords(ctx, o, cfg); err != nil {
		return err
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(stepStart).Seconds()))
	slog.Info("Step 5/8: Complete - Words extracted and linked")

	// Step 6: Create verification materialized view
	msg = "Step 6/8: Creating verification view"
	gn.Info(msg)
	slog.Info(msg)
	stepStart = time.Now()
	if msg, err = createVerificationView(ctx, o, cfg); err != nil {
		return err
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(stepStart).Seconds()))
	slog.Info("Step 6/8: Complete - Verification view created")

	// Step 7: Build abbreviation lookup table
	msg = "Step 7/8: Building abbreviation lookup"
	gn.Info(msg)
	slog.Info(msg)
	stepStart = time.Now()
	if msg, err = buildAbbrLookup(ctx, o, cfg); err != nil {
		return err
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(stepStart).Seconds()))
	slog.Info("Step 7/8: Complete - Abbreviation lookup built")

	// Step 8: Run VACUUM ANALYZE
	msg = "Step 8/8: Running VACUUM ANALYZE"
	gn.Info(msg)
	slog.Info(msg)
	stepStart = time.Now()
	if err := vacuumAnalyze(ctx, o, cfg); err != nil {
		return err
	}
	gn.Message(
		"<em>VACUUM ANALYZE completed</em> %s",
		gnfmt.TimeString(time.Since(stepStart).Seconds()),
	)
	slog.Info("Step 8/8: Complete - VACUUM ANALYZE finished")

	totalDuration := time.Since(startTime)
	slog.Info("Optimization complete",
		"duration", gnfmt.TimeString(totalDuration.Seconds()))
	gn.Info("Optimization complete. Elapsed time: <em>%s</em>",
		gnfmt.TimeString(totalDuration.Seconds()))
	return nil
}
