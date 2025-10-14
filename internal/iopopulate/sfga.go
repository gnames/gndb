package iopopulate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnames/gndb/pkg/populate"
	"github.com/sfborg/sflib"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// resolveSFGAFile finds the SFGA file in the parent directory matching the given ID.
// Matches pattern: {4-digit-ID}*.zip or {4-digit-ID}*.sqlite
// Returns error if 0 or 2+ files match.
func resolveSFGAFile(parentDir string, id int) (string, error) {
	// Format ID with leading zeros (4 digits)
	idPattern := fmt.Sprintf("%04d", id)

	// Read directory
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return "", fmt.Errorf("failed to read parent directory %s: %w", parentDir, err)
	}

	// Find matching files
	var matches []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		// Match files starting with the ID pattern
		if strings.HasPrefix(filename, idPattern) {
			// Must be .zip or .sqlite file
			if strings.HasSuffix(filename, ".zip") || strings.HasSuffix(filename, ".sqlite") {
				matches = append(matches, filename)
			}
		}
	}

	// Handle ambiguity
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no files found matching ID %d (pattern %s*.{zip,sqlite}) in %s", id, idPattern, parentDir)
	case 1:
		return filepath.Join(parentDir, matches[0]), nil
	default:
		return "", fmt.Errorf("found %d files matching ID %d in %s: %v - please keep only one version", len(matches), id, parentDir, matches)
	}
}

// fetchSFGA fetches and extracts an SFGA file to the cache directory.
// For local files, it resolves by ID pattern and uses sflib Archive.Fetch.
// For URLs, it constructs the URL and fetches.
// Returns the path to the extracted SQLite file.
func fetchSFGA(ctx context.Context, source populate.DataSourceConfig, cacheDir string) (string, error) {
	var sfgaPath string
	var err error

	// Determine if parent is URL or local directory
	isURL := populate.IsValidURL(source.Parent)

	if isURL {
		// For URLs, construct the file path by appending ID-based filename
		// The actual filename will be discovered by trying common patterns
		// For now, we'll construct a basic pattern and let sflib handle it
		idPattern := fmt.Sprintf("%04d", source.ID)
		sfgaPath = strings.TrimSuffix(source.Parent, "/") + "/" + idPattern + ".sqlite.zip"
	} else {
		// For local directories, resolve the exact filename
		sfgaPath, err = resolveSFGAFile(source.Parent, source.ID)
		if err != nil {
			return "", err
		}
	}

	// Create Archive for fetching
	arc := sflib.NewSfga()

	// Fetch and extract to cache directory
	err = arc.Fetch(sfgaPath, cacheDir)
	if err != nil {
		return "", fmt.Errorf("failed to fetch SFGA from %s: %w", sfgaPath, err)
	}

	// Get the path to the extracted SQLite file
	sqlitePath := arc.DbPath()
	if sqlitePath == "" {
		return "", fmt.Errorf("failed to get database path after fetching %s", sfgaPath)
	}

	return sqlitePath, nil
}

// openSFGA opens a SQLite database and returns a database handle.
func openSFGA(sqlitePath string) (*sql.DB, error) {
	// Check if file exists
	if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("SFGA file does not exist: %s", sqlitePath)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database %s: %w", sqlitePath, err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to SQLite database %s: %w", sqlitePath, err)
	}

	return db, nil
}
