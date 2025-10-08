// Package populate implements Populator interface for importing SFGA data into PostgreSQL.
// This is an impure I/O package that reads SFGA files and performs bulk inserts.
package populate

import (
	"context"
	"fmt"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/database"
	"github.com/gnames/gndb/pkg/lifecycle"
)

// PopulatorImpl implements the Populator interface.
type PopulatorImpl struct {
	operator database.Operator
}

// NewPopulator creates a new Populator.
func NewPopulator(op database.Operator) lifecycle.Populator {
	return &PopulatorImpl{operator: op}
}

// Populate imports data from SFGA sources into the database.
// This is a stub implementation that returns "not implemented" error.
func (p *PopulatorImpl) Populate(ctx context.Context, cfg *config.Config) error {
	pool := p.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	// TODO: Implement population logic
	// This will include:
	// 1. Read sources.yaml configuration
	// 2. Open SFGA files using sflib
	// 3. Parse CoLDP data from SFGA
	// 4. Transform to PostgreSQL schema (data-model.md)
	// 5. Bulk insert using pgx CopyFrom for performance
	// 6. Log progress for long-running imports

	return fmt.Errorf("population not yet implemented")
}
