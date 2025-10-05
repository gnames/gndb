package populate_test

import (
	"fmt"
	"os"
	"path/filepath"
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
			name:            "full format with all metadata",
			filename:        "0001_col_2025-10-03_v1.2.3.sql",
			expectedID:      1,
			expectedVersion: "1.2.3",
			expectedDate:    "2025-10-03",
		},
		{
			name:            "with zip compression",
			filename:        "0012_gbif_2025-01-15_v2.0.sql.zip",
			expectedID:      12,
			expectedVersion: "2.0",
			expectedDate:    "2025-01-15",
		},
		{
			name:            "sqlite format",
			filename:        "1005_custom_source.sqlite",
			expectedID:      1005,
			expectedVersion: "",
			expectedDate:    "",
		},
		{
			name:            "with path",
			filename:        "/path/to/data/0003_worms_2024-12-01_v3.1.4.sql.zip",
			expectedID:      3,
			expectedVersion: "3.1.4",
			expectedDate:    "2024-12-01",
		},
		{
			name:            "with URL",
			filename:        "https://example.com/data/0025_mydata_2025-03-20_v1.0.sqlite.zip",
			expectedID:      25,
			expectedVersion: "1.0",
			expectedDate:    "2025-03-20",
		},
		{
			name:            "version without dots",
			filename:        "0007_source_v10.sql",
			expectedID:      7,
			expectedVersion: "10",
			expectedDate:    "",
		},
		{
			name:            "date only",
			filename:        "0100_data_2025-05-01.sql",
			expectedID:      100,
			expectedVersion: "",
			expectedDate:    "2025-05-01",
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
	// Create a temporary test file
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test SFGA file
	sfgaPath := filepath.Join(tmpDir, "1001.sql")
	err = os.WriteFile(sfgaPath, []byte("-- test sql"), 0644)
	require.NoError(t, err)

	// Create minimal YAML config
	yamlContent := fmt.Sprintf(`data_sources:
  - file: %s
`, sfgaPath)

	configPath := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load config
	config, err := populate.LoadSourcesConfig(configPath)
	require.NoError(t, err)
	require.Len(t, config.DataSources, 1)

	ds := config.DataSources[0]
	assert.Equal(t, sfgaPath, ds.File)
	assert.Equal(t, 1001, *ds.ID)
}

