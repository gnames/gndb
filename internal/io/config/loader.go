// Package config provides I/O operations for loading configuration from files and flags.
// This is an impure package that handles file system and flag operations.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gnames/gndb/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Load reads configuration from a YAML file and returns a validated Config.
// If configPath is empty, it searches default locations:
//   - ./gndb.yaml
//   - ~/.config/gndb/gndb.yaml
//
// Returns error if file is malformed or validation fails.
func Load(configPath string) (*config.Config, error) {
	v := viper.New()

	// Set config file type
	v.SetConfigType("yaml")

	if configPath != "" {
		// Use explicit config path
		v.SetConfigFile(configPath)
	} else {
		// Search default locations
		v.SetConfigName("gndb")

		// Current directory
		v.AddConfigPath(".")

		// User config directory
		homeDir, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(filepath.Join(homeDir, ".config", "gndb"))
		}
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// If no config file found, return defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return config.Defaults(), nil
		}
		// For explicit config path that doesn't exist, return error
		if configPath != "" {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// For default search paths, return defaults if not found
		return config.Defaults(), nil
	}

	// Unmarshal into Config struct
	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// BindFlags binds cobra command flags to viper and returns updated config.
// CLI flags take precedence over config file values.
func BindFlags(cmd *cobra.Command, cfg *config.Config) (*config.Config, error) {
	v := viper.New()

	// Bind flags to viper
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return nil, fmt.Errorf("failed to bind flags: %w", err)
	}

	// Manually override config with flag values if set
	if v.IsSet("host") {
		cfg.Database.Host = v.GetString("host")
	}
	if v.IsSet("port") {
		cfg.Database.Port = v.GetInt("port")
	}
	if v.IsSet("user") {
		cfg.Database.User = v.GetString("user")
	}
	if v.IsSet("password") {
		cfg.Database.Password = v.GetString("password")
	}
	if v.IsSet("database") {
		cfg.Database.Database = v.GetString("database")
	}
	if v.IsSet("ssl-mode") {
		cfg.Database.SSLMode = v.GetString("ssl-mode")
	}

	// Validate updated config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration after flag binding: %w", err)
	}

	return cfg, nil
}
