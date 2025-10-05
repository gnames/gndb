package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gnames/gndb/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Unset all GNDB_ environment variables to ensure a clean test environment,
	// then restore them after the test suite finishes.
	gndbVars := make(map[string]string)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GNDB_") {
			parts := strings.SplitN(env, "=", 2)
			gndbVars[parts[0]] = parts[1]
			os.Unsetenv(parts[0])
		}
	}
	t.Cleanup(func() {
		for key, val := range gndbVars {
			os.Setenv(key, val)
		}
	})

	baseConfigContent := `
database:
  host: config-host
  port: 5432
  user: config-user
  password: config-pass
  database: config-db
  ssl_mode: disable
  max_connections: 20
  min_connections: 2
  max_conn_lifetime: 60
  max_conn_idle_time: 10
import:
  batch_size: 5000
optimization:
  concurrent_indexes: false
logging:
  level: info
  format: text
`

	testCases := []struct {
		name             string
		configContent    string
		envVars          map[string]string
		expectedConfig   func() *config.Config
		expectedSource   string
		expectSourcePath bool
	}{
		{
			name:          "env vars override config file",
			configContent: baseConfigContent,
			envVars: map[string]string{
				"GNDB_DATABASE_HOST":                   "env-host",
				"GNDB_DATABASE_PORT":                   "5433",
				"GNDB_DATABASE_USER":                   "env-user",
				"GNDB_DATABASE_PASSWORD":               "env-pass",
				"GNDB_DATABASE_DATABASE":               "env-db",
				"GNDB_DATABASE_SSL_MODE":               "require",
				"GNDB_DATABASE_MAX_CONNECTIONS":        "50",
				"GNDB_DATABASE_MIN_CONNECTIONS":        "5",
				"GNDB_IMPORT_BATCH_SIZE":               "10000",
				"GNDB_OPTIMIZATION_CONCURRENT_INDEXES": "true",
				"GNDB_LOGGING_LEVEL":                   "debug",
				"GNDB_LOGGING_FORMAT":                  "json",
			},
			expectedConfig: func() *config.Config {
				cfg := config.Defaults()
				cfg.Database.Host = "env-host"
				cfg.Database.Port = 5433
				cfg.Database.User = "env-user"
				cfg.Database.Password = "env-pass"
				cfg.Database.Database = "env-db"
				cfg.Database.SSLMode = "require"
				cfg.Database.MaxConnections = 50
				cfg.Database.MinConnections = 5
				cfg.Import.BatchSize = 10000
				cfg.Optimization.ConcurrentIndexes = true
				cfg.Optimization.StatisticsTargets = nil
				cfg.Logging.Level = "debug"
				cfg.Logging.Format = "json"
				return cfg
			},
			expectedSource:   "file",
			expectSourcePath: true,
		},
		{
			name:          "no config file, env vars only",
			configContent: "", // No config file
			envVars: map[string]string{
				"GNDB_DATABASE_HOST":     "env-only-host",
				"GNDB_DATABASE_USER":     "testuser",
				"GNDB_IMPORT_BATCH_SIZE": "8000",
			},
			expectedConfig: func() *config.Config {
				cfg := config.Defaults()
				cfg.Database.Host = "env-only-host"
				cfg.Database.User = "testuser"
				cfg.Import.BatchSize = 8000
				cfg.Optimization.StatisticsTargets = nil
				return cfg
			},
			expectedSource:   "defaults+env",
			expectSourcePath: false,
		},
		{
			name:          "config file only, no env vars",
			configContent: baseConfigContent,
			envVars:       nil,
			expectedConfig: func() *config.Config {
				cfg := config.Defaults()
				cfg.Database.Host = "config-host"
				cfg.Database.User = "config-user"
				cfg.Database.Password = "config-pass"
				cfg.Database.Database = "config-db"
				cfg.Optimization.StatisticsTargets = nil
				return cfg
			},
			expectedSource:   "file",
			expectSourcePath: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := ""

			if tc.configContent != "" {
				configPath = filepath.Join(tempDir, "gndb.yaml")
				err := os.WriteFile(configPath, []byte(tc.configContent), 0644)
				require.NoError(t, err)
			} else {
				// Ensure no config file is found in default locations
				t.Setenv("HOME", tempDir)
			}

			for key, value := range tc.envVars {
				t.Setenv(key, value)
			}

			// Load config
			result, err := Load(configPath)
			require.NoError(t, err)

			// Get expected config
			expected := tc.expectedConfig()

			// Verify config values
			assert.Equal(t, expected, result.Config, "Loaded config should match expected config")

			// Verify source metadata
			assert.Equal(t, tc.expectedSource, result.Source, "Source should match")
			if tc.expectSourcePath {
				assert.NotEmpty(t, result.SourcePath, "SourcePath should not be empty")
			} else {
				assert.Empty(t, result.SourcePath, "SourcePath should be empty")
			}
		})
	}
}
