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
  3. Enables required extensions (pg_trgm for fuzzy matching)
  4. Records schema version

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

	// Enable required extensions
	fmt.Println("Enabling PostgreSQL extensions...")
	if err := op.EnableExtension(ctx, "pg_trgm"); err != nil {
		return fmt.Errorf("failed to enable pg_trgm extension: %w", err)
	}
	fmt.Println("✓ pg_trgm extension enabled")

	// Set schema version
	version := "1.0.0"
	if err := op.SetSchemaVersion(ctx, version, "Initial schema created with GORM"); err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}
	fmt.Printf("✓ Schema version set to %s\n", version)

	// Verify all tables were created
	models := schema.AllModels()
	fmt.Printf("\nCreated %d tables:\n", len(models))

	// Get table names from GORM
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying DB: %w", err)
	}
	defer sqlDB.Close()

	// Query for table names
	rows, err := sqlDB.Query(`
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY tablename
	`)
	if err != nil {
		return fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		fmt.Printf("  - %s\n", tableName)
	}

	fmt.Println("\n✓ Database schema creation complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  - Run 'gndb populate' to import data from SFGA files")
	fmt.Println("  - Run 'gndb restructure' to create indexes and optimize")

	return nil
}
