package main

import (
	"context"
	"fmt"

	iodatabase "github.com/gnames/gndb/internal/io/database"
	ioschema "github.com/gnames/gndb/internal/io/schema"
	"github.com/gnames/gndb/pkg/database"
	"github.com/gnames/gndb/pkg/lifecycle"
	"github.com/spf13/cobra"
)

func getMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Applies database migrations",
		Long:  "Applies all pending database migrations to bring the schema to the latest version.",
		RunE:  runMigrate,
	}
	return cmd
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := getConfig()

	// Create database operator
	var op database.Operator = iodatabase.NewPgxOperator()
	if err := op.Connect(ctx, &cfg.Database); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer op.Close()

	fmt.Printf("Connected to database: %s@%s:%d/%s\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)

	// Create schema manager
	var sm lifecycle.SchemaManager = ioschema.NewManager(op)

	// Run GORM AutoMigrate to migrate schema
	fmt.Println("Applying database migrations...")
	if err := sm.Migrate(ctx, cfg); err != nil {
		return fmt.Errorf("failed to migrate schema: %w", err)
	}

	fmt.Println("\n✓ Database migration complete!")
	return nil
}
