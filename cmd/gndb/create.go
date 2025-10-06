package main

import (
	"context"
	"fmt"

	"github.com/gnames/gndb/internal/io/database"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
	op := database.NewPgxOperator()
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

	// Build DSN for GORM
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Database,
		cfg.Database.SSLMode,
	)

	// Connect with GORM
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect with GORM: %w", err)
	}

	// Run GORM AutoMigrate to create schema
	fmt.Println("Creating schema using GORM AutoMigrate...")
	if err := schema.Migrate(db); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	fmt.Println("✓ Schema created successfully")

	// Set collation for string columns (critical for correct sorting)
	fmt.Println("Setting collation for string columns...")
	if err := op.SetCollation(ctx); err != nil {
		return fmt.Errorf("failed to set collation: %w", err)
	}
	fmt.Println("✓ Collation set successfully")

	// Note: No PostgreSQL extensions needed
	// Fuzzy matching is handled by gnmatcher (bloom filters, suffix tries)
	// This database only stores canonical forms for exact lookups

	// Verify all tables were created
	tables, err := op.ListTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	models := schema.AllModels()
	fmt.Printf("\nCreated %d tables:\n", len(models))
	for _, tableName := range tables {
		fmt.Printf("  - %s\n", tableName)
	}

	fmt.Println("\n✓ Database schema creation complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  - Run 'gndb populate' to import data from SFGA files")
	fmt.Println("  - Run 'gndb restructure' to create indexes and optimize")

	return nil
}
