package contracts

import (
	"context"

	"github.com/gnames/gndb/pkg/config"
)

// DatabaseOperator defines the interface for database operations.
type DatabaseOperator interface {
	// Connect establishes a connection to the database.
	Connect(context.Context, *config.DatabaseConfig) error

	// Close closes the database connection.
	Close() error

	// CreateSchema creates the database schema.
	CreateSchema(ctx context.Context, ddlStatements []string, force bool) error

	// TableExists checks if a table exists in the database.
	TableExists(ctx context.Context, tableName string) (bool, error)

	// DropAllTables drops all tables in the public schema.
	DropAllTables(ctx context.Context) error

	// ExecuteDDL executes a single DDL statement.
	ExecuteDDL(ctx context.Context, ddl string) error

	// ExecuteDDLBatch executes a batch of DDL statements.
	ExecuteDDLBatch(ctx context.Context, ddlStatements []string) error

	// GetSchemaVersion returns the current schema version.
	GetSchemaVersion(ctx context.Context) (string, error)

	// SetSchemaVersion sets the current schema version.
	SetSchemaVersion(ctx context.Context, version, description string) error
}
