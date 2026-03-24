package ioexport

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/gndb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_ImplementsInterface verifies New returns a gndb.Exporter.
func TestNew_ImplementsInterface(t *testing.T) {
	cfg := config.New()
	op := iodb.NewPgxOperator()

	var _ gndb.Exporter = New(cfg, op)
}

// TestNew_StoresConfig verifies cfg and operator are stored.
func TestNew_StoresConfig(t *testing.T) {
	cfg := config.New()
	op := iodb.NewPgxOperator()

	e := New(cfg, op).(*exporter)

	assert.Same(t, cfg, e.cfg)
	assert.Equal(t, op, e.operator)
	assert.Nil(t, e.sources)
}

// TestInit_NilPool_ReturnsNotConnectedError verifies Init fails fast
// when the operator has no pool (never connected).
func TestInit_NilPool_ReturnsNotConnectedError(t *testing.T) {
	cfg := config.New()
	op := iodb.NewPgxOperator() // pool is nil before Connect

	e := New(cfg, op)
	err := e.(*exporter).Init(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

// TestEnsureOutputDir_CreatesNewDirectory verifies the output directory
// is created when it does not exist.
func TestEnsureOutputDir_CreatesNewDirectory(t *testing.T) {
	dir := t.TempDir()
	newDir := fmt.Sprintf("%s/subdir/output", dir)

	cfg := config.New()
	cfg.Update([]config.Option{config.OptExportOutputDir(newDir)})

	e := &exporter{cfg: cfg}
	err := e.ensureOutputDir()

	require.NoError(t, err)
	_, statErr := os.Stat(newDir)
	assert.NoError(t, statErr, "Directory should exist after ensureOutputDir")
}

// TestEnsureOutputDir_ExistingDirectory verifies no error when the
// directory already exists.
func TestEnsureOutputDir_ExistingDirectory(t *testing.T) {
	dir := t.TempDir()

	cfg := config.New()
	cfg.Update([]config.Option{config.OptExportOutputDir(dir)})

	e := &exporter{cfg: cfg}
	err := e.ensureOutputDir()

	assert.NoError(t, err)
}

// TestEnsureOutputDir_EmptyOutputDir_UsesDot verifies that an empty
// OutputDir defaults to "." without error.
func TestEnsureOutputDir_EmptyOutputDir_UsesDot(t *testing.T) {
	cfg := config.New()
	// OutputDir is "" by default

	e := &exporter{cfg: cfg}
	err := e.ensureOutputDir()

	assert.NoError(t, err)
}

// Integration tests (require PostgreSQL)

// TestInit_Integration_LoadsAllSources verifies Init loads all sources
// from data_sources when no SourceIDs filter is set.
func TestInit_Integration_LoadsAllSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg, op := mustConnectTestDB(t)
	defer op.Close()

	e := New(cfg, op).(*exporter)
	err := e.Init(context.Background())

	require.NoError(t, err)
	assert.NotEmpty(t, e.sources, "Should load at least one source")
}

// TestInit_Integration_FiltersBySourceID verifies Init returns only
// the requested source IDs when SourceIDs is set.
func TestInit_Integration_FiltersBySourceID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg, op := mustConnectTestDB(t)
	defer op.Close()

	// Load all first to find a valid ID.
	e := New(cfg, op).(*exporter)
	err := e.Init(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, e.sources)

	firstID := e.sources[0].ID

	// Now filter to that single ID.
	cfg.Update([]config.Option{
		config.OptExportSourceIDs([]int{firstID}),
	})
	e2 := New(cfg, op).(*exporter)
	err = e2.Init(context.Background())

	require.NoError(t, err)
	require.Len(t, e2.sources, 1)
	assert.Equal(t, firstID, e2.sources[0].ID)
}

// TestInit_Integration_NoMatchingIDs_ReturnsError verifies Init returns
// a NoSourcesError when none of the requested IDs exist.
func TestInit_Integration_NoMatchingIDs_ReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg, op := mustConnectTestDB(t)
	defer op.Close()

	cfg.Update([]config.Option{
		config.OptExportSourceIDs([]int{999999}),
	})

	e := New(cfg, op).(*exporter)
	err := e.Init(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "999999")
}

// mustConnectTestDB connects to the test database and returns cfg and op.
// Skips the test if the database is not configured.
func mustConnectTestDB(t *testing.T) (*config.Config, db.Operator) {
	t.Helper()

	host := os.Getenv("GNDB_TEST_DB_HOST")
	if host == "" {
		host = "localhost"
	}

	database := os.Getenv("GNDB_TEST_DB_DATABASE")
	if database == "" {
		t.Skip("GNDB_TEST_DB_DATABASE not set — skipping integration test")
	}

	user := os.Getenv("GNDB_TEST_DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("GNDB_TEST_DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	dbCfg := &config.DatabaseConfig{
		Host:     host,
		Port:     5432,
		User:     user,
		Password: password,
		Database: database,
		SSLMode:  "disable",
	}

	cfg := config.New()
	cfg.Update([]config.Option{
		config.OptExportOutputDir(t.TempDir()),
	})

	op := iodb.NewPgxOperator()
	err := op.Connect(context.Background(), dbCfg)
	require.NoError(t, err, "Failed to connect to test database")

	return cfg, op
}
