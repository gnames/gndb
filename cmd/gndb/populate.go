package main

import (
	"context"
	"fmt"

	"github.com/gnames/gndb/internal/io/database"
	"github.com/gnames/gndb/internal/io/populate"
	"github.com/spf13/cobra"
)

func getPopulateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "populate",
		Short: "Populates the database with data",
		Long: `Populates the database with nomenclature data from configured data sources.

This command:
  1. Connects to PostgreSQL using configuration settings
  2. Reads SFGA data sources from configuration
  3. Imports data using bulk insert operations for performance
  4. Logs progress for long-running imports

Examples:
  gndb populate
  gndb populate --config custom.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig()
			ctx := context.Background()

			// Create database operator
			op := database.NewPgxOperator()
			err := op.Connect(ctx, &cfg.Database)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer op.Close()

			// Create populator
			populator := populate.NewPopulator(op)

			// Run population
			log.Info("starting database population")
			err = populator.Populate(ctx, cfg)
			if err != nil {
				return fmt.Errorf("population failed: %w", err)
			}

			log.Info("database population complete")
			return nil
		},
	}
	return cmd
}
