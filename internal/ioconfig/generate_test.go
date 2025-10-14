package ioconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gnames/gndb/pkg/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	configDir, err := GetConfigDir()
	require.NoError(t, err)

	expectedDir := filepath.Join(tempHome, ".config", "gndb")
	assert.Equal(t, expectedDir, configDir)
}

func TestGetCacheDir(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	cacheDir, err := GetCacheDir()
	require.NoError(t, err)

	expectedDir := filepath.Join(tempHome, ".cache", "gndb")
	assert.Equal(t, expectedDir, cacheDir)
}

func TestGetDefaultConfigPath(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	configPath, err := GetDefaultConfigPath()
	require.NoError(t, err)

	expectedDir := filepath.Join(tempHome, ".config", "gndb")
	expectedPath := filepath.Join(expectedDir, "config.yaml")

	assert.Equal(t, expectedPath, configPath)
	assert.True(t, filepath.IsAbs(configPath))
}

func TestGenerateDefaultConfig(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	t.Run("creates config and sources files", func(t *testing.T) {
		configPath, err := GenerateDefaultConfig()
		require.NoError(t, err)

		// Verify config.yaml
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, templates.ConfigYAML, string(content))
		err = ValidateGeneratedConfig(configPath)
		assert.NoError(t, err, "generated config should be valid")

		// Verify sources.yaml
		sourcesPath := filepath.Join(filepath.Dir(configPath), "sources.yaml")
		content, err = os.ReadFile(sourcesPath)
		require.NoError(t, err)
		assert.Equal(t, templates.SourcesYAML, string(content))

		// Clean up created files for next subtest if needed, but t.TempDir handles it.
		os.Remove(configPath)
		os.Remove(sourcesPath)
	})

	t.Run("does not overwrite existing files", func(t *testing.T) {
		configPath, err := GetDefaultConfigPath()
		require.NoError(t, err)
		sourcesPath, err := GetDefaultSourcesPath()
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Dir(configPath), 0755)
		require.NoError(t, err)

		// Pre-create config.yaml
		existingContent := "existing config"
		err = os.WriteFile(configPath, []byte(existingContent), 0644)
		require.NoError(t, err)

		// Generate should not overwrite config.yaml but should create sources.yaml
		_, err = GenerateDefaultConfig()
		require.NoError(t, err)

		// Verify config.yaml was not changed
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, existingContent, string(content))

		// Verify sources.yaml was created
		_, err = os.Stat(sourcesPath)
		assert.NoError(t, err, "sources.yaml should have been created")

		os.Remove(configPath)
		os.Remove(sourcesPath)
	})

	t.Run("errors if both files exist", func(t *testing.T) {
		configPath, err := GetDefaultConfigPath()
		require.NoError(t, err)
		sourcesPath, err := GetDefaultSourcesPath()
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Dir(configPath), 0755)
		require.NoError(t, err)

		// Pre-create both files
		err = os.WriteFile(configPath, []byte("dummy"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(sourcesPath, []byte("dummy"), 0644)
		require.NoError(t, err)

		// Generate should return an error
		_, err = GenerateDefaultConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config files already exist")
	})
}

func TestConfigFileExists(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Case 1: File does not exist
	exists, err := ConfigFileExists()
	require.NoError(t, err)
	assert.False(t, exists)

	// Case 2: File exists
	configPath, err := GetDefaultConfigPath()
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	require.NoError(t, err)
	file, err := os.Create(configPath)
	require.NoError(t, err)
	file.Close()

	exists, err = ConfigFileExists()
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestValidateGeneratedConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		err := os.WriteFile(configPath, []byte(templates.ConfigYAML), 0644)
		require.NoError(t, err)

		err = ValidateGeneratedConfig(configPath)
		assert.NoError(t, err)
	})

	t.Run("invalid config", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		invalidYAML := "database: { port: not-a-number }"
		err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
		require.NoError(t, err)

		err = ValidateGeneratedConfig(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid YAML")
	})
}
