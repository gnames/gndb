// Package parserpool provides a pool of gnparser instances for concurrent name parsing.
// This is a pure package - parsing is computation, not I/O.
package parserpool

import (
	"fmt"
	"runtime"

	"github.com/gnames/gnlib/ent/nomcode"
	"github.com/gnames/gnparser"
	"github.com/gnames/gnparser/ent/parsed"
)

// Pool provides a pool of gnparser instances for concurrent parsing.
// It maintains separate pools for botanical and zoological nomenclatural codes.
type Pool interface {
	// Parse parses a scientific name string using the specified nomenclatural code.
	// It retrieves a parser from the appropriate pool, parses the name, and returns
	// the parser to the pool. This method is safe for concurrent use.
	Parse(nameString string, code nomcode.Code) (parsed.Parsed, error)

	// Close shuts down the parser pools and releases resources.
	// After calling Close, the pool should not be used.
	Close()
}

// PoolImpl implements the Pool interface using gnparser.NewPool.
type PoolImpl struct {
	botanicalCh  chan gnparser.GNparser
	zoologicalCh chan gnparser.GNparser
	poolSize     int
}

// NewPool creates a new parser pool with the specified number of workers.
// If jobsNum is 0, it defaults to runtime.NumCPU().
// It creates two separate pools: one for botanical and one for zoological parsing.
// Total parsers created = 2 * poolsSize (one pool per nomenclatural code).
func NewPool(jobsNum int) Pool {
	poolSize := jobsNum
	if poolSize == 0 {
		poolSize = runtime.NumCPU()
	}

	// Create botanical parser pool (nomcode.Botanical)
	// WithDetails(true) is required to populate the Words field needed for T025
	botanicalCfg := gnparser.NewConfig(
		gnparser.OptCode(nomcode.Botanical),
		gnparser.OptWithDetails(true),
	)
	botanicalCh := gnparser.NewPool(botanicalCfg, poolSize)

	// Create zoological parser pool (nomcode.Zoological is default)
	// WithDetails(true) is required to populate the Words field needed for T025
	zoologicalCfg := gnparser.NewConfig(
		gnparser.OptCode(nomcode.Zoological),
		gnparser.OptWithDetails(true),
	)
	zoologicalCh := gnparser.NewPool(zoologicalCfg, poolSize)

	return &PoolImpl{
		botanicalCh:  botanicalCh,
		zoologicalCh: zoologicalCh,
		poolSize:     poolSize,
	}
}

// Parse parses a scientific name string using the specified nomenclatural code.
// It selects the appropriate parser pool based on the code, retrieves a parser,
// parses the name, returns the parser to the pool, and returns the parsed result.
func (p *PoolImpl) Parse(nameString string, code nomcode.Code) (parsed.Parsed, error) {
	// Select the appropriate channel based on nomenclatural code
	var ch chan gnparser.GNparser
	switch code {
	case nomcode.Botanical:
		ch = p.botanicalCh
	case nomcode.Zoological:
		ch = p.zoologicalCh
	default:
		return parsed.Parsed{}, fmt.Errorf("unsupported nomenclatural code: %v", code)
	}

	// Get a parser from the pool (blocks if all parsers are busy)
	parser := <-ch

	// Parse the name string
	result := parser.ParseName(nameString)

	// Return the parser to the pool
	ch <- parser

	return result, nil
}

// Close shuts down both parser pools and releases resources.
// It closes the channels and drains any remaining parsers.
func (p *PoolImpl) Close() {
	// Close botanical pool
	if p.botanicalCh != nil {
		close(p.botanicalCh)
		// Drain the channel
		for range p.botanicalCh {
		}
	}

	// Close zoological pool
	if p.zoologicalCh != nil {
		close(p.zoologicalCh)
		// Drain the channel
		for range p.zoologicalCh {
		}
	}
}
