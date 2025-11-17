// Package ioschema implements SchemaManager interface for
// database schema management. This is an impure I/O package
// that wraps GORM AutoMigrate functionality.
package ioschema

import (
	"context"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/gndb"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// manager implements the gndb.SchemaManager interface
// using GORM AutoMigrate.
type manager struct {
	operator db.Operator
}

// NewManager creates a new SchemaManager.
func NewManager(op db.Operator) gndb.SchemaManager {
	return &manager{operator: op}
}

// Create creates the initial database schema using
// GORM AutoMigrate. Also applies collation settings for
// correct scientific name sorting.
func (m *manager) Create(
	ctx context.Context,
	cfg *config.Config,
) error {
	pool := m.operator.Pool()
	if pool == nil {
		return NotConnectedError()
	}

	db := stdlib.OpenDBFromPool(pool)

	// Connect with GORM
	gormDB, err := gorm.Open(
		postgres.New(postgres.Config{Conn: db}),
		&gorm.Config{},
	)
	if err != nil {
		return GORMConnectionError(err)
	}

	// Run GORM AutoMigrate to create schema
	if err := schema.Migrate(gormDB); err != nil {
		return CreateSchemaError(err)
	}

	// Set collation for string columns
	// (critical for correct sorting)
	if err := m.setCollation(ctx); err != nil {
		return err
	}

	return nil
}

// Migrate updates the database schema to the latest version
// using GORM AutoMigrate.
func (m *manager) Migrate(
	ctx context.Context,
	cfg *config.Config,
) error {
	pool := m.operator.Pool()
	if pool == nil {
		return NotConnectedError()
	}

	db := stdlib.OpenDBFromPool(pool)

	// Connect with GORM
	gormDB, err := gorm.Open(
		postgres.New(postgres.Config{Conn: db}),
		&gorm.Config{},
	)
	if err != nil {
		return GORMConnectionError(err)
	}

	// Run GORM AutoMigrate
	if err := schema.Migrate(gormDB); err != nil {
		return MigrateSchemaError(err)
	}

	return nil
}

// setCollation sets "C" collation on specified varchar
// columns. This is critical for correct sorting and
// comparison of scientific names.
func (m *manager) setCollation(ctx context.Context) error {
	pool := m.operator.Pool()
	if pool == nil {
		return NotConnectedError()
	}

	type columnDef struct {
		table, column string
	}

	columns := []columnDef{
		{"name_strings", "name"},
		{"canonicals", "name"},
		{"canonical_fulls", "name"},
		{"canonical_stems", "name"},
		{"words", "normalized"},
		{"words", "modified"},
		{"vernacular_strings", "name"},
	}

	qStr := `ALTER TABLE %s ALTER COLUMN %s ` +
		`TYPE TEXT COLLATE "C"`

	for _, col := range columns {
		q := formatCollationSQL(qStr, col.table,
			col.column)
		if _, err := pool.Exec(ctx, q); err != nil {
			return CollationError(col.table, col.column, err)
		}
	}

	return nil
}
