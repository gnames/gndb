package database_test

import (
	"context"
	"os"
	"testing"

	"github.com/gnames/gndb/internal/io/database"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These are integration tests that require PostgreSQL.
// For local testing, ensure PostgreSQL is running:
//
// Option 1: Use Docker with default credentials:
//   docker run -d --name gndb-test -e POSTGRES_PASSWORD=test -p 5432:5432 postgres:15
//
// Option 2: Use .envrc or environment variables:
//   export GNDB_DATABASE_USER=your_user
//   export GNDB_DATABASE_PASSWORD=your_password
//
// Skip these tests in CI without testcontainers support using:
//   go test -short (these tests will be skipped)

func getTestConfig() *config.DatabaseConfig {
	// Start with defaults
	cfg := config.Defaults()

	// Override with environment variables if present
	if user := os.Getenv("GNDB_DATABASE_USER"); user != "" {
		cfg.Database.User = user
	}
	if password := os.Getenv("GNDB_DATABASE_PASSWORD"); password != "" {
		cfg.Database.Password = password
	}

	// Always use test database
	cfg.Database.Database = "gndb_test"

	return &cfg.Database
}

func TestPgxOperator_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err, "Connect should succeed with valid config")

	defer op.Close()

	// Verify connection works
	err = op.EnableExtension(ctx, "plpgsql") // plpgsql is always available
	assert.NoError(t, err, "Should be able to execute commands after Connect")
}

func TestPgxOperator_Connect_InvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	cfg := getTestConfig()
	cfg.Host = "invalid-host-that-does-not-exist"

	err := op.Connect(ctx, cfg)
	assert.Error(t, err, "Connect should fail with invalid host")
}

func TestPgxOperator_TableExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err)
	defer op.Close()

	// Clean up any existing test table
	_ = op.ExecuteDDL(ctx, "DROP TABLE IF EXISTS test_table_exists CASCADE")

	// Table should not exist initially
	exists, err := op.TableExists(ctx, "test_table_exists")
	require.NoError(t, err)
	assert.False(t, exists, "Table should not exist initially")

	// Create table
	err = op.ExecuteDDL(ctx, "CREATE TABLE test_table_exists (id SERIAL PRIMARY KEY)")
	require.NoError(t, err)

	// Table should now exist
	exists, err = op.TableExists(ctx, "test_table_exists")
	require.NoError(t, err)
	assert.True(t, exists, "Table should exist after creation")

	// Clean up
	_ = op.ExecuteDDL(ctx, "DROP TABLE test_table_exists")
}

func TestPgxOperator_ExecuteDDL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err)
	defer op.Close()

	// Clean up
	_ = op.ExecuteDDL(ctx, "DROP TABLE IF EXISTS test_execute_ddl CASCADE")

	// Execute valid DDL
	err = op.ExecuteDDL(ctx, "CREATE TABLE test_execute_ddl (id SERIAL PRIMARY KEY, name TEXT)")
	require.NoError(t, err)

	// Verify table was created
	exists, err := op.TableExists(ctx, "test_execute_ddl")
	require.NoError(t, err)
	assert.True(t, exists)

	// Execute invalid DDL should fail and rollback
	err = op.ExecuteDDL(ctx, "CREATE TABLE invalid syntax here")
	assert.Error(t, err, "Invalid DDL should fail")

	// Clean up
	_ = op.ExecuteDDL(ctx, "DROP TABLE test_execute_ddl")
}

func TestPgxOperator_ExecuteDDLBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err)
	defer op.Close()

	// Clean up
	_ = op.ExecuteDDL(ctx, "DROP TABLE IF EXISTS test_batch1 CASCADE")
	_ = op.ExecuteDDL(ctx, "DROP TABLE IF EXISTS test_batch2 CASCADE")

	// Execute batch of DDL statements
	ddlStatements := []string{
		"CREATE TABLE test_batch1 (id SERIAL PRIMARY KEY)",
		"CREATE TABLE test_batch2 (id SERIAL PRIMARY KEY, ref_id INT REFERENCES test_batch1(id))",
	}

	err = op.ExecuteDDLBatch(ctx, ddlStatements)
	require.NoError(t, err)

	// Verify both tables exist
	exists1, _ := op.TableExists(ctx, "test_batch1")
	exists2, _ := op.TableExists(ctx, "test_batch2")
	assert.True(t, exists1)
	assert.True(t, exists2)

	// Clean up
	_ = op.ExecuteDDL(ctx, "DROP TABLE test_batch2")
	_ = op.ExecuteDDL(ctx, "DROP TABLE test_batch1")
}

