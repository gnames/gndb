package iooptimize_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gnames/gndb/internal/iooptimize"
	"github.com/gnames/gndb/pkg/parserpool"
	"github.com/gnames/gnlib/ent/nomcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheManager_NewCacheManager(t *testing.T) {
	tests := []struct {
		name    string
		cacheDir string
		wantErr bool
	}{
		{
			name:    "creates cache directory successfully",
			cacheDir: filepath.Join(t.TempDir(), "test-cache"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := iooptimize.NewCacheManager(tt.cacheDir)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cm)

				// Verify directory was created
				_, err := os.Stat(tt.cacheDir)
				assert.NoError(t, err)
			}
		})
	}
}

func TestCacheManager_OpenClose(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "test-cache")
	cm, err := iooptimize.NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Test Open
	err = cm.Open()
	assert.NoError(t, err)

	// Test Close
	err = cm.Close()
	assert.NoError(t, err)

	// Test double close (should not error)
	err = cm.Close()
	assert.NoError(t, err)
}

func TestCacheManager_StoreParsed(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "test-cache")
	cm, err := iooptimize.NewCacheManager(cacheDir)
	require.NoError(t, err)

	err = cm.Open()
	require.NoError(t, err)
	defer cm.Close()

	// Create a parsed name using parser pool
	pool := parserpool.NewPool(1)
	defer pool.Close()
	parsed, err := pool.Parse("Homo sapiens Linnaeus 1758", nomcode.Zoological)
	require.NoError(t, err)

	// Store parsed result
	nameStringID := "test-uuid-12345"
	err = cm.StoreParsed(nameStringID, &parsed)
	assert.NoError(t, err)
}

func TestCacheManager_GetParsed(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "test-cache")
	cm, err := iooptimize.NewCacheManager(cacheDir)
	require.NoError(t, err)

	err = cm.Open()
	require.NoError(t, err)
	defer cm.Close()

	// Create and store a parsed name using parser pool
	pool := parserpool.NewPool(1)
	defer pool.Close()
	parsed, err := pool.Parse("Homo sapiens Linnaeus 1758", nomcode.Zoological)
	require.NoError(t, err)

	nameStringID := "test-uuid-12345"
	err = cm.StoreParsed(nameStringID, &parsed)
	require.NoError(t, err)

	// Retrieve parsed result
	retrieved, err := cm.GetParsed(nameStringID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Homo sapiens", retrieved.CanonicalSimple)
	assert.Equal(t, "Homo sapiens", retrieved.CanonicalFull)

	// Test retrieving non-existent key
	notFound, err := cm.GetParsed("non-existent-key")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestCacheManager_StoreAndRetrieveMultiple(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "test-cache")
	cm, err := iooptimize.NewCacheManager(cacheDir)
	require.NoError(t, err)

	err = cm.Open()
	require.NoError(t, err)
	defer cm.Close()

	pool := parserpool.NewPool(1)
	defer pool.Close()

	// Store multiple parsed names
	testCases := []struct {
		id   string
		name string
		expectedCanonical string
	}{
		{"id-1", "Homo sapiens", "Homo sapiens"},
		{"id-2", "Mus musculus Linnaeus 1758", "Mus musculus"},
		{"id-3", "Felis catus", "Felis catus"},
	}

	// Store all
	for _, tc := range testCases {
		parsed, err := pool.Parse(tc.name, nomcode.Zoological)
		require.NoError(t, err)
		err = cm.StoreParsed(tc.id, &parsed)
		require.NoError(t, err)
	}

	// Retrieve all and verify
	for _, tc := range testCases {
		retrieved, err := cm.GetParsed(tc.id)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, tc.expectedCanonical, retrieved.CanonicalSimple, "ID: %s", tc.id)
	}
}

func TestCacheManager_OperationsOnClosedCache(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "test-cache")
	cm, err := iooptimize.NewCacheManager(cacheDir)
	require.NoError(t, err)

	// Try to store without opening
	pool := parserpool.NewPool(1)
	defer pool.Close()
	parsed, err := pool.Parse("Homo sapiens", nomcode.Zoological)
	require.NoError(t, err)
	err = cm.StoreParsed("test-id", &parsed)
	assert.Error(t, err, "should error when cache not open")

	// Try to retrieve without opening
	_, err = cm.GetParsed("test-id")
	assert.Error(t, err, "should error when cache not open")
}

func TestCacheManager_Cleanup(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "test-cache")
	cm, err := iooptimize.NewCacheManager(cacheDir)
	require.NoError(t, err)

	err = cm.Open()
	require.NoError(t, err)

	// Store some data
	pool := parserpool.NewPool(1)
	defer pool.Close()
	parsed, err := pool.Parse("Homo sapiens", nomcode.Zoological)
	require.NoError(t, err)
	err = cm.StoreParsed("test-id", &parsed)
	require.NoError(t, err)

	// Cleanup
	err = cm.Cleanup()
	assert.NoError(t, err)

	// Verify directory is empty (cleaned)
	entries, err := os.ReadDir(cacheDir)
	assert.NoError(t, err)
	assert.Empty(t, entries, "cache directory should be empty after cleanup")
}

func TestCacheManager_UnparsedName(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "test-cache")
	cm, err := iooptimize.NewCacheManager(cacheDir)
	require.NoError(t, err)

	err = cm.Open()
	require.NoError(t, err)
	defer cm.Close()

	// Create an unparsed name (gibberish)
	pool := parserpool.NewPool(1)
	defer pool.Close()
	parsed, err := pool.Parse("12345 !!@#$", nomcode.Zoological)
	require.NoError(t, err)

	// Store unparsed result
	nameStringID := "unparsed-id"
	err = cm.StoreParsed(nameStringID, &parsed)
	assert.NoError(t, err)

	// Retrieve and verify empty canonicals
	retrieved, err := cm.GetParsed(nameStringID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Empty(t, retrieved.CanonicalSimple)
	assert.Empty(t, retrieved.CanonicalFull)
}
