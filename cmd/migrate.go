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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/pkg/gndb"
	"github.com/spf13/cobra"
)

// getMigrateCmd returns the migrate command.
// Extracted as a function to facilitate testing and dynamic
// command registration.
func getMigrateCmd() *cobra.Command {
	var recreateViews bool
	var dryRun bool

	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate database schema to latest version",
		Long: `Migrate updates the database schema to match the current model state.

This command uses Atlas declarative migrations:
  1. Creates a temporary dev schema and applies the desired model state
  2. Inspects both the dev schema (desired) and the live database (current)
  3. Computes the diff and shows the planned SQL changes
  4. Prompts for confirmation before applying (use --dry-run to skip)
  5. Applies approved changes and optionally recreates materialized views

Atlas computes the exact diff regardless of how many versions behind the
database is — no migration files required.

This command is safe: it only applies changes you explicitly approve.
Does NOT delete tables or columns unless they appear in the diff and
you confirm the plan.

After migration, run 'gndb optimize' to recreate materialized views,
or use --recreate-views to do it in the same step.

Examples:
  gndb migrate
  gndb migrate --dry-run
  gndb migrate --recreate-views
  gndb migrate -v`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate(cmd, args, recreateViews, dryRun)
		},
	}

	migrateCmd.Flags().BoolVarP(&recreateViews, "recreate-views", "v",
		false, "recreate materialized views after migration")
	migrateCmd.Flags().BoolVarP(&dryRun, "dry-run", "n",
		false, "show planned changes without applying them")

	return migrateCmd
}

func runMigrate(
	_ *cobra.Command,
	_ []string,
	recreateViews bool,
	dryRun bool,
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

	gn.Info("Computing schema diff...")

	opts := gndb.MigrateOptions{
		RecreateViews: recreateViews,
		DryRun:        dryRun,
		Confirm:       confirmMigrate(dryRun),
	}

	if err := sm.Migrate(ctx, opts); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	return nil
}

// confirmMigrate returns a Confirm callback that displays planned SQL
// and prompts the user for approval (or just prints for dry-run).
func confirmMigrate(dryRun bool) func([]string) bool {
	return func(stmts []string) bool {
		gn.Info("Planned changes:")
		fmt.Println()
		for _, stmt := range stmts {
			fmt.Println(stmt)
		}

		if dryRun {
			gn.Info("Dry-run mode: no changes applied.")
			return false
		}

		fmt.Print("\nApply changes? [y/N]: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.TrimSpace(scanner.Text())
			if strings.ToLower(response) == "y" {
				return true
			}
		}

		gn.Info("Migration cancelled. No changes applied.")
		return false
	}
}
