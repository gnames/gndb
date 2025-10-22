// Package populate implements cache management for SFGA file handling.
package iopopulate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gnames/gndb/internal/ioconfig"
)

// clearCache removes all files from the cache directory and ensures it exists.
// If the directory doesn't exist, it creates it.
func clearCache(cacheDir string) error {
	// Remove the entire cache directory if it exists
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	// Recreate the cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	return nil
}

// prepareCacheDir returns the SFGA cache directory path and ensures it's empty.
// Cache location: ~/.cache/gndb/sfga/ (all platforms)
//
// Cache lifecycle:
//   - Initial clear at start of population process
//   - Additional clear before fetching each source (prevents "too many database files" error)
//   - Cache always contains the most recently processed source
//   - Useful for debugging failed imports (inspect cached SFGA files)
func prepareCacheDir() (string, error) {
	// Get base cache directory from config
	baseCache, err := ioconfig.GetCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Append "sfga" subdirectory
	sfgaCache := filepath.Join(baseCache, "sfga")

	// Clear cache directory
	if err := clearCache(sfgaCache); err != nil {
		return "", fmt.Errorf("failed to clear cache: %w", err)
	}

	return sfgaCache, nil
}
