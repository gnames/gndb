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

	// Test import batch size default
	assert.Equal(t, 5000, cfg.Import.BatchSize)

	// Test optimization defaults
	assert.False(t, cfg.Optimization.ConcurrentIndexes)
	assert.Equal(t, 1000, cfg.Optimization.StatisticsTargets["name_strings.canonical_simple"])
	assert.Equal(t, 100, cfg.Optimization.StatisticsTargets["taxa.rank"])

	// Test logging defaults
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "text", cfg.Logging.Format)

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
				},
				Import: config.ImportConfig{
					BatchSize: 5000,
				},
				Logging: config.LoggingConfig{
					Format: "invalid",
				},
			},
			errMsg: "logging.format must be 'json' or 'text'",
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
				Import: config.ImportConfig{
					BatchSize: 5000,
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
				Import: config.ImportConfig{
					BatchSize: 5000,
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
				Import: config.ImportConfig{
					BatchSize: 5000,
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
				},
				Import: config.ImportConfig{
					BatchSize: -100,
				},
			},
			errMsg: "import.batch_size cannot be negative",
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
		name     string
		config   *config.Config
		expected *config.Config
	}{
		{
			name:   "empty config gets all defaults",
			config: &config.Config{},
			expected: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "postgres",
					Password:        "postgres",
					Database:        "gnames",
					SSLMode:         "disable",
					MaxConnections:  20,
					MinConnections:  2,
					MaxConnLifetime: 60,
					MaxConnIdleTime: 10,
				},
				Import: config.ImportConfig{
					BatchSize: 5000,
				},
				Optimization: config.OptimizationConfig{
					ConcurrentIndexes: false,
				},
				Logging: config.LoggingConfig{
					Level:  "info",
					Format: "text",
				},
			},
		},
		{
			name: "partial config gets missing defaults",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host: "custom-host",
					Port: 3333,
				},
			},
			expected: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "custom-host",
					Port:            3333,
					User:            "postgres",
					Password:        "postgres",
					Database:        "gnames",
					SSLMode:         "disable",
					MaxConnections:  20,
					MinConnections:  2,
					MaxConnLifetime: 60,
					MaxConnIdleTime: 10,
				},
				Import: config.ImportConfig{
					BatchSize: 5000,
				},
				Optimization: config.OptimizationConfig{
					ConcurrentIndexes: false,
				},
				Logging: config.LoggingConfig{
					Level:  "info",
					Format: "text",
				},
			},
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
				Import: config.ImportConfig{
					BatchSize: 10000,
				},
				Optimization: config.OptimizationConfig{
					ConcurrentIndexes: true,
				},
				Logging: config.LoggingConfig{
					Level:  "debug",
					Format: "json",
				},
			},
			expected: &config.Config{
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
				Import: config.ImportConfig{
					BatchSize: 10000,
				},
				Optimization: config.OptimizationConfig{
					ConcurrentIndexes: true,
				},
				Logging: config.LoggingConfig{
					Level:  "debug",
					Format: "json",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.MergeWithDefaults()
			assert.Equal(t, tt.expected, tt.config)
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
		Import: config.ImportConfig{
			BatchSize: 5000,
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
