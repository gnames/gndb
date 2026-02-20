package iooptimize

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
)

// vacuumAnalyze runs VACUUM ANALYZE on the entire database to
// reclaim space and update query planner statistics.
//
// This is Step 6 of the optimize workflow - a gndb enhancement
// not present in gnidump. VACUUM reclaims storage occupied by
// dead tuples. ANALYZE updates statistics used by the query
// planner.
//
// Note: This operation cannot run inside a transaction block.
//
// Reference: PostgreSQL documentation on VACUUM and FR-004
// requirement
func vacuumAnalyze(
	ctx context.Context,
	opt *optimizer,
	_ *config.Config,
) error {
	pool := opt.operator.Pool()
	if pool == nil {
		return &gn.Error{
			Code: errcode.OptimizerVacuumError,
			Msg:  "Database connection lost",
			Err:  fmt.Errorf("pool is nil"),
		}
	}

	slog.Info("Running VACUUM ANALYZE on database")
	timeStart := time.Now()

	// VACUUM ANALYZE must be executed outside of a transaction
	_, err := pool.Exec(ctx, "VACUUM ANALYZE")
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerVacuumError,
			Msg:  "Failed to run VACUUM ANALYZE",
			Err:  fmt.Errorf("vacuum: %w", err),
		}
	}

	elapsed := time.Since(timeStart)
	slog.Info(
		"VACUUM ANALYZE completed",
		"duration", elapsed.String(),
	)

	return nil
}
