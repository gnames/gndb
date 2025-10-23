package populate_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/gnames/gndb/pkg/populate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectedID      int
		expectedVersion string
		expectedDate    string
	}{
		{
			name:            "minimal format",
			filename:        "1001.sql",
			expectedID:      1001,
			expectedVersion: "",
			expectedDate:    "",
		},
		{
			name:            "full format with version prefix",
			filename:        "0001_col_2025-10-03_v2024.1.sql",
			expectedID:      1,
			expectedVersion: "v2024.1",
			expectedDate:    "2025-10-03",
		},
		{
			name:            "version matching date (common pattern)",
			filename:        "0002_gbif_2024-12-15_2024-12-15.sql.zip",
			expectedID:      2,
			expectedVersion: "2024-12-15",
			expectedDate:    "2024-12-15",
		},
		{
			name:            "date only, no version",
			filename:        "0003_worms_2025-01-01.sqlite",
			expectedID:      3,
			expectedVersion: "",
			expectedDate:    "2025-01-01",
		},
		{
			name:            "complex version string",
			filename:        "0004_itis_2025-02-01_v2025.1-beta.3.sql.zip",
			expectedID:      4,
			expectedVersion: "v2025.1-beta.3",
			expectedDate:    "2025-02-01",
		},
		{
			name:            "version with underscores",
			filename:        "0005_ncbi_2025-03-15_2025_03_15.sqlite",
			expectedID:      5,
			expectedVersion: "2025_03_15",
			expectedDate:    "2025-03-15",
		},
		{
			name:            "sqlite format no date",
			filename:        "1005_custom_source.sqlite",
			expectedID:      1005,
			expectedVersion: "",
			expectedDate:    "",
		},
		{
			name:            "with path",
			filename:        "/path/to/data/0006_worms_2024-12-01_v3.1.4.sql.zip",
			expectedID:      6,
			expectedVersion: "v3.1.4",
			expectedDate:    "2024-12-01",
		},
		{
			name:            "with URL",
			filename:        "https://example.com/data/0025_mydata_2025-03-20_v1.0.sqlite.zip",
			expectedID:      25,
			expectedVersion: "v1.0",
			expectedDate:    "2025-03-20",
		},
		{
			name:            "date only with path",
			filename:        "0100_data_2025-05-01.sql",
			expectedID:      100,
			expectedVersion: "",
			expectedDate:    "2025-05-01",
		},
		{
			name:            "version without v prefix",
			filename:        "0007_source_2025-01-01_2025.1.sql",
			expectedID:      7,
			expectedVersion: "2025.1",
			expectedDate:    "2025-01-01",
		},
		{
			name:            "arbitrary version string",
			filename:        "0008_test_2025-01-01_release-candidate-3.sql",
			expectedID:      8,
			expectedVersion: "release-candidate-3",
			expectedDate:    "2025-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := populate.ParseFilename(tt.filename)
			assert.Equal(t, tt.expectedID, meta.ID, "ID mismatch")
			assert.Equal(t, tt.expectedVersion, meta.Version, "Version mismatch")
			assert.Equal(t, tt.expectedDate, meta.ReleaseDate, "Date mismatch")
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"http URL", "http://example.com/data.sql", true},
		{"https URL", "https://example.com/data.sql", true},
		{"local path", "/path/to/file.sql", false},
		{"relative path", "data/file.sql", false},
		{"current dir", "./file.sql", false},
		{"just filename", "file.sql", false},
		{"ftp not valid", "ftp://example.com/data.sql", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := populate.IsValidURL(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadSourcesConfig_Minimal(t *testing.T) {
	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create parent directory for SFGA files
	parentDir := filepath.Join(tmpDir, "sfga")
	err = os.MkdirAll(parentDir, 0755)
	require.NoError(t, err)

	// Create minimal YAML config
	yamlContent := fmt.Sprintf(`data_sources:
  - id: 1001
    parent: %s
`, parentDir)

	configPath := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load config
	config, err := populate.LoadSourcesConfig(configPath)
	require.NoError(t, err)
	require.Len(t, config.DataSources, 1)

	ds := config.DataSources[0]
	assert.Equal(t, parentDir, ds.Parent)
	assert.Equal(t, 1001, ds.ID)
}

func TestLoadSourcesConfig_FullConfig(t *testing.T) {
	yamlContent := `data_sources:
  - id: 1
    parent: https://example.com/data/
    title: Catalogue of Life
    title_short: CoL
    description: Global taxonomic backbone
    home_url: https://www.catalogueoflife.org
    data_url: https://www.catalogueoflife.org/data
    data_source_type: taxonomic
    is_curated: true
    is_auto_curated: false
    has_classification: true
    is_outlink_ready: true
    outlink_url: "https://www.catalogueoflife.org/data/taxon/{}"
    outlink_id_column: "taxon.col__id"
`
	tmpfile, err := os.CreateTemp("", "sources-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(yamlContent)
	require.NoError(t, err)
	tmpfile.Close()

	config, err := populate.LoadSourcesConfig(tmpfile.Name())
	require.NoError(t, err)
	require.Len(t, config.DataSources, 1)

	ds := config.DataSources[0]
	assert.Equal(t, "https://example.com/data/", ds.Parent)
	assert.Equal(t, 1, ds.ID)
	assert.Equal(t, "Catalogue of Life", ds.Title)
	assert.Equal(t, "CoL", ds.TitleShort)
	assert.Equal(t, "Global taxonomic backbone", ds.Description)
	assert.Equal(t, "https://www.catalogueoflife.org", ds.HomeURL)
	assert.Equal(t, "https://www.catalogueoflife.org/data", ds.DataURL)
	assert.Equal(t, "taxonomic", ds.DataSourceType)
	assert.True(t, ds.IsCurated)
	assert.False(t, ds.IsAutoCurated)
	assert.True(t, ds.HasClassification)
	assert.Equal(t, "https://www.catalogueoflife.org/data/taxon/{}", ds.OutlinkURL)
	assert.True(t, ds.IsOutlinkReady)
	assert.Equal(t, "taxon.col__id", ds.OutlinkIDColumn)
}

func TestLoadSourcesConfig_MultipleDataSources(t *testing.T) {
	// Create temporary directory and parent directories
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create parent directories
	parent1 := filepath.Join(tmpDir, "parent1")
	parent2 := filepath.Join(tmpDir, "parent2")
	err = os.MkdirAll(parent1, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(parent2, 0755)
	require.NoError(t, err)

	yamlContent := fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s
    title: Catalogue of Life
  - id: 2
    parent: %s
    title: GBIF Backbone
  - id: 1001
    parent: https://example.com/sfga/
`, parent1, parent2)

	configPath := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	config, err := populate.LoadSourcesConfig(configPath)
	require.NoError(t, err)
	require.Len(t, config.DataSources, 3)

	assert.Equal(t, 1, config.DataSources[0].ID)
	assert.Equal(t, 2, config.DataSources[1].ID)
	assert.Equal(t, 1001, config.DataSources[2].ID)
}

func TestLoadSourcesConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		yamlTemplate string
		setupFunc    func(tmpDir string) string // returns yaml content with paths
		expectedErr  string
	}{
		{
			name: "missing id",
			setupFunc: func(tmpDir string) string {
				parentDir := filepath.Join(tmpDir, "parent")
				_ = os.MkdirAll(parentDir, 0755)
				return fmt.Sprintf(`data_sources:
  - parent: %s
    title: Test
`, parentDir)
			},
			expectedErr: "id is required",
		},
		{
			name: "missing parent",
			setupFunc: func(tmpDir string) string {
				return `data_sources:
  - id: 1
    title: Test
`
			},
			expectedErr: "parent directory or URL is required",
		},
		{
			name: "parent directory does not exist",
			setupFunc: func(tmpDir string) string {
				return fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s/nonexistent
`, tmpDir)
			},
			expectedErr: "does not exist",
		},
		{
			name: "invalid data_source_type",
			setupFunc: func(tmpDir string) string {
				parentDir := filepath.Join(tmpDir, "parent")
				_ = os.MkdirAll(parentDir, 0755)
				return fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s
    data_source_type: invalid
`, parentDir)
			},
			expectedErr: "invalid data_source_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "sources-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			yamlContent := tt.setupFunc(tmpDir)

			configPath := filepath.Join(tmpDir, "sources.yaml")
			err = os.WriteFile(configPath, []byte(yamlContent), 0644)
			require.NoError(t, err)

			_, err = populate.LoadSourcesConfig(configPath)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestLoadSourcesConfig_FileNotFound(t *testing.T) {
	_, err := populate.LoadSourcesConfig("nonexistent.yaml")
	assert.Error(t, err)
}

func TestLoadSourcesConfig_OutlinkWarnings(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func(tmpDir string) string
		expectedWarning string
	}{
		{
			name: "outlink_ready without outlink_url",
			setupFunc: func(tmpDir string) string {
				parentDir := filepath.Join(tmpDir, "parent")
				_ = os.MkdirAll(parentDir, 0755)
				return fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s
    is_outlink_ready: true
`, parentDir)
			},
			expectedWarning: "outlink_url is required",
		},
		{
			name: "outlink_url without placeholder",
			setupFunc: func(tmpDir string) string {
				parentDir := filepath.Join(tmpDir, "parent")
				_ = os.MkdirAll(parentDir, 0755)
				return fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s
    is_outlink_ready: true
    outlink_url: "https://example.com/taxon/123"
`, parentDir)
			},
			expectedWarning: "must contain {} placeholder",
		},
		{
			name: "invalid outlink_id_column table",
			setupFunc: func(tmpDir string) string {
				parentDir := filepath.Join(tmpDir, "parent")
				_ = os.MkdirAll(parentDir, 0755)
				return fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s
    is_outlink_ready: true
    outlink_url: "https://example.com/{}"
    outlink_id_column: "invalid_table.col__id"
`, parentDir)
			},
			expectedWarning: "invalid table 'invalid_table'",
		},
		{
			name: "invalid outlink_id_column column",
			setupFunc: func(tmpDir string) string {
				parentDir := filepath.Join(tmpDir, "parent")
				_ = os.MkdirAll(parentDir, 0755)
				return fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s
    is_outlink_ready: true
    outlink_url: "https://example.com/{}"
    outlink_id_column: "taxon.invalid_column"
`, parentDir)
			},
			expectedWarning: "column not valid for this table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "sources-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			yamlContent := tt.setupFunc(tmpDir)

			configPath := filepath.Join(tmpDir, "sources.yaml")
			err = os.WriteFile(configPath, []byte(yamlContent), 0644)
			require.NoError(t, err)

			config, err := populate.LoadSourcesConfig(configPath)
			require.NoError(t, err, "Should not return error for outlink issues")
			require.NotNil(t, config)

			// Should have warnings
			assert.Greater(t, len(config.Warnings), 0, "Should have warnings")
			if len(config.Warnings) > 0 {
				assert.Contains(t, config.Warnings[0].Message, tt.expectedWarning)
				assert.Equal(t, 1, config.Warnings[0].DataSourceID)
				assert.NotEmpty(t, config.Warnings[0].Suggestion)
			}

			// Outlink should be disabled
			assert.False(t, config.DataSources[0].IsOutlinkReady, "IsOutlinkReady should be set to false")
		})
	}
}

func TestGenerateExampleConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "sources.yaml")
	err = populate.GenerateExampleConfig(configPath)
	require.NoError(t, err)

	// Verify file was created
	stat, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Greater(t, stat.Size(), int64(100), "Config file should have substantial content")

	// Verify file is valid YAML (just check it can be read)
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "data_sources:", "Should contain data_sources section")
	assert.Contains(t, string(content), "# sources.yaml", "Should contain header comment")
}

func TestGenerateExampleConfig_FileExists(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "sources-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	err = populate.GenerateExampleConfig(tmpfile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

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
			filtered, err := populate.FilterSources(sources, tt.filter)

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
			description: "Range 10-20 should match IDs 10, 15, 20 (silently skip 11-14, 16-19)",
		},
		{
			name:        "range from start",
			filter:      "-10",
			expectedIDs: []int{1, 5, 10},
			expectError: false,
			description: "Range -10 should match IDs 1-10 (silently skip 2-4, 6-9)",
		},
		{
			name:        "range to end",
			filter:      "100-",
			expectedIDs: []int{100, 200, 1000},
			expectError: false,
			description: "Range 100- should match from 100 to max ID",
		},
		{
			name:        "range with all gaps",
			filter:      "25-45",
			expectedIDs: nil,
			expectError: true,
			description: "Range with no matching sources should return error with warning",
		},
		{
			name:        "mix of IDs and ranges",
			filter:      "1,10-20,100",
			expectedIDs: []int{1, 10, 15, 20, 100},
			expectError: false,
			description: "Mix of explicit IDs and ranges",
		},
		{
			name:        "explicit ID not found (should warn)",
			filter:      "1,999,1000",
			expectedIDs: []int{1, 1000},
			expectError: false,
			description: "Explicit ID 999 not found - should warn but continue",
		},
		{
			name:        "range with one gap at boundary",
			filter:      "5-15",
			expectedIDs: []int{5, 10, 15},
			expectError: false,
			description: "Range should work even with gaps (6-9, 11-14 missing)",
		},
		{
			name:        "invalid range format",
			filter:      "10-20-30",
			expectedIDs: nil,
			expectError: true,
			description: "Invalid range format should error",
		},
		{
			name:        "reversed range",
			filter:      "20-10",
			expectedIDs: nil,
			expectError: true,
			description: "Start > end should error",
		},
		{
			name:        "multiple ranges",
			filter:      "1-5,15-20,100-200",
			expectedIDs: []int{1, 5, 15, 20, 100, 200},
			expectError: false,
			description: "Multiple ranges should work",
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
			filtered, err := populate.FilterSources(sources, tt.filter)

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
		})
	}
}

func TestParentPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		id          int
		parentPath  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid local directory",
			id:          123,
			parentPath:  "",
			expectError: false,
		},
		{
			name:        "valid URL",
			id:          1,
			parentPath:  "https://example.com/sfga/",
			expectError: false,
		},
		{
			name:        "nonexistent directory",
			id:          1,
			parentPath:  "/nonexistent/path/to/sfga",
			expectError: true,
			errorMsg:    "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "sources-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create parent directory if needed for local path test
			parentPath := tt.parentPath
			if parentPath == "" {
				parentPath = filepath.Join(tmpDir, "parent")
				err = os.MkdirAll(parentPath, 0755)
				require.NoError(t, err)
			}

			// Create config YAML
			yamlContent := fmt.Sprintf(`data_sources:
  - id: %d
    parent: %s
`, tt.id, parentPath)

			configPath := filepath.Join(tmpDir, "sources.yaml")
			err = os.WriteFile(configPath, []byte(yamlContent), 0644)
			require.NoError(t, err)

			config, err := populate.LoadSourcesConfig(configPath)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.Len(t, config.DataSources, 1)
				assert.Equal(t, tt.id, config.DataSources[0].ID)
				assert.Equal(t, parentPath, config.DataSources[0].Parent)
			}
		})
	}
}

func TestValidateOutlinkIDColumn_ValidFormats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	parentDir := filepath.Join(tmpDir, "parent")
	err = os.MkdirAll(parentDir, 0755)
	require.NoError(t, err)

	tests := []struct {
		name            string
		outlinkIDColumn string
	}{
		// Valid taxon columns
		{
			name:            "taxon.col__id",
			outlinkIDColumn: "taxon.col__id",
		},
		{
			name:            "taxon.col__alternative_id",
			outlinkIDColumn: "taxon.col__alternative_id",
		},
		{
			name:            "taxon.gn__local_id",
			outlinkIDColumn: "taxon.gn__local_id",
		},
		{
			name:            "taxon.gn__global_id",
			outlinkIDColumn: "taxon.gn__global_id",
		},
		// Valid name columns
		{
			name:            "name.col__id",
			outlinkIDColumn: "name.col__id",
		},
		{
			name:            "name.col__alternative_id",
			outlinkIDColumn: "name.col__alternative_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent := fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s
    is_outlink_ready: true
    outlink_url: "https://example.com/taxon/{}"
    outlink_id_column: "%s"
`, parentDir, tt.outlinkIDColumn)

			configPath := filepath.Join(tmpDir, "sources_"+tt.name+".yaml")
			err = os.WriteFile(configPath, []byte(yamlContent), 0644)
			require.NoError(t, err)

			config, err := populate.LoadSourcesConfig(configPath)
			assert.NoError(t, err, "Should accept valid format: %s", tt.outlinkIDColumn)
			if err == nil {
				require.Len(t, config.DataSources, 1)
				assert.Equal(t, tt.outlinkIDColumn, config.DataSources[0].OutlinkIDColumn)
			}
		})
	}
}

