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

func TestValidate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		errMsg string
	}{
		{
			name: "missing database host",
			config: &config.Config{
				Database: config.DatabaseConfig{
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
			},
			errMsg: "database.host is required",
		},
		{
			name: "missing database port",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
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
			},
			errMsg: "database.port is required",
		},
		{
			name: "missing database user",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					Database:        "gnames",
					MaxConnections:  20,
					MinConnections:  2,
					MaxConnLifetime: 60,
					MaxConnIdleTime: 10,
				},
				Import: config.ImportConfig{
					BatchSize: 5000,
				},
			},
			errMsg: "database.user is required",
		},
		{
			name: "missing database name",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "postgres",
					MaxConnections:  20,
					MinConnections:  2,
					MaxConnLifetime: 60,
					MaxConnIdleTime: 10,
				},
				Import: config.ImportConfig{
					BatchSize: 5000,
				},
			},
			errMsg: "database.database is required",
		},
		{
			name: "invalid batch size",
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
					BatchSize: 0,
				},
			},
			errMsg: "import.batch_size must be positive",
		},
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
			name: "invalid max_connections zero",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					User:            "postgres",
					Database:        "gnames",
					MaxConnections:  0,
					MinConnections:  2,
					MaxConnLifetime: 60,
					MaxConnIdleTime: 10,
				},
				Import: config.ImportConfig{
					BatchSize: 5000,
				},
			},
			errMsg: "database.max_connections must be at least 1",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
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
