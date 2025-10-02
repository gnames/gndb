// Package contracts defines interfaces for database optimization operations.
package contracts

import (
	"context"
)

// Optimizer defines operations for database restructure and performance tuning.
// Implemented by: internal/io/database/optimizer.go
// Used by: pkg/restructure
type Optimizer interface {
	// CreateIndexes creates all secondary indexes for a table.
	// Uses CONCURRENTLY mode to avoid blocking reads (slower but safer).
	// Returns list of created index names.
	CreateIndexes(ctx context.Context, tableName string, indexDDL []string, concurrent bool) ([]string, error)

	// CreateMaterializedView creates a materialized view from a SQL query.
	CreateMaterializedView(ctx context.Context, viewName, query string) error

	// RefreshMaterializedViews refreshes all materialized views in order.
	// Use concurrent=true to allow reads during refresh (slower but non-blocking).
	RefreshMaterializedViews(ctx context.Context, viewNames []string, concurrent bool) error

	// TuneStatistics sets statistics targets for high-cardinality columns.
	// Higher targets (e.g., 1000) improve query planner accuracy.
	TuneStatistics(ctx context.Context, columnTargets []ColumnStatTarget) error

	// EnableTrigram enables pg_trgm extension for fuzzy text matching.
	// Idempotent: safe to call multiple times.
	EnableTrigram(ctx context.Context) error

	// VacuumFull performs VACUUM FULL on large tables to reclaim space.
	// WARNING: Locks table for duration (use after bulk operations only).
	VacuumFull(ctx context.Context, tableName string) error

	// AnalyzeTables updates query planner statistics for specified tables.
	// Run after index creation and bulk imports.
	AnalyzeTables(ctx context.Context, tableNames []string) error

	// GetIndexStats returns index usage statistics for optimization analysis.
	GetIndexStats(ctx context.Context, tableName string) ([]IndexStat, error)

	// GetCacheHitRatio returns the cache hit ratio (should be >95%).
	// Low ratios indicate insufficient shared_buffers or poor query patterns.
	GetCacheHitRatio(ctx context.Context) (float64, error)

	// GetSlowQueries returns queries with execution time above threshold.
	// Requires pg_stat_statements extension enabled.
	GetSlowQueries(ctx context.Context, minDurationMs int64) ([]SlowQuery, error)

	// SetConnectionPoolSize configures the maximum number of database connections.
	// Recommended: 20-30 connections for 1000 queries/sec with PgBouncer.
	SetConnectionPoolSize(maxConnections int)
}

// ColumnStatTarget defines a column and its statistics target.
type ColumnStatTarget struct {
	TableName  string
	ColumnName string
	Target     int // Default: 100, High-cardinality: 1000, Low-cardinality: 10
}

// IndexStat represents index usage statistics.
type IndexStat struct {
	IndexName   string
	TableName   string
	ScanCount   int64 // Number of index scans
	TuplesRead  int64 // Tuples read via index
	TuplesFetch int64 // Tuples fetched via index
	SizeBytes   int64 // Index size in bytes
}

// SlowQuery represents a query with high execution time.
type SlowQuery struct {
	Query        string
	Calls        int64
	TotalTimeMs  float64
	MeanTimeMs   float64
	MinTimeMs    float64
	MaxTimeMs    float64
	StdDevTimeMs float64
}

// OptimizationStats summarizes restructure operation results.
type OptimizationStats struct {
	IndexesCreated      int
	IndexCreationTimeMs int64
	MaterializedViews   int
	ViewRefreshTimeMs   int64
	StatisticsUpdated   int
	VacuumTimeMs        int64
	TotalDurationMs     int64
	DatabaseSizeBytes   int64
	CacheHitRatio       float64
}
