// Package testing provides shared test utilities for integration tests.
// This is an internal package for test infrastructure only.
package testing

import (
	"github.com/gnames/gndb/internal/io/config"
	pkgconfig "github.com/gnames/gndb/pkg/config"
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
func GetTestConfig() *pkgconfig.Config {
	// Load config using the standard config system
	result, err := config.Load("")

	var cfg *pkgconfig.Config
	if err != nil {
		// No config file found, use defaults
		cfg = pkgconfig.Defaults()
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
func GetTestDatabaseConfig() *pkgconfig.DatabaseConfig {
	cfg := GetTestConfig()
	return &cfg.Database
}
