package db

import (
	"context"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Operator defines the interface for basic database management operations.
// It provides connection lifecycle management and exposes the pgxpool.Pool for
// high-level lifecycle components (SchemaManager, Populator, Optimizer) to execute
// their specialized SQL operations internally.
//
// Design rationale:
// - Keeps interface minimal to avoid bloat with mixed semantics
// - Pool() enables components to use performance-critical features (CopyFrom for bulk inserts)
// - Schema creation uses schema.Migrate; schema updates use Atlas via SchemaManager
type Operator interface {
	// Connect establishes a connection pool to the database.
	Connect(context.Context, *config.DatabaseConfig) error

	// Close closes the database connection pool.
	Close() error

	// Pool returns the underlying pgxpool.Pool for high-level components to execute
	// specialized SQL operations. Components use this for transactions, bulk inserts
	// (CopyFrom), and custom queries.
	Pool() *pgxpool.Pool

	// TableExists checks if a table exists in the database.
	TableExists(ctx context.Context, tableName string) (bool, error)

	// HasTables checks if the database has any tables in the public schema.
	// Used to determine if schema creation should prompt for confirmation.
	HasTables(ctx context.Context) (bool, error)

	// DropAllTables drops all tables in the public schema.
	// Used during schema initialization when overwriting existing data.
	DropAllTables(ctx context.Context) error

	// DropMaterializedViews drops all materialized views in the public schema.
	// Used during migration to allow ALTER TABLE operations on dependent tables.
	DropMaterializedViews(ctx context.Context) error

	// CreateMaterializedViews creates all materialized views for the database.
	// Used after migration and during optimization.
	CreateMaterializedViews(ctx context.Context) error

	// GetDataSources returns DataSource records for the given IDs.
	// If ids is empty, all data sources are returned.
	GetDataSources(ctx context.Context, ids []int) ([]schema.DataSource, error)

	// DeleteDatasets removes all records belonging to the given data source IDs
	// from name_string_indices, vernacular_string_indices, and data_sources.
	// Orphaned name_strings/canonicals are cleaned up by the optimize command.
	DeleteDatasets(ctx context.Context, ids []int) error
}
