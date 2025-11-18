package iopopulate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearCache(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses file system in short mode")
	}

	tests := []struct {
		name      string
		setupFunc func(dir string) error
	}{
		{
			name: "clear existing directory with files",
			setupFunc: func(dir string) error {
				// Create directory with some files
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				f, err := os.Create(filepath.Join(dir, "test.txt"))
				if err != nil {
					return err
				}
				return f.Close()
			},
		},
		{
			name: "clear non-existent directory",
			setupFunc: func(_ string) error {
				// Don't create anything
				return nil
			},
		},
		{
			name: "clear empty directory",
			setupFunc: func(dir string) error {
				return os.MkdirAll(dir, 0755)
			},
		},
		{
			name: "clear directory with subdirectories",
			setupFunc: func(dir string) error {
				subdir := filepath.Join(dir, "subdir", "nested")
				if err := os.MkdirAll(subdir, 0755); err != nil {
					return err
				}
				f, err := os.Create(filepath.Join(subdir, "nested.txt"))
				if err != nil {
					return err
				}
				return f.Close()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create unique temp directory for this test.
			tmpDir := t.TempDir()
			cacheDir := filepath.Join(tmpDir, "cache")

			// Setup test conditions.
			err := tt.setupFunc(cacheDir)
			require.NoError(t, err)

			// Clear cache.
			err = clearCache(cacheDir)
			require.NoError(t, err)

			// Verify directory exists and is empty.
			entries, err := os.ReadDir(cacheDir)
			require.NoError(t, err)
			assert.Empty(t, entries)
		})
	}
}

func TestPrepareCacheDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses file system in short mode")
	}

	// Create temporary home directory.
	tmpHome := t.TempDir()

	// First call - creates directory.
	cacheDir, err := prepareCacheDir(tmpHome)
	require.NoError(t, err)
	assert.Contains(t, cacheDir, "sfga")

	// Verify directory exists.
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Create a file in the cache.
	testFile := filepath.Join(cacheDir, "test.sqlite")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Second call - should clear existing files.
	cacheDir2, err := prepareCacheDir(tmpHome)
	require.NoError(t, err)
	assert.Equal(t, cacheDir, cacheDir2)

	// Verify file was removed.
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))

	// Verify directory still exists and is empty.
	entries, err := os.ReadDir(cacheDir2)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestPrepareCacheDirPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses file system in short mode")
	}

	tmpHome := t.TempDir()

	cacheDir, err := prepareCacheDir(tmpHome)
	require.NoError(t, err)

	// Verify path structure contains expected components.
	assert.Contains(t, cacheDir, ".cache")
	assert.Contains(t, cacheDir, "gndb")
	assert.Contains(t, cacheDir, "sfga")

	// Verify it ends with "sfga".
	assert.Equal(t, "sfga", filepath.Base(cacheDir))
}
