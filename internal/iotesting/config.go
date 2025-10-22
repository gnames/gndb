// Package testing provides shared test utilities for integration tests.
// This is an internal package for test infrastructure only.
package iotesting

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gnames/gndb/internal/ioconfig"
	"github.com/gnames/gndb/pkg/config"
)

const (
	// TestDatabaseName is the database name used for all integration tests.
	// This ensures tests never accidentally run against production databases.
	TestDatabaseName = "gndb_test"
)

// GetTestConfig returns a configuration suitable for integration tests.
// It loads the standard config (from file or defaults) and overrides the
// database name to TestDatabaseName for safety.
//
// Usage in integration tests:
//
//	func TestSomething(t *testing.T) {
//	    if testing.Short() {
//	        t.Skip("Skipping integration test")
//	    }
//	    cfg := testing.GetTestConfig()
//	    // ... use cfg for database operations
//	}
func GetTestConfig() *config.Config {
	// Load config using the standard config system
	result, err := ioconfig.Load("")

	var cfg *config.Config
	if err != nil {
		// No config file found, use defaults
		cfg = config.Defaults()
	} else {
		cfg = result.Config
	}

	// Ensure defaults are merged
	cfg.MergeWithDefaults()

	// Always use test database for safety
	cfg.Database.Database = TestDatabaseName

	return cfg
}

// GetTestDatabaseConfig returns only the database configuration for tests.
// This is useful when you only need database config without the full Config struct.
func GetTestDatabaseConfig() *config.DatabaseConfig {
	cfg := GetTestConfig()
	return &cfg.Database
}

// SetupTempConfigDir creates a temporary config directory for a test and sets
// the GNDB_CONFIG_DIR environment variable to point to it. The directory is
// automatically cleaned up when the test finishes.
//
// This prevents tests from accidentally modifying production config files in
// ~/.config/gndb/. All tests that need to write config/sources files should
// use this function.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    tempConfigDir := iotesting.SetupTempConfigDir(t)
//	    // tempConfigDir is now set as GNDB_CONFIG_DIR
//	    // Write test sources.yaml to tempConfigDir
//	    // Cleanup happens automatically via t.Cleanup()
//	}
//
// Returns the absolute path to the temporary config directory.
func SetupTempConfigDir(t *testing.T) string {
	t.Helper()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "gndb-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}

	// Set environment variable to override GetConfigDir()
	originalConfigDir := os.Getenv("GNDB_CONFIG_DIR")
	err = os.Setenv("GNDB_CONFIG_DIR", tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to set GNDB_CONFIG_DIR: %v", err)
	}

	// Register cleanup to restore original environment and remove temp dir
	t.Cleanup(func() {
		if originalConfigDir != "" {
			os.Setenv("GNDB_CONFIG_DIR", originalConfigDir)
		} else {
			os.Unsetenv("GNDB_CONFIG_DIR")
		}
		os.RemoveAll(tempDir)
	})

	return tempDir
}

// SetupTempCacheDir creates a temporary cache directory for a test and sets
// the GNDB_CACHE_DIR environment variable to point to it. The directory is
// automatically cleaned up when the test finishes.
//
// This prevents tests from accidentally using/polluting production cache in
// ~/.cache/gndb/. Tests that download SFGA files should use this function.
//
// Returns the absolute path to the temporary cache directory.
func SetupTempCacheDir(t *testing.T) string {
	t.Helper()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "gndb-test-cache-*")
	if err != nil {
		t.Fatalf("Failed to create temp cache dir: %v", err)
	}

	// Set environment variable to override GetCacheDir()
	originalCacheDir := os.Getenv("GNDB_CACHE_DIR")
	err = os.Setenv("GNDB_CACHE_DIR", tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to set GNDB_CACHE_DIR: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		if originalCacheDir != "" {
			os.Setenv("GNDB_CACHE_DIR", originalCacheDir)
		} else {
			os.Unsetenv("GNDB_CACHE_DIR")
		}
		os.RemoveAll(tempDir)
	})

	return tempDir
}

// WriteTempSourcesYAML writes a sources.yaml file to the temporary config directory.
// Must be called after SetupTempConfigDir().
//
// Usage:
//
//	tempConfigDir := iotesting.SetupTempConfigDir(t)
//	iotesting.WriteTempSourcesYAML(t, tempConfigDir, `
//	data_sources:
//	  - id: 1000
//	    parent: /path/to/testdata
//	`)
func WriteTempSourcesYAML(t *testing.T, configDir, content string) {
	t.Helper()

	sourcesPath := filepath.Join(configDir, "sources.yaml")
	err := os.WriteFile(sourcesPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp sources.yaml: %v", err)
	}
}
