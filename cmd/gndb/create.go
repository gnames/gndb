package main

import (
	"context"
	"fmt"

	io_database "github.com/gnames/gndb/internal/io/database"
	io_schema "github.com/gnames/gndb/internal/io/schema"
	"github.com/gnames/gndb/pkg/database"
	"github.com/gnames/gndb/pkg/lifecycle"
	"github.com/spf13/cobra"
)

var (
	forceCreate bool
)

func getCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create database schema",
		Long: `Create the GNverifier database schema from scratch.

This command:
  1. Connects to PostgreSQL using configuration settings
  2. Creates all base tables using GORM AutoMigrate
  3. Records schema version

Note: Fuzzy matching is handled by gnmatcher (external to database).
This database stores canonical forms for exact lookups only.

Use --force to drop existing tables before creating schema.

Examples:
  gndb create
  gndb create --force
  gndb create --config custom.yaml`,
		RunE: runCreate,
	}

	cmd.Flags().BoolVar(&forceCreate, "force", false,
		"drop existing tables before creating schema (destructive)")

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := getConfig()

	// Create database operator
	var op database.Operator = io_database.NewPgxOperator()
	if err := op.Connect(ctx, &cfg.Database); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer op.Close()

	fmt.Printf("Connected to database: %s@%s:%d/%s\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)

	// If force flag is set, drop all existing tables
	if forceCreate {
		fmt.Println("Dropping all existing tables (--force enabled)...")
		if err := op.DropAllTables(ctx); err != nil {
			return fmt.Errorf("failed to drop tables: %w", err)
		}
		fmt.Println("✓ All tables dropped")
	}

	// Create schema manager
	var sm lifecycle.SchemaManager = io_schema.NewManager(op)

	// Run GORM AutoMigrate to create schema
	fmt.Println("Creating schema using GORM AutoMigrate...")
	if err := sm.Create(ctx, cfg); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	fmt.Println("\n✓ Database schema creation complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  - Run 'gndb populate' to import data from SFGA files")
	fmt.Println("  - Run 'gndb optimize' to create indexes and optimize")

	return nil
}
