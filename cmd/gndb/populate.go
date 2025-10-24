package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/gnames/gndb/internal/ioconfig"
	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/iopopulate"
	"github.com/gnames/gndb/pkg/populate"
	"github.com/gnames/gnlib"
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
				configDir, err := ioconfig.GetConfigDir()
				if err != nil {
					return fmt.Errorf("failed to get config directory: %w", err)
				}
				sourcesYAMLPath = filepath.Join(configDir, "sources.yaml")
			}

			// Load sources configuration
			sourcesConfig, err := populate.LoadSourcesConfig(sourcesYAMLPath)
			if err != nil {
				return fmt.Errorf(
					"failed to load sources configuration from %s: %w",
					sourcesYAMLPath,
					err,
				)
			}

			// Display configuration warnings (non-fatal issues)
			if len(sourcesConfig.Warnings) > 0 {
				fmt.Println("\n" + gnlib.FormatMessage("<warn>Configuration Issues Detected</warn>", nil))
				fmt.Println(gnlib.FormatMessage(
					"Found <em>%d</em> issue(s) in sources configuration:",
					[]any{len(sourcesConfig.Warnings)},
				))
				fmt.Println()

				for _, warning := range sourcesConfig.Warnings {
					fmt.Printf("  • Data Source ID %d:\n", warning.DataSourceID)
					fmt.Printf("    Field: %s\n", warning.Field)
					fmt.Printf("    Issue: %s\n", warning.Message)
					if warning.Suggestion != "" {
						fmt.Printf("    Suggestion: %s\n", warning.Suggestion)
					}
					fmt.Println()
				}

				fmt.Println(gnlib.FormatMessage(
					"<warn>Datasets with issues may be imported but could have limited functionality.</warn>",
					nil,
				))

				// Prompt user to confirm continuation (defaults to yes)
				response, err := promptUser("Continue with import? [Y/n]: ")
				if err != nil {
					return fmt.Errorf("failed to read user input: %w", err)
				}

				if response == "no" {
					return fmt.Errorf("import cancelled by user")
				}
			}

			// Filter sources based on --sources flag
			filteredSources, err := populate.FilterSources(
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
					slog.Info("Dataset version override", "version", releaseVersion)
				}
				if hasReleaseDate {
					// Date override will be applied during population
					slog.Info("Dataset release date override", "date", releaseDate)
				}
			}

			// Update sources config with filtered list
			sourcesConfig.DataSources = filteredSources

			// Extract source IDs and set in Config for Populate()
			sourceIDs := make([]int, len(filteredSources))
			for i, src := range filteredSources {
				sourceIDs[i] = src.ID
			}
			cfg.Populate.SourceIDs = sourceIDs
			cfg.Populate.ReleaseVersion = releaseVersion
			cfg.Populate.ReleaseDate = releaseDate

			// Create database operator
			op := iodb.NewPgxOperator()
			err = op.Connect(ctx, &cfg.Database)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer op.Close()

			// Create populator
			populator := iopopulate.NewPopulator(op)

			// Run population
			err = populator.Populate(ctx, cfg)
			if err != nil {
				return fmt.Errorf("population failed: %w", err)
			}

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

// promptUser displays a message and reads user input from stdin.
// Defaults to "yes" - user must explicitly type "n" or "no" to decline.
// Any other input (including empty/Enter) is treated as "yes".
func promptUser(message string) (string, error) {
	fmt.Print(message)

	var response string
	// Scanln returns error on empty input, but we want to allow that as default "yes"
	_, err := fmt.Scanln(&response)
	if err != nil && err.Error() != "unexpected newline" {
		// Real error, not just empty input
		return "", err
	}

	response = strings.ToLower(strings.TrimSpace(response))

	// Explicit "no" or "n" means decline, everything else (including empty) means yes
	if response == "n" || response == "no" {
		return "no", nil
	}

	return "yes", nil
}
