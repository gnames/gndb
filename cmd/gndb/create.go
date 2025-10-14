package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/pkg/db"
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
  2. Checks for existing tables and prompts for confirmation if found
  3. Creates all base tables using GORM AutoMigrate
  4. Records schema version

Note: Fuzzy matching is handled by gnmatcher (external to database).
This database stores canonical forms for exact lookups only.

Use --force to skip confirmation and drop existing tables automatically.

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
	var op db.Operator = iodb.NewPgxOperator()
	if err := op.Connect(ctx, &cfg.Database); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer op.Close()

	fmt.Printf("Connected to database: %s@%s:%d/%s\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)

	// Check if database has existing tables
	hasTables, err := op.HasTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for existing tables: %w", err)
	}

	// Handle existing tables
	if hasTables {
		if forceCreate {
			// Force flag set - drop without prompting
			fmt.Println("Dropping all existing tables (--force enabled)...")
			if err := op.DropAllTables(ctx); err != nil {
				return fmt.Errorf("failed to drop tables: %w", err)
			}
			fmt.Println("✓ All tables dropped")
		} else {
			// Prompt user for confirmation
			fmt.Println("\n⚠️  Warning: Database contains existing tables.")
			fmt.Println("Creating schema will drop ALL existing tables and data.")
			fmt.Print("\nDo you want to continue? (yes/no): ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "yes" && response != "y" {
				fmt.Println("Aborted. No changes made to the database.")
				return nil
			}

			// User confirmed - drop tables
			fmt.Println("Dropping all existing tables...")
			if err := op.DropAllTables(ctx); err != nil {
				return fmt.Errorf("failed to drop tables: %w", err)
			}
			fmt.Println("✓ All tables dropped")
		}
	}

	// Create schema manager
	var sm lifecycle.SchemaManager = ioschema.NewManager(op)

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
