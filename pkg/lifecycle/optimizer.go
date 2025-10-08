package lifecycle

import (
	"context"

	"github.com/gnames/gndb/pkg/config"
)

// Optimizer defines the interface for applying performance optimizations to the database.
// Optimization includes creating indexes, materialized views, and denormalized tables
// for fast name verification, vernacular name lookup, and synonym resolution.
//
// Optimization always rebuilds from scratch:
// - Drops existing optimization artifacts (indexes, materialized views)
// - Recreates them with latest algorithms
// - Ensures algorithm improvements are applied even when data hasn't changed
type Optimizer interface {
	// Optimize applies performance optimizations by dropping and recreating
	// all optimization artifacts (indexes, materialized views, denormalized tables).
	Optimize(ctx context.Context, cfg *config.Config) error
}
