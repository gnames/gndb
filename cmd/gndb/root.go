package main

import (
	"fmt"
	"log/slog"

	"github.com/gnames/gndb/internal/ioconfig"
	gndb "github.com/gnames/gndb/pkg"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

func getRootCmd() *cobra.Command {
	// Auto-generate config file on first run if it doesn't exist
	// This runs before any command or flag processing, ensuring configs
	// are created even when just running 'gndb -V' or 'gndb --help'
	if cfgFile == "" {
		exists, err := ioconfig.ConfigFileExists()
		if err == nil {
			if !exists {
				// Generate default config (best-effort, errors will be reported later if config is actually needed)
				_, _ = ioconfig.GenerateDefaultConfig()
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
		Version: fmt.Sprintf("%s (build: %s)", gndb.Version, gndb.Build),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			result, err := ioconfig.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}
			cfg = result.Config

			// Initialize logger with config
			lg := logger.New(&cfg.Logging)
			slog.SetDefault(lg)

			// Display config source using logger
			var source string
			switch result.Source {
			case "file":
				source = result.SourcePath
			case "defaults+env":
				source = "defaults with environment overrides"
			case "defaults":
				source = "built-in defaults"
			}
			slog.Info("Config loaded", "source", source)

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
