package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_EnvVarOverride_DatabaseHost(t *testing.T) {
	// Create temp config file with default host
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gndb.yaml")
	configContent := `
database:
  host: config-file-host
  port: 5432
  user: postgres
  password: postgres
  database: gnames
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variable to override host
	t.Setenv("GNDB_DATABASE_HOST", "env-override-host")
	t.Setenv("GNDB_DATABASE_USER", "env-user")

	// Load config
	result, err := Load(configPath)
	require.NoError(t, err)
	cfg := result.Config

	// Verify environment variable overrode config file
	assert.Equal(t, "env-override-host", cfg.Database.Host)
	// Other values should remain from config file
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "env-user", cfg.Database.User)
	// Verify source is file
	assert.Equal(t, "file", result.Source)
	assert.NotEmpty(t, result.SourcePath)
}

func TestLoad_EnvVarOverride_NestedField(t *testing.T) {
	// Create temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gndb.yaml")
	configContent := `
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: gnames
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variable for nested field
	t.Setenv("GNDB_DATABASE_MAX_CONNECTIONS", "50")
	t.Setenv("GNDB_DATABASE_MIN_CONNECTIONS", "5")

	// Load config
	result, err := Load(configPath)
	require.NoError(t, err)
	cfg := result.Config

	// Verify environment variables overrode config file
	assert.Equal(t, 50, cfg.Database.MaxConnections)
	assert.Equal(t, 5, cfg.Database.MinConnections)
}

func TestLoad_EnvVarOverride_ImportBatchSize(t *testing.T) {
	// Create temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gndb.yaml")
	configContent := `
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: gnames
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variable for import batch size
	t.Setenv("GNDB_IMPORT_BATCH_SIZE", "10000")

	// Load config
	result, err := Load(configPath)
	require.NoError(t, err)
	cfg := result.Config

	// Verify environment variable overrode config file
	assert.Equal(t, 10000, cfg.Import.BatchSize)
}

func TestLoad_EnvVarOverride_LoggingLevel(t *testing.T) {
	// Create temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gndb.yaml")
	configContent := `
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: gnames
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variables for logging
	t.Setenv("GNDB_LOGGING_LEVEL", "debug")
	t.Setenv("GNDB_LOGGING_FORMAT", "json")

	// Load config
	result, err := Load(configPath)
	require.NoError(t, err)
	cfg := result.Config

	// Verify environment variables overrode config file
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestLoad_EnvVarOverride_BooleanField(t *testing.T) {
	// Create temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gndb.yaml")
	configContent := `
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: gnames
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variable for boolean field
	t.Setenv("GNDB_OPTIMIZATION_CONCURRENT_INDEXES", "true")

	// Load config
	result, err := Load(configPath)
	require.NoError(t, err)
	cfg := result.Config

	// Verify environment variable overrode config file
	assert.True(t, cfg.Optimization.ConcurrentIndexes)
}

func TestLoad_EnvVarOverride_MultipleFields(t *testing.T) {
	// Create temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gndb.yaml")
	configContent := `
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: gnames
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set multiple environment variables
	t.Setenv("GNDB_DATABASE_HOST", "prod-db.example.com")
	t.Setenv("GNDB_DATABASE_PORT", "5433")
	t.Setenv("GNDB_DATABASE_USER", "gndb_user")
	t.Setenv("GNDB_DATABASE_PASSWORD", "secret123")
	t.Setenv("GNDB_DATABASE_DATABASE", "gnverifier_prod")
	t.Setenv("GNDB_DATABASE_SSL_MODE", "require")
	t.Setenv("GNDB_IMPORT_BATCH_SIZE", "8000")
	t.Setenv("GNDB_LOGGING_LEVEL", "warn")

	// Load config
	result, err := Load(configPath)
	require.NoError(t, err)
	cfg := result.Config

	// Verify all environment variables overrode config file
	assert.Equal(t, "prod-db.example.com", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "gndb_user", cfg.Database.User)
	assert.Equal(t, "secret123", cfg.Database.Password)
	assert.Equal(t, "gnverifier_prod", cfg.Database.Database)
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.Equal(t, 8000, cfg.Import.BatchSize)
	assert.Equal(t, "warn", cfg.Logging.Level)
}

func TestLoad_NoConfigFile_EnvVarsOnly(t *testing.T) {
	// No config file, only environment variables
	// Override HOME to a temp directory to ensure no config file exists
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	t.Setenv("GNDB_DATABASE_HOST", "env-only-host")
	t.Setenv("GNDB_DATABASE_PORT", "5432")
	t.Setenv("GNDB_DATABASE_USER", "testuser")
	t.Setenv("GNDB_DATABASE_PASSWORD", "testpass")
	t.Setenv("GNDB_DATABASE_DATABASE", "testdb")

	// Load config without config file (will use defaults + env vars)
	result, err := Load("")
	require.NoError(t, err)
	cfg := result.Config

	// Verify environment variables overrode defaults
	assert.Equal(t, "env-only-host", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "testuser", cfg.Database.User)
	assert.Equal(t, "testpass", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.Database)

	// Other values should be defaults
	assert.Equal(t, 20, cfg.Database.MaxConnections) // default
	assert.Equal(t, 5000, cfg.Import.BatchSize)      // default

	// Verify source is defaults+env
	assert.Equal(t, "defaults+env", result.Source)
	assert.Empty(t, result.SourcePath)
}

func TestLoad_PrecedenceOrder(t *testing.T) {
	// This test verifies precedence: env vars > config file
	// (flag precedence is tested separately via BindFlags)

	// Create temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gndb.yaml")
	configContent := `
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set some env vars to override, leave others from config file
	t.Setenv("GNDB_DATABASE_HOST", "env-host")
	t.Setenv("GNDB_DATABASE_USER", "env-user")
	// Don't set password, port, etc - should come from config file

	// Load config
	result, err := Load(configPath)
	require.NoError(t, err)
	cfg := result.Config

	// Verify env vars take precedence
	assert.Equal(t, "env-host", cfg.Database.Host)
	assert.Equal(t, "env-user", cfg.Database.User)
}
