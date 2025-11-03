package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/iopopulate"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetPopulateCmd_Exists verifies getPopulateCmd returns
// a valid command.
func TestGetPopulateCmd_Exists(t *testing.T) {
	cmd := getPopulateCmd()
	require.NotNil(t, cmd, "Populate command should exist")
	assert.Equal(t, "populate", cmd.Use,
		"Command name should be populate")
}

// TestGetPopulateCmd_ShortDescription verifies short
// description.
func TestGetPopulateCmd_ShortDescription(t *testing.T) {
	cmd := getPopulateCmd()

	assert.NotEmpty(t, cmd.Short,
		"Short description should not be empty")
	assert.Contains(t, cmd.Short, "SFGA",
		"Short description should mention SFGA")
}

// TestGetPopulateCmd_LongDescription verifies long
// description.
func TestGetPopulateCmd_LongDescription(t *testing.T) {
	cmd := getPopulateCmd()

	assert.NotEmpty(t, cmd.Long,
		"Long description should not be empty")
	assert.Contains(t, cmd.Long, "SFGA",
		"Long description should mention SFGA")
	assert.Contains(t, cmd.Long, "sources.yaml",
		"Long description should mention config")
}

// TestGetPopulateCmd_HasRunE verifies run function is set.
func TestGetPopulateCmd_HasRunE(t *testing.T) {
	cmd := getPopulateCmd()

	assert.NotNil(t, cmd.RunE,
		"RunE should be set")
}

// TestGetPopulateCmd_SourceIDsFlag verifies --source-ids
// flag exists.
func TestGetPopulateCmd_SourceIDsFlag(t *testing.T) {
	cmd := getPopulateCmd()

	flag := cmd.Flags().Lookup("source-ids")
	require.NotNil(t, flag,
		"--source-ids flag should exist")

	assert.Equal(t, "s", flag.Shorthand,
		"Short form should be -s")
	assert.Contains(t, flag.Usage, "source IDs",
		"Usage should mention source IDs")
}

// TestGetPopulateCmd_ReleaseVersionFlag verifies
// --release-version flag exists.
func TestGetPopulateCmd_ReleaseVersionFlag(t *testing.T) {
	cmd := getPopulateCmd()

	flag := cmd.Flags().Lookup("release-version")
	require.NotNil(t, flag,
		"--release-version flag should exist")

	assert.Equal(t, "r", flag.Shorthand,
		"Short form should be -r")
	assert.Contains(t, flag.Usage, "version",
		"Usage should mention version")
}

// TestGetPopulateCmd_ReleaseDateFlag verifies
// --release-date flag exists.
func TestGetPopulateCmd_ReleaseDateFlag(t *testing.T) {
	cmd := getPopulateCmd()

	flag := cmd.Flags().Lookup("release-date")
	require.NotNil(t, flag,
		"--release-date flag should exist")

	assert.Equal(t, "d", flag.Shorthand,
		"Short form should be -d")
	assert.Contains(t, flag.Usage, "date",
		"Usage should mention date")
}

// TestGetPopulateCmd_FlatClassificationFlag verifies
// --flat-classification flag exists.
func TestGetPopulateCmd_FlatClassificationFlag(t *testing.T) {
	cmd := getPopulateCmd()

	flag := cmd.Flags().Lookup("flat-classification")
	require.NotNil(t, flag,
		"--flat-classification flag should exist")

	assert.Equal(t, "f", flag.Shorthand,
		"Short form should be -f")
	assert.Contains(t, flag.Usage, "classification",
		"Usage should mention classification")
}

// TestGetPopulateCmd_IndependentInstances verifies each
// call returns independent instance.
func TestGetPopulateCmd_IndependentInstances(t *testing.T) {
	cmd1 := getPopulateCmd()
	cmd2 := getPopulateCmd()

	// Should be different instances
	assert.NotSame(t, cmd1, cmd2,
		"Each call should return new instance")

	// Modifying one shouldn't affect the other
	cmd1.Short = "test1"
	cmd2.Short = "test2"

	assert.Equal(t, "test1", cmd1.Short)
	assert.Equal(t, "test2", cmd2.Short)
}

// Integration Tests

