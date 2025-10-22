package config_test

import (
	"testing"

	"github.com/gnames/gndb/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	cfg := config.Defaults()

	// Test database defaults
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "postgres", cfg.Database.User)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, 20, cfg.Database.MaxConnections)
	assert.Equal(t, 2, cfg.Database.MinConnections)
	assert.Equal(t, 60, cfg.Database.MaxConnLifetime)
	assert.Equal(t, 10, cfg.Database.MaxConnIdleTime)

	// Test database batch size default
	assert.Equal(t, 50000, cfg.Database.BatchSize)

	// Test optimization defaults
	assert.False(t, cfg.Optimization.ConcurrentIndexes)
	assert.Equal(t, 1000, cfg.Optimization.StatisticsTargets["name_strings.canonical_simple"])
	assert.Equal(t, 100, cfg.Optimization.StatisticsTargets["taxa.rank"])

	// Test logging defaults
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "tint", cfg.Logging.Format)

	// Test JobsNumber default is set to number of CPUs (> 0)
	assert.Greater(t, cfg.JobsNumber, 0, "JobsNumber should default to number of CPUs")

	// Defaults should be valid
	err := cfg.Validate()
	require.NoError(t, err, "default config should be valid")
}

func TestValidate_InvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		errMsg string
	}{
		{
			name: "invalid logging format",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "postgres",
					Database:        "gnames",
					MaxConnections:  20,
					MinConnections:  2,
					MaxConnLifetime: 60,
					MaxConnIdleTime: 10,
					BatchSize:       50000,
				},
				Logging: config.LoggingConfig{
					Format: "invalid",
				},
			},
			errMsg: "logging.format must be 'tint', 'json' or 'text'",
		},
		{
			name: "invalid logging level",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "postgres",
					Database:        "gnames",
					MaxConnections:  20,
					MinConnections:  2,
					MaxConnLifetime: 60,
					MaxConnIdleTime: 10,
				},
				Logging: config.LoggingConfig{
					Level: "invalid",
				},
			},
			errMsg: "logging.level must be 'debug', 'info', 'warn', or 'error'",
		},
		{
			name: "invalid min_connections exceeds max",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "postgres",
					Database:        "gnames",
					MaxConnections:  10,
					MinConnections:  20,
					MaxConnLifetime: 60,
					MaxConnIdleTime: 10,
				},
			},
			errMsg: "database.min_connections cannot exceed max_connections",
		},
		{
			name: "invalid negative max_conn_lifetime",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "postgres",
					Database:        "gnames",
					MaxConnections:  20,
					MinConnections:  2,
					MaxConnLifetime: -1,
					MaxConnIdleTime: 10,
				},
			},
			errMsg: "database.max_conn_lifetime cannot be negative",
		},
		{
			name: "invalid negative batch_size",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "postgres",
					Database:        "gnames",
					MaxConnections:  20,
					MinConnections:  2,
					MaxConnLifetime: 60,
					MaxConnIdleTime: 10,
					BatchSize:       -100,
				},
			},
			errMsg: "database.batch_size cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestMergeWithDefaults(t *testing.T) {
	tests := []struct {
		name                 string
		config               *config.Config
		checkJobsNumber      bool
		expectedJobsNumber   int
		verifyFieldsManually bool
	}{
		{
			name:            "empty config gets all defaults",
			config:          &config.Config{},
			checkJobsNumber: true,
		},
		{
			name: "partial config gets missing defaults",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host: "custom-host",
					Port: 3333,
				},
			},
			checkJobsNumber: true,
		},
		{
			name: "complete config unchanged",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "custom-host",
					Port:            3333,
					User:            "custom-user",
					Password:        "custom-pass",
					Database:        "custom-db",
					SSLMode:         "require",
					MaxConnections:  50,
					MinConnections:  5,
					MaxConnLifetime: 120,
					MaxConnIdleTime: 30,
				},
				Optimization: config.OptimizationConfig{
					ConcurrentIndexes: true,
				},
				Logging: config.LoggingConfig{
					Level:  "debug",
					Format: "json",
				},
				JobsNumber: 16, // Custom value
			},
			checkJobsNumber:    true,
			expectedJobsNumber: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store original values for custom fields
			originalHost := tt.config.Database.Host
			originalPort := tt.config.Database.Port
			originalUser := tt.config.Database.User
			originalBatchSize := tt.config.Database.BatchSize
			originalLevel := tt.config.Logging.Level
			originalFormat := tt.config.Logging.Format
			originalJobsNumber := tt.config.JobsNumber

			tt.config.MergeWithDefaults()

			// Check that custom values are preserved
			if originalHost != "" {
				assert.Equal(t, originalHost, tt.config.Database.Host)
			} else {
				assert.Equal(t, "localhost", tt.config.Database.Host)
			}

			if originalPort != 0 {
				assert.Equal(t, originalPort, tt.config.Database.Port)
			} else {
				assert.Equal(t, 5432, tt.config.Database.Port)
			}

			if originalUser != "" {
				assert.Equal(t, originalUser, tt.config.Database.User)
			} else {
				assert.Equal(t, "postgres", tt.config.Database.User)
			}

			if originalBatchSize != 0 {
				assert.Equal(t, originalBatchSize, tt.config.Database.BatchSize)
			} else {
				assert.Equal(t, 50000, tt.config.Database.BatchSize)
			}

			if originalLevel != "" {
				assert.Equal(t, originalLevel, tt.config.Logging.Level)
			} else {
				assert.Equal(t, "info", tt.config.Logging.Level)
			}

			if originalFormat != "" {
				assert.Equal(t, originalFormat, tt.config.Logging.Format)
			} else {
				assert.Equal(t, "tint", tt.config.Logging.Format)
			}

			// Check JobsNumber
			if tt.checkJobsNumber {
				if originalJobsNumber != 0 {
					assert.Equal(
						t,
						tt.expectedJobsNumber,
						tt.config.JobsNumber,
						"Custom JobsNumber should be preserved",
					)
				} else {
					assert.Greater(t, tt.config.JobsNumber, 0, "JobsNumber should be set to runtime.NumCPU() when 0")
				}
			}
		})
	}
}

func TestValidate_CompleteConfig(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "postgres",
			Password:        "secret",
			Database:        "gndb_test",
			SSLMode:         "require",
			MaxConnections:  50,
			MinConnections:  5,
			MaxConnLifetime: 120,
			MaxConnIdleTime: 30,
		},
		Optimization: config.OptimizationConfig{
			ConcurrentIndexes: true,
			StatisticsTargets: map[string]int{
				"name_strings.canonical_simple": 1000,
			},
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err, "complete config should be valid")
}
