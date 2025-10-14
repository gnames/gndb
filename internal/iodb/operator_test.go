package iodb_test

import (
	"context"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These are integration tests that require PostgreSQL.
//
// Configuration is loaded using the full config system:
//   1. Environment variables (GNDB_DATABASE_* via .envrc)
//   2. Config file (~/.config/gndb/config.yaml)
//   3. Built-in defaults (postgres/postgres/gndb_test)
//
// Configuration examples:
//
// Option 1: Use .envrc (recommended for local development):
//   export GNDB_DATABASE_USER=your_user
//   export GNDB_DATABASE_PASSWORD=your_password
//   # Database name is always forced to "gndb_test" for safety
//
// Option 2: Use config.yaml:
//   database:
//     user: your_user
//     password: your_password
//   # Database name is always forced to "gndb_test" for safety
//
// Option 3: Use Docker with default credentials:
//   docker run -d --name gndb-test -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:15
//   # Tests will use defaults: postgres/postgres/gndb_test
//
// Skip these tests in CI without testcontainers support using:
//   go test -short (these tests will be skipped)

func TestPgxOperator_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := iodb.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, iotesting.GetTestDatabaseConfig())
	require.NoError(t, err, "Connect should succeed with valid config")

	defer op.Close()

	// Verify connection works by checking if we can query tables
	exists, err := op.TableExists(ctx, "nonexistent_table")
	assert.NoError(t, err, "Should be able to execute commands after Connect")
	assert.False(t, exists)
}

func TestPgxOperator_Connect_InvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := iodb.NewPgxOperator()
	ctx := context.Background()

	cfg := iotesting.GetTestDatabaseConfig()
	cfg.Host = "invalid-host-that-does-not-exist"

	err := op.Connect(ctx, cfg)
	assert.Error(t, err, "Connect should fail with invalid host")
}

func TestPgxOperator_TableExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := iodb.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, iotesting.GetTestDatabaseConfig())
	require.NoError(t, err)
	defer op.Close()

	// Clean up any existing test table
	_, _ = op.Pool().Exec(ctx, "DROP TABLE IF EXISTS test_table_exists CASCADE")

	// Table should not exist initially
	exists, err := op.TableExists(ctx, "test_table_exists")
	require.NoError(t, err)
	assert.False(t, exists, "Table should not exist initially")

	// Create table
	_, err = op.Pool().Exec(ctx, "CREATE TABLE test_table_exists (id SERIAL PRIMARY KEY)")
	require.NoError(t, err)

	// Table should now exist
	exists, err = op.TableExists(ctx, "test_table_exists")
	require.NoError(t, err)
	assert.True(t, exists, "Table should exist after creation")

	// Clean up
	_, _ = op.Pool().Exec(ctx, "DROP TABLE test_table_exists")
}

func TestPgxOperator_DropAllTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := iodb.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, iotesting.GetTestDatabaseConfig())
	require.NoError(t, err)
	defer op.Close()

	// Create some test tables
	_, _ = op.Pool().Exec(ctx, "CREATE TABLE IF NOT EXISTS drop_test1 (id SERIAL PRIMARY KEY)")
	_, _ = op.Pool().Exec(ctx, "CREATE TABLE IF NOT EXISTS drop_test2 (id SERIAL PRIMARY KEY)")

	// Drop all tables
	err = op.DropAllTables(ctx)
	require.NoError(t, err)

	// Verify tables are gone
	exists1, _ := op.TableExists(ctx, "drop_test1")
	exists2, _ := op.TableExists(ctx, "drop_test2")
	assert.False(t, exists1, "drop_test1 should be dropped")
	assert.False(t, exists2, "drop_test2 should be dropped")
}
