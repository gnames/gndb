package main

import (
	"context"
	"testing"

	"github.com/gnames/gndb/internal/io/database"
	"github.com/gnames/gndb/internal/io/schema"
	iotesting "github.com/gnames/gndb/internal/io/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that requires PostgreSQL.
// See operator_test.go for configuration instructions.
// Skip with: go test -short

// TestMigrateCommand_Integration tests the complete migrate workflow end-to-end.
// This test verifies:
//  1. Database connection
//  2. Schema creation (prerequisite)
//  3. Schema migration via GORM AutoMigrate
//  4. Verification that migration is idempotent
func TestMigrateCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Create database operator
	op := database.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err, "Should connect to database")
	defer op.Close()

	// Clean up any existing tables first
	_ = op.DropAllTables(ctx)

	// Create schema manager
	sm := schema.NewManager(op)

	// First, create the initial schema
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "Initial schema creation should succeed")

	// Verify initial tables exist
	exists, err := op.TableExists(ctx, "name_strings")
	require.NoError(t, err)
	require.True(t, exists, "name_strings should exist after initial creation")

	// Now run migration (should be idempotent on existing schema)
	err = sm.Migrate(ctx, cfg)
	require.NoError(t, err, "Migration should succeed on existing schema")

	// Verify tables still exist after migration
	expectedTables := []string{
		"data_sources",
		"name_strings",
		"canonicals",
		"canonical_fulls",
		"canonical_stems",
		"name_string_indices",
		"words",
		"word_name_strings",
		"vernacular_strings",
		"vernacular_string_indices",
	}

	for _, table := range expectedTables {
		exists, err := op.TableExists(ctx, table)
		require.NoError(t, err, "Should be able to check table existence for %s", table)
		assert.True(t, exists, "Table %s should exist after migration", table)
	}

	// Clean up
	err = op.DropAllTables(ctx)
	assert.NoError(t, err, "Should be able to drop tables after test")
}

// TestMigrateCommand_Integration_Idempotent tests running migrate multiple times.
// GORM AutoMigrate should handle multiple runs gracefully.
func TestMigrateCommand_Integration_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Create database operator
	op := database.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean up
	_ = op.DropAllTables(ctx)

	// Create schema manager
	sm := schema.NewManager(op)

	// Create initial schema
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "Initial schema creation should succeed")

	// Run migration first time
	err = sm.Migrate(ctx, cfg)
	require.NoError(t, err, "First migration should succeed")

	// Run migration second time (should be idempotent)
	err = sm.Migrate(ctx, cfg)
	require.NoError(t, err, "Second migration should succeed (idempotent)")

	// Run migration third time (verify truly idempotent)
	err = sm.Migrate(ctx, cfg)
	require.NoError(t, err, "Third migration should succeed (idempotent)")

	// Verify tables still exist after multiple migrations
	exists, err := op.TableExists(ctx, "name_strings")
	require.NoError(t, err)
	assert.True(t, exists, "name_strings should exist after multiple migrations")

	exists, err = op.TableExists(ctx, "data_sources")
	require.NoError(t, err)
	assert.True(t, exists, "data_sources should exist after multiple migrations")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestMigrateCommand_Integration_WithoutInitialSchema tests migrate on empty database.
// GORM AutoMigrate should create schema if it doesn't exist.
func TestMigrateCommand_Integration_WithoutInitialSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Create database operator
	op := database.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean up - ensure no tables exist
	_ = op.DropAllTables(ctx)

	// Verify no tables exist
	hasTables, err := op.HasTables(ctx)
	require.NoError(t, err)
	require.False(t, hasTables, "Database should be empty initially")

	// Create schema manager
	sm := schema.NewManager(op)

	// Run migration on empty database (should create schema)
	err = sm.Migrate(ctx, cfg)
	require.NoError(t, err, "Migration should create schema on empty database")

	// Verify tables were created
	exists, err := op.TableExists(ctx, "name_strings")
	require.NoError(t, err)
	assert.True(t, exists, "name_strings should exist after migration on empty database")

	exists, err = op.TableExists(ctx, "data_sources")
	require.NoError(t, err)
	assert.True(t, exists, "data_sources should exist after migration on empty database")

	// Verify all expected tables exist
	hasTables, err = op.HasTables(ctx)
	require.NoError(t, err)
	assert.True(t, hasTables, "Database should have tables after migration")

	// Clean up
	_ = op.DropAllTables(ctx)
}
