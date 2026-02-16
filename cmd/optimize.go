/*
Copyright © 2025 Dmitry Mozzherin <dmozzherin@gmail.com>
*/
package cmd

import (
	"context"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/iooptimize"
	"github.com/spf13/cobra"
)

// getOptimizeCmd returns the optimize command.
func getOptimizeCmd() *cobra.Command {
	optimizeCmd := &cobra.Command{
		Use:   "optimize",
		Short: "Optimize database for gnverifier",
		Long: `Optimize the database for fast name verification queries.

This command prepares your database for use with gnverifier by reparsing
name strings with the latest gnparser, creating canonical forms, building
word indexes for fuzzy matching, and creating materialized views.

Prerequisites:
  - Database must be created (run 'gndb create' first)
  - Database must be populated (run 'gndb populate' first)

Performance:
  Optimization may take 20-90 minutes depending on dataset size.
  Progress bars will show the current status.

Examples:
  # Optimize with default settings
  gndb optimize`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOptimize(cmd, args)
		},
	}

	return optimizeCmd
}

func runOptimize(
	_ *cobra.Command,
	_ []string,
) error {
	ctx := context.Background()

	// Create database operator
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	if err != nil {
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

	// Create optimizer
	optimizer := iooptimize.NewOptimizer(op)

	// Run optimize
	gn.Info("Starting database optimization...")
	if err := optimizer.Optimize(ctx, cfg); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}
	gn.Info(`Database optimization is complete!

Your database is now ready for gnverifier.
You can re-run 'gndb optimize' anytime to apply the latest algorithm updates.`)

	return nil
}
