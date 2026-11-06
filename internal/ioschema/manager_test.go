package ioschema

import (
	"context"
	"os"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/gndb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManager_ImplementsInterface verifies manager
// implements gndb.SchemaManager interface.
func TestManager_ImplementsInterface(t *testing.T) {
	op := iodb.NewPgxOperator()
	cfg := config.New()
	var _ gndb.SchemaManager = NewManager(op, cfg)
}

// TestNewManager_CreatesManager verifies manager creation.
func TestNewManager_CreatesManager(t *testing.T) {
	op := iodb.NewPgxOperator()
	cfg := config.New()
	mgr := NewManager(op, cfg)
	require.NotNil(t, mgr)
}

// TestManager_PrivateStruct verifies struct is private.
func TestManager_PrivateStruct(t *testing.T) {
	// This test verifies that the manager struct
	// is not exported. If this compiles, the pattern
	// is correct.
	op := iodb.NewPgxOperator()
	cfg := config.New()
	var _ gndb.SchemaManager = NewManager(op, cfg)

	// Cannot create: var m *manager  (would fail to compile)
	// Can only use: NewManager() returns interface
}

// Integration tests would require:
// - Database connection
// - GORM setup
// - Schema migration testing
// These are better suited for E2E tests

// TestManager_Migrate_KeepsViews verifies that a migration that applies
// nothing leaves materialized views intact. A dry-run and a cancelled
// migration both end with Confirm returning false.
func TestManager_Migrate_KeepsViews(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	database := os.Getenv("GNDB_TEST_DB_DATABASE")
	if database == "" {
		t.Skip("Database not configured")
	}

	cfg := config.New()
	cfg.Update([]config.Option{config.OptDatabaseDatabase(database)})
	if host := os.Getenv("GNDB_TEST_DB_HOST"); host != "" {
		cfg.Update([]config.Option{config.OptDatabaseHost(host)})
	}
	if user := os.Getenv("GNDB_TEST_DB_USER"); user != "" {
		cfg.Update([]config.Option{config.OptDatabaseUser(user)})
	}
	if pass := os.Getenv("GNDB_TEST_DB_PASSWORD"); pass != "" {
		cfg.Update([]config.Option{config.OptDatabasePassword(pass)})
	}

	ctx := context.Background()
	op := iodb.NewPgxOperator()
	require.NoError(t, op.Connect(ctx, &cfg.Database))
	defer op.Close()
	pool := op.Pool()

	const view = "migrate_keeps_views_test"
	dropView := "DROP MATERIALIZED VIEW IF EXISTS " + view
	_, err := pool.Exec(ctx, dropView)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "CREATE MATERIALIZED VIEW "+view+" AS SELECT 1")
	require.NoError(t, err)
	defer func() { _, _ = pool.Exec(ctx, dropView) }()

	opts := gndb.MigrateOptions{
		DryRun:  true,
		Confirm: func([]string) bool { return false },
	}
	require.NoError(t, NewManager(op, cfg).Migrate(ctx, opts))

	var exists bool
	err = pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_matviews WHERE matviewname = $1)",
		view,
	).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "view should survive a migration without changes")
}
