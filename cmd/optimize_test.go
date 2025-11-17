package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/iooptimize"
	"github.com/gnames/gndb/internal/iopopulate"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetOptimizeCmd_Exists verifies getOptimizeCmd
// returns a valid command.
func TestGetOptimizeCmd_Exists(t *testing.T) {
	cmd := getOptimizeCmd()
	require.NotNil(t, cmd,
		"Optimize command should exist")
	assert.Equal(t, "optimize", cmd.Use,
		"Command name should be optimize")
}

// TestGetOptimizeCmd_ShortDescription verifies short
// description.
func TestGetOptimizeCmd_ShortDescription(t *testing.T) {
	cmd := getOptimizeCmd()

	assert.NotEmpty(t, cmd.Short,
		"Short description should not be empty")
	assert.Contains(t, cmd.Short, "gnverifier",
		"Short description should mention gnverifier")
}

// TestGetOptimizeCmd_LongDescription verifies long
// description.
func TestGetOptimizeCmd_LongDescription(t *testing.T) {
	cmd := getOptimizeCmd()

	assert.NotEmpty(t, cmd.Long,
		"Long description should not be empty")
	assert.Contains(t, cmd.Long, "Prerequisites",
		"Long description should mention prerequisites")
	assert.Contains(t, cmd.Long, "create",
		"Long description should mention create command")
	assert.Contains(t, cmd.Long, "populate",
		"Long description should mention populate command")
}

// TestGetOptimizeCmd_HasRunE verifies run function is set.
func TestGetOptimizeCmd_HasRunE(t *testing.T) {
	cmd := getOptimizeCmd()

	assert.NotNil(t, cmd.RunE,
		"RunE should be set")
}

// TestGetOptimizeCmd_IndependentInstances verifies each
// call returns independent instance.
func TestGetOptimizeCmd_IndependentInstances(t *testing.T) {
	cmd1 := getOptimizeCmd()
	cmd2 := getOptimizeCmd()

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

// TestOptimize_EndToEnd_VASCAN tests the complete
// optimization workflow using source 1002 (VASCAN)
// from testdata. This test verifies all 6 optimization
// phases complete successfully.
func TestOptimize_EndToEnd_VASCAN(t *testing.T) {
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

	// Create sources.yaml in test config dir pointing to
	// actual testdata
	testdataPath, err := filepath.Abs(
		filepath.Join("..", "testdata"),
	)
	require.NoError(t, err)

	destSourcesPath := filepath.Join(
		configDir, "sources.yaml",
	)

	// Create sources.yaml with absolute path to testdata
	sourcesYAML := fmt.Sprintf(`data_sources:
  # VASCAN
  - id: 1002
    parent: "%s"
    title_short: "VASCAN"
    is_curated: true
    has_classification: true
`, testdataPath)

	err = os.WriteFile(
		destSourcesPath,
		[]byte(sourcesYAML),
		0o644,
	)
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

	// Populate database with VASCAN data
	populator := iopopulate.New(testCfg, op)
	err = populator.Populate()
	require.NoError(t, err, "Populate should succeed")

	// Verify pre-optimization state
	pool := op.Pool()
	require.NotNil(t, pool)

	var nsCount int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM name_strings",
	).Scan(&nsCount)
	require.NoError(t, err)
	assert.Greater(t, nsCount, 0,
		"NameStrings should exist before optimization")

	// Verify words table is empty before optimization
	var wordsCountBefore int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM words",
	).Scan(&wordsCountBefore)
	require.NoError(t, err)
	assert.Equal(t, 0, wordsCountBefore,
		"Words table should be empty before optimization")

	// Verify verification view doesn't exist
	var viewExists bool
	err = pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT FROM pg_matviews
			WHERE matviewname = 'verification'
		)`,
	).Scan(&viewExists)
	require.NoError(t, err)
	assert.False(t, viewExists,
		"Verification view should not exist yet")

	// Run optimize
	optimizer := iooptimize.NewOptimizer(op)
	err = optimizer.Optimize(ctx, testCfg)
	require.NoError(t, err,
		"Optimize should succeed")

	// Verify Phase 1: Reparsing updated canonical forms
	var canonicalCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM name_strings
		 WHERE canonical_id IS NOT NULL`,
	).Scan(&canonicalCount)
	require.NoError(t, err)
	assert.Greater(t, canonicalCount, 0,
		"Canonical forms should be populated")

	// Verify Phase 2: Vernacular language normalization
	var vernacularCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM vernacular_string_indices
		 WHERE lang_code IS NOT NULL AND lang_code != ''`,
	).Scan(&vernacularCount)
	require.NoError(t, err)
	// Should have normalized language codes
	t.Logf("Vernacular indices with lang_code: %d",
		vernacularCount)

	// Verify Phase 3: Orphan removal
	// Check no name_strings exist without indices
	var orphanCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM name_strings ns
		 LEFT OUTER JOIN name_string_indices nsi
		 ON ns.id = nsi.name_string_id
		 WHERE nsi.name_string_id IS NULL`,
	).Scan(&orphanCount)
	require.NoError(t, err)
	assert.Equal(t, 0, orphanCount,
		"No orphan name_strings should remain")

	// Verify Phase 4: Word extraction
	var wordsCount int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM words",
	).Scan(&wordsCount)
	require.NoError(t, err)
	assert.Greater(t, wordsCount, 0,
		"Words should be extracted")

	var wordNameStringCount int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM word_name_strings",
	).Scan(&wordNameStringCount)
	require.NoError(t, err)
	assert.Greater(t, wordNameStringCount, 0,
		"Word-name linkages should be created")

	// Verify Phase 5: Verification view creation
	err = pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT FROM pg_matviews
			WHERE matviewname = 'verification'
		)`,
	).Scan(&viewExists)
	require.NoError(t, err)
	assert.True(t, viewExists,
		"Verification view should exist")

	var verificationCount int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM verification",
	).Scan(&verificationCount)
	require.NoError(t, err)
	assert.Greater(t, verificationCount, 0,
		"Verification view should have records")

	// Verify verification view has required indexes
	var indexCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM pg_indexes
		 WHERE tablename = 'verification'`,
	).Scan(&indexCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, indexCount, 3,
		"Verification view should have at least 3 indexes")

	// Verify Phase 6: VACUUM ANALYZE completed
	// (no direct verification, but command succeeded)

	t.Logf("VASCAN optimization successful:")
	t.Logf("  - NameStrings: %d", nsCount)
	t.Logf("  - Canonical forms: %d", canonicalCount)
	t.Logf("  - Words: %d", wordsCount)
	t.Logf("  - Word linkages: %d", wordNameStringCount)
	t.Logf("  - Verification records: %d",
		verificationCount)
	t.Logf("  - Verification indexes: %d", indexCount)
}

