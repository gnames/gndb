package config_test

import (
	"os"
	"path/filepath"
	"testing"

	ioconfig "github.com/gnames/gndb/internal/io/config"
	"github.com/gnames/gndb/pkg/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ExplicitPath(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	yamlContent := `
database:
  host: testhost
  port: 5433
  user: testuser
  password: testpass
  database: testdb
  ssl_mode: require
  max_connections: 30
  min_connections: 5
  max_conn_lifetime: 90
  max_conn_idle_time: 15

import:
  batch_size: 1000

optimization:
  concurrent_indexes: true

logging:
  level: debug
  format: json
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load config
	cfg, err := ioconfig.Load(configFile)
	require.NoError(t, err)

	// Verify database config
	assert.Equal(t, "testhost", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "testuser", cfg.Database.User)
	assert.Equal(t, "testpass", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.Database)
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.Equal(t, 30, cfg.Database.MaxConnections)
	assert.Equal(t, 5, cfg.Database.MinConnections)
	assert.Equal(t, 90, cfg.Database.MaxConnLifetime)
	assert.Equal(t, 15, cfg.Database.MaxConnIdleTime)

	// Verify import config
	assert.Equal(t, 1000, cfg.Import.BatchSize)

	// Verify optimization config
	assert.True(t, cfg.Optimization.ConcurrentIndexes)

	// Verify logging config
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestLoad_MissingRequiredField(t *testing.T) {
	// Create config file with missing required field
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid-config.yaml")

	yamlContent := `
database:
  host: testhost
  port: 5432
  # missing user field

import:
  batch_size: 1000
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load should fail validation
	_, err = ioconfig.Load(configFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid configuration")
}

func TestLoad_MalformedYAML(t *testing.T) {
	// Create malformed YAML file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "malformed.yaml")

	yamlContent := `
database:
  host: testhost
  port: not_a_number
  user: testuser
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load should fail
	_, err = ioconfig.Load(configFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal config")
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Load with empty path (search default locations)
	cfg, err := ioconfig.Load("")

	// Should return defaults when file not found
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify it's using defaults
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "postgres", cfg.Database.User)
}

func TestLoad_ExplicitPathNotFound(t *testing.T) {
	// Load with explicit non-existent file path
	_, err := ioconfig.Load("/nonexistent/path/config.yaml")

	// Should return error for explicit path that doesn't exist
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestBindFlags(t *testing.T) {
	// Start with default config
	cfg := config.Defaults()

	// Create a cobra command with flags
	cmd := &cobra.Command{
		Use: "test",
	}

	cmd.Flags().String("host", "", "database host")
	cmd.Flags().Int("port", 0, "database port")
	cmd.Flags().String("user", "", "database user")
	cmd.Flags().String("password", "", "database password")
	cmd.Flags().String("database", "", "database name")
	cmd.Flags().String("ssl-mode", "", "SSL mode")

	// Set some flags
	require.NoError(t, cmd.Flags().Set("host", "flaghost"))
	require.NoError(t, cmd.Flags().Set("port", "9999"))
	require.NoError(t, cmd.Flags().Set("user", "flaguser"))

	// Bind flags to config
	updatedCfg, err := ioconfig.BindFlags(cmd, cfg)
	require.NoError(t, err)

	// Verify flag values override defaults
	assert.Equal(t, "flaghost", updatedCfg.Database.Host)
	assert.Equal(t, 9999, updatedCfg.Database.Port)
	assert.Equal(t, "flaguser", updatedCfg.Database.User)

	// Verify unset flags retain original values
	assert.Equal(t, cfg.Database.Database, updatedCfg.Database.Database)
	assert.Equal(t, cfg.Database.SSLMode, updatedCfg.Database.SSLMode)
}

func TestBindFlags_InvalidConfig(t *testing.T) {
	// Start with config
	cfg := config.Defaults()

	// Create command with flags
	cmd := &cobra.Command{
		Use: "test",
	}

	cmd.Flags().String("host", "", "database host")
	cmd.Flags().Int("port", 0, "database port")

	// Set invalid host (empty host will fail validation)
	require.NoError(t, cmd.Flags().Set("host", ""))

	// Bind flags should fail validation
	_, err := ioconfig.BindFlags(cmd, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid configuration")
}
