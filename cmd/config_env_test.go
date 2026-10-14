package cmd

import (
	"testing"

	"github.com/gnames/gndb/internal/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envKeys are environment variable suffixes for config keys.
var envKeys = []string{
	"DATABASE_HOST", "DATABASE_PORT", "DATABASE_USER", "DATABASE_PASSWORD",
	"DATABASE_DATABASE", "DATABASE_SSL_MODE", "DATABASE_BATCH_SIZE",
	"LOG_LEVEL", "LOG_FORMAT", "LOG_DESTINATION", "JOBS_NUMBER",
}

func setupConfigHome(t *testing.T) string {
	t.Helper()
	for _, k := range envKeys {
		t.Setenv(k, "")
		t.Setenv("GNDB_"+k, "")
	}
	home := t.TempDir()
	require.NoError(t, iofs.EnsureDirs(home))
	require.NoError(t, iofs.EnsureConfigFile(home))
	return home
}

// TestInitConfigIgnoresUnprefixedEnv checks that generic variables like
// DATABASE_HOST, set for other programs, do not configure gndb.
func TestInitConfigIgnoresUnprefixedEnv(t *testing.T) {
	home := setupConfigHome(t)
	def, err := initConfig(home)
	require.NoError(t, err)

	t.Setenv("DATABASE_HOST", "other.example.org")
	t.Setenv("DATABASE_PASSWORD", "other-secret")
	t.Setenv("DATABASE_DATABASE", "otherdb")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("JOBS_NUMBER", "77")

	cfg, err := initConfig(home)
	require.NoError(t, err)
	assert.Equal(t, def.Database.Host, cfg.Database.Host)
	assert.Equal(t, def.Database.Password, cfg.Database.Password)
	assert.Equal(t, def.Database.Database, cfg.Database.Database)
	assert.Equal(t, def.Log.Level, cfg.Log.Level)
	assert.Equal(t, def.JobsNumber, cfg.JobsNumber)
}

func TestInitConfigPrefixedEnv(t *testing.T) {
	home := setupConfigHome(t)
	t.Setenv("GNDB_DATABASE_HOST", "db.example.org")
	t.Setenv("GNDB_DATABASE_PORT", "6543")
	t.Setenv("GNDB_DATABASE_PASSWORD", "secret")
	t.Setenv("GNDB_DATABASE_SSL_MODE", "require")
	t.Setenv("GNDB_DATABASE_BATCH_SIZE", "1000")
	t.Setenv("GNDB_LOG_LEVEL", "debug")
	t.Setenv("GNDB_LOG_DESTINATION", "stderr")
	t.Setenv("GNDB_JOBS_NUMBER", "3")

	cfg, err := initConfig(home)
	require.NoError(t, err)
	assert.Equal(t, "db.example.org", cfg.Database.Host)
	assert.Equal(t, 6543, cfg.Database.Port)
	assert.Equal(t, "secret", cfg.Database.Password)
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.Equal(t, 1000, cfg.Database.BatchSize)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "stderr", cfg.Log.Destination)
	assert.Equal(t, 3, cfg.JobsNumber)
}
