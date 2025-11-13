package populate_test

import (
	"sort"
	"testing"

	"github.com/gnames/gndb/pkg/populate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: I/O tests (LoadSourcesConfig, GenerateExampleConfig) have been moved to
// internal/iopopulate/sources_test.go because they test file system operations.
// This file focuses on pure logic: filtering, parsing, validation.

func TestFilterSources(t *testing.T) {
	// Helper to create test sources
	createSource := func(id int) populate.DataSourceConfig {
		return populate.DataSourceConfig{
			ID:     id,
			Parent: "https://example.com/sfga/",
		}
	}

	sources := []populate.DataSourceConfig{
		createSource(1),
		createSource(5),
		createSource(10),
		createSource(100),
		createSource(1000),
		createSource(1001),
		createSource(2000),
	}

	tests := []struct {
		name        string
		filter      string
		expectedIDs []int
		expectError bool
	}{
		{
			name:        "empty filter returns all",
			filter:      "",
			expectedIDs: []int{1, 5, 10, 100, 1000, 1001, 2000},
			expectError: false,
		},
		{
			name:        "main filter returns ID < 1000",
			filter:      "main",
			expectedIDs: []int{1, 5, 10, 100},
			expectError: false,
		},
		{
			name:        "exclude main returns ID >= 1000",
			filter:      "exclude main",
			expectedIDs: []int{1000, 1001, 2000},
			expectError: false,
		},
		{
			name:        "single ID",
			filter:      "5",
			expectedIDs: []int{5},
			expectError: false,
		},
		{
			name:        "comma-separated IDs",
			filter:      "1,10,1001",
			expectedIDs: []int{1, 10, 1001},
			expectError: false,
		},
		{
			name:        "comma-separated with spaces",
			filter:      "5, 100, 2000",
			expectedIDs: []int{5, 100, 2000},
			expectError: false,
		},
		{
			name:        "non-existent ID returns error with warning",
			filter:      "9999",
			expectedIDs: nil,
			expectError: true,
		},
		{
			name:        "invalid ID format",
			filter:      "abc",
			expectedIDs: nil,
			expectError: true,
		},
		{
			name:        "mixed valid and invalid",
			filter:      "1,invalid,5",
			expectedIDs: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, warnings, err := populate.FilterSources(sources, tt.filter)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, filtered, len(tt.expectedIDs))

				actualIDs := make([]int, len(filtered))
				for i, src := range filtered {
					actualIDs[i] = src.ID
				}
				assert.Equal(t, tt.expectedIDs, actualIDs)
			}
			// Warnings are optional - caller can check if needed
			_ = warnings
		})
	}
}

func TestFilterSources_Ranges(t *testing.T) {
	// Create test sources with gaps: 1, 5, 10, 15, 20, 50, 100, 200, 1000
	createSource := func(id int) populate.DataSourceConfig {
		return populate.DataSourceConfig{
			ID:     id,
			Parent: "https://example.com/sfga/",
		}
	}

	sources := []populate.DataSourceConfig{
		createSource(1),
		createSource(5),
		createSource(10),
		createSource(15),
		createSource(20),
		createSource(50),
		createSource(100),
		createSource(200),
		createSource(1000),
	}

	tests := []struct {
		name        string
		filter      string
		expectedIDs []int
		expectError bool
		description string
	}{
		{
			name:        "simple range",
			filter:      "10-20",
			expectedIDs: []int{10, 15, 20},
			expectError: false,
			description: "Should include all IDs in range [10,20]",
		},
		{
			name:        "range from start",
			filter:      "-20",
			expectedIDs: []int{1, 5, 10, 15, 20},
			expectError: false,
			description: "Should include all IDs from 1 to 20",
		},
		{
			name:        "range to end",
			filter:      "100-",
			expectedIDs: []int{100, 200, 1000},
			expectError: false,
			description: "Should include all IDs from 100 to end",
		},
		{
			name:        "mixed ranges and individual IDs",
			filter:      "1,10-20,100",
			expectedIDs: []int{1, 10, 15, 20, 100},
			expectError: false,
			description: "Should combine individual IDs and ranges",
		},
		{
			name:        "overlapping ranges and IDs",
			filter:      "1-10,5,10-20",
			expectedIDs: []int{1, 5, 10, 15, 20},
			expectError: false,
			description: "Overlapping ranges should deduplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, warnings, err := populate.FilterSources(sources, tt.filter)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				require.NoError(t, err, tt.description)

				actualIDs := make([]int, len(filtered))
				for i, src := range filtered {
					actualIDs[i] = src.ID
				}

				// Sort for comparison (order may vary)
				sort.Ints(actualIDs)
				expectedSorted := make([]int, len(tt.expectedIDs))
				copy(expectedSorted, tt.expectedIDs)
				sort.Ints(expectedSorted)

				assert.Equal(t, expectedSorted, actualIDs, tt.description)
			}
			// Warnings are optional - caller can check if needed
			_ = warnings
		})
	}
}

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectedID      int
		expectedVersion string
		expectedDate    string
	}{
		{
			name:            "full format with version",
			filename:        "0001_col_2025-10-03_v2024.1.sqlite.zip",
			expectedID:      1,
			expectedVersion: "v2024.1",
			expectedDate:    "2025-10-03",
		},
		{
			name:            "date as version",
			filename:        "0002_gbif_2024-12-15_2024-12-15.sql.zip",
			expectedID:      2,
			expectedVersion: "2024-12-15",
			expectedDate:    "2024-12-15",
		},
		{
			name:            "no version",
			filename:        "0003_worms_2025-01-01.sqlite",
			expectedID:      3,
			expectedVersion: "",
			expectedDate:    "2025-01-01",
		},
		{
			name:            "minimal format",
			filename:        "1001.sql",
			expectedID:      1001,
			expectedVersion: "",
			expectedDate:    "",
		},
		{
			name:            "with path",
			filename:        "/data/sfga/0001_col_2025-10-03_v2024.1.sqlite.zip",
			expectedID:      1,
			expectedVersion: "v2024.1",
			expectedDate:    "2025-10-03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := populate.ParseFilename(tt.filename)
			assert.Equal(t, tt.expectedID, metadata.ID)
			assert.Equal(t, tt.expectedVersion, metadata.Version)
			assert.Equal(t, tt.expectedDate, metadata.ReleaseDate)
		})
	}
}

func TestExtractOutlinkID(t *testing.T) {
	tests := []struct {
		name     string
		column   string
		value    string
		expected string
	}{
		{
			name:     "regular column returns value as-is",
			column:   "taxon.col__id",
			value:    "12345",
			expected: "12345",
		},
		{
			name:     "alternative_id with gnoutlink namespace",
			column:   "taxon.col__alternative_id",
			value:    "wikidata:Q123,gnoutlink:Homo_sapiens",
			expected: "Homo_sapiens",
		},
		{
			name:     "name alternative_id with gnoutlink",
			column:   "name.col__alternative_id",
			value:    "gnoutlink:url-encoded-name",
			expected: "url-encoded-name",
		},
		{
			name:     "alternative_id without gnoutlink",
			column:   "taxon.col__alternative_id",
			value:    "wikidata:Q123",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := populate.ExtractOutlinkID(tt.column, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid HTTP URL",
			input:    "http://example.com",
			expected: true,
		},
		{
			name:     "valid HTTPS URL",
			input:    "https://example.com/path",
			expected: true,
		},
		{
			name:     "local path",
			input:    "/home/user/data",
			expected: false,
		},
		{
			name:     "relative path",
			input:    "./data",
			expected: false,
		},
		{
			name:     "tilde path",
			input:    "~/data",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := populate.IsValidURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
