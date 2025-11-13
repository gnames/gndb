package iopopulate

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnames/gndb/pkg/sources"
	"gopkg.in/yaml.v3"
)

// LoadSourcesConfig reads and validates sources.yaml from disk.
// It performs both data structure validation (via sources.SourcesConfig.Validate)
// and file system validation (directory existence checks).
func LoadSourcesConfig(path string) (*sources.SourcesConfig, error) {
	// Read file from disk
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read sources config file: %w", err)
	}

	// Parse YAML
	var config sources.SourcesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse sources config: %w", err)
	}

	// Validate data structure (pure validation)
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Perform file system validation for local directories
	if err := validateSourcesFileSystem(&config); err != nil {
		return nil, err
	}

	// Log configuration warnings
	for _, w := range config.Warnings {
		slog.Warn("Source configuration warning",
			"source_id", w.DataSourceID,
			"field", w.Field,
			"message", w.Message,
			"suggestion", w.Suggestion)
	}

	return &config, nil
}

// validateSourcesFileSystem checks that parent directories exist for local paths.
// This is the I/O layer validation that was removed from pkg/sources validation.
func validateSourcesFileSystem(config *sources.SourcesConfig) error {
	for i, ds := range config.DataSources {
		// Skip URLs - they'll be validated at fetch time
		if sources.IsValidURL(ds.Parent) {
			continue
		}

		// Expand ~ if needed
		parentPath := ds.Parent
		if strings.HasPrefix(parentPath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf(
					"data source %d: failed to expand ~: %w",
					i+1,
					err,
				)
			}
			parentPath = filepath.Join(homeDir, parentPath[2:])
		}

		// Check directory exists
		stat, err := os.Stat(parentPath)
		if os.IsNotExist(err) {
			return fmt.Errorf(
				"data source %d: parent directory does not exist: %s",
				i+1,
				ds.Parent,
			)
		}
		if err != nil {
			return fmt.Errorf(
				"data source %d: failed to check parent directory: %w",
				i+1,
				err,
			)
		}
		if !stat.IsDir() {
			return fmt.Errorf(
				"data source %d: parent path is not a directory: %s",
				i+1,
				ds.Parent,
			)
		}
	}
	return nil
}
