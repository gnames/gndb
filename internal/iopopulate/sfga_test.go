package iopopulate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateIDPatterns(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		expected []string
	}{
		{
			name:     "single digit ID",
			id:       1,
			expected: []string{"0001", "001", "01", "1"},
		},
		{
			name:     "two digit ID",
			id:       42,
			expected: []string{"0042", "042", "42"},
		},
		{
			name:     "three digit ID",
			id:       196,
			expected: []string{"0196", "196"},
		},
		{
			name:     "four digit ID",
			id:       1234,
			expected: []string{"1234"},
		},
		{
			name:     "boundary 10",
			id:       10,
			expected: []string{"0010", "010", "10"},
		},
		{
			name:     "boundary 100",
			id:       100,
			expected: []string{"0100", "100"},
		},
		{
			name:     "boundary 1000",
			id:       1000,
			expected: []string{"1000"},
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
			name:     "matches dash separator",
			filename: "0001-file.sqlite",
			patterns: []string{"0001", "001", "01", "1"},
			expected: true,
		},
		{
			name:     "matches underscore separator",
			filename: "0001_file.sqlite",
			patterns: []string{"0001", "001", "01", "1"},
			expected: true,
		},
		{
			name:     "matches dot (extension only)",
			filename: "0001.sqlite",
			patterns: []string{"0001", "001", "01", "1"},
			expected: true,
		},
		{
			name:     "no match - different ID",
			filename: "0002-file.sqlite",
			patterns: []string{"0001", "001", "01", "1"},
			expected: false,
		},
		{
			name:     "no match - ID is substring",
			filename: "1001-file.sqlite",
			patterns: []string{"0001", "001", "01", "1"},
			expected: false,
		},
		{
			name:     "matches shorter pattern",
			filename: "01-file.sqlite",
			patterns: []string{"0001", "001", "01", "1"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesIDPattern(tt.filename, tt.patterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSFGAFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "sql extension",
			filename: "0001.sql",
			expected: true,
		},
		{
			name:     "sql.zip extension",
			filename: "0001.sql.zip",
			expected: true,
		},
		{
			name:     "sqlite extension",
			filename: "0001.sqlite",
			expected: true,
		},
		{
			name:     "sqlite.zip extension",
			filename: "0001.sqlite.zip",
			expected: true,
		},
		{
			name:     "txt extension",
			filename: "0001.txt",
			expected: false,
		},
		{
			name:     "csv extension",
			filename: "0001.csv",
			expected: false,
		},
		{
			name:     "no extension",
			filename: "0001",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSFGAFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFileTypePriority(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected int
	}{
		{
			name:     "sqlite.zip has highest priority",
			filename: "0001.sqlite.zip",
			expected: 4,
		},
		{
			name:     "sql.zip has second priority",
			filename: "0001.sql.zip",
			expected: 3,
		},
		{
			name:     "sqlite has third priority",
			filename: "0001.sqlite",
			expected: 2,
		},
		{
			name:     "sql has fourth priority",
			filename: "0001.sql",
			expected: 1,
		},
		{
			name:     "unknown extension",
			filename: "0001.txt",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFileTypePriority(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSelectLatestFile(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "empty list",
			files:    []string{},
			expected: "",
		},
		{
			name:     "single file",
			files:    []string{"0001.sqlite"},
			expected: "0001.sqlite",
		},
		{
			name:     "select latest date",
			files:    []string{"0001-2023-01-01.sqlite", "0001-2024-01-01.sqlite"},
			expected: "0001-2024-01-01.sqlite",
		},
		{
			name:     "same date - prefer sqlite.zip",
			files:    []string{"0001-2024-01-01.sql", "0001-2024-01-01.sqlite.zip"},
			expected: "0001-2024-01-01.sqlite.zip",
		},
		{
			name:     "no dates - use priority",
			files:    []string{"0001.sql", "0001.sqlite.zip"},
			expected: "0001.sqlite.zip",
		},
		{
			name:     "mixed dates and no dates",
			files:    []string{"0001.sqlite", "0001-2020-01-01.sqlite"},
			expected: "0001-2020-01-01.sqlite",
		},
		{
			name:     "multiple dates - select latest",
			files: []string{
				"0001-2022-06-15.sql",
				"0001-2023-12-01.sqlite",
				"0001-2023-01-01.sqlite.zip",
			},
			expected: "0001-2023-12-01.sqlite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectLatestFile(tt.files)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSFGAFilename(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		wantVersion  string
		wantRevDate  string
	}{
		{
			name:         "full metadata with version",
			filename:     "1000_ruhoff_2023-08-22_v1.0.0.sqlite.zip",
			wantVersion:  "1.0.0",
			wantRevDate:  "2023-08-22",
		},
		{
			name:         "date only",
			filename:     "0147-vascan-2025-08-25.sqlite.zip",
			wantVersion:  "",
			wantRevDate:  "2025-08-25",
		},
		{
			name:         "no metadata",
			filename:     "1000.sql",
			wantVersion:  "",
			wantRevDate:  "",
		},
		{
			name:         "version with two parts",
			filename:     "0001_test_2024-01-01_v2.3.sqlite",
			wantVersion:  "2.3",
			wantRevDate:  "2024-01-01",
		},
		{
			name:         "version without v prefix",
			filename:     "0001_test_2024-01-01_1.5.2.sqlite",
			wantVersion:  "1.5.2",
			wantRevDate:  "2024-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSFGAFilename(tt.filename)
			assert.Equal(t, tt.filename, result.Filename)
			assert.Equal(t, tt.wantVersion, result.Version)
			assert.Equal(t, tt.wantRevDate, result.RevisionDate)
		})
	}
}

func TestResolveSFGAFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses file system in short mode")
	}

	// Create temporary directory with test files.
	tmpDir := t.TempDir()

	// Create test files.
	testFiles := []string{
		"0001-test-2024-01-01.sqlite",
		"0002-test-2023-06-15.sqlite",
		"0002-test-2024-06-15.sqlite.zip",
		"0003.sql",
	}
	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		err := os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	tests := []struct {
		name        string
		id          int
		wantFile    string
		wantWarning bool
		wantErr     bool
	}{
		{
			name:        "single match",
			id:          1,
			wantFile:    "0001-test-2024-01-01.sqlite",
			wantWarning: false,
			wantErr:     false,
		},
		{
			name:        "multiple matches - select latest",
			id:          2,
			wantFile:    "0002-test-2024-06-15.sqlite.zip",
			wantWarning: true,
			wantErr:     false,
		},
		{
			name:        "simple file without date",
			id:          3,
			wantFile:    "0003.sql",
			wantWarning: false,
			wantErr:     false,
		},
		{
			name:        "no match",
			id:          999,
			wantFile:    "",
			wantWarning: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, warning, err := resolveSFGAFile(tmpDir, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, filepath.Join(tmpDir, tt.wantFile), path)

			if tt.wantWarning {
				assert.NotEmpty(t, warning)
			} else {
				assert.Empty(t, warning)
			}
		})
	}
}
