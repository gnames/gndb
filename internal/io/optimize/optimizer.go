// Package optimize implements Optimizer interface for database performance optimization.
// This is an impure I/O package that creates indexes, materialized views, and statistics.
package optimize

import (
	"context"
	"fmt"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/database"
	"github.com/gnames/gndb/pkg/lifecycle"
)

// OptimizerImpl implements the Optimizer interface.
type OptimizerImpl struct {
	operator database.Operator
}

// NewOptimizer creates a new Optimizer.
func NewOptimizer(op database.Operator) lifecycle.Optimizer {
	return &OptimizerImpl{operator: op}
}

// Optimize applies performance optimizations by dropping and recreating
// all optimization artifacts (indexes, materialized views, denormalized tables).
func (o *OptimizerImpl) Optimize(ctx context.Context, cfg *config.Config) error {
	pool := o.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	// TODO: Implement optimization logic
	// This will include:
	// 1. Drop existing indexes/materialized views
	// 2. Create indexes for name verification
	// 3. Create materialized views for vernacular lookups
	// 4. Set statistics targets
	// 5. Run VACUUM ANALYZE

	return fmt.Errorf("optimization not yet implemented")
}

// VacuumAnalyze runs VACUUM ANALYZE on specified tables.
func (o *OptimizerImpl) VacuumAnalyze(ctx context.Context, tableNames []string) error {
	pool := o.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	for _, table := range tableNames {
		sql := fmt.Sprintf("VACUUM ANALYZE %s", table)
		if _, err := pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("failed to vacuum analyze %s: %w", table, err)
		}
	}

	return nil
}

// CreateIndexConcurrently creates an index without blocking writes.
func (o *OptimizerImpl) CreateIndexConcurrently(ctx context.Context, indexDDL string) error {
	pool := o.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	// CONCURRENTLY cannot be run inside a transaction
	if _, err := pool.Exec(ctx, indexDDL); err != nil {
		return fmt.Errorf("failed to create index concurrently: %w", err)
	}

	return nil
}

// RefreshMaterializedView refreshes a materialized view.
func (o *OptimizerImpl) RefreshMaterializedView(ctx context.Context, viewName string, concurrently bool) error {
	pool := o.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	sql := "REFRESH MATERIALIZED VIEW"
	if concurrently {
		sql += " CONCURRENTLY"
	}
	sql += fmt.Sprintf(" %s", viewName)

	if _, err := pool.Exec(ctx, sql); err != nil {
		return fmt.Errorf("failed to refresh materialized view %s: %w", viewName, err)
	}

	return nil
}

// SetStatisticsTarget sets the statistics target for a column.
func (o *OptimizerImpl) SetStatisticsTarget(ctx context.Context, tableName, columnName string, target int) error {
	pool := o.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET STATISTICS %d", tableName, columnName, target)
	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("failed to set statistics target: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
