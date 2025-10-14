package main

import (
	"context"
	"fmt"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/iooptimize"
	"github.com/spf13/cobra"
)

func getOptimizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Optimizes the database",
		Long:  "Applies performance-critical optimizations to the database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig()
			ctx := context.Background()

			// Create database operator
			op := iodb.NewPgxOperator()
			err := op.Connect(ctx, &cfg.Database)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer op.Close()

			// Create optimizer
			optimizer := iooptimize.NewOptimizer(op)

			// Run optimization
			lg.Info("starting database optimization")
			err = optimizer.Optimize(ctx, cfg)
			if err != nil {
				return fmt.Errorf("optimization failed: %w", err)
			}

			lg.Info("database optimization complete")
			return nil
		},
	}
	return cmd
}
