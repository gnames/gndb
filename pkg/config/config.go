// Package config provides configuration types and validation for GNdb.
// This is a pure package with no I/O dependencies.
package config

import (
	"fmt"
)

// Config represents the complete GNdb configuration.
type Config struct {
	// Database contains PostgreSQL connection settings.
	Database DatabaseConfig `mapstructure:"database"`

	// Import contains settings for SFGA data import operations.
	Import ImportConfig `mapstructure:"import"`

	// Optimization contains settings for database restructure phase.
	Optimization OptimizationConfig `mapstructure:"optimization"`

	// Logging contains application logging settings.
	Logging LoggingConfig `mapstructure:"logging"`
}

// DatabaseConfig contains PostgreSQL connection parameters.
type DatabaseConfig struct {
	// Host is the PostgreSQL server hostname or IP address.
	// Default: "localhost"
	Host string `mapstructure:"host"`

	// Port is the PostgreSQL server port number.
	// Default: 5432
	Port int `mapstructure:"port"`

	// User is the PostgreSQL database username.
	// Default: "postgres"
	User string `mapstructure:"user"`

	// Password is the PostgreSQL database password.
	// Optional: can be empty for trust authentication or set via environment.
	Password string `mapstructure:"password"`

	// Database is the PostgreSQL database name to connect to.
	// Default: "gndb"
	Database string `mapstructure:"database"`

	// SSLMode specifies the SSL connection mode.
	// Valid values: "disable", "require", "verify-ca", "verify-full"
	// Default: "disable"
	SSLMode string `mapstructure:"ssl_mode"`
}

// ImportConfig contains settings for SFGA data import.
type ImportConfig struct {
	// BatchSize defines the number of records to insert per transaction
	// during SFGA import. Applies to all record types.
	// Larger batches are faster but use more memory. Tune based on available RAM.
	// Default: 5000
	BatchSize int `mapstructure:"batch_size"`
}

// OptimizationConfig contains settings for database restructure phase.
type OptimizationConfig struct {
	// ConcurrentIndexes determines whether indexes are created concurrently.
	// - false: Faster index creation but locks tables (recommended for initial setup)
	// - true: Slower but allows reads during index creation (for production)
	// Default: false
	ConcurrentIndexes bool `mapstructure:"concurrent_indexes"`

	// StatisticsTargets sets the statistics target for specific columns.
	// Higher values (e.g., 1000) improve query planning for high-cardinality columns.
	// Map key format: "table.column"
	// Default targets:
	//   - "name_strings.canonical_simple": 1000
	//   - "taxa.rank": 100
	StatisticsTargets map[string]int `mapstructure:"statistics_targets"`
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	// Level is the logging level.
	// Valid values: "debug", "info", "warn", "error"
	// Default: "info"
	Level string `mapstructure:"level"`

	// Format is the log output format.
	// Valid values: "json", "text"
	// Default: "text"
	Format string `mapstructure:"format"`
}

// Validate checks that all required configuration fields are set correctly.
func (c *Config) Validate() error {
	// Validate database connection parameters
	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if c.Database.Port == 0 {
		return fmt.Errorf("database.port is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database.database is required")
	}

	// Validate batch size is positive
	if c.Import.BatchSize <= 0 {
		return fmt.Errorf("import.batch_size must be positive")
	}

	// Validate logging format
	if c.Logging.Format != "" && c.Logging.Format != "json" && c.Logging.Format != "text" {
		return fmt.Errorf("logging.format must be 'json' or 'text'")
	}

	return nil
}

// Defaults returns a Config with sensible default values.
func Defaults() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Database: "gndb",
			SSLMode:  "disable",
		},
		Import: ImportConfig{
			BatchSize: 5000,
		},
		Optimization: OptimizationConfig{
			ConcurrentIndexes: false, // Faster for initial setup, locks tables
			StatisticsTargets: map[string]int{
				"name_strings.canonical_simple": 1000,
				"taxa.rank":                     100,
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}
