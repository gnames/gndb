// Package schema implements SchemaManager interface for database schema management.
// This is an impure I/O package that wraps GORM AutoMigrate functionality.
package schema

import (
	"context"
	"fmt"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/database"
	"github.com/gnames/gndb/pkg/lifecycle"
	"github.com/gnames/gndb/pkg/schema"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Manager implements the SchemaManager interface using GORM AutoMigrate.
type Manager struct {
	operator database.Operator
}

// NewManager creates a new SchemaManager.
func NewManager(op database.Operator) lifecycle.SchemaManager {
	return &Manager{operator: op}
}

// Create creates the initial database schema using GORM AutoMigrate.
// Also applies collation settings for correct scientific name sorting.
func (m *Manager) Create(ctx context.Context, cfg *config.Config) error {
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
	if err := schema.Migrate(db); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Set collation for string columns (critical for correct sorting)
	if err := m.setCollation(ctx); err != nil {
		return fmt.Errorf("failed to set collation: %w", err)
	}

	return nil
}

// Migrate updates the database schema to the latest version using GORM AutoMigrate.
func (m *Manager) Migrate(ctx context.Context, cfg *config.Config) error {
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

	// Run GORM AutoMigrate
	if err := schema.Migrate(db); err != nil {
		return fmt.Errorf("failed to migrate schema: %w", err)
	}

	return nil
}

// setCollation sets "C" collation on specified varchar columns.
// This is critical for correct sorting and comparison of scientific names.
func (m *Manager) setCollation(ctx context.Context) error {
	pool := m.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	type columnDef struct {
		table, column string
		varchar       int
	}
	columns := []columnDef{
		{"name_strings", "name", 500},
		{"canonicals", "name", 255},
		{"canonical_fulls", "name", 255},
		{"canonical_stems", "name", 255},
		{"words", "normalized", 255},
		{"words", "modified", 255},
		{"vernacular_strings", "name", 255},
	}

	qStr := `ALTER TABLE %s ALTER COLUMN %s TYPE VARCHAR(%d) COLLATE "C"`

	for _, col := range columns {
		q := fmt.Sprintf(qStr, col.table, col.column, col.varchar)
		if _, err := pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("failed to set collation on %s.%s: %w", col.table, col.column, err)
		}
	}

	return nil
}
