package contracts

import (
	"io"
)

// SFGAReader defines the interface for reading SFGA data.
type SFGAReader interface {
	// Read reads data from the SFGA data source.
	Read(path string) (io.ReadCloser, error)
}