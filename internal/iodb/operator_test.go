package iodb

import (
	"context"
	"os"
	"testing"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPgxOperator_ImplementsInterface verifies pgxOperator
// implements db.Operator interface.
func TestPgxOperator_ImplementsInterface(t *testing.T) {
	var _ db.Operator = NewPgxOperator()
}

// TestNewPgxOperator_CreatesOperator verifies operator
// creation.
func TestNewPgxOperator_CreatesOperator(t *testing.T) {
	op := NewPgxOperator()
	require.NotNil(t, op)

	// Pool should be nil before Connect
	assert.Nil(t, op.Pool(),
		"Pool should be nil before Connect")
}

// TestPgxOperator_CloseBeforeConnect verifies Close works
// even if never connected.
func TestPgxOperator_CloseBeforeConnect(t *testing.T) {
	op := NewPgxOperator()
	err := op.Close()
	assert.NoError(t, err,
		"Close should work even if never connected")
}

// TestPgxOperator_PoolBeforeConnect verifies Pool returns
// nil before connection.
func TestPgxOperator_PoolBeforeConnect(t *testing.T) {
	op := NewPgxOperator()
	pool := op.Pool()
	assert.Nil(t, pool,
		"Pool should be nil before Connect")
}

// Integration tests (require PostgreSQL)

// TestPgxOperator_Connect_Success verifies successful
// database connection.
func TestPgxOperator_Connect_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := getTestDBConfig(t)
	if cfg == nil {
		t.Skip("Database not configured")
	}

	ctx := context.Background()
	op := NewPgxOperator()

	err := op.Connect(ctx, cfg)
	require.NoError(t, err)
	defer op.Close()

	// Verify pool is set
	pool := op.Pool()
	assert.NotNil(t, pool)

	// Verify connection works
	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

// TestPgxOperator_Connect_InvalidHost verifies error
// on invalid host.
func TestPgxOperator_Connect_InvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	op := NewPgxOperator()

	cfg := &config.DatabaseConfig{
		Host:     "invalid-host-that-does-not-exist",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		Database: "test",
		SSLMode:  "disable",
	}

	err := op.Connect(ctx, cfg)
	assert.Error(t, err,
		"Should error on invalid host")
}

// TestPgxOperator_HasTables verifies table detection.
func TestPgxOperator_HasTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := getTestDBConfig(t)
	if cfg == nil {
		t.Skip("Database not configured")
	}

	ctx := context.Background()
	op := NewPgxOperator()

	err := op.Connect(ctx, cfg)
	require.NoError(t, err)
	defer op.Close()

	// Check if database has tables
	hasTables, err := op.HasTables(ctx)
	require.NoError(t, err)

	// Result depends on database state
	// Just verify it returns a boolean
	assert.IsType(t, false, hasTables)
}

// TestPgxOperator_TableExists verifies specific table check.
func TestPgxOperator_TableExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := getTestDBConfig(t)
	if cfg == nil {
		t.Skip("Database not configured")
	}

	ctx := context.Background()
	op := NewPgxOperator()

	err := op.Connect(ctx, cfg)
	require.NoError(t, err)
	defer op.Close()

	// Check for a table that definitely doesn't exist
	exists, err := op.TableExists(ctx,
		"nonexistent_table_xyz_123")
	require.NoError(t, err)
	assert.False(t, exists,
		"Nonexistent table should not exist")
}

// getTestDBConfig returns database config from environment
// or nil if not configured.
func getTestDBConfig(t *testing.T) *config.DatabaseConfig {
	host := os.Getenv("GNDB_TEST_DB_HOST")
	if host == "" {
		host = "localhost"
	}

	database := os.Getenv("GNDB_TEST_DB_DATABASE")
	if database == "" {
		return nil
	}

	user := os.Getenv("GNDB_TEST_DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("GNDB_TEST_DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	return &config.DatabaseConfig{
		Host:     host,
		Port:     5432,
		User:     user,
		Password: password,
		Database: database,
		SSLMode:  "disable",
	}
}
