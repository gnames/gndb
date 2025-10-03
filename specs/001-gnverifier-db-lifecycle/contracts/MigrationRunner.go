package contracts

import (
	"context"

	"github.com/gnames/gndb/pkg/config"
)

// MigrationRunner defines the interface for running database migrations.
type MigrationRunner interface {
	// Migrate runs the database migrations.
	Migrate(ctx context.Context, cfg *config.Config) error
}