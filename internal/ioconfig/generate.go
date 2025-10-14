package ioconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/templates"
	"gopkg.in/yaml.v3"
)

// GetConfigDir returns the configuration directory for GNdb.
// Uses ~/.config/gndb/ on all platforms (Linux, macOS, Windows) for consistency.
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "gndb")
	return configDir, nil
}

// GetCacheDir returns the cache directory for GNdb.
// Uses ~/.cache/gndb/ on all platforms (Linux, macOS, Windows) for consistency.
func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".cache", "gndb")
	return cacheDir, nil
}

// GetDefaultConfigPath returns the full path to the default config file.
func GetDefaultConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}

// GetDefaultSourcesPath returns the full path to the default sources file.
func GetDefaultSourcesPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sources.yaml"), nil
}

// GenerateDefaultConfig creates documented default config and sources files at the platform-specific location.
// Returns the config path where files were created, or error if generation fails.
// Does NOT overwrite existing files.
func GenerateDefaultConfig() (string, error) {
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return "", err
	}

	sourcesPath, err := GetDefaultSourcesPath()
	if err != nil {
		return "", err
	}

	// Check if config file already exists
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	sourcesExists := false
	if _, err := os.Stat(sourcesPath); err == nil {
		sourcesExists = true
	}

	// If both exist, return error
	if configExists && sourcesExists {
		return "", fmt.Errorf("config files already exist at %s", filepath.Dir(configPath))
	}

	// Create parent directories if they don't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config.yaml if it doesn't exist
	if !configExists {
		if err := os.WriteFile(configPath, []byte(templates.ConfigYAML), 0644); err != nil {
			return "", fmt.Errorf("failed to write config file: %w", err)
		}
	}

	// Write sources.yaml if it doesn't exist
	if !sourcesExists {
		if err := os.WriteFile(sourcesPath, []byte(templates.SourcesYAML), 0644); err != nil {
			return "", fmt.Errorf("failed to write sources file: %w", err)
		}
	}

	// Ensure files are synced to disk
	if file, err := os.Open(configPath); err == nil {
		_ = file.Sync()
		file.Close()
	}
	if file, err := os.Open(sourcesPath); err == nil {
		_ = file.Sync()
		file.Close()
	}

	return configPath, nil
}

// ConfigFileExists checks if a config file exists at the default location.
func ConfigFileExists() (bool, error) {
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(configPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ValidateGeneratedConfig reads and validates a generated config file.
// Used for testing to ensure generated YAML is valid.
func ValidateGeneratedConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Merge with defaults since generated config has all values commented out
	cfg.MergeWithDefaults()

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}
