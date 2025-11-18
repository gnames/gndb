package config_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gnames/gndb/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses file system in short mode")
	}

	tempHome := t.TempDir()

	tests := []struct {
		msg string
		fn  func(string) string
		res string
	}{
		{
			msg: "config dir",
			fn:  config.ConfigDir,
			res: filepath.Join(tempHome, ".config", "gndb"),
		},
		{
			msg: "cache dir",
			fn:  config.CacheDir,
			res: filepath.Join(tempHome, ".cache", "gndb"),
		},
		{
			msg: "log dir",
			fn:  config.LogDir,
			res: filepath.Join(tempHome, ".local", "share", "gndb", "logs"),
		},
	}

	for _, v := range tests {
		res := v.fn(tempHome)
		assert.Equal(t, v.res, res, v.msg)
	}
}

func TestNew(t *testing.T) {
	cfg := config.New()

	t.Run("creates valid default config", func(t *testing.T) {
		require.NotNil(t, cfg)

		// Database defaults
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "postgres", cfg.Database.User)
		assert.Equal(t, "postgres", cfg.Database.Password)
		assert.Equal(t, "gnames", cfg.Database.Database)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
		assert.Equal(t, 50_000, cfg.Database.BatchSize)

		// Log defaults
		assert.Equal(t, "json", cfg.Log.Format)
		assert.Equal(t, "info", cfg.Log.Level)
		assert.Equal(t, "file", cfg.Log.Destination)

		// JobsNumber defaults to CPU count
		assert.Equal(t, runtime.NumCPU(), cfg.JobsNumber)
	})
}

func TestOptionDatabaseHost(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sets valid host",
			input:    "db.example.com",
			expected: "db.example.com",
		},
		{
			name:     "trims whitespace",
			input:    "  db.example.com  ",
			expected: "db.example.com",
		},
		{
			name:     "ignores empty string",
			input:    "",
			expected: "localhost", // Should keep default
		},
		{
			name:     "ignores whitespace-only",
			input:    "   ",
			expected: "localhost", // Should keep default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			opt := config.OptDatabaseHost(tt.input)
			cfg.Update([]config.Option{opt})
			assert.Equal(t, tt.expected, cfg.Database.Host)
		})
	}
}

func TestOptionDatabasePort(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "sets valid port",
			input:    3306,
			expected: 3306,
		},
		{
			name:     "ignores zero",
			input:    0,
			expected: 5432, // Should keep default
		},
		{
			name:     "ignores negative",
			input:    -100,
			expected: 5432, // Should keep default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			opt := config.OptDatabasePort(tt.input)
			cfg.Update([]config.Option{opt})
			assert.Equal(t, tt.expected, cfg.Database.Port)
		})
	}
}

func TestOptionDatabaseSSLMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sets valid ssl mode - disable",
			input:    "disable",
			expected: "disable",
		},
		{
			name:     "sets valid ssl mode - require",
			input:    "require",
			expected: "require",
		},
		{
			name:     "sets valid ssl mode - verify-ca",
			input:    "verify-ca",
			expected: "verify-ca",
		},
		{
			name:     "sets valid ssl mode - verify-full",
			input:    "verify-full",
			expected: "verify-full",
		},
		{
			name:     "normalizes to lowercase",
			input:    "REQUIRE",
			expected: "require",
		},
		{
			name:     "ignores invalid value",
			input:    "invalid",
			expected: "disable", // Should keep default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			opt := config.OptDatabaseSSLMode(tt.input)
			cfg.Update([]config.Option{opt})
			assert.Equal(t, tt.expected, cfg.Database.SSLMode)
		})
	}
}

func TestOptionLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sets valid log level - debug",
			input:    "debug",
			expected: "debug",
		},
		{
			name:     "sets valid log level - info",
			input:    "info",
			expected: "info",
		},
		{
			name:     "sets valid log level - warn",
			input:    "warn",
			expected: "warn",
		},
		{
			name:     "sets valid log level - error",
			input:    "error",
			expected: "error",
		},
		{
			name:     "normalizes to lowercase",
			input:    "DEBUG",
			expected: "debug",
		},
		{
			name:     "ignores invalid value",
			input:    "trace",
			expected: "info", // Should keep default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			opt := config.OptLogLevel(tt.input)
			cfg.Update([]config.Option{opt})
			assert.Equal(t, tt.expected, cfg.Log.Level)
		})
	}
}

func TestOptionLogFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sets valid format - json",
			input:    "json",
			expected: "json",
		},
		{
			name:     "sets valid format - text",
			input:    "text",
			expected: "text",
		},
		{
			name:     "sets valid format - tint",
			input:    "tint",
			expected: "tint",
		},
		{
			name:     "ignores invalid value",
			input:    "xml",
			expected: "json", // Should keep default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			opt := config.OptLogFormat(tt.input)
			cfg.Update([]config.Option{opt})
			assert.Equal(t, tt.expected, cfg.Log.Format)
		})
	}
}

func TestOptionBatchSize(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "sets valid batch size",
			input:    10000,
			expected: 10000,
		},
		{
			name:     "ignores zero",
			input:    0,
			expected: 50_000, // Should keep default
		},
		{
			name:     "ignores negative",
			input:    -1000,
			expected: 50_000, // Should keep default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			opt := config.OptDatabaseBatchSize(tt.input)
			cfg.Update([]config.Option{opt})
			assert.Equal(t, tt.expected, cfg.Database.BatchSize)
		})
	}
}

func TestOptionJobsNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "sets valid jobs number",
			input:    8,
			expected: 8,
		},
		{
			name:     "ignores zero",
			input:    0,
			expected: runtime.NumCPU(), // Should keep default
		},
		{
			name:     "ignores negative",
			input:    -5,
			expected: runtime.NumCPU(), // Should keep default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			opt := config.OptJobsNumber(tt.input)
			cfg.Update([]config.Option{opt})
			assert.Equal(t, tt.expected, cfg.JobsNumber)
		})
	}
}

func TestOptionPopulateSourceIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "sets source IDs",
			input:    []int{1, 3, 5},
			expected: []int{1, 3, 5},
		},
		{
			name:     "ignores empty slice",
			input:    []int{},
			expected: nil, // Should keep default (nil)
		},
		{
			name:     "ignores nil",
			input:    nil,
			expected: nil, // Should keep default (nil)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			opt := config.OptPopulateSourceIDs(tt.input)
			cfg.Update([]config.Option{opt})
			assert.Equal(t, tt.expected, cfg.Populate.SourceIDs)
		})
	}
}

func TestOptionPopulateWithFlatClassification(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		input    *bool
		expected *bool
	}{
		{
			name:     "sets to true",
			input:    &trueVal,
			expected: &trueVal,
		},
		{
			name:     "sets to false",
			input:    &falseVal,
			expected: &falseVal,
		},
		{
			name:     "ignores nil",
			input:    nil,
			expected: nil, // Should keep default (nil)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			opt := config.OptPopulateWithFlatClassification(tt.input)
			cfg.Update([]config.Option{opt})
			if tt.expected == nil {
				assert.Nil(t, cfg.Populate.WithFlatClassification)
			} else {
				require.NotNil(t, cfg.Populate.WithFlatClassification)
				assert.Equal(t, *tt.expected, *cfg.Populate.WithFlatClassification)
			}
		})
	}
}

