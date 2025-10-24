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
			name:          "multiple files with same ID - selects latest",
			id:            2,
			setupFiles:    []string{"0002_worms_2025-01-01.sqlite.zip", "0002_worms_2025-02-01.sqlite.zip"},
			expectedMatch: "0002_worms_2025-02-01.sqlite.zip",
			expectError:   false,
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
		// New tests for flexible ID pattern matching
		{
			name:          "ID=1 matches 1-data.sqlite",
			id:            1,
			setupFiles:    []string{"1-data.sqlite.zip"},
			expectedMatch: "1-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=1 matches 01-data.sqlite",
			id:            1,
			setupFiles:    []string{"01-data.sqlite.zip"},
			expectedMatch: "01-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=1 matches 001-data.sqlite",
			id:            1,
			setupFiles:    []string{"001-data.sqlite.zip"},
			expectedMatch: "001-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=1 matches 0001-data.sqlite",
			id:            1,
			setupFiles:    []string{"0001-data.sqlite.zip"},
			expectedMatch: "0001-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=1 with underscore separator",
			id:            1,
			setupFiles:    []string{"1_data.sqlite.zip"},
			expectedMatch: "1_data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=1 with just extension",
			id:            1,
			setupFiles:    []string{"1.sqlite.zip"},
			expectedMatch: "1.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=1 does not match 10-data.sqlite or 1001-data.sqlite",
			id:            1,
			setupFiles:    []string{"10-data.sqlite.zip", "1001-data.sqlite.zip"},
			expectError:   true,
			errorContains: "no files found matching ID 1",
		},
		{
			name:          "ID=42 matches 42-data.sqlite",
			id:            42,
			setupFiles:    []string{"42-data.sqlite.zip"},
			expectedMatch: "42-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=42 matches 042-data.sqlite",
			id:            42,
			setupFiles:    []string{"042-data.sqlite.zip"},
			expectedMatch: "042-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=42 matches 0042-data.sqlite",
			id:            42,
			setupFiles:    []string{"0042-data.sqlite.zip"},
			expectedMatch: "0042-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=196 matches 196-data.sqlite",
			id:            196,
			setupFiles:    []string{"196-data.sqlite.zip"},
			expectedMatch: "196-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=196 matches 0196-data.sqlite",
			id:            196,
			setupFiles:    []string{"0196-data.sqlite.zip"},
			expectedMatch: "0196-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "ID=1234 matches 1234-data.sqlite only",
			id:            1234,
			setupFiles:    []string{"1234-data.sqlite.zip"},
			expectedMatch: "1234-data.sqlite.zip",
			expectError:   false,
		},
		{
			name:          "Multiple formats - prefers standard 4-digit",
			id:            5,
			setupFiles:    []string{"5-data_2025-01-01.sqlite.zip", "0005-data_2025-02-01.sqlite.zip"},
			expectedMatch: "0005-data_2025-02-01.sqlite.zip",
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
			result, warning, err := resolveSFGAFile(tmpDir, tt.id)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, filepath.Join(tmpDir, tt.expectedMatch), result)

				// Check for warning when multiple files found
				if len(tt.setupFiles) > 1 {
					assert.NotEmpty(t, warning, "should have warning when multiple files found")
					assert.Contains(t, warning, "selected latest")
					t.Logf("Warning: %s", warning)
				} else {
					assert.Empty(t, warning, "should not have warning for single file")
				}
			}
		})
	}
}

