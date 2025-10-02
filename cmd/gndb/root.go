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
  - restructure: Optimize with indexes and materialized views`,
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			var err error
			cfg, err = config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
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
