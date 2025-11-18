package gndb

import (
	"context"
)

// SchemaManager defines the interface for database schema management.
// It uses GORM AutoMigrate to handle both initial schema creation and migrations.
// Schema management is idempotent - safe to run multiple times.
// Config is provided during construction via NewManager.
type SchemaManager interface {
	// Create creates the initial database schema using GORM AutoMigrate.
	// Also applies collation settings for correct scientific name sorting.
	// If tables already exist, behavior depends on user confirmation
	// via DropAllTables.
	Create(ctx context.Context) error

	// Migrate updates the database schema to the latest version.
	// Drops materialized views before migration (required for ALTER TABLE),
	// runs GORM AutoMigrate, and optionally recreates views.
	Migrate(ctx context.Context, recreateViews bool) error
}
