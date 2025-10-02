package lifecycle

import (
	"context"

	"github.com/gnames/gndb/pkg/config"
)

// SchemaManager defines the interface for database schema management.
// It uses GORM AutoMigrate to handle both initial schema creation and migrations.
// Schema management is idempotent - safe to run multiple times.
type SchemaManager interface {
	// Create creates the initial database schema using GORM AutoMigrate.
	// Also applies collation settings for correct scientific name sorting.
	// If tables already exist, behavior depends on user confirmation via DropAllTables.
	Create(ctx context.Context, cfg *config.Config) error

	// Migrate updates the database schema to the latest version using GORM AutoMigrate.
	// GORM handles schema version tracking automatically.
	Migrate(ctx context.Context, cfg *config.Config) error
}
