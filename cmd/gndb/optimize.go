package main

import (
	"context"
	"fmt"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/iooptimize"
	"github.com/gnames/gnlib"
	"github.com/spf13/cobra"
)

func getOptimizeCmd() *cobra.Command {
	var (
		jobs      int
		batchSize int
	)

	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Optimizes the database for gnverifier",
		Long: `Optimizes the database for fast name verification queries.

This command prepares your database for use with gnverifier by creating indexes,
updating statistics, and applying performance optimizations. It's safe to run
multiple times - each run ensures your database uses the latest algorithms.

Prerequisites:
  - Database must be created (run 'gndb create' first)
  - Database must be populated (run 'gndb populate' first)

Performance:
  Use --jobs to control how many CPU cores to use (default: all available cores).
  Larger values speed up optimization but use more system resources.

Examples:
  # Optimize with default settings
  gndb optimize

  # Use more workers for faster optimization on powerful servers
  gndb optimize --jobs 100

  # Adjust batch size for memory-constrained systems
  gndb optimize --batch-size 10000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig()
			ctx := context.Background()

			// Override config with CLI flags if provided
			if cmd.Flags().Changed("jobs") {
				cfg.JobsNumber = jobs
			}
			if cmd.Flags().Changed("batch-size") {
				cfg.Database.BatchSize = batchSize
			}

			// Create database operator
			op := iodb.NewPgxOperator()

			// Connect to database (errors propagate from iodb package)
			err := op.Connect(ctx, &cfg.Database)
			if err != nil {
				gnlib.PrintUserMessage(err)
				return err
			}
			defer op.Close()

			// Check if database is ready (errors propagate from iodb package)
			err = op.CheckReadyForOptimization(ctx, &cfg.Database)
			if err != nil {
				gnlib.PrintUserMessage(err)
				return err
			}

			// Create optimizer
			optimizer := iooptimize.NewOptimizer(op)

			// Optimize (errors propagate from iooptimize package)
			err = optimizer.Optimize(ctx, cfg)
			if err != nil {
				gnlib.PrintUserMessage(err)
				return err
			}

			// Display success message
			successMsg := gnlib.FormatMessage(`
<em>✓ Your database is now optimized and ready for gnverifier.</em>
You can re-run <em>gndb optimize</em> anytime to apply the latest
algorithm updates.
`,
				nil,
			)
			fmt.Println(successMsg)

			return nil
		},
	}

	// Add flags
	cmd.Flags().IntVar(&jobs, "jobs", 0,
		"Number of concurrent workers for parsing and normalization (default: number of CPU cores)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 0,
		"Batch size for word processing (default: 50000)")

	return cmd
}
