package gndb

import (
	"context"
)

// MigrateOptions configures the behavior of schema migration.
type MigrateOptions struct {
	// RecreateViews triggers recreation of materialized views after migration.
	RecreateViews bool

	// DryRun shows planned changes without applying them.
	DryRun bool

	// Confirm is called with the list of SQL statements to be applied.
	// It should display the statements to the user and return true if
	// migration should proceed. Returning false cancels the migration.
	Confirm func(stmts []string) bool
}

// SchemaManager defines the interface for database schema management.
// Config is provided during construction via NewManager.
type SchemaManager interface {
	// Create creates the initial database schema from the GORM models.
	// Also applies collation settings for correct scientific name sorting.
	Create(ctx context.Context) error

	// Migrate updates the database schema to match the current model state
	// using Atlas declarative migrations. It inspects the current database,
	// computes a diff against the desired schema, shows the planned SQL to
	// the user via opts.Confirm, and applies changes on approval.
	Migrate(ctx context.Context, opts MigrateOptions) error
}
