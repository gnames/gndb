// Package config provides configuration management for GNdb.
//
// This package has no I/O dependencies (no file operations, no network calls).
// Validation functions may write user-facing warnings via gn.Warn().
//
// # Configuration Sources
//
// Precedence (highest to lowest): CLI flags > env vars > config.yaml > defaults
//
// # Design Principles
//
// - Default config (from New()) is always valid - no validation needed
// - All mutations go through Option functions - the only way to modify Config
// - Invalid options are rejected with gn.Warn() - config remains in valid state
// - ToOptions() converts persistent fields (those in config.yaml)
// - Environment variables match ToOptions() fields exactly
//
// # Persistent vs Runtime Fields
//
// Persistent fields (in ToOptions, config.yaml, and env vars):
//   - Database: host, port, user, password, database, ssl_mode, batch_size
//   - Log: level, format, destination
//   - General: jobs_number
//
// Runtime-only fields (CLI flags only):
//   - Populate.SourceIDs, ReleaseVersion, ReleaseDate, WithFlatClassification
//     (per-command)
//   - HomeDir (set once at startup)
//
// # Environment Variables
//
// Use GNDB_ prefix with underscores for nesting:
//
//	GNDB_DATABASE_HOST=localhost
//	GNDB_DATABASE_PORT=5432
//	GNDB_LOG_LEVEL=info
//	GNDB_JOBS_NUMBER=8
//
// See .envrc.example for complete list with defaults.
package config

import (
	"runtime"
)

// Config represents the complete GNdb configuration.
type Config struct {
	// Database contains PostgreSQL connection settings.
	Database DatabaseConfig `mapstructure:"database" yaml:"database"`

	// Populate contains settings specific to the populate command.
	Populate PopulateConfig `mapstructure:"populate" yaml:"populate"`

	Log LogConfig `mapstructure:"log" yaml:"log"`

	// JobsNumber is the number of concurrent workers for parallel operations.
	// Default value is set accoring to the number of available threads.
	JobsNumber int `mapstructure:"jobs_number" yaml:"jobs_number"`

	// HomeDir determines where config, cache and logs directories reside.
	// It must be set by CLI during init, there is no default value for it.
	HomeDir string
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

	// BatchSize defines the number of records to process per batch for bulk operations.
	// Used in both populate (data import) and optimize (word extraction) phases.
	// Larger batches are faster but use more memory. Tune based on available RAM.
	// Typical values: 5000-100000 depending on record size and available memory.
	BatchSize int `mapstructure:"batch_size" yaml:"batch_size"`
}

// PopulateConfig contains settings specific to the populate command.
type PopulateConfig struct {
	// SourceIDs is the list of data source IDs to import.
	// Empty slice means import all sources from sources.yaml.
	// The CLI filters sources and passes only the IDs to process.
	// Populate() will load sources.yaml and look up each source by ID.
	SourceIDs []int `mapstructure:"source_ids" yaml:"source_ids"`

	// ReleaseVersion overrides the version for the data source being imported.
	// Only valid when importing a single source (len(SourceIDs) == 1).
	// The CLI validates this constraint before calling Populate().
	ReleaseVersion string `mapstructure:"release_version" yaml:"release_version"`

	// ReleaseDate overrides the release date for the data source being imported.
	// Format: YYYY-MM-DD. Only valid when importing a single source (len(SourceIDs) == 1).
	// The CLI validates this constraint before calling Populate().
	ReleaseDate string `mapstructure:"release_date" yaml:"release_date"`

	// WithFlatClassification is true if the 'flat' version of classification
	// is preferable instead of parent/child hierarchical classification.
	// Note: If flat classification does not exist in SFGA, classification breadcrumbs
	// will stay empty even if parent/child hierarchy exists.
	// Default: false (hierarchical parent/child classification is preferred)
	WithFlatClassification *bool `mapstructure:"with_flat_classification" yaml:"with_flat_classification"`
}

// LogConfig provides typical settings for application logs.
type LogConfig struct {
	// Format can be 'json', 'text' or 'tint' (user-facing and colored).
	Format string `mapstructure:"format"      yaml:"format"`
	// Level of logging -- 'error', 'warn', 'info', 'debug'
	Level string `mapstructure:"level"       yaml:"level"`
	// Destination can be a log file (to default place), STDERR or STDOUT
	Destination string `mapstructure:"destination" yaml:"destination"`
}

// New creates a Config with sensible default values.
// The returned config is always valid and ready to use.
// Default values can be overridden using Option functions via Update().
func New() *Config {
	res := &Config{
		Database: DatabaseConfig{
			Host:      "localhost",
			Port:      5432,
			User:      "postgres",
			Password:  "postgres",
			Database:  "gnames",
			SSLMode:   "disable",
			BatchSize: 50_000, // Batch size for bulk operations (populate, optimize)
		},
		Log: LogConfig{
			Format: "json",
			Level:  "info",
			// for now file is rewritten every time the log starts
			Destination: "file",
		},
		JobsNumber: runtime.NumCPU(), // Default to number of CPU threads
	}

	return res
}
