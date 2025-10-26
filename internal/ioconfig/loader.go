// Package config provides I/O operations for loading configuration from files and flags.
// This is an impure package that handles file system and flag operations.
package ioconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/gnames/gndb/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// LoadResult contains the loaded configuration and metadata about the source.
type LoadResult struct {
	Config     *config.Config
	SourcePath string // Path to config file used, or empty if using defaults
	Source     string // "file", "defaults", or "defaults+env"
}

// Load reads configuration from a YAML file and returns a validated Config with source info.
// If configPath is empty, it searches default locations:
//   - ./gndb.yaml
//   - ~/.config/gndb/gndb.yaml
//
// Returns error if file is malformed or validation fails.
func Load(configPath string) (*LoadResult, error) {
	v := viper.New()

	// Set config file type
	v.SetConfigType("yaml")

	// Enable environment variable overrides
	// Precedence: flags > env vars > config file > defaults
	v.SetEnvPrefix("GNDB")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults BEFORE reading config - this allows env vars to work with AutomaticEnv()
	// Even if config file exists, defaults ensure viper knows which keys to check for env vars
	defaults := config.Defaults()
	v.SetDefault("database.host", defaults.Database.Host)
	v.SetDefault("database.port", defaults.Database.Port)
	v.SetDefault("database.user", defaults.Database.User)
	v.SetDefault("database.password", defaults.Database.Password)
	v.SetDefault("database.database", defaults.Database.Database)
	v.SetDefault("database.ssl_mode", defaults.Database.SSLMode)
	v.SetDefault("database.max_connections", defaults.Database.MaxConnections)
	v.SetDefault("database.min_connections", defaults.Database.MinConnections)
	v.SetDefault("database.max_conn_lifetime", defaults.Database.MaxConnLifetime)
	v.SetDefault("database.max_conn_idle_time", defaults.Database.MaxConnIdleTime)
	v.SetDefault("database.batch_size", defaults.Database.BatchSize)
	v.SetDefault("import.min_sfga_version", defaults.Import.MinSfgaVersion)
	v.SetDefault("optimization.concurrent_indexes", defaults.Optimization.ConcurrentIndexes)
	v.SetDefault("logging.level", defaults.Logging.Level)
	v.SetDefault("logging.format", defaults.Logging.Format)

	if configPath != "" {
		// Use explicit config path
		v.SetConfigFile(configPath)
	} else {
		// Try explicit default path first
		defaultPath, err := GetDefaultConfigPath()
		if err == nil {
			if _, statErr := os.Stat(defaultPath); statErr == nil {
				// Config file exists at default location, use it explicitly
				v.SetConfigFile(defaultPath)
			}
		}
		// If no explicit default path worked, viper will use defaults + env vars
	}

	// Read config file (if it exists)
	configFileRead := false
	usedConfigPath := ""

	if err := v.ReadInConfig(); err != nil {
		// If no config file found and no explicit path, continue with defaults + env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if configPath != "" {
				// Explicit path that doesn't exist is an error
				return nil, fmt.Errorf("config file not found: %s", configPath)
			}
			// No config file in default locations - will use defaults + env vars
			configFileRead = false
		} else {
			// Other error reading config
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		configFileRead = true
		usedConfigPath = v.ConfigFileUsed()
	}

	// Unmarshal into Config struct
	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Merge with defaults to fill in missing values
	cfg.MergeWithDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Determine source
	source := "defaults"
	if configFileRead {
		source = "file"
	} else if hasEnvVars() {
		source = "defaults+env"
	}

	return &LoadResult{
		Config:     &cfg,
		SourcePath: usedConfigPath,
		Source:     source,
	}, nil
}

// hasEnvVars checks if any GNDB_* environment variables are set.
func hasEnvVars() bool {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GNDB_") {
			return true
		}
	}
	return false
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
