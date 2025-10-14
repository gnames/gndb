package iopopulate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearCache(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantError bool
	}{
		{
			name: "clear existing cache with files",
			setup: func(t *testing.T) string {
				// Create temporary directory
				cacheDir := filepath.Join(t.TempDir(), "cache")
				require.NoError(t, os.MkdirAll(cacheDir, 0755))

				// Add some test files
				testFiles := []string{"file1.sqlite", "file2.sql", "subdir/file3.txt"}
				for _, f := range testFiles {
					path := filepath.Join(cacheDir, f)
					require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
					require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
				}

				return cacheDir
			},
			wantError: false,
		},
		{
			name: "clear non-existent cache directory",
			setup: func(t *testing.T) string {
				// Return path to non-existent directory
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantError: false,
		},
		{
			name: "clear empty cache directory",
			setup: func(t *testing.T) string {
				cacheDir := filepath.Join(t.TempDir(), "empty")
				require.NoError(t, os.MkdirAll(cacheDir, 0755))
				return cacheDir
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheDir := tt.setup(t)

			err := clearCache(cacheDir)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify directory exists and is empty
				entries, err := os.ReadDir(cacheDir)
				require.NoError(t, err)
				assert.Empty(t, entries, "cache directory should be empty")
			}
		})
	}
}

func TestPrepareCacheDir(t *testing.T) {
	tests := []struct {
		name      string
		wantError bool
	}{
		{
			name:      "prepare cache directory successfully",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run prepareCacheDir
			cacheDir, err := prepareCacheDir()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, cacheDir, "cache directory path should not be empty")

				// Verify path ends with "sfga"
				assert.Equal(t, "sfga", filepath.Base(cacheDir), "cache directory should end with 'sfga'")

				// Verify directory exists
				info, err := os.Stat(cacheDir)
				require.NoError(t, err)
				assert.True(t, info.IsDir(), "cache path should be a directory")

				// Verify directory is empty
				entries, err := os.ReadDir(cacheDir)
				require.NoError(t, err)
				assert.Empty(t, entries, "cache directory should be empty")

				// Clean up
				// The cache is in user's actual ~/.cache/gndb/sfga, so we should clean it
				require.NoError(t, clearCache(cacheDir))
			}
		})
	}
}

func TestPrepareCacheDir_IntegrationWithGetCacheDir(t *testing.T) {
	// This test verifies that prepareCacheDir correctly uses config.GetCacheDir()

	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	// Verify the path structure
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	expectedBase := filepath.Join(homeDir, ".cache", "gndb", "sfga")
	assert.Equal(t, expectedBase, cacheDir, "cache directory should be ~/.cache/gndb/sfga")

	// Clean up
	require.NoError(t, clearCache(cacheDir))
}
