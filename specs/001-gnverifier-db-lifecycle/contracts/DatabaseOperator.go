// Package contracts defines interfaces for database lifecycle operations.
// These interfaces separate pure business logic (pkg/) from impure I/O (internal/io/).
package contracts

import (
	"context"
)

// DatabaseOperator defines operations for PostgreSQL database lifecycle management.
// Implemented by: internal/io/database/operator.go
// Used by: pkg/schema, pkg/migrate, pkg/populate, pkg/restructure
type DatabaseOperator interface {
	// Connect establishes a connection pool to PostgreSQL.
	// Returns error if connection fails or database is unreachable.
	Connect(ctx context.Context, dsn string) error

	// Close releases all database connections.
	Close() error

	// CreateSchema creates all base tables from DDL definitions.
	// Does NOT create indexes (indexes added during restructure phase).
	// Returns error if tables already exist (unless force=true).
	CreateSchema(ctx context.Context, ddlStatements []string, force bool) error

	// TableExists checks if a table exists in the current database.
	TableExists(ctx context.Context, tableName string) (bool, error)

	// DropAllTables drops all tables in the public schema (destructive operation).
	// Used when force=true in CreateSchema.
	DropAllTables(ctx context.Context) error

	// ExecuteDDL executes a single DDL statement (CREATE, ALTER, DROP).
	// Runs in a transaction; rolls back on error.
	ExecuteDDL(ctx context.Context, ddl string) error

	// ExecuteDDLBatch executes multiple DDL statements in a single transaction.
	// All-or-nothing: rolls back all on first error.
	ExecuteDDLBatch(ctx context.Context, ddlStatements []string) error

	// GetSchemaVersion returns the current schema version from schema_versions table.
	// Returns empty string if table doesn't exist or has no rows.
	GetSchemaVersion(ctx context.Context) (string, error)

	// SetSchemaVersion inserts or updates the current schema version.
	SetSchemaVersion(ctx context.Context, version, description string) error

	// EnableExtension enables a PostgreSQL extension (e.g., pg_trgm).
	// Idempotent: safe to call multiple times.
	EnableExtension(ctx context.Context, extensionName string) error

	// VacuumAnalyze runs VACUUM ANALYZE on specified tables.
	// Used after bulk imports and index creation.
	VacuumAnalyze(ctx context.Context, tableNames []string) error

	// CreateIndexConcurrently creates an index without blocking writes.
	// Safer for production but slower than regular CREATE INDEX.
	CreateIndexConcurrently(ctx context.Context, indexDDL string) error

	// RefreshMaterializedView refreshes a materialized view.
	// Use concurrently=true to avoid blocking reads during refresh.
	RefreshMaterializedView(ctx context.Context, viewName string, concurrently bool) error

	// SetStatisticsTarget sets the statistics target for a column.
	// Higher values (e.g., 1000) improve query planning for high-cardinality columns.
	SetStatisticsTarget(ctx context.Context, tableName, columnName string, target int) error

	// GetDatabaseSize returns the total size of the database in bytes.
	GetDatabaseSize(ctx context.Context) (int64, error)

	// GetTableSize returns the total size of a table (including indexes) in bytes.
	GetTableSize(ctx context.Context, tableName string) (int64, error)
}

// DDLGenerator defines how Go models generate PostgreSQL DDL.
// Implemented by: pkg/schema models (DataSource, NameString, etc.)
type DDLGenerator interface {
	// TableDDL returns the CREATE TABLE statement for this model.
	TableDDL() string

	// IndexDDL returns CREATE INDEX statements for this model (restructure phase).
	// Returns empty slice if no indexes needed.
	IndexDDL() []string

	// TableName returns the PostgreSQL table name for this model.
	TableName() string
}
