package config

import (
	"strings"
)

// Option is a function that modifies a Config.
// Options validate inputs and reject invalid values with warnings.
type Option func(*Config)

// OptDatabaseHost sets the PostgreSQL server hostname or IP address.
func OptDatabaseHost(s string) Option {
	s = strings.TrimSpace(s)
	return func(c *Config) {
		if isValidString("Database Host", s) {
			c.Database.Host = s
		}
	}
}

// OptDatabasePort sets the PostgreSQL server port number.
func OptDatabasePort(i int) Option {
	return func(c *Config) {
		if isValidInt("Database Port", i) {
			c.Database.Port = i
		}
	}
}

// OptDatabaseUser sets the PostgreSQL database username.
func OptDatabaseUser(s string) Option {
	s = strings.TrimSpace(s)
	return func(c *Config) {
		if isValidString("Database User", s) {
			c.Database.User = s
		}
	}
}

// OptDatabasePassword sets the PostgreSQL database password.
func OptDatabasePassword(s string) Option {
	s = strings.TrimSpace(s)
	return func(c *Config) {
		if isValidString("Database Password", s) {
			c.Database.Password = s
		}
	}
}

// OptDatabaseDatabase sets the PostgreSQL database name to connect to.
func OptDatabaseDatabase(s string) Option {
	s = strings.TrimSpace(s)
	return func(c *Config) {
		if isValidString("Database Name", s) {
			c.Database.Database = s
		}
	}
}

// OptDatabaseSSLMode sets the SSL connection mode.
// Valid values: "disable", "require", "verify-ca", "verify-full".
func OptDatabaseSSLMode(s string) Option {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return func(c *Config) {
		if isValidEnum("Database.SSLMode", s) {
			c.Database.SSLMode = s
		}
	}
}

// OptDatabaseBatchSize sets the number of records to process per batch.
// Used for bulk operations in populate and optimize phases.
func OptDatabaseBatchSize(i int) Option {
	return func(c *Config) {
		if isValidInt("Batch Size", i) {
			c.Database.BatchSize = i
		}
	}
}

// OptPopulateSourceIDs sets the list of data source IDs to import.
// Empty slice means import all sources from sources.yaml.
// Runtime-only field - not in ToOptions().
func OptPopulateSourceIDs(ii []int) Option {
	return func(c *Config) {
		if len(ii) > 0 {
			c.Populate.SourceIDs = ii
		}
	}
}

// OptPopulateReleaseVersion overrides the version for a single-source import.
// Only valid when importing one source. CLI validates this constraint.
// Runtime-only field - not in ToOptions().
func OptPopulateReleaseVersion(s string) Option {
	s = strings.TrimSpace(s)
	return func(c *Config) {
		if isValidString("Release Version", s) {
			c.Populate.ReleaseVersion = s
		}
	}
}

// OptPopulateReleaseDate overrides the release date for a single-source import.
// Format: YYYY-MM-DD. Only valid when importing one source.
// Runtime-only field - not in ToOptions().
func OptPopulateReleaseDate(s string) Option {
	s = strings.TrimSpace(s)
	return func(c *Config) {
		if isValidString("Release Date", s) {
			c.Populate.ReleaseDate = s
		}
	}
}

// OptPopulateWithFlatClassification sets whether to prefer flat classification
// over hierarchical parent/child classification from SFGA.
// Uses pointer to distinguish between unset (nil) and false.
// Runtime-only field - not in ToOptions().
func OptPopulateWithFlatClassification(b *bool) Option {
	return func(c *Config) {
		if b != nil {
			c.Populate.WithFlatClassification = b
		}
	}
}

// OptLogLevel sets the logging level.
// Valid values: "debug", "info", "warn", "error".
func OptLogLevel(s string) Option {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return func(c *Config) {
		if isValidEnum("Log.Level", s) {
			c.Log.Level = s
		}
	}
}

// OptLogFormat sets the log output format.
// Valid values: "json", "text", "tint".
func OptLogFormat(s string) Option {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return func(c *Config) {
		if isValidEnum("Log.Format", s) {
			c.Log.Format = s
		}
	}
}

// OptLogDestination sets where logs are written.
// Valid values: "file", "stdin", "stdout".
func OptLogDestination(s string) Option {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return func(c *Config) {
		if isValidEnum("Log.Destination", s) {
			c.Log.Destination = s
		}
	}
}

// OptJobsNumber sets the number of concurrent workers for parallel operations.
// Default is runtime.NumCPU().
func OptJobsNumber(i int) Option {
	return func(c *Config) {
		if isValidInt("Jobs Number", i) {
			c.JobsNumber = i
		}
	}
}

// OptHomeDir sets the home directory for config, cache, and log locations.
// Set once at startup from os.UserHomeDir().
// Runtime-only field - not in ToOptions().
func OptHomeDir(s string) Option {
	s = strings.TrimSpace(s)
	return func(c *Config) {
		if isValidString("Home Directory", s) {
			c.HomeDir = s
		}
	}
}
