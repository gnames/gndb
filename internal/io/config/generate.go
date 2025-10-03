package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gnames/gndb/pkg/config"
	"gopkg.in/yaml.v3"
)

// GetConfigDir returns the platform-specific configuration directory for GNdb.
// - Linux/macOS: ~/.config/gndb/
// - Windows: %APPDATA%\gndb\
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Use ~/.config/gndb on all Unix-like systems (Linux, macOS)
	// Use %APPDATA%\gndb on Windows
	var configDir string
	if filepath.Separator == '/' {
		// Unix-like (Linux, macOS)
		configDir = filepath.Join(homeDir, ".config", "gndb")
	} else {
		// Windows
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		configDir = filepath.Join(appData, "gndb")
	}

	return configDir, nil
}

// GetDefaultConfigPath returns the full path to the default config file.
func GetDefaultConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "gndb.yaml"), nil
}

// GenerateDefaultConfig creates a documented default config file at the platform-specific location.
// Returns the path where the config was created, or error if generation fails.
// Does NOT overwrite existing config files.
func GenerateDefaultConfig() (string, error) {
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return "", err
	}

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		// File exists, don't overwrite
		return "", fmt.Errorf("config file already exists at %s", configPath)
	}

	// Create parent directories if they don't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// Get default configuration
	defaults := config.Defaults()

	// Create documented YAML content using yaml.Marshal for proper encoding
	yamlContent := `# GNdb Configuration File
# This file was auto-generated. Edit as needed.
#
# Configuration precedence (highest to lowest):
#   1. CLI flags (--host, --port, etc.)
#   2. Environment variables (GNDB_*)
#   3. This config file
#   4. Built-in defaults
#
# For all environment variables, see: go doc github.com/gnames/gndb/pkg/config

database:
  host: ` + defaults.Database.Host + `
  port: ` + fmt.Sprintf("%d", defaults.Database.Port) + `
  user: ` + defaults.Database.User + `
  password: ` + defaults.Database.Password + `
  database: ` + defaults.Database.Database + `
  ssl_mode: ` + defaults.Database.SSLMode + `
  max_connections: ` + fmt.Sprintf("%d", defaults.Database.MaxConnections) + `
  min_connections: ` + fmt.Sprintf("%d", defaults.Database.MinConnections) + `
  max_conn_lifetime: ` + fmt.Sprintf("%d", defaults.Database.MaxConnLifetime) + `
  max_conn_idle_time: ` + fmt.Sprintf("%d", defaults.Database.MaxConnIdleTime) + `

import:
  batch_size: ` + fmt.Sprintf("%d", defaults.Import.BatchSize) + `

optimization:
  concurrent_indexes: ` + fmt.Sprintf("%t", defaults.Optimization.ConcurrentIndexes) + `

logging:
  level: ` + defaults.Logging.Level + `
  format: ` + defaults.Logging.Format + `
`

	// Write config file
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	// Ensure file is synced to disk (in case viper reads immediately after)
	// This shouldn't be necessary but adding as defensive measure
	file, err := os.Open(configPath)
	if err == nil {
		file.Sync()
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

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}
