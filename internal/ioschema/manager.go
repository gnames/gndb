// Package ioschema implements SchemaManager interface for
// database schema management. This is an impure I/O package
// that uses schema.Migrate for initial creation and Atlas
// declarative migrations for updates.
package ioschema

import (
	"context"
	"fmt"
	"time"

	atlasPG "ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/migrate"
	atlasschema "ariga.io/atlas/sql/schema"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/gndb"
	gndbschema "github.com/gnames/gndb/pkg/schema"
	"github.com/jackc/pgx/v5/stdlib"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// manager implements the gndb.SchemaManager interface.
type manager struct {
	operator db.Operator
	cfg      *config.Config
}

// NewManager creates a new SchemaManager.
func NewManager(
	op db.Operator,
	cfg *config.Config,
) gndb.SchemaManager {
	return &manager{
		operator: op,
		cfg:      cfg,
	}
}

// Create creates the initial database schema using
// schema.Migrate. Also applies collation settings for
// correct scientific name sorting.
func (m *manager) Create(ctx context.Context) error {
	pool := m.operator.Pool()
	if pool == nil {
		return NotConnectedError()
	}

	db := stdlib.OpenDBFromPool(pool)

	// Connect with GORM
	gormDB, err := gorm.Open(
		gormpg.New(gormpg.Config{Conn: db}),
		&gorm.Config{},
	)
	if err != nil {
		return GORMConnectionError(err)
	}

	// Create schema from GORM models
	if err := gndbschema.Migrate(gormDB); err != nil {
		return CreateSchemaError(err)
	}

	// Set collation for string columns
	// (critical for correct sorting)
	if err := m.setCollation(ctx); err != nil {
		return err
	}

	return nil
}

// Migrate updates the database schema to match the current model state
// using Atlas declarative migrations. It:
//  1. Creates a temporary dev schema and applies the GORM models there
//     to represent the desired schema state.
//  2. Inspects both the dev schema (desired) and the public schema (current).
//  3. Computes the diff and generates SQL statements.
//  4. Calls opts.Confirm with the SQL — proceeds only if it returns true.
//  5. Applies the changes and optionally recreates materialized views.
func (m *manager) Migrate(
	ctx context.Context,
	opts gndb.MigrateOptions,
) error {
	pool := m.operator.Pool()
	if pool == nil {
		return NotConnectedError()
	}

	// Drop materialized views before migration
	// (required for ALTER TABLE operations)
	if err := m.operator.DropMaterializedViews(ctx); err != nil {
		return err
	}

	// Open the Atlas Postgres driver on the shared connection pool.
	sqlDB := stdlib.OpenDBFromPool(pool)
	drv, err := atlasPG.Open(sqlDB)
	if err != nil {
		return AtlasDriverError(err)
	}

	// Inspect desired state by applying the GORM models to a
	// temporary dev schema, then reading it back via Atlas.
	desired, err := m.inspectDesiredSchema(ctx, drv)
	if err != nil {
		return err
	}

	// Inspect current state from the public schema.
	current, err := drv.InspectSchema(ctx, "public", nil)
	if err != nil {
		return AtlasInspectError("public", err)
	}

	// Compute the diff.
	changes, err := drv.SchemaDiff(current, desired)
	if err != nil {
		return AtlasDiffError(err)
	}

	if len(changes) == 0 {
		gn.Info("Schema is already up to date.")
		if opts.RecreateViews {
			return m.operator.CreateMaterializedViews(ctx)
		}
		return nil
	}

	// Translate changes to SQL statements for review.
	inPlace := migrate.PlanOption(func(o *migrate.PlanOptions) {
		o.Mode = migrate.PlanModeInPlace
	})
	plan, err := drv.PlanChanges(ctx, "", changes, inPlace)
	if err != nil {
		return AtlasPlanError(err)
	}

	stmts := make([]string, len(plan.Changes))
	for i, c := range plan.Changes {
		stmts[i] = c.Cmd
	}

	// Show the plan to the user and ask for confirmation.
	if !opts.Confirm(stmts) {
		return nil
	}

	// Apply changes.
	if err := drv.ApplyChanges(ctx, changes, inPlace); err != nil {
		return MigrateSchemaError(err)
	}

	// Re-apply collation on the public schema after structural changes.
	if err := m.setCollation(ctx); err != nil {
		return err
	}

	if opts.RecreateViews {
		return m.operator.CreateMaterializedViews(ctx)
	}

	return nil
}

// inspectDesiredSchema returns the Atlas schema representation of what
// the database should look like according to the current GORM models.
//
// It works by:
//  1. Creating a temporary dev schema
//  2. Applying the GORM models there to build the desired table structure
//  3. Applying collation so it matches production expectations
//  4. Inspecting it with Atlas and normalising the schema name to "public"
//  5. Dropping the dev schema on return
func (m *manager) inspectDesiredSchema(
	ctx context.Context,
	drv migrate.Driver,
) (*atlasschema.Schema, error) {
	pool := m.operator.Pool()

	devSchema := fmt.Sprintf("gndb_dev_%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx,
		"CREATE SCHEMA "+devSchema); err != nil {
		return nil, AtlasDevSchemaError(err)
	}
	defer pool.Exec(ctx, //nolint:errcheck
		"DROP SCHEMA IF EXISTS "+devSchema+" CASCADE")

	devDSN := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&search_path=%s",
		m.cfg.Database.User,
		m.cfg.Database.Password,
		m.cfg.Database.Host,
		m.cfg.Database.Port,
		m.cfg.Database.Database,
		m.cfg.Database.SSLMode,
		devSchema,
	)
	devGormDB, err := gorm.Open(gormpg.Open(devDSN), &gorm.Config{})
	if err != nil {
		return nil, GORMConnectionError(err)
	}
	devSQLDB, err := devGormDB.DB()
	if err != nil {
		return nil, GORMConnectionError(err)
	}
	defer devSQLDB.Close()

	if err := gndbschema.Migrate(devGormDB); err != nil {
		return nil, CreateSchemaError(err)
	}

	if err := m.setCollationOnSchema(ctx, devSchema); err != nil {
		return nil, err
	}

	result, err := drv.InspectSchema(ctx, devSchema, nil)
	if err != nil {
		return nil, AtlasInspectError(devSchema, err)
	}
	result.Name = "public"
	for _, t := range result.Tables {
		if t.Schema != nil {
			t.Schema = result
		}
	}

	return result, nil
}

// setCollation applies "C" collation to the public schema.
func (m *manager) setCollation(ctx context.Context) error {
	return m.setCollationOnSchema(ctx, "public")
}

// setCollationOnSchema sets "C" collation on string columns
// within the named schema. This is critical for correct sorting
// and comparison of scientific names.
func (m *manager) setCollationOnSchema(
	ctx context.Context,
	schemaName string,
) error {
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

	for _, col := range columns {
		q := fmt.Sprintf(
			`ALTER TABLE %s.%s ALTER COLUMN %s TYPE TEXT COLLATE "C"`,
			schemaName, col.table, col.column,
		)
		if _, err := pool.Exec(ctx, q); err != nil {
			return CollationError(col.table, col.column, err)
		}
	}

	return nil
}
