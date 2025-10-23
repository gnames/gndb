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
	// Uses temp directory to avoid touching production cache

	// Setup temp cache directory
	tempCacheDir, err := os.MkdirTemp("", "gndb-cache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempCacheDir)

	// Override GNDB_CACHE_DIR for this test
	originalCacheDir := os.Getenv("GNDB_CACHE_DIR")
	err = os.Setenv("GNDB_CACHE_DIR", tempCacheDir)
	require.NoError(t, err)
	defer func() {
		if originalCacheDir != "" {
			os.Setenv("GNDB_CACHE_DIR", originalCacheDir)
		} else {
			os.Unsetenv("GNDB_CACHE_DIR")
		}
	}()

	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	// Verify the path structure (should be under temp dir now)
	expectedBase := filepath.Join(tempCacheDir, "sfga")
	assert.Equal(t, expectedBase, cacheDir, "cache directory should be under temp directory")

	// Clean up
	require.NoError(t, clearCache(cacheDir))
}

// TestMultiSourceCacheCleaning tests that cache is properly cleared between processing
// multiple data sources, preventing "too many database files" error from sflib.
//
// This test simulates the scenario where multiple sources are processed sequentially,
// and verifies that:
// 1. Cache is cleared before each source fetch
// 2. Only one source's files exist in cache at a time
// 3. No accumulation of database files from previous sources
func TestMultiSourceCacheCleaning(t *testing.T) {
	// Create temp cache directory
	cacheDir := filepath.Join(t.TempDir(), "sfga-cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	// Simulate processing 3 different data sources
	sources := []struct {
		id       int
		filename string
	}{
		{id: 1, filename: "source1.sqlite"},
		{id: 2, filename: "source2.sqlite"},
		{id: 3, filename: "source3.sqlite"},
	}

	for _, source := range sources {
		// Clear cache before processing each source (this is what processSource does)
		err := clearCache(cacheDir)
		require.NoError(t, err, "cache clear should succeed for source %d", source.id)

		// Verify cache is empty after clear
		entries, err := os.ReadDir(cacheDir)
		require.NoError(t, err)
		assert.Empty(t, entries, "cache should be empty before fetching source %d", source.id)

		// Simulate fetching/extracting SFGA file for this source
		// In real scenario, this would be done by fetchSFGA() -> sflib.Fetch()
		testFile := filepath.Join(cacheDir, source.filename)
		err = os.WriteFile(testFile, []byte("test data for source"), 0644)
		require.NoError(t, err)

		// Also create some additional files that sflib might create
		additionalFiles := []string{
			filepath.Join(cacheDir, "metadata.json"),
			filepath.Join(cacheDir, "subdir", "data.txt"),
		}
		for _, f := range additionalFiles {
			require.NoError(t, os.MkdirAll(filepath.Dir(f), 0755))
			require.NoError(t, os.WriteFile(f, []byte("additional"), 0644))
		}

		// Verify cache now contains files for current source
		entries, err = os.ReadDir(cacheDir)
		require.NoError(t, err)
		assert.NotEmpty(t, entries, "cache should contain files for source %d", source.id)

		// Verify the expected file exists
		_, err = os.Stat(testFile)
		assert.NoError(t, err, "source file should exist for source %d", source.id)

		// If this were not the last source, the next iteration would clear the cache
		// Let's verify that files from previous sources don't accumulate
		if source.id < len(sources) {
			// Count SQLite files in cache
			sqliteFiles := 0
			walkErr := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() && filepath.Ext(path) == ".sqlite" {
					sqliteFiles++
				}
				return nil
			})
			require.NoError(t, walkErr, "filepath.Walk should succeed")
			assert.Equal(t, 1, sqliteFiles,
				"should have exactly 1 SQLite file in cache for source %d", source.id)
		}
	}

	// After processing all sources, clear cache one final time
	err := clearCache(cacheDir)
	require.NoError(t, err)

	// Verify final cleanup
	entries, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "cache should be empty after final cleanup")
}

// TestCacheCleaning_PreventsSFLibError tests that clearing cache before each
// source prevents the "too many database files" error that would occur if
// multiple SQLite files existed in the cache directory.
func TestCacheCleaning_PreventsSFLibError(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "sfga-cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	// Simulate the problematic scenario: multiple SQLite files in cache
	// (this would happen without proper cache clearing between sources)
	problematicFiles := []string{
		"0001_source1.sqlite",
		"0002_source2.sqlite",
		"0003_source3.sqlite",
	}

	for _, filename := range problematicFiles {
		path := filepath.Join(cacheDir, filename)
		err := os.WriteFile(path, []byte("database content"), 0644)
		require.NoError(t, err)
	}

	// Verify we have multiple SQLite files (the problematic state)
	entries, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	assert.Len(t, entries, 3, "should have 3 SQLite files before clearing")

	// Clear cache (this is what processSource does before each fetchSFGA)
	err = clearCache(cacheDir)
	require.NoError(t, err)

	// Verify cache is now empty (preventing the "too many database files" error)
	entries, err = os.ReadDir(cacheDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "cache should be empty after clearing")

	// Now it's safe to fetch a new source
	newSourceFile := filepath.Join(cacheDir, "0004_new_source.sqlite")
	err = os.WriteFile(newSourceFile, []byte("new source"), 0644)
	require.NoError(t, err)

	// Verify only the new source exists
	entries, err = os.ReadDir(cacheDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "should have exactly 1 file after fetching new source")
	assert.Equal(t, "0004_new_source.sqlite", entries[0].Name())
}
