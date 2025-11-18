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

	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/spf13/cobra"
)

// getMigrateCmd returns the migrate command.
// Extracted as a function to facilitate testing and dynamic
// command registration.
func getMigrateCmd() *cobra.Command {
	var recreateViews bool

	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate database schema to latest version",
		Long: `Migrate updates the database schema to the latest version.

This command:
  1. Connects to PostgreSQL using configuration settings
  2. Checks if database schema exists
  3. Drops materialized views (required for ALTER TABLE)
  4. Runs GORM AutoMigrate to update schema
  5. Optionally recreates materialized views

GORM AutoMigrate:
  - Adds new tables if they don't exist
  - Adds new columns to existing tables
  - Adds missing indexes
  - Does NOT delete columns or tables (safe)

Use this command after updating gndb to get schema changes.
After migration, run 'gndb populate' then 'gndb optimize' to
rebuild views with fresh data.

Examples:
  gndb migrate
  gndb migrate --recreate-views
  gndb migrate -v`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate(cmd, args, recreateViews)
		},
	}

	migrateCmd.Flags().BoolVarP(&recreateViews, "recreate-views", "v",
		false, "recreate materialized views after migration")

	return migrateCmd
}

func runMigrate(
	_ *cobra.Command,
	_ []string,
	recreateViews bool,
) error {
	ctx := context.Background()

	// Create database operator
	op := iodb.NewPgxOperator()
	if err := op.Connect(ctx, &cfg.Database); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}
	defer op.Close()

	gn.Info("Connected to database: <em>%s@%s:%d/%s</em>",
		cfg.Database.User, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.Database)

	// Check if database has tables
	hasTables, err := op.HasTables(ctx)
	if err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	if !hasTables {
		gn.Warn(`Warning: Database appears to be empty.
	Run 'gndb create' first to initialize the schema.`)
		return nil
	}

	// Create schema manager and run migration
	sm := ioschema.NewManager(op, cfg)

	gn.Info("Migrating schema to latest version...")
	if err := sm.Migrate(ctx, recreateViews); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	// Report result
	if recreateViews {
		gn.Info("Schema migration complete. Materialized views recreated.")
	} else {
		gn.Info(`Schema migration complete.
   Materialized views were dropped.
   Run '<em>gndb optimize</em>' to recreate them after populating data.`)
	}

	return nil
}
