package iofs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureDirs_CreatesDirectories verifies all required
// directories are created.
func TestEnsureDirs_CreatesDirectories(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	// Verify config directory exists
	configDir := filepath.Join(tmpDir, ".config", "gndb")
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir(),
		"Config directory should exist")

	// Verify cache directory exists
	cacheDir := filepath.Join(tmpDir, ".cache", "gndb")
	info, err = os.Stat(cacheDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir(),
		"Cache directory should exist")

	// Verify log directory exists
	logDir := filepath.Join(tmpDir, ".local", "share", "gndb",
		"logs")
	info, err = os.Stat(logDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir(),
		"Log directory should exist")
}

// TestEnsureDirs_Idempotent verifies multiple calls work.
func TestEnsureDirs_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// First call
	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	// Second call should succeed
	err = EnsureDirs(tmpDir)
	require.NoError(t, err)

	// Third call should still succeed
	err = EnsureDirs(tmpDir)
	require.NoError(t, err)
}

// TestEnsureDirs_PermissionsCorrect verifies directory
// permissions are set correctly.
func TestEnsureDirs_PermissionsCorrect(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	configDir := filepath.Join(tmpDir, ".config", "gndb")
	info, err := os.Stat(configDir)
	require.NoError(t, err)

	// Check permissions (0755)
	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0755), mode,
		"Directory should have 0755 permissions")
}

// TestTouchDir_CreatesNewDirectory verifies new directory
// creation.
func TestTouchDir_CreatesNewDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "test", "subdir")

	err := touchDir(newDir)
	require.NoError(t, err)

	info, err := os.Stat(newDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestTouchDir_ExistingDirectory verifies existing directory
// is not modified.
func TestTouchDir_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	existingDir := filepath.Join(tmpDir, "existing")

	// Create directory first
	err := os.MkdirAll(existingDir, 0755)
	require.NoError(t, err)

	// Get original info
	originalInfo, err := os.Stat(existingDir)
	require.NoError(t, err)

	// Call touchDir on existing directory
	err = touchDir(existingDir)
	require.NoError(t, err)

	// Verify directory still exists and unchanged
	newInfo, err := os.Stat(existingDir)
	require.NoError(t, err)
	assert.True(t, newInfo.IsDir())
	assert.Equal(t, originalInfo.Mode(), newInfo.Mode())
}

// TestEnsureConfigFile_CreatesFile verifies config file
// is created.
func TestEnsureConfigFile_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()

	// First ensure directories exist
	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	// Create config file
	err = EnsureConfigFile(tmpDir)
	require.NoError(t, err)

	// Verify file exists
	configPath := filepath.Join(tmpDir, ".config", "gndb",
		"config.yaml")
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.False(t, info.IsDir(),
		"Config file should be a file, not directory")

	// Verify file is not empty
	assert.Greater(t, info.Size(), int64(0),
		"Config file should not be empty")
}

// TestEnsureConfigFile_ContentCorrect verifies config file
// content matches embedded template.
func TestEnsureConfigFile_ContentCorrect(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	err = EnsureConfigFile(tmpDir)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, ".config", "gndb",
		"config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Verify content matches embedded ConfigYAML
	assert.Equal(t, ConfigYAML, string(content),
		"Config file content should match embedded template")
}

// TestEnsureConfigFile_Idempotent verifies existing file
// is not overwritten.
func TestEnsureConfigFile_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	// Create config file
	err = EnsureConfigFile(tmpDir)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, ".config", "gndb",
		"config.yaml")

	// Modify the file
	customContent := "# Custom config\ndatabase:\n  host: myhost"
	err = os.WriteFile(configPath, []byte(customContent),
		0644)
	require.NoError(t, err)

	// Call EnsureConfigFile again
	err = EnsureConfigFile(tmpDir)
	require.NoError(t, err)

	// Verify file still has custom content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, customContent, string(content),
		"Existing config file should not be overwritten")
}

