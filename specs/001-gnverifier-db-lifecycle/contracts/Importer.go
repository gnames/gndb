package contracts

import (
	"context"

	"github.com/gnames/gndb/pkg/config"
)

// Importer defines the interface for importing data into the database.
type Importer interface {
	// Import imports data from the given data source.
	Import(ctx context.Context, cfg *config.Config, ds *DataSource) error
}

// DataSource represents a data source to be imported.
type DataSource struct {
	Path string
}