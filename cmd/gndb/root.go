package main

import (
	"fmt"

	"github.com/gnames/gndb/internal/io/config"
	pkgconfig "github.com/gnames/gndb/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *pkgconfig.Config
)

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gndb",
		Short: "GNdb manages GNverifier database lifecycle",
		Long: `GNdb is a CLI tool for managing the complete lifecycle of the GNverifier
PostgreSQL database, from schema creation through data population and optimization.

The tool provides four main phases:
  - create: Create database schema and extensions
  - migrate: Apply schema migrations
  - populate: Import data from SFGA files
  - restructure: Optimize with indexes and materialized views

Configuration precedence (highest to lowest):
  1. CLI flags (--host, --port, etc.)
  2. Environment variables (GNDB_*)
  3. Config file (gndb.yaml)
  4. Built-in defaults

Environment Variables:
  All configuration can be set via GNDB_* environment variables.
  Nested fields use underscores (database.host → GNDB_DATABASE_HOST).

  Examples:
    GNDB_DATABASE_HOST              PostgreSQL host
    GNDB_DATABASE_PORT              PostgreSQL port
    GNDB_DATABASE_USER              PostgreSQL user
    GNDB_DATABASE_PASSWORD          PostgreSQL password
    GNDB_DATABASE_DATABASE          Database name
    GNDB_IMPORT_BATCH_SIZE          Import batch size
    GNDB_LOGGING_LEVEL              Log level (debug/info/warn/error)

  See 'go doc github.com/gnames/gndb/pkg/config' for complete list.`,
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Auto-generate config file on first run if it doesn't exist
			if cfgFile == "" {
				// Check if default config exists
				exists, err := config.ConfigFileExists()
				if err != nil {
					return fmt.Errorf("failed to check config file: %w", err)
				}

				if !exists {
					// Generate default config
					generatedPath, err := config.GenerateDefaultConfig()
					if err != nil {
						// Only warn, don't fail - can use defaults
						fmt.Printf("Warning: could not generate config file: %v\n", err)
					} else {
						fmt.Printf("Generated default config at: %s\n", generatedPath)
					}
				}
			}

			// Load configuration
			result, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}
			cfg = result.Config

			// Display config source
			switch result.Source {
			case "file":
				fmt.Printf("Using config from: %s\n", result.SourcePath)
			case "defaults+env":
				fmt.Println("Using built-in defaults with environment variable overrides")
			case "defaults":
				fmt.Println("Using built-in defaults (no config file)")
			}

			return nil
		},
	}

	// Persistent flags available to all subcommands
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default: ./gndb.yaml or ~/.config/gndb/gndb.yaml)")

	// Override version flag to use -V (consistent with other gn projects)
	rootCmd.Flags().BoolP("version", "V", false, "version for gndb")

	// Add subcommands
	rootCmd.AddCommand(getCreateCmd())
	// TODO: Add migrate, populate, restructure commands in future tasks

	return rootCmd
}

// getConfig returns the loaded configuration (for use in subcommands)
func getConfig() *pkgconfig.Config {
	return cfg
}