// TestOptimize_EmptyDatabase verifies optimizer handles
// empty database gracefully.
func TestOptimize_EmptyDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := getTestDBConfig(t)
	if cfg == nil {
		t.Skip("Database not configured")
	}

	ctx := context.Background()

	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

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
		config.OptJobsNumber(4),
	})

	// Connect to database
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &testCfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean database
	hasTables, err := op.HasTables(ctx)
	require.NoError(t, err)

	if hasTables {
		err = op.DropAllTables(ctx)
		require.NoError(t, err)
	}

	// Create empty schema
	mgr := ioschema.NewManager(op)
	err = mgr.Create(ctx, testCfg)
	require.NoError(t, err)

	// Run optimize on empty database
	optimizer := iooptimize.NewOptimizer(op)
	err = optimizer.Optimize(ctx, testCfg)

	// Should succeed even with no data
	require.NoError(t, err,
		"Optimize should handle empty database")

	// Verify verification view was created
	// (even if empty)
	pool := op.Pool()
	var viewExists bool
	err = pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT FROM pg_matviews
			WHERE matviewname = 'verification'
		)`,
	).Scan(&viewExists)
	require.NoError(t, err)
	assert.True(t, viewExists,
		"Verification view should exist even if empty")
}

// TestOptimize_Idempotent verifies running optimize
// multiple times is safe.
func TestOptimize_Idempotent(t *testing.T) {
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

	// Create sources.yaml
	testdataPath, err := filepath.Abs(
		filepath.Join("..", "testdata"),
	)
	require.NoError(t, err)

	destSourcesPath := filepath.Join(
		configDir, "sources.yaml",
	)

	sourcesYAML := fmt.Sprintf(`data_sources:
  - id: 1002
    parent: "%s"
    title_short: "VASCAN"
    is_curated: true
    has_classification: true
`, testdataPath)

	err = os.WriteFile(
		destSourcesPath,
		[]byte(sourcesYAML),
		0o644,
	)
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
		config.OptPopulateSourceIDs([]int{1002}),
		config.OptJobsNumber(4),
	})

	// Connect to database
	op := iodb.NewPgxOperator()
	err = op.Connect(ctx, &testCfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean and setup database
	hasTables, err := op.HasTables(ctx)
	require.NoError(t, err)

	if hasTables {
		err = op.DropAllTables(ctx)
		require.NoError(t, err)
	}

	mgr := ioschema.NewManager(op)
	err = mgr.Create(ctx, testCfg)
	require.NoError(t, err)

	// Populate database
	populator := iopopulate.New(testCfg, op)
	err = populator.Populate()
	require.NoError(t, err)

	// Run optimize first time
	optimizer := iooptimize.NewOptimizer(op)
	err = optimizer.Optimize(ctx, testCfg)
	require.NoError(t, err, "First optimize should succeed")

	// Get counts after first run
	pool := op.Pool()
	var wordsCount1 int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM words",
	).Scan(&wordsCount1)
	require.NoError(t, err)

	var verificationCount1 int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM verification",
	).Scan(&verificationCount1)
	require.NoError(t, err)

	// Run optimize second time
	err = optimizer.Optimize(ctx, testCfg)
	require.NoError(t, err,
		"Second optimize should succeed")

	// Get counts after second run
	var wordsCount2 int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM words",
	).Scan(&wordsCount2)
	require.NoError(t, err)

	var verificationCount2 int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM verification",
	).Scan(&verificationCount2)
	require.NoError(t, err)

	// Counts should be consistent
	assert.Equal(t, wordsCount1, wordsCount2,
		"Words count should be stable across runs")
	assert.Equal(t,
		verificationCount1,
		verificationCount2,
		"Verification count should be stable across runs")

	t.Logf("Idempotency verified:")
	t.Logf("  - Words: %d (both runs)", wordsCount1)
	t.Logf("  - Verification: %d (both runs)",
		verificationCount1)
}