func TestValidateOutlinkIDColumn_InvalidFormats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	parentDir := filepath.Join(tmpDir, "parent")
	err = os.MkdirAll(parentDir, 0755)
	require.NoError(t, err)

	tests := []struct {
		name            string
		outlinkIDColumn string
		expectedWarning string
	}{
		{
			name:            "no dot",
			outlinkIDColumn: "taxon_col__id",
			expectedWarning: "invalid outlink_id_column format",
		},
		{
			name:            "too many dots",
			outlinkIDColumn: "taxon.col__id.extra",
			expectedWarning: "invalid outlink_id_column format",
		},
		{
			name:            "invalid table name",
			outlinkIDColumn: "invalid_table.col__id",
			expectedWarning: "invalid table",
		},
		{
			name:            "invalid column name",
			outlinkIDColumn: "taxon.invalid_column",
			expectedWarning: "column not valid for this table",
		},
		{
			name:            "synonym table not supported",
			outlinkIDColumn: "synonym.col__id",
			expectedWarning: "invalid table",
		},
		{
			name:            "empty string",
			outlinkIDColumn: "",
			expectedWarning: "outlink_id_column is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yamlContent string
			if tt.outlinkIDColumn == "" {
				yamlContent = fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s
    is_outlink_ready: true
    outlink_url: "https://example.com/taxon/{}"
`, parentDir)
			} else {
				yamlContent = fmt.Sprintf(`data_sources:
  - id: 1
    parent: %s
    is_outlink_ready: true
    outlink_url: "https://example.com/taxon/{}"
    outlink_id_column: "%s"
`, parentDir, tt.outlinkIDColumn)
			}

			configPath := filepath.Join(tmpDir, "sources_"+tt.name+".yaml")
			err = os.WriteFile(configPath, []byte(yamlContent), 0644)
			require.NoError(t, err)

			config, err := populate.LoadSourcesConfig(configPath)
			require.NoError(t, err, "Should not return error for invalid outlink format: %s", tt.outlinkIDColumn)
			require.NotNil(t, config)

			// Should have warnings
			assert.Greater(t, len(config.Warnings), 0, "Should have warnings for invalid format: %s", tt.outlinkIDColumn)
			if len(config.Warnings) > 0 {
				assert.Contains(t, config.Warnings[0].Message, tt.expectedWarning)
			}

			// Outlink should be disabled
			assert.False(t, config.DataSources[0].IsOutlinkReady, "IsOutlinkReady should be disabled")
		})
	}
}

func TestExtractOutlinkID(t *testing.T) {
	tests := []struct {
		name       string
		columnName string
		value      string
		expected   string
	}{
		{
			name:       "direct column returns value as-is",
			columnName: "col__id",
			value:      "123456",
			expected:   "123456",
		},
		{
			name:       "col__name_id returns value as-is",
			columnName: "col__name_id",
			value:      "name_789",
			expected:   "name_789",
		},
		{
			name:       "col__local_id returns value as-is",
			columnName: "col__local_id",
			value:      "local_123",
			expected:   "local_123",
		},
		{
			name:       "alternative_id with gnoutlink namespace",
			columnName: "col__alternative_id",
			value:      "gnoutlink:Homo_sapiens",
			expected:   "Homo_sapiens",
		},
		{
			name:       "alternative_id with multiple namespaces",
			columnName: "col__alternative_id",
			value:      "wikidata:Q123,gbif:456789,gnoutlink:Species_name",
			expected:   "Species_name",
		},
		{
			name:       "alternative_id with gnoutlink at start",
			columnName: "col__alternative_id",
			value:      "gnoutlink:abc123,wikidata:Q999",
			expected:   "abc123",
		},
		{
			name:       "alternative_id with spaces around commas",
			columnName: "col__alternative_id",
			value:      "wikidata:Q123 , gnoutlink:encoded_id , gbif:789",
			expected:   "encoded_id",
		},
		{
			name:       "alternative_id without gnoutlink namespace",
			columnName: "col__alternative_id",
			value:      "wikidata:Q123,gbif:456",
			expected:   "",
		},
		{
			name:       "alternative_id with empty gnoutlink value",
			columnName: "col__alternative_id",
			value:      "wikidata:Q123,gnoutlink:,gbif:456",
			expected:   "",
		},
		{
			name:       "alternative_id gnoutlink only",
			columnName: "col__alternative_id",
			value:      "gnoutlink:single_value",
			expected:   "single_value",
		},
		{
			name:       "alternative_id with URL-encoded value",
			columnName: "col__alternative_id",
			value:      "gnoutlink:Homo%20sapiens%20%28L.%29",
			expected:   "Homo%20sapiens%20%28L.%29",
		},
		{
			name:       "empty value returns empty",
			columnName: "col__id",
			value:      "",
			expected:   "",
		},
		{
			name:       "alternative_id empty value returns empty",
			columnName: "col__alternative_id",
			value:      "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := populate.ExtractOutlinkID(tt.columnName, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}