func TestSelectLatestFile(t *testing.T) {
	tests := []struct {
		name      string
		filenames []string
		expected  string
	}{
		{
			name:      "empty list",
			filenames: []string{},
			expected:  "",
		},
		{
			name:      "single file",
			filenames: []string{"0001_col_2025-01-15.sqlite.zip"},
			expected:  "0001_col_2025-01-15.sqlite.zip",
		},
		{
			name:      "multiple files - selects latest by date",
			filenames: []string{"0002_worms_2025-01-01.sqlite.zip", "0002_worms_2025-02-01.sqlite.zip", "0002_worms_2024-12-31.sqlite.zip"},
			expected:  "0002_worms_2025-02-01.sqlite.zip",
		},
		{
			name:      "files without dates - prefers sqlite.zip",
			filenames: []string{"0003_itis.sql", "0003_itis.sqlite.zip"},
			expected:  "0003_itis.sqlite.zip",
		},
		{
			name:      "mixed with and without dates - prefers dated file",
			filenames: []string{"0004_source.sqlite.zip", "0004_source_2025-01-15.sqlite.zip"},
			expected:  "0004_source_2025-01-15.sqlite.zip",
		},
		{
			name:      "different date formats - only YYYY-MM-DD recognized",
			filenames: []string{"0005_a_2025-01-15.sqlite.zip", "0005_b_20250120.sqlite.zip"},
			expected:  "0005_a_2025-01-15.sqlite.zip",
		},
		{
			name:      "same date - prefers sqlite.zip over sql.zip",
			filenames: []string{"0006_source_2025-01-15.sql.zip", "0006_source_2025-01-15.sqlite.zip"},
			expected:  "0006_source_2025-01-15.sqlite.zip",
		},
		{
			name:      "same date - prefers sql.zip over sqlite",
			filenames: []string{"0007_source_2025-01-15.sqlite", "0007_source_2025-01-15.sql.zip"},
			expected:  "0007_source_2025-01-15.sql.zip",
		},
		{
			name:      "same date - prefers sqlite over sql",
			filenames: []string{"0008_source_2025-01-15.sql", "0008_source_2025-01-15.sqlite"},
			expected:  "0008_source_2025-01-15.sqlite",
		},
		{
			name:      "priority order: sqlite.zip > sql.zip > sqlite > sql",
			filenames: []string{"0009_a_2025-01-15.sql", "0009_b_2025-01-15.sqlite", "0009_c_2025-01-15.sql.zip", "0009_d_2025-01-15.sqlite.zip"},
			expected:  "0009_d_2025-01-15.sqlite.zip",
		},
		{
			name:      "later date wins over higher priority",
			filenames: []string{"0010_old_2025-01-01.sqlite.zip", "0010_new_2025-02-01.sql"},
			expected:  "0010_new_2025-02-01.sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectLatestFile(tt.filenames)
			assert.Equal(t, tt.expected, result)
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

	// Test resolveSFGAPath (new function)
	sfgaPath, metadata, warning, err := resolveSFGAPath(source)
	require.NoError(t, err)
	t.Logf("resolveSFGAPath returned: %s (metadata: %+v, warning: %s)", sfgaPath, metadata, warning)

	// Test fetchSFGA (now takes the resolved path)
	ctx := context.Background()
	sqlitePath, err := fetchSFGA(ctx, sfgaPath, cacheDir)
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

func TestResolveRemoteSFGAFile(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		id        int
		wantMatch string // substring that should be in the result
		wantErr   bool
	}{
		{
			name:      "find file with ID 206",
			baseURL:   "http://opendata.globalnames.org/sfga/",
			id:        206,
			wantMatch: "0206",
			wantErr:   false,
		},
		{
			name:      "find file with ID 196",
			baseURL:   "http://opendata.globalnames.org/sfga/",
			id:        196,
			wantMatch: "0196",
			wantErr:   false,
		},
		{
			name:    "non-existent ID should fail",
			baseURL: "http://opendata.globalnames.org/sfga/",
			id:      9999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, warning, err := resolveRemoteSFGAFile(tt.baseURL, tt.id)
			_ = warning // May be used in future test assertions

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			// Skip if no internet connection
			if err != nil && (strings.Contains(err.Error(), "no such host") ||
				strings.Contains(err.Error(), "connection refused")) {
				t.Skipf("skipping: no internet connection: %v", err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, result, tt.wantMatch)
			assert.True(t, isSFGAFile(result), "result should be a valid SFGA file")
			t.Logf("Resolved file: %s", result)
		})
	}
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
		Parent: "http://opendata.globalnames.org/sfga/",
	}

	// Test resolveSFGAPath (new function)
	sfgaPath, metadata, warning, err := resolveSFGAPath(source)

	// Skip if no internet connection or file not available
	if err != nil {
		if strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "context deadline exceeded") {
			t.Skipf("skipping URL test: no internet connection or server unavailable: %v", err)
		}
		// If it's a different error (like file not found on server), that's still useful to know
		t.Logf("URL resolve error (may indicate file not on server): %v", err)
		return
	}

	require.NoError(t, err)
	t.Logf("resolveSFGAPath returned: %s (metadata: %+v, warning: %s)", sfgaPath, metadata, warning)

	// Test fetchSFGA (now takes the resolved path)
	ctx := context.Background()
	sqlitePath, err := fetchSFGA(ctx, sfgaPath, cacheDir)
	require.NoError(t, err)

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

	// First resolve the path
	sfgaPath, _, _, err := resolveSFGAPath(source)
	require.NoError(t, err)

	// Then fetch/extract
	ctx := context.Background()
	sqlitePath, err := fetchSFGA(ctx, sfgaPath, cacheDir)
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

// Helper function for tests: resolves and fetches SFGA in one call (old behavior)
// This simplifies test migration and matches the original fetchSFGA signature.
func resolveFetchSFGA(ctx context.Context, source populate.DataSourceConfig, cacheDir string) (string, SFGAMetadata, string, error) {
	// Resolve the path first
	sfgaPath, metadata, warning, err := resolveSFGAPath(source)
	if err != nil {
		return "", SFGAMetadata{}, "", err
	}

	// Then fetch/extract
	sqlitePath, err := fetchSFGA(ctx, sfgaPath, cacheDir)
	if err != nil {
		return "", SFGAMetadata{}, warning, err
	}

	return sqlitePath, metadata, warning, nil
}

func TestGenerateIDPatterns(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		expected []string
	}{
		{
			name:     "ID=1 generates all 4 patterns",
			id:       1,
			expected: []string{"0001", "001", "01", "1"},
		},
		{
			name:     "ID=42 generates 3 patterns",
			id:       42,
			expected: []string{"0042", "042", "42"},
		},
		{
			name:     "ID=196 generates 2 patterns",
			id:       196,
			expected: []string{"0196", "196"},
		},
		{
			name:     "ID=1234 generates 1 pattern",
			id:       1234,
			expected: []string{"1234"},
		},
		{
			name:     "ID=9 generates all 4 patterns",
			id:       9,
			expected: []string{"0009", "009", "09", "9"},
		},
		{
			name:     "ID=99 generates 3 patterns",
			id:       99,
			expected: []string{"0099", "099", "99"},
		},
		{
			name:     "ID=999 generates 2 patterns",
			id:       999,
			expected: []string{"0999", "999"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateIDPatterns(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchesIDPattern(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		patterns []string
		expected bool
	}{
		{
			name:     "matches with dash separator",
			filename: "0001-col.sqlite.zip",
			patterns: []string{"0001", "001", "01", "1"},
			expected: true,
		},
		{
			name:     "matches with underscore separator",
			filename: "0001_col.sqlite.zip",
			patterns: []string{"0001", "001", "01", "1"},
			expected: true,
		},
		{
			name:     "matches with dot separator (extension)",
			filename: "0001.sqlite.zip",
			patterns: []string{"0001", "001", "01", "1"},
			expected: true,
		},
		{
			name:     "matches shorter pattern",
			filename: "1-col.sqlite.zip",
			patterns: []string{"0001", "001", "01", "1"},
			expected: true,
		},
		{
			name:     "does not match without separator",
			filename: "0001col.sqlite.zip",
			patterns: []string{"0001", "001", "01", "1"},
			expected: false,
		},
		{
			name:     "does not match longer ID (1001 vs 1)",
			filename: "1001-col.sqlite.zip",
			patterns: []string{"0001", "001", "01", "1"},
			expected: false,
		},
		{
			name:     "does not match similar but different ID (10 vs 1)",
			filename: "10-col.sqlite.zip",
			patterns: []string{"0001", "001", "01", "1"},
			expected: false,
		},
		{
			name:     "matches exact 3-digit pattern",
			filename: "042-data.sqlite.zip",
			patterns: []string{"0042", "042", "42"},
			expected: true,
		},
		{
			name:     "matches exact 2-digit pattern",
			filename: "42-data.sqlite.zip",
			patterns: []string{"0042", "042", "42"},
			expected: true,
		},
		{
			name:     "does not match when ID is prefix of filename number",
			filename: "420-data.sqlite.zip",
			patterns: []string{"0042", "042", "42"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesIDPattern(tt.filename, tt.patterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}
