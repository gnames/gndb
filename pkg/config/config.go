// Package config provides configuration types and validation for GNdb.
// This is a pure package with no I/O dependencies.
//
// # Configuration Loading
//
// Configuration is loaded with the following precedence (highest to lowest):
//  1. CLI flags (--host, --port, etc.)
//  2. Environment variables (GNDB_*)
//  3. Config file (gndb.yaml)
//  4. Defaults
//
// # Environment Variables
//
// All configuration fields can be overridden using environment variables with the GNDB_ prefix.
// Nested fields use underscores instead of dots.
//
// Database configuration:
//
//	GNDB_DATABASE_HOST              - PostgreSQL host (string, default: "localhost")
//	GNDB_DATABASE_PORT              - PostgreSQL port (int, default: 5432)
//	GNDB_DATABASE_USER              - PostgreSQL user (string, default: "postgres")
//	GNDB_DATABASE_PASSWORD          - PostgreSQL password (string, default: "postgres")
//	GNDB_DATABASE_DATABASE          - PostgreSQL database name (string, default: "gnames")
//	GNDB_DATABASE_SSL_MODE          - SSL mode: disable/require/verify-ca/verify-full (string, default: "disable")
//	GNDB_DATABASE_MAX_CONNECTIONS   - Maximum connections in pool (int, default: 20)
//	GNDB_DATABASE_MIN_CONNECTIONS   - Minimum connections in pool (int, default: 2)
//	GNDB_DATABASE_MAX_CONN_LIFETIME - Max connection lifetime in minutes (int, default: 60)
//	GNDB_DATABASE_MAX_CONN_IDLE_TIME - Max connection idle time in minutes (int, default: 10)
//
// Import configuration:
//
//	GNDB_IMPORT_BATCH_SIZE          - Batch size for imports (int, default: 5000)
//
// Optimization configuration:
//
//	GNDB_OPTIMIZATION_CONCURRENT_INDEXES - Create indexes concurrently (bool, default: false)
//
// Logging configuration:
//
//	GNDB_LOGGING_LEVEL              - Log level: debug/info/warn/error (string, default: "info")
//	GNDB_LOGGING_FORMAT             - Log format: json/text (string, default: "text")
//
// # Example Usage
//
//	# Override database connection via environment variables
//	export GNDB_DATABASE_HOST=prod-db.example.com
//	export GNDB_DATABASE_PASSWORD=secret123
//	gndb create
//
//	# CLI flags still take highest precedence
//	gndb create --host=override-db.example.com  # Uses override-db, not prod-db
package config

import (
	"fmt"
)

// Config represents the complete GNdb configuration.
type Config struct {
	// Database contains PostgreSQL connection settings.
	Database DatabaseConfig `mapstructure:"database" yaml:"database"`

	// Import contains settings for SFGA data import operations.
	Import ImportConfig `mapstructure:"import" yaml:"import"`

	// Optimization contains settings for database restructure phase.
	Optimization OptimizationConfig `mapstructure:"optimization" yaml:"optimization"`

	// Logging contains application logging settings.
	Logging LoggingConfig `mapstructure:"logging" yaml:"logging"`
}

// DatabaseConfig contains PostgreSQL connection parameters.
type DatabaseConfig struct {
	// Host is the PostgreSQL server hostname or IP address.
	Host string `mapstructure:"host" yaml:"host"`

	// Port is the PostgreSQL server port number.
	Port int `mapstructure:"port" yaml:"port"`

	// User is the PostgreSQL database username.
	User string `mapstructure:"user" yaml:"user"`

	// Password is the PostgreSQL database password.
	Password string `mapstructure:"password" yaml:"password"`

	// Database is the PostgreSQL database name to connect to.
	Database string `mapstructure:"database" yaml:"database"`

	// SSLMode specifies the SSL connection mode.
	// Valid values: "disable", "require", "verify-ca", "verify-full"
	SSLMode string `mapstructure:"ssl_mode" yaml:"ssl_mode"`

	// MaxConnections is the maximum number of connections in the pgxpool.
	// Used for concurrent data import operations with multiple goroutines.
	// Higher values enable more parallelism but consume more database resources.
	MaxConnections int `mapstructure:"max_connections" yaml:"max_connections"`

	// MinConnections is the minimum number of connections maintained in the pool.
	// Keeping connections warm reduces latency for new operations.
	MinConnections int `mapstructure:"min_connections" yaml:"min_connections"`

	// MaxConnLifetime is the maximum duration (in minutes) a connection can be reused.
	// After this time, connections are closed and recreated to prevent stale connections.
	// Set to 0 for unlimited lifetime.
	MaxConnLifetime int `mapstructure:"max_conn_lifetime" yaml:"max_conn_lifetime"`

	// MaxConnIdleTime is the maximum duration (in minutes) a connection can be idle.
	// Idle connections beyond this time are closed to free resources.
	MaxConnIdleTime int `mapstructure:"max_conn_idle_time" yaml:"max_conn_idle_time"`
}

// ImportConfig contains settings for SFGA data import.
type ImportConfig struct {
	// BatchSize defines the number of records to insert per transaction
	// during SFGA import. Applies to all record types.
	// Larger batches are faster but use more memory. Tune based on available RAM.
	BatchSize int `mapstructure:"batch_size" yaml:"batch_size"`
}

// OptimizationConfig contains settings for database restructure phase.
type OptimizationConfig struct {
	// ConcurrentIndexes determines whether indexes are created concurrently.
	// - false: Faster index creation but locks tables (recommended for initial setup)
	// - true: Slower but allows reads during index creation (for production)
	ConcurrentIndexes bool `mapstructure:"concurrent_indexes" yaml:"concurrent_indexes"`

	// StatisticsTargets sets the statistics target for specific columns.
	// Higher values (e.g., 1000) improve query planning for high-cardinality columns.
	// Map key format: "table.column"
	StatisticsTargets map[string]int `mapstructure:"statistics_targets" yaml:"statistics_targets"`
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	// Level is the logging level.
	// Valid values: "debug", "info", "warn", "error"
	Level string `mapstructure:"level" yaml:"level"`

	// Format is the log output format.
	// Valid values: "json", "text"
	Format string `mapstructure:"format" yaml:"format"`
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

	// Validate connection pool settings
	if c.Database.MaxConnections < 1 {
		return fmt.Errorf("database.max_connections must be at least 1")
	}
	if c.Database.MinConnections < 0 {
		return fmt.Errorf("database.min_connections cannot be negative")
	}
	if c.Database.MinConnections > c.Database.MaxConnections {
		return fmt.Errorf("database.min_connections cannot exceed max_connections")
	}
	if c.Database.MaxConnLifetime < 0 {
		return fmt.Errorf("database.max_conn_lifetime cannot be negative")
	}
	if c.Database.MaxConnIdleTime < 0 {
		return fmt.Errorf("database.max_conn_idle_time cannot be negative")
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
			Host:            "localhost",
			Port:            5432,
			User:            "postgres",
			Password:        "postgres",
			Database:        "gnames",
			SSLMode:         "disable",
			MaxConnections:  20, // Allows 20 concurrent workers for import
			MinConnections:  2,  // Keep 2 connections warm
			MaxConnLifetime: 60, // 1 hour in minutes
			MaxConnIdleTime: 10, // 10 minutes
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
