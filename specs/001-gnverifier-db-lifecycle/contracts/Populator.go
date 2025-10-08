package contracts

import (
	"context"

	"github.com/gnames/gndb/pkg/config"
)

// Populator defines the interface for populating the database with SFGA data.
// It uses github.com/sfborg/sflib to read SFGA files and imports data into PostgreSQL.
type Populator interface {
	// Populate imports data from SFGA sources into the database.
	// It reads the sources configuration, connects to SFGA files using sflib,
	// transforms CoLDP data to PostgreSQL schema, and performs batch inserts.
	Populate(ctx context.Context, cfg *config.Config) error
}
