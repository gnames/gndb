package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gnames/gndb/internal/io/config"
	"github.com/gnames/gndb/internal/io/database"
	"github.com/gnames/gndb/internal/io/populate"
	gnpopulate "github.com/gnames/gndb/pkg/populate"
	"github.com/spf13/cobra"
)

func getPopulateCmd() *cobra.Command {
	var (
		sourcesFilter   string
		releaseVersion  string
		releaseDate     string
		sourcesYAMLPath string
	)

	cmd := &cobra.Command{
		Use:   "populate",
		Short: "Populates the database with data",
		Long: `Populates the database with nomenclature data from SFGA sources.

This command:
  1. Connects to PostgreSQL using configuration settings
  2. Loads sources.yaml from config directory
  3. Filters sources based on --sources flag (optional)
  4. Imports data using bulk insert operations for performance
  5. Logs progress for long-running imports

Source Filtering:
  --sources "main"        Import only official sources (ID < 1000)
  --sources "exclude main" Import only custom sources (ID >= 1000)
  --sources "1,3,5"       Import only sources with specified IDs
  (no --sources flag)     Import all sources in sources.yaml

Override Flags (single source only):
  --release-version       Override release version from filename
  --release-date          Override release date from filename
  Note: These flags only work when importing a single source

Examples:
  # Import all sources from sources.yaml
  gndb populate

  # Import only official sources (ID < 1000)
  gndb populate --sources main

  # Import only custom sources (ID >= 1000)
  gndb populate --sources "exclude main"

  # Import specific sources by ID
  gndb populate --sources 1,3,5

  # Import single source with version override
  gndb populate --sources 1 --release-version "2024.1"

  # Import single source with date override
  gndb populate --sources 2 --release-date "2024-12-15"

  # Use custom sources.yaml location
  gndb populate --sources-yaml /path/to/sources.yaml`,
		// RunE runs but returns error, so the error handling is a responsibility of
		// the framework.
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig()
			ctx := context.Background()

			// Determine sources.yaml path
			if sourcesYAMLPath == "" {
				configDir, err := config.GetConfigDir()
				if err != nil {
					return fmt.Errorf("failed to get config directory: %w", err)
				}
				sourcesYAMLPath = filepath.Join(configDir, "sources.yaml")
			}

			// Load sources configuration
			sourcesConfig, err := gnpopulate.LoadSourcesConfig(sourcesYAMLPath)
			if err != nil {
				return fmt.Errorf(
					"failed to load sources configuration from %s: %w",
					sourcesYAMLPath,
					err,
				)
			}

			// Filter sources based on --sources flag
			filteredSources, err := gnpopulate.FilterSources(
				sourcesConfig.DataSources,
				sourcesFilter,
			)
			if err != nil {
				return fmt.Errorf("failed to filter sources: %w", err)
			}

			if len(filteredSources) == 0 {
				return fmt.Errorf(
					"no sources selected for import. Check your --sources filter or sources.yaml configuration",
				)
			}

			// Validate override flags (CLI constraint: overrides only work with single source)
			hasReleaseVersion := cmd.Flags().Changed("release-version")
			hasReleaseDate := cmd.Flags().Changed("release-date")

			if err := validateOverrideFlags(len(filteredSources), hasReleaseVersion, hasReleaseDate); err != nil {
				return err
			}

			// Apply overrides if single source
			if len(filteredSources) == 1 {
				if hasReleaseVersion {
					// Version override will be applied during population
					lg.Info("release version override", "version", releaseVersion)
				}
				if hasReleaseDate {
					// Date override will be applied during population
					lg.Info("release date override", "date", releaseDate)
				}
			}

			// Update sources config with filtered list
			sourcesConfig.DataSources = filteredSources

			lg.Info("sources loaded", "count", len(filteredSources), "filter", sourcesFilter)

			// Extract source IDs and set in Config for Populate()
			sourceIDs := make([]int, len(filteredSources))
			for i, src := range filteredSources {
				sourceIDs[i] = src.ID
			}
			cfg.Populate.SourceIDs = sourceIDs
			cfg.Populate.ReleaseVersion = releaseVersion
			cfg.Populate.ReleaseDate = releaseDate

			// Create database operator
			op := database.NewPgxOperator()
			err = op.Connect(ctx, &cfg.Database)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer op.Close()

			// Create populator
			populator := populate.NewPopulator(op)

			// Run population
			lg.Info("starting database population")
			err = populator.Populate(ctx, cfg)
			if err != nil {
				return fmt.Errorf("population failed: %w", err)
			}

			lg.Info("database population complete")
			return nil
		},
	}

	cmd.Flags().
		StringVar(&sourcesFilter, "sources", "", "Filter sources to import: 'main' (ID < 1000), 'exclude main' (ID >= 1000), or comma-separated IDs '1,3,5'")
	cmd.Flags().
		StringVar(&releaseVersion, "release-version", "", "Override release version (only for single source)")
	cmd.Flags().
		StringVar(&releaseDate, "release-date", "", "Override release date in YYYY-MM-DD format (only for single source)")
	cmd.Flags().
		StringVar(&sourcesYAMLPath, "sources-yaml", "", "Path to sources.yaml configuration file (default: ~/.config/gndb/sources.yaml)")

	return cmd
}

// validateOverrideFlags validates that release version and release date overrides
// are only used with a single source (CLI-specific constraint).
func validateOverrideFlags(sourceCount int, hasReleaseVersion, hasReleaseDate bool) error {
	if sourceCount <= 1 {
		return nil // Single source or no sources - OK
	}

	if hasReleaseVersion {
		return fmt.Errorf(
			"cannot override release version with multiple sources (%d sources selected). Use --sources to select a single source (e.g., --sources 1)",
			sourceCount,
		)
	}

	if hasReleaseDate {
		return fmt.Errorf(
			"cannot override release date with multiple sources (%d sources selected). Use --sources to select a single source (e.g., --sources 2)",
			sourceCount,
		)
	}

	return nil
}
