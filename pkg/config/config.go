// Package config provides configuration types and validation for GNdb.
// This is a pure package with no I/O dependencies.
package config

import (
	"fmt"
)

// Config represents the complete GNdb configuration.
type Config struct {
	Database     DatabaseConfig     `mapstructure:"database"`
	Import       ImportConfig       `mapstructure:"import"`
	Optimization OptimizationConfig `mapstructure:"optimization"`
	Logging      LoggingConfig      `mapstructure:"logging"`
}

// DatabaseConfig contains PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// ImportConfig contains settings for SFGA data import.
type ImportConfig struct {
	BatchSizes BatchSizes `mapstructure:"batch_sizes"`
}

// BatchSizes defines batch sizes for different record types during import.
type BatchSizes struct {
	Names       int `mapstructure:"names"`
	Taxa        int `mapstructure:"taxa"`
	References  int `mapstructure:"references"`
	Synonyms    int `mapstructure:"synonyms"`
	Vernaculars int `mapstructure:"vernaculars"`
}

// OptimizationConfig contains settings for database restructure phase.
type OptimizationConfig struct {
	ConcurrentIndexes bool           `mapstructure:"concurrent_indexes"`
	StatisticsTargets map[string]int `mapstructure:"statistics_targets"`
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
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

	// Validate batch sizes are positive
	if c.Import.BatchSizes.Names <= 0 {
		return fmt.Errorf("import.batch_sizes.names must be positive")
	}
	if c.Import.BatchSizes.Taxa <= 0 {
		return fmt.Errorf("import.batch_sizes.taxa must be positive")
	}
	if c.Import.BatchSizes.References <= 0 {
		return fmt.Errorf("import.batch_sizes.references must be positive")
	}
	if c.Import.BatchSizes.Synonyms <= 0 {
		return fmt.Errorf("import.batch_sizes.synonyms must be positive")
	}
	if c.Import.BatchSizes.Vernaculars <= 0 {
		return fmt.Errorf("import.batch_sizes.vernaculars must be positive")
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
			BatchSizes: BatchSizes{
				Names:       5000,
				Taxa:        2000,
				References:  1000,
				Synonyms:    3000,
				Vernaculars: 3000,
			},
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
