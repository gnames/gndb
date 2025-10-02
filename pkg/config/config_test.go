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

	// Test import batch size defaults
	assert.Equal(t, 5000, cfg.Import.BatchSizes.Names)
	assert.Equal(t, 2000, cfg.Import.BatchSizes.Taxa)
	assert.Equal(t, 1000, cfg.Import.BatchSizes.References)
	assert.Equal(t, 3000, cfg.Import.BatchSizes.Synonyms)
	assert.Equal(t, 3000, cfg.Import.BatchSizes.Vernaculars)

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
					Port:     5432,
					User:     "postgres",
					Database: "gndb",
				},
				Import: config.ImportConfig{
					BatchSizes: config.BatchSizes{
						Names:       5000,
						Taxa:        2000,
						References:  1000,
						Synonyms:    3000,
						Vernaculars: 3000,
					},
				},
			},
			errMsg: "database.host is required",
		},
		{
			name: "missing database port",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:     "localhost",
					User:     "postgres",
					Database: "gndb",
				},
				Import: config.ImportConfig{
					BatchSizes: config.BatchSizes{
						Names:       5000,
						Taxa:        2000,
						References:  1000,
						Synonyms:    3000,
						Vernaculars: 3000,
					},
				},
			},
			errMsg: "database.port is required",
		},
		{
			name: "missing database user",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "gndb",
				},
				Import: config.ImportConfig{
					BatchSizes: config.BatchSizes{
						Names:       5000,
						Taxa:        2000,
						References:  1000,
						Synonyms:    3000,
						Vernaculars: 3000,
					},
				},
			},
			errMsg: "database.user is required",
		},
		{
			name: "missing database name",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host: "localhost",
					Port: 5432,
					User: "postgres",
				},
				Import: config.ImportConfig{
					BatchSizes: config.BatchSizes{
						Names:       5000,
						Taxa:        2000,
						References:  1000,
						Synonyms:    3000,
						Vernaculars: 3000,
					},
				},
			},
			errMsg: "database.database is required",
		},
		{
			name: "invalid batch size - names",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Database: "gndb",
				},
				Import: config.ImportConfig{
					BatchSizes: config.BatchSizes{
						Names:       0,
						Taxa:        2000,
						References:  1000,
						Synonyms:    3000,
						Vernaculars: 3000,
					},
				},
			},
			errMsg: "import.batch_sizes.names must be positive",
		},
		{
			name: "invalid logging format",
			config: &config.Config{
				Database: config.DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Database: "gndb",
				},
				Import: config.ImportConfig{
					BatchSizes: config.BatchSizes{
						Names:       5000,
						Taxa:        2000,
						References:  1000,
						Synonyms:    3000,
						Vernaculars: 3000,
					},
				},
				Logging: config.LoggingConfig{
					Format: "invalid",
				},
			},
			errMsg: "logging.format must be 'json' or 'text'",
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
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "secret",
			Database: "gndb_test",
			SSLMode:  "require",
		},
		Import: config.ImportConfig{
			BatchSizes: config.BatchSizes{
				Names:       5000,
				Taxa:        2000,
				References:  1000,
				Synonyms:    3000,
				Vernaculars: 3000,
			},
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