func TestPgxOperator_CreateSchema_WithGORM(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err)
	defer op.Close()

	// Get all models for migration
	models := schema.AllModels()

	// Use GORM to generate schema (this is what we'll actually use)
	// For now, just verify we can create schema_versions table
	ddl := `
		CREATE TABLE IF NOT EXISTS schema_versions (
			version TEXT PRIMARY KEY,
			description TEXT,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	// Test CreateSchema with force=false (should succeed first time)
	err = op.CreateSchema(ctx, []string{ddl}, false)
	require.NoError(t, err)

	// Verify table exists
	exists, err := op.TableExists(ctx, "schema_versions")
	require.NoError(t, err)
	assert.True(t, exists)

	// Test CreateSchema with force=true (should drop and recreate)
	err = op.CreateSchema(ctx, []string{ddl}, true)
	require.NoError(t, err)

	// Verify table still exists after force recreation
	exists, err = op.TableExists(ctx, "schema_versions")
	require.NoError(t, err)
	assert.True(t, exists)

	// Prevent unused variable error
	_ = models
}

func TestPgxOperator_EnableExtension(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err)
	defer op.Close()

	// Enable pg_trgm extension (common for name matching)
	err = op.EnableExtension(ctx, "pg_trgm")
	require.NoError(t, err)

	// Enabling again should be idempotent
	err = op.EnableExtension(ctx, "pg_trgm")
	assert.NoError(t, err, "EnableExtension should be idempotent")
}

func TestPgxOperator_SchemaVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err)
	defer op.Close()

	// Create schema_versions table
	ddl := `
		CREATE TABLE IF NOT EXISTS schema_versions (
			version TEXT PRIMARY KEY,
			description TEXT,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_ = op.ExecuteDDL(ctx, "DROP TABLE IF EXISTS schema_versions CASCADE")
	err = op.ExecuteDDL(ctx, ddl)
	require.NoError(t, err)

	// Get version when table is empty
	version, err := op.GetSchemaVersion(ctx)
	require.NoError(t, err)
	assert.Empty(t, version, "Version should be empty initially")

	// Set version
	err = op.SetSchemaVersion(ctx, "1.0.0", "Initial schema")
	require.NoError(t, err)

	// Get version
	version, err = op.GetSchemaVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version)

	// Set another version
	err = op.SetSchemaVersion(ctx, "1.1.0", "Added indexes")
	require.NoError(t, err)

	// Get latest version
	version, err = op.GetSchemaVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", version, "Should return most recent version")

	// Clean up
	_ = op.ExecuteDDL(ctx, "DROP TABLE schema_versions")
}

func TestPgxOperator_DropAllTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err)
	defer op.Close()

	// Create some test tables
	_ = op.ExecuteDDL(ctx, "CREATE TABLE IF NOT EXISTS drop_test1 (id SERIAL PRIMARY KEY)")
	_ = op.ExecuteDDL(ctx, "CREATE TABLE IF NOT EXISTS drop_test2 (id SERIAL PRIMARY KEY)")

	// Drop all tables
	err = op.DropAllTables(ctx)
	require.NoError(t, err)

	// Verify tables are gone
	exists1, _ := op.TableExists(ctx, "drop_test1")
	exists2, _ := op.TableExists(ctx, "drop_test2")
	assert.False(t, exists1, "drop_test1 should be dropped")
	assert.False(t, exists2, "drop_test2 should be dropped")
}

func TestPgxOperator_DatabaseSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err)
	defer op.Close()

	// Get database size
	size, err := op.GetDatabaseSize(ctx)
	require.NoError(t, err)
	assert.Greater(t, size, int64(0), "Database size should be positive")
}

func TestPgxOperator_TableSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	op := database.NewPgxOperator()
	ctx := context.Background()

	err := op.Connect(ctx, getTestConfig())
	require.NoError(t, err)
	defer op.Close()

	// Create test table
	_ = op.ExecuteDDL(ctx, "DROP TABLE IF EXISTS size_test CASCADE")
	err = op.ExecuteDDL(ctx, "CREATE TABLE size_test (id SERIAL PRIMARY KEY, data TEXT)")
	require.NoError(t, err)

	// Get table size
	size, err := op.GetTableSize(ctx, "size_test")
	require.NoError(t, err)
	assert.Greater(t, size, int64(0), "Table size should be positive")

	// Clean up
	_ = op.ExecuteDDL(ctx, "DROP TABLE size_test")
}