// TestEnsureConfigFile_PermissionsCorrect verifies file
// permissions are set correctly.
func TestEnsureConfigFile_PermissionsCorrect(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	err = EnsureConfigFile(tmpDir)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, ".config", "gndb",
		"config.yaml")
	info, err := os.Stat(configPath)
	require.NoError(t, err)

	// Check permissions (0644)
	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0644), mode,
		"Config file should have 0644 permissions")
}

// TestConfigYAML_Embedded verifies embedded config is
// not empty.
func TestConfigYAML_Embedded(t *testing.T) {
	assert.NotEmpty(t, ConfigYAML,
		"Embedded ConfigYAML should not be empty")
	assert.Contains(t, ConfigYAML, "database",
		"ConfigYAML should contain database section")
	assert.Contains(t, ConfigYAML, "log",
		"ConfigYAML should contain log section")
}

// TestEnsureSourcesFile_CreatesFile verifies sources file
// is created.
func TestEnsureSourcesFile_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()

	// First ensure directories exist
	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	// Create sources file
	err = EnsureSourcesFile(tmpDir)
	require.NoError(t, err)

	// Verify file exists
	sourcesPath := filepath.Join(tmpDir, ".config", "gndb",
		"sources.yaml")
	info, err := os.Stat(sourcesPath)
	require.NoError(t, err)
	assert.False(t, info.IsDir(),
		"Sources file should be a file, not directory")

	// Verify file is not empty
	assert.Greater(t, info.Size(), int64(0),
		"Sources file should not be empty")
}

// TestEnsureSourcesFile_ContentCorrect verifies sources
// file content matches embedded template.
func TestEnsureSourcesFile_ContentCorrect(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	err = EnsureSourcesFile(tmpDir)
	require.NoError(t, err)

	sourcesPath := filepath.Join(tmpDir, ".config", "gndb",
		"sources.yaml")
	content, err := os.ReadFile(sourcesPath)
	require.NoError(t, err)

	// Verify content matches embedded SourcesYAML
	assert.Equal(t, SourcesYAML, string(content),
		"Sources file content should match embedded template")
}

// TestEnsureSourcesFile_Idempotent verifies existing file
// is not overwritten.
func TestEnsureSourcesFile_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	// Create sources file
	err = EnsureSourcesFile(tmpDir)
	require.NoError(t, err)

	sourcesPath := filepath.Join(tmpDir, ".config", "gndb",
		"sources.yaml")

	// Modify the file
	customContent := "# Custom sources\ndata_sources:\n  - id: 999"
	err = os.WriteFile(sourcesPath, []byte(customContent),
		0644)
	require.NoError(t, err)

	// Call EnsureSourcesFile again
	err = EnsureSourcesFile(tmpDir)
	require.NoError(t, err)

	// Verify file still has custom content
	content, err := os.ReadFile(sourcesPath)
	require.NoError(t, err)
	assert.Equal(t, customContent, string(content),
		"Existing sources file should not be overwritten")
}

// TestEnsureSourcesFile_PermissionsCorrect verifies file
// permissions are set correctly.
func TestEnsureSourcesFile_PermissionsCorrect(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureDirs(tmpDir)
	require.NoError(t, err)

	err = EnsureSourcesFile(tmpDir)
	require.NoError(t, err)

	sourcesPath := filepath.Join(tmpDir, ".config", "gndb",
		"sources.yaml")
	info, err := os.Stat(sourcesPath)
	require.NoError(t, err)

	// Check permissions (0644)
	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0644), mode,
		"Sources file should have 0644 permissions")
}

// TestSourcesYAML_Embedded verifies embedded sources is
// not empty.
func TestSourcesYAML_Embedded(t *testing.T) {
	assert.NotEmpty(t, SourcesYAML,
		"Embedded SourcesYAML should not be empty")
	assert.Contains(t, SourcesYAML, "data_sources",
		"SourcesYAML should contain data_sources section")
	assert.Contains(t, SourcesYAML, "Catalogue of Life",
		"SourcesYAML should contain example sources")
	assert.Contains(t, SourcesYAML, "outlink_url",
		"SourcesYAML should document outlink configuration")
}
