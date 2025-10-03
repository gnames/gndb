package contracts

import (
	"context"

	"github.com/gnames/gndb/pkg/config"
)

// Optimizer defines the interface for optimizing the database.
type Optimizer interface {
	// Optimize optimizes the database.
	Optimize(ctx context.Context, cfg *config.Config) error
}
