/*
Copyright © 2025 Dmitry Mozzherin <dmozzherin@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/iopopulate"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/spf13/cobra"
)

// getPopulateCmd returns the populate command.
// Extracted as a function to facilitate testing and dynamic
// command registration.
func getPopulateCmd() *cobra.Command {
	var (
		sourceIDs          []int
		releaseVersion     string
		releaseDate        string
		flatClassification bool
	)

	populateCmd := &cobra.Command{
		Use:   "populate",
		Short: "Populate database with SFGA data",
		Long: `Import nomenclature data from SFGA sources.

This command:
  1. Connects to PostgreSQL using configuration settings
  2. Reads sources.yaml to discover SFGA data sources
  3. Downloads/opens SFGA SQLite files (local or remote)
  4. Imports data in phases:
     - Metadata (DataSource records)
     - Name-strings (NameString, Canonical, etc.)
     - Vernacular names (VernacularString)
     - Taxonomic hierarchy (classification tree)
     - Name indices (NameStringIndex)
  5. Reports progress and statistics

SFGA data sources configured in: ~/.config/gndb/sources.yaml
Each source has an ID (< 1000 official, >= 1000 custom).

Override flags (--release-version, --release-date) only work
when importing a single source.

Examples:
  # Import all sources from sources.yaml
  gndb populate

  # Import specific sources only
  gndb populate --source-ids 1,11,132
  gndb populate -s 1,11,132

  # Override release version for single source
  gndb populate -s 1 -r "2024.01" -d "2024-01-15"

  # Use flat classification
  gndb populate --flat-classification`,
		Aliases: []string{"add"},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runPopulate(
				cmd, sourceIDs, releaseVersion,
				releaseDate, flatClassification,
			)
			if err != nil {
				gn.PrintErrorMessage(err)
			}
			return err
		},
	}

	populateCmd.Flags().IntSliceVarP(
		&sourceIDs, "source-ids", "s", []int{},
		"data source IDs to import (empty = all)",
	)
	populateCmd.Flags().StringVarP(
		&releaseVersion, "release-version", "r", "",
		"override version (single source only)",
	)
	populateCmd.Flags().StringVarP(
		&releaseDate, "release-date", "d", "",
		"override date YYYY-MM-DD (single source only)",
	)
	populateCmd.Flags().BoolVarP(
		&flatClassification, "flat-classification", "f", false,
		"use flat classification",
	)

	return populateCmd
}

func runPopulate(
	cmd *cobra.Command,
	sourceIDs []int,
	releaseVersion string,
	releaseDate string,
	flatClassification bool,
) error {
	ctx := context.Background()

	// Validate override flags (single source constraint)
	hasVersion := cmd.Flags().Changed("release-version")
	hasDate := cmd.Flags().Changed("release-date")
	hasSourceIDs := cmd.Flags().Changed("source-ids")

	if hasSourceIDs && len(sourceIDs) > 1 {
		if hasVersion {
			gn.Warn(`<warn>Cannot override release version with multiple sources</warn>
   <warn>Use --source-ids to select a single source</warn>`)
			err := fmt.Errorf("invalid flag combination")
			slog.Error("invalid flag combination", "error", err)
			return err
		}
		if hasDate {
			gn.Warn(`<warn>Cannot override release date with multiple sources</warn>"
   <warn>Use --source-ids to select a single source</warn>`)
			err := fmt.Errorf("invalid flag combination")
			slog.Error("invalid flag combination", "error", err)
			return err
		}
	}

	// Build options from explicitly set flags
	var populateOpts []config.Option

	if hasSourceIDs {
		populateOpts = append(
			populateOpts,
			config.OptPopulateSourceIDs(sourceIDs),
		)
	}

	if hasVersion {
		populateOpts = append(
			populateOpts,
			config.OptPopulateReleaseVersion(releaseVersion),
		)
	}

	if hasDate {
		populateOpts = append(
			populateOpts,
			config.OptPopulateReleaseDate(releaseDate),
		)
	}

	if cmd.Flags().Changed("flat-classification") {
		populateOpts = append(
			populateOpts,
			config.OptPopulateWithFlatClassification(
				&flatClassification,
			),
		)
	}

	// Apply populate-specific options to config
	if len(populateOpts) > 0 {
		cfg.Update(populateOpts)
	}

	// Create database operator
	op := iodb.NewPgxOperator()
	if err := op.Connect(ctx, &cfg.Database); err != nil {
		return err
	}
	defer op.Close()

	gn.Info("Connected to database: <em>%s@%s:%d/%s</em>",
		cfg.Database.User, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.Database)

	// Check if database has tables
	hasTables, err := op.HasTables(ctx)
	if err != nil {
		return err
	}

	if !hasTables {
		err = &gn.Error{
			Code: errcode.DBEmptyDatabaseError,
			Msg: `<err>Database appears to be empty.</err>
   Run <em>'gndb create'</em> first to initialize the schema.`,
			Err: errors.New("cannot insert data into empty database"),
		}
		return err
	}

	// Create populator
	populator := iopopulate.New(cfg, op)

	// Run populate
	gn.Info("Starting data population from SFGA sources...")
	if err := populator.Populate(); err != nil {
		return err
	}

	gn.Info(`Next steps:
	 - Run '<em>gndb populate</em>' until you get all data you need
	 - Run '<em>gndb optimize</em>' to create indexes
	 - Database is ready for verification
`)

	return nil
}
