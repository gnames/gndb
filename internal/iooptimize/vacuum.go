package iooptimize

import (
	"context"
	"log/slog"
	"time"

	"github.com/gnames/gndb/pkg/config"
)

// vacuumAnalyze runs VACUUM ANALYZE on the entire database to reclaim space
// and update query planner statistics.
//
// This is Step 6 of the optimize workflow - a gndb enhancement not present in gnidump.
// VACUUM reclaims storage occupied by dead tuples.
// ANALYZE updates statistics used by the query planner.
//
// Note: This operation cannot run inside a transaction block.
//
// Reference: PostgreSQL documentation on VACUUM and FR-004 requirement
func vacuumAnalyze(ctx context.Context, opt *OptimizerImpl, _ *config.Config) error {
	slog.Info("Running VACUUM ANALYZE on database...")
	timeStart := time.Now()

	// VACUUM ANALYZE must be executed outside of a transaction
	_, err := opt.operator.Pool().Exec(ctx, "VACUUM ANALYZE")
	if err != nil {
		slog.Error("Failed to run VACUUM ANALYZE", "error", err)
		return err
	}

	elapsed := time.Since(timeStart)
	slog.Info("VACUUM ANALYZE completed", "duration", elapsed.String())

	return nil
}
