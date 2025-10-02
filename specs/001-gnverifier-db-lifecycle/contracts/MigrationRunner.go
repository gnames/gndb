// Package contracts defines interfaces for database migration operations.
package contracts

import (
	"context"
)

// MigrationRunner defines operations for managing database schema migrations.
// Implemented by: internal/io/database/migrations.go (wraps Atlas SDK)
// Used by: pkg/migrate
type MigrationRunner interface {
	// GetCurrentVersion returns the current schema version applied to the database.
	// Returns empty string if no migrations have been applied yet.
	GetCurrentVersion(ctx context.Context) (string, error)

	// GetPendingMigrations returns a list of migrations not yet applied.
	// Migrations are identified by version (timestamp prefix).
	GetPendingMigrations(ctx context.Context) ([]Migration, error)

	// ApplyMigration applies a single migration to the database.
	// Runs in a transaction; rolls back on error.
	// Updates schema_versions table with new version.
	ApplyMigration(ctx context.Context, migration Migration) error

	// ApplyAll applies all pending migrations in order.
	// Stops on first error; does not roll back previous successful migrations.
	ApplyAll(ctx context.Context) (*MigrationResult, error)

	// Rollback reverts the last N migrations.
	// WARNING: Not all migrations are reversible (e.g., DROP COLUMN).
	// Returns error if down migrations are not available.
	Rollback(ctx context.Context, steps int) error

	// ValidateMigrations checks integrity of migration files using atlas.sum.
	// Returns error if checksums don't match (migration files tampered).
	ValidateMigrations(ctx context.Context) error

	// GenerateMigration creates a new migration file based on schema diff.
	// Compares current database schema to desired schema (from Go models).
	// Writes SQL to migrations/{timestamp}_{name}.sql.
	GenerateMigration(ctx context.Context, name string, models []interface{}) (string, error)

	// DryRun simulates migration application without executing SQL.
	// Returns SQL statements that would be executed.
	DryRun(ctx context.Context) ([]string, error)
}

// Migration represents a single schema migration.
type Migration struct {
	Version     string // Timestamp prefix (e.g., "20251002120000")
	Name        string // Descriptive name (e.g., "add_cardinality_index")
	Description string
	UpSQL       string // Forward migration SQL
	DownSQL     string // Rollback SQL (optional)
	AppliedAt   string // ISO 8601 timestamp (empty if not applied)
}

// MigrationResult summarizes the result of applying migrations.
type MigrationResult struct {
	Applied        []Migration // Successfully applied migrations
	Failed         *Migration  // Migration that caused failure (nil if all succeeded)
	Error          error       // Error that caused failure
	CurrentVersion string      // Schema version after migrations
}
