package iosources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSourcesConfig_Minimal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses file system in short mode")
	}

	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create parent directory for SFGA files
	parentDir := filepath.Join(tmpDir, "sfga")
	err = os.MkdirAll(parentDir, 0755)
	require.NoError(t, err)

	// Create minimal sources.yaml
	yamlContent := `
data_sources:
  - id: 1001
    parent: ` + parentDir + `
`

	configPath := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load config
	config, err := loadSourcesConfig(configPath)
	require.NoError(t, err)
	require.Len(t, config.DataSources, 1)

	// Check first data source
	ds := config.DataSources[0]
	assert.Equal(t, 1001, ds.ID)
}

func TestLoadSourcesConfig_FileNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses file system in short mode")
	}

	_, err := loadSourcesConfig("nonexistent.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read sources config file")
}

func TestLoadSourcesConfig_DirectoryNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses file system in short mode")
	}

	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create sources.yaml with non-existent parent directory
	yamlContent := `
data_sources:
  - id: 1001
    parent: /nonexistent/directory/that/does/not/exist
`

	configPath := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load config - should fail with directory not found
	_, err = loadSourcesConfig(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parent directory does not exist")
}

func TestLoadSourcesConfig_URLsSkipFileSystemCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses file system in short mode")
	}

	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "sources-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create sources.yaml with URL parent (no file system check)
	yamlContent := `
data_sources:
  - id: 1
    parent: http://opendata.globalnames.org/sfga/
`

	configPath := filepath.Join(tmpDir, "sources.yaml")
	err = os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load config - should succeed even though URL doesn't exist locally
	config, err := loadSourcesConfig(configPath)
	require.NoError(t, err)
	require.Len(t, config.DataSources, 1)
	assert.Equal(t, 1, config.DataSources[0].ID)
	assert.Equal(
		t,
		"http://opendata.globalnames.org/sfga/",
		config.DataSources[0].Parent,
	)
}
