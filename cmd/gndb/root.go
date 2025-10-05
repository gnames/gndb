package main

import (
	"fmt"
	"log/slog"

	"github.com/gnames/gndb/internal/io/config"
	pkgconfig "github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *pkgconfig.Config
	log     *slog.Logger
)

func getRootCmd() *cobra.Command {
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

			// Initialize logger with config
			log = logger.New(&cfg.Logging)

			// Display config source using logger
			switch result.Source {
			case "file":
				log.Info("config loaded", "source", "file", "path", result.SourcePath)
			case "defaults+env":
				log.Info("config loaded", "source", "defaults with environment overrides")
			case "defaults":
				log.Info("config loaded", "source", "built-in defaults")
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
		getRestructureCmd(),
	)

	return rootCmd
}

// getConfig returns the loaded configuration (for use in subcommands)
func getConfig() *pkgconfig.Config {
	return cfg
}
