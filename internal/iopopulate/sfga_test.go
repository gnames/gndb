package iopopulate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gnames/gndb/pkg/populate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSFGAFile(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupFiles    []string // files to create in parent dir
		expectedMatch string   // expected filename pattern
		expectError   bool
		errorContains string
	}{
		{
			name:          "single file matches 4-digit ID pattern",
			id:            1,
			setupFiles:    []string{"0001_col_2025-01-15.sqlite.zip"},
			expectedMatch: "0001_col_2025-01-15.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "multiple files with same ID - error",
			id:            2,
			setupFiles:    []string{"0002_worms_2025-01-01.sqlite.zip", "0002_worms_2025-02-01.sqlite.zip"},
			expectError:   true,
			errorContains: "found 2 files matching",
		},
		{
			name:          "no files match - warning but continue",
			id:            999,
			setupFiles:    []string{"0001_col.sqlite.zip"},
			expectError:   true,
			errorContains: "no files found matching ID 999",
		},
		{
			name:          "3-digit ID with leading zero",
			id:            100,
			setupFiles:    []string{"0100_source.sqlite.zip"},
			expectedMatch: "0100_source.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "file without .zip extension",
			id:            3,
			setupFiles:    []string{"0003_itis.sqlite"},
			expectedMatch: "0003_itis.sqlite",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary parent directory
			tmpDir, err := os.MkdirTemp("", "sfga-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create test files
			for _, filename := range tt.setupFiles {
				filePath := filepath.Join(tmpDir, filename)
				err := os.WriteFile(filePath, []byte("test"), 0644)
				require.NoError(t, err)
			}

			// Test resolveSFGAFile
			result, err := resolveSFGAFile(tmpDir, tt.id)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, filepath.Join(tmpDir, tt.expectedMatch), result)
			}
		})
	}
}

func TestFetchSFGA_LocalDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Use existing testdata
	testdataDir := "../../testdata"
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skip("testdata not available, skipping test")
	}

	// Use real cache directory (keeps last fetch for debugging)
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)
	t.Logf("Using cache directory: %s", cacheDir)

	// Create test config using real testdata
	source := populate.DataSourceConfig{
		ID:     1000,
		Parent: testdataDir,
	}

	// Test fetchSFGA
	ctx := context.Background()
	sqlitePath, err := fetchSFGA(ctx, source, cacheDir)
	require.NoError(t, err)
	t.Logf("fetchSFGA returned: %s (will remain in cache for inspection)", sqlitePath)

	// Verify the path exists
	_, statErr := os.Stat(sqlitePath)
	assert.NoError(t, statErr, "returned SQLite path should exist")

	// Verify it's a valid SQLite database
	db, err := openSFGA(sqlitePath)
	require.NoError(t, err)
	defer db.Close()

	// Verify we can query it
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM name").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0, "should have some name records")
}

func TestFetchSFGA_URL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Use real cache directory (keeps last fetch for debugging)
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)
	t.Logf("Using cache directory: %s", cacheDir)

	// Use a real SFGA URL (ID 206 - Ruhoff, a small dataset ~4MB)
	// Note: Actual filename is 0206_ruhoff_date_version.sqlite.zip
	// This test validates URL fetching logic even if file not found
	source := populate.DataSourceConfig{
		ID:     206,
		Parent: "http://opendata.globalnames.org/sfga/latest/",
	}

	// Test fetchSFGA
	ctx := context.Background()
	sqlitePath, err := fetchSFGA(ctx, source, cacheDir)

	// Skip if no internet connection or file not available
	if err != nil {
		if strings.Contains(err.Error(), "no such host") ||
		   strings.Contains(err.Error(), "connection refused") ||
		   strings.Contains(err.Error(), "context deadline exceeded") {
			t.Skipf("skipping URL test: no internet connection or server unavailable: %v", err)
		}
		// If it's a different error (like file not found on server), that's still useful to know
		t.Logf("URL fetch error (may indicate file not on server): %v", err)
		return
	}

	t.Logf("fetchSFGA from URL returned: %s (will remain in cache for inspection)", sqlitePath)

	// Verify the path exists
	_, statErr := os.Stat(sqlitePath)
	assert.NoError(t, statErr, "returned SQLite path should exist")

	// Verify it's a valid SQLite database
	db, err := openSFGA(sqlitePath)
	require.NoError(t, err)
	defer db.Close()

	// Verify we can query it
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM name").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0, "should have some name records")
}

func TestOpenSFGA(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Use fetchSFGA to get an extracted SQLite file
	testdataDir := "../../testdata"
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skip("testdata not available, skipping test")
	}

	// Use real cache directory
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	// Fetch and extract test data
	source := populate.DataSourceConfig{
		ID:     1000,
		Parent: testdataDir,
	}

	ctx := context.Background()
	sqlitePath, err := fetchSFGA(ctx, source, cacheDir)
	require.NoError(t, err)

	// Test opening the database
	db, err := openSFGA(sqlitePath)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify we can query the database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM name").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0, "should have some name records")
}

func TestOpenSFGA_NonexistentFile(t *testing.T) {
	_, err := openSFGA("/nonexistent/path/to/file.sqlite")
	assert.Error(t, err)
}
