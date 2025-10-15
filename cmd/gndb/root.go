package main

import (
	"fmt"
	"log/slog"

	"github.com/gnames/gndb/internal/ioconfig"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
	lg      *slog.Logger
)

func getRootCmd() *cobra.Command {
	// Auto-generate config file on first run if it doesn't exist
	// This runs before any command or flag processing, ensuring configs
	// are created even when just running 'gndb -V' or 'gndb --help'
	if cfgFile == "" {
		exists, err := ioconfig.ConfigFileExists()
		if err == nil {
			if !exists {
				// Generate default config
				_, err := ioconfig.GenerateDefaultConfig()
				if err != nil {
					// Silently continue - config generation is best-effort at this stage
					// Errors will be reported later if config is actually needed
				}
			} else {
				// Config files already exist - inform the user
				configPath, _ := ioconfig.GetDefaultConfigPath()
				sourcesPath, _ := ioconfig.GetDefaultSourcesPath()
				slog.Info("Using existing config files", "config", configPath, "sources", sourcesPath)
			}
		}
	}

	rootCmd := &cobra.Command{
		Use:   "gndb",
		Short: "GNdb manages GNverifier database lifecycle",
		Long: `GNdb is a command-line tool for managing the lifecycle of a PostgreSQL
database for GNverifier. It allows users to set up and maintain a local
GNverifier instance with custom data sources.

The tool supports the following functionalities:

- Database Schema Management: Create and migrate the database schema.
- Data Population: Populate the database with nomenclature data.
- Database Optimization: Optimize the database for fast name verification.

Configuration is managed through a gndb.yaml file, environment variables
(with GNDB_ prefix), and command-line flags.

For more information, see the project's README file.`,
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			result, err := ioconfig.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}
			cfg = result.Config

			// Initialize logger with config
			lg = logger.New(&cfg.Logging)

			// Display config source using logger
			switch result.Source {
			case "file":
				lg.Info("config loaded", "source", "file", "path", result.SourcePath)
			case "defaults+env":
				lg.Info("config loaded", "source", "defaults with environment overrides")
			case "defaults":
				lg.Info("config loaded", "source", "built-in defaults")
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
	rootCmd.AddCommand(
		getCreateCmd(),
		getMigrateCmd(),
		getPopulateCmd(),
		getOptimizeCmd(),
	)

	return rootCmd
}

// getConfig returns the loaded configuration (for use in subcommands)
func getConfig() *config.Config {
	return cfg
}