func TestMultipleOptions(t *testing.T) {
	t.Run("applies multiple options in order", func(t *testing.T) {
		cfg := config.New()

		opts := []config.Option{
			config.OptDatabaseHost("custom.host.com"),
			config.OptDatabasePort(3306),
			config.OptDatabaseUser("myuser"),
			config.OptLogLevel("debug"),
			config.OptJobsNumber(16),
		}

		cfg.Update(opts)

		assert.Equal(t, "custom.host.com", cfg.Database.Host)
		assert.Equal(t, 3306, cfg.Database.Port)
		assert.Equal(t, "myuser", cfg.Database.User)
		assert.Equal(t, "debug", cfg.Log.Level)
		assert.Equal(t, 16, cfg.JobsNumber)

		// Unchanged fields keep defaults
		assert.Equal(t, "postgres", cfg.Database.Password)
		assert.Equal(t, "json", cfg.Log.Format)
	})

	t.Run("later options override earlier ones", func(t *testing.T) {
		cfg := config.New()

		opts := []config.Option{
			config.OptDatabaseHost("first.host.com"),
			config.OptDatabaseHost("second.host.com"),
		}

		cfg.Update(opts)

		assert.Equal(t, "second.host.com", cfg.Database.Host)
	})
}

func TestToOptions(t *testing.T) {
	t.Run("converts config to options correctly", func(t *testing.T) {
		// Create config with custom values
		original := config.New()
		opts := []config.Option{
			config.OptDatabaseHost("test.host.com"),
			config.OptDatabasePort(3306),
			config.OptDatabaseUser("testuser"),
			config.OptDatabasePassword("testpass"),
			config.OptDatabaseDatabase("testdb"),
			config.OptDatabaseSSLMode("require"),
			config.OptDatabaseBatchSize(10000),
			config.OptLogLevel("debug"),
			config.OptLogFormat("text"),
			config.OptLogDestination("stdout"),
			config.OptJobsNumber(8),
		}
		original.Update(opts)

		// Convert to options and apply to new config
		convertedOpts := original.ToOptions()
		newCfg := config.New()
		newCfg.Update(convertedOpts)

		// Verify persistent fields match
		assert.Equal(t, original.Database.Host, newCfg.Database.Host)
		assert.Equal(t, original.Database.Port, newCfg.Database.Port)
		assert.Equal(t, original.Database.User, newCfg.Database.User)
		assert.Equal(t, original.Database.Password, newCfg.Database.Password)
		assert.Equal(t, original.Database.Database, newCfg.Database.Database)
		assert.Equal(t, original.Database.SSLMode, newCfg.Database.SSLMode)
		assert.Equal(t, original.Database.BatchSize, newCfg.Database.BatchSize)
		assert.Equal(t, original.Log.Level, newCfg.Log.Level)
		assert.Equal(t, original.Log.Format, newCfg.Log.Format)
		assert.Equal(t, original.Log.Destination, newCfg.Log.Destination)
		assert.Equal(t, original.JobsNumber, newCfg.JobsNumber)
	})

	t.Run("excludes runtime-only fields", func(t *testing.T) {
		cfg := config.New()
		cfg.Update([]config.Option{
			config.OptHomeDir("/custom/home"),
			config.OptPopulateSourceIDs([]int{1, 2, 3}),
			config.OptPopulateReleaseVersion("v1.0.0"),
			config.OptPopulateReleaseDate("2025-01-01"),
		})

		// These fields should not be in ToOptions() output
		opts := cfg.ToOptions()
		newCfg := config.New()
		newCfg.Update(opts)

		// Runtime fields should remain at defaults in newCfg
		assert.Equal(t, "", newCfg.HomeDir)
		assert.Nil(t, newCfg.Populate.SourceIDs)
		assert.Equal(t, "", newCfg.Populate.ReleaseVersion)
		assert.Equal(t, "", newCfg.Populate.ReleaseDate)
	})

	t.Run("excludes WithFlatClassification runtime field", func(t *testing.T) {
		trueVal := true
		cfg := config.New()
		cfg.Update([]config.Option{
			config.OptPopulateWithFlatClassification(&trueVal),
		})

		// ToOptions should NOT include WithFlatClassification (runtime-only)
		opts := cfg.ToOptions()
		newCfg := config.New()
		newCfg.Update(opts)

		// Runtime field should remain nil in newCfg
		assert.Nil(t, newCfg.Populate.WithFlatClassification)
	})
}