// TestPopulate_EndToEnd_VASCAN tests the complete populate
// workflow using source 1002 (VASCAN) from testdata.
func TestPopulate_EndToEnd_VASCAN(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := getTestDBConfig(t)
	if cfg == nil {
		t.Skip("Database not configured")
	}

	ctx := context.Background()

	// Create temporary home directory for test
	tmpHome := t.TempDir()

	// Create config directory structure
	configDir := config.ConfigDir(tmpHome)
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	// Create sources.yaml in test config dir pointing to actual testdata
	// Get absolute path to testdata directory
	testdataPath, err := filepath.Abs(filepath.Join("..", "testdata"))
	require.NoError(t, err)

	destSourcesPath := filepath.Join(configDir, "sources.yaml")

	// Create sources.yaml with absolute path to testdata
	sourcesYAML := fmt.Sprintf(`data_sources:
  # VASCAN
  - id: 1002
    parent: "%s"
    title_short: "VASCAN"
    is_curated: true
    has_classification: true
`, testdataPath)

	err = os.WriteFile(destSourcesPath, []byte(sourcesYAML), 0o644)
	require.NoError(t, err)

	// Create cache directory
	cacheDir := config.CacheDir(tmpHome)
	err = os.MkdirAll(cacheDir, 0o755)
	require.NoError(t, err)

	// Setup test config
	testCfg := config.New()
	testCfg.Update([]config.Option{
		config.OptHomeDir(tmpHome),
		config.OptDatabaseHost(cfg.Host),
		config.OptDatabasePort(cfg.Port),
		config.OptDatabaseUser(cfg.User),
		config.OptDatabasePassword(cfg.Password),
		config.OptDatabaseDatabase(cfg.Database),
		config.OptDatabaseSSLMode(cfg.SSLMode),
		config.OptPopulateSourceIDs([]int{1002}), // VASCAN
		config.OptJobsNumber(4),
	})

	// Connect to database
	op := iodb.NewPgxOperator()
	err = op.Connect(ctx, &testCfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean database for fresh test
	hasTables, err := op.HasTables(ctx)
	require.NoError(t, err)

	if hasTables {
		// Drop all existing tables
		err = op.DropAllTables(ctx)
		require.NoError(t, err)
	}

	// Create fresh schema
	mgr := ioschema.NewManager(op)
	err = mgr.Create(ctx, testCfg)
	require.NoError(t, err)

	// Run populate
	populator := iopopulate.New(testCfg, op)
	err = populator.Populate(ctx)
	require.NoError(t, err, "Populate should succeed")

	// Verify data was imported
	pool := op.Pool()
	require.NotNil(t, pool)

	// Check DataSource was created
	var dsCount int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM data_sources WHERE id = 1002",
	).Scan(&dsCount)
	require.NoError(t, err)
	assert.Greater(t, dsCount, 0,
		"DataSource 1002 should exist")

	// Check NameStrings were imported
	var nsCount int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM name_strings",
	).Scan(&nsCount)
	require.NoError(t, err)
	assert.Greater(t, nsCount, 0,
		"NameStrings should be imported")

	// Check NameStringIndices were created
	var nsiCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM name_string_indices
		 WHERE data_source_id = 1002`,
	).Scan(&nsiCount)
	require.NoError(t, err)
	assert.Greater(t, nsiCount, 0,
		"NameStringIndices should be created for VASCAN")

	// Verify specific VASCAN data characteristics
	// VASCAN has classification data
	var withClassification int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM name_string_indices
		 WHERE data_source_id = 1002
		 AND classification IS NOT NULL
		 AND classification != ''`,
	).Scan(&withClassification)
	require.NoError(t, err)
	assert.Greater(t, withClassification, 0,
		"VASCAN should have classification data")

	// Check vernacular strings were imported
	var vernCount int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM vernacular_strings",
	).Scan(&vernCount)
	require.NoError(t, err)
	assert.Greater(t, vernCount, 0,
		"Vernacular strings should be imported")

	t.Logf("VASCAN import successful:")
	t.Logf("  - NameStrings: %d", nsCount)
	t.Logf("  - NameStringIndices: %d", nsiCount)
	t.Logf("  - With classification: %d", withClassification)
	t.Logf("  - Vernacular strings: %d", vernCount)
}

// getTestDBConfig returns database config from environment
// or nil if not configured.
func getTestDBConfig(t *testing.T) *config.DatabaseConfig {
	t.Helper()

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
