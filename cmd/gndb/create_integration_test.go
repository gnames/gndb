package main

import (
	"context"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that requires PostgreSQL.
// See operator_test.go for configuration instructions.
// Skip with: go test -short

// TestCreateCommand_Integration tests the complete create workflow end-to-end.
// This test verifies:
//  1. Database connection
//  2. Schema creation via GORM AutoMigrate
//  3. Table existence verification
//  4. Collation settings
func TestCreateCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Create database operator
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err, "Should connect to database")
	defer op.Close()

	// Clean up any existing tables first
	_ = op.DropAllTables(ctx)

	// Create schema manager
	sm := ioschema.NewManager(op)

	// Execute schema creation
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "Schema creation should succeed")

	// Verify core tables were created
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
		assert.True(t, exists, "Table %s should exist after schema creation", table)
	}

	// Verify collation was set on name_strings.name column
	// Query the column definition
	query := `
		SELECT collation_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'name_strings'
		  AND column_name = 'name'
	`
	var collation string
	err = op.Pool().QueryRow(ctx, query).Scan(&collation)
	require.NoError(t, err, "Should be able to query collation")
	assert.Equal(t, "C", collation, "Collation should be set to 'C' for name_strings.name")

	// Clean up - drop all tables
	err = op.DropAllTables(ctx)
	assert.NoError(t, err, "Should be able to drop tables after test")
}

// TestCreateCommand_Integration_Idempotent tests that running create twice works.
// The second run should use GORM AutoMigrate which is idempotent.
func TestCreateCommand_Integration_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Create database operator
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean up
	_ = op.DropAllTables(ctx)

	// Create schema manager
	sm := ioschema.NewManager(op)

	// First creation
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "First schema creation should succeed")

	// Drop all tables to simulate fresh start
	err = op.DropAllTables(ctx)
	require.NoError(t, err)

	// Second creation (should also succeed)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "Second schema creation should succeed (idempotent)")

	// Verify tables still exist
	exists, err := op.TableExists(ctx, "name_strings")
	require.NoError(t, err)
	assert.True(t, exists, "name_strings table should exist after second create")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestCreateCommand_Integration_HasTables tests the HasTables functionality
// used by the create command to prompt users.
func TestCreateCommand_Integration_HasTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Create database operator
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean up first
	_ = op.DropAllTables(ctx)

	// Should have no tables initially
	hasTables, err := op.HasTables(ctx)
	require.NoError(t, err)
	assert.False(t, hasTables, "Database should have no tables initially")

	// Create schema
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Should now have tables
	hasTables, err = op.HasTables(ctx)
	require.NoError(t, err)
	assert.True(t, hasTables, "Database should have tables after schema creation")

	// Clean up
	_ = op.DropAllTables(ctx)
}