func TestLoadSourcesConfig_FullConfig(t *testing.T) {
	yamlContent := `data_sources:
  - file: https://example.com/data/0001_col_2025-10-03_v1.2.3.sql.zip
    id: 1
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
    outlink_id_field: record_id
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
	assert.Equal(t, "https://example.com/data/0001_col_2025-10-03_v1.2.3.sql.zip", ds.File)
	assert.Equal(t, 1, *ds.ID)
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
	assert.Equal(t, "record_id", ds.OutlinkIDField)
}

func TestLoadSourcesConfig_MultipleDataSources(t *testing.T) {
	// Create temporary directory and test files
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test SFGA files
	file1 := filepath.Join(tmpDir, "0001_col.sql")
	file2 := filepath.Join(tmpDir, "0002_gbif.sql")
	err = os.WriteFile(file1, []byte("-- test sql"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("-- test sql"), 0644)
	require.NoError(t, err)

	yamlContent := fmt.Sprintf(`data_sources:
  - file: %s
    title: Catalogue of Life
  - file: %s
    title: GBIF Backbone
  - file: https://example.com/1001_custom.sql
`, file1, file2)

	configPath := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	config, err := populate.LoadSourcesConfig(configPath)
	require.NoError(t, err)
	require.Len(t, config.DataSources, 3)

	assert.Equal(t, 1, *config.DataSources[0].ID)
	assert.Equal(t, 2, *config.DataSources[1].ID)
	assert.Equal(t, 1001, *config.DataSources[2].ID)
}

func TestLoadSourcesConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name         string
		yamlTemplate string
		setupFunc    func(tmpDir string) string // returns yaml content with paths
		expectedErr  string
	}{
		{
			name: "missing file",
			yamlTemplate: `data_sources:
  - id: 1
    title: Test
`,
			setupFunc: func(tmpDir string) string {
				return `data_sources:
  - id: 1
    title: Test
`
			},
			expectedErr: "file path is required",
		},
		{
			name: "no ID in filename",
			setupFunc: func(tmpDir string) string {
				// Create file without ID in name
				testFile := filepath.Join(tmpDir, "noID.sql")
				_ = os.WriteFile(testFile, []byte("-- test"), 0644)
				return fmt.Sprintf(`data_sources:
  - file: %s
`, testFile)
			},
			expectedErr: "cannot extract ID",
		},
		{
			name: "file does not exist",
			setupFunc: func(tmpDir string) string {
				return fmt.Sprintf(`data_sources:
  - file: %s/nonexistent_0001.sql
`, tmpDir)
			},
			expectedErr: "does not exist",
		},
		{
			name: "invalid data_source_type",
			setupFunc: func(tmpDir string) string {
				testFile := filepath.Join(tmpDir, "0001_test.sql")
				_ = os.WriteFile(testFile, []byte("-- test"), 0644)
				return fmt.Sprintf(`data_sources:
  - file: %s
    data_source_type: invalid
`, testFile)
			},
			expectedErr: "invalid data_source_type",
		},
		{
			name: "outlink_ready without outlink_url",
			setupFunc: func(tmpDir string) string {
				testFile := filepath.Join(tmpDir, "0001_test.sql")
				_ = os.WriteFile(testFile, []byte("-- test"), 0644)
				return fmt.Sprintf(`data_sources:
  - file: %s
    is_outlink_ready: true
`, testFile)
			},
			expectedErr: "outlink_url is required",
		},
		{
			name: "outlink_url without placeholder",
			setupFunc: func(tmpDir string) string {
				testFile := filepath.Join(tmpDir, "0001_test.sql")
				_ = os.WriteFile(testFile, []byte("-- test"), 0644)
				return fmt.Sprintf(`data_sources:
  - file: %s
    is_outlink_ready: true
    outlink_url: "https://example.com/taxon/123"
`, testFile)
			},
			expectedErr: "must contain {} placeholder",
		},
		{
			name: "invalid outlink_id_field",
			setupFunc: func(tmpDir string) string {
				testFile := filepath.Join(tmpDir, "0001_test.sql")
				_ = os.WriteFile(testFile, []byte("-- test"), 0644)
				return fmt.Sprintf(`data_sources:
  - file: %s
    is_outlink_ready: true
    outlink_url: "https://example.com/taxon/{}"
    outlink_id_field: "invalid_field"
`, testFile)
			},
			expectedErr: "invalid outlink_id_field",
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

func TestIDExtraction_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectedID  int
		expectError bool
	}{
		{
			name:        "ID from filename only",
			filename:    "0123_data.sql",
			expectedID:  123,
			expectError: false,
		},
		{
			name:        "ID from yaml only (should fail - need ID in filename)",
			filename:    "data_source.sql",
			expectedID:  0,
			expectError: true,
		},
		{
			name:        "leading zeros preserved in filename",
			filename:    "0001_col.sql",
			expectedID:  1,
			expectError: false,
		},
		{
			name:        "four digit ID",
			filename:    "9999_test.sql",
			expectedID:  9999,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "sources-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create the SFGA file
			sfgaPath := filepath.Join(tmpDir, tt.filename)
			err = os.WriteFile(sfgaPath, []byte("-- test sql"), 0644)
			require.NoError(t, err)

			// Create config YAML
			yamlContent := fmt.Sprintf(`data_sources:
  - file: %s
`, sfgaPath)

			configPath := filepath.Join(tmpDir, "sources.yaml")
			err = os.WriteFile(configPath, []byte(yamlContent), 0644)
			require.NoError(t, err)

			config, err := populate.LoadSourcesConfig(configPath)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, config.DataSources, 1)
				assert.Equal(t, tt.expectedID, *config.DataSources[0].ID)
			}
		})
	}
}
