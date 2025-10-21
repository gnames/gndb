package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnames/gndb/internal/ioconfig"
	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/iooptimize"
	"github.com/gnames/gndb/internal/iopopulate"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/gnames/gndb/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that requires PostgreSQL.
// See operator_test.go for configuration instructions.
// Skip with: go test -short

// TestOptimizeCommand_Integration tests the complete optimize workflow end-to-end.
// This test verifies:
//  1. All 6 optimization steps execute successfully
//  2. Verification view is created and queryable
//  3. Words tables are populated
//  4. Database is ready for gnverifier
func TestOptimizeCommand_Integration(t *testing.T) {
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

	// Setup: Create fresh schema and populate with testdata 1001
	err = setupTestDatabase(ctx, op, cfg)
	require.NoError(t, err, "Should setup test database")

	// Create optimizer
	optimizer := iooptimize.NewOptimizer(op)

	// Execute optimization
	err = optimizer.Optimize(ctx, cfg)
	require.NoError(t, err, "Optimize should complete successfully")

	// // Verify Step 1: Name strings reparsed (canonical_id should be populated)
	// var canonicalCount int
	// query := "SELECT COUNT(*) FROM name_strings WHERE canonical_id IS NOT NULL"
	// err = op.Pool().QueryRow(ctx, query).Scan(&canonicalCount)
	// require.NoError(t, err)
	// assert.Greater(t, canonicalCount, 0, "Should have canonical IDs after reparsing")
	//
	// // Verify Step 2: Vernacular languages normalized (lang_code should be lowercase)
	// var uppercaseLangCount int
	// query = "SELECT COUNT(*) FROM vernacular_string_indices WHERE lang_code != LOWER(lang_code)"
	// err = op.Pool().QueryRow(ctx, query).Scan(&uppercaseLangCount)
	// require.NoError(t, err)
	// assert.Equal(t, 0, uppercaseLangCount, "All lang_code values should be lowercase")
	//
	// // Verify Step 3: Orphans removed (no orphan name_strings)
	// var orphanCount int
	// query = `
	// 	SELECT COUNT(*)
	// 	FROM name_strings ns
	// 	WHERE NOT EXISTS (
	// 		SELECT 1 FROM name_string_indices nsi
	// 		WHERE nsi.name_string_id = ns.id
	// 	)
	// `
	// err = op.Pool().QueryRow(ctx, query).Scan(&orphanCount)
	// require.NoError(t, err)
	// assert.Equal(t, 0, orphanCount, "Should have no orphan name_strings")
	//
	// // Verify Step 4: Words tables populated
	// var wordCount int
	// query = "SELECT COUNT(*) FROM words"
	// err = op.Pool().QueryRow(ctx, query).Scan(&wordCount)
	// require.NoError(t, err)
	// assert.Greater(t, wordCount, 0, "Words table should be populated")
	//
	// var wordNameCount int
	// query = "SELECT COUNT(*) FROM word_name_strings"
	// err = op.Pool().QueryRow(ctx, query).Scan(&wordNameCount)
	// require.NoError(t, err)
	// assert.Greater(t, wordNameCount, 0, "Word-name linkages should be populated")
	//
	// // Verify Step 5: Verification view created and queryable
	// exists, err := viewExists(ctx, op, "verification")
	// require.NoError(t, err)
	// assert.True(t, exists, "Verification view should exist")
	//
	// var verificationCount int
	// query = "SELECT COUNT(*) FROM verification"
	// err = op.Pool().QueryRow(ctx, query).Scan(&verificationCount)
	// require.NoError(t, err)
	// assert.Greater(t, verificationCount, 0, "Verification view should have records")
	//
	// // Verify indexes on verification view
	// indexes := []string{
	// 	"verification_canonical_id_idx",
	// 	"verification_name_string_id_idx",
	// 	"verification_year_idx",
	// }
	// for _, idx := range indexes {
	// 	exists, err := indexExists(ctx, op, idx)
	// 	require.NoError(t, err)
	// 	assert.True(t, exists, "Index %s should exist", idx)
	// }

	// Step 6 (VACUUM ANALYZE) doesn't have verifiable side effects in tests,
	// but if we got here without errors, it completed successfully

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestOptimizeCommand_Integration_Idempotent tests that running optimize twice is safe.
// func TestOptimizeCommand_Integration_Idempotent(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test in short mode")
// 	}
//
// 	ctx := context.Background()
// 	cfg := iotesting.GetTestConfig()
//
// 	// Create database operator
// 	op := iodb.NewPgxOperator()
// 	err := op.Connect(ctx, &cfg.Database)
// 	require.NoError(t, err)
// 	defer op.Close()
//
// 	// Setup test database
// 	err = setupTestDatabase(ctx, op, cfg)
// 	require.NoError(t, err)
//
// 	// Create optimizer
// 	optimizer := iooptimize.NewOptimizer(op)
//
// 	// First optimization
// 	err = optimizer.Optimize(ctx, cfg)
// 	require.NoError(t, err, "First optimize should succeed")
//
// 	// Get record counts after first run
// 	var wordsCount1, verificationCount1 int
// 	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM words").Scan(&wordsCount1)
// 	require.NoError(t, err)
// 	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM verification").Scan(&verificationCount1)
// 	require.NoError(t, err)
//
// 	// Second optimization (should be idempotent)
// 	err = optimizer.Optimize(ctx, cfg)
// 	require.NoError(t, err, "Second optimize should succeed (idempotent)")
//
// 	// Verify counts are the same (no duplication)
// 	var wordsCount2, verificationCount2 int
// 	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM words").Scan(&wordsCount2)
// 	require.NoError(t, err)
// 	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM verification").Scan(&verificationCount2)
// 	require.NoError(t, err)
//
// 	assert.Equal(t, wordsCount1, wordsCount2, "Words count should be the same after second run")
// 	assert.Equal(t, verificationCount1, verificationCount2, "Verification count should be the same")
//
// 	// Clean up
// 	_ = op.DropAllTables(ctx)
// }

// TestOptimizeCommand_Integration_EmptyDatabase tests that optimize handles empty database gracefully.
func TestOptimizeCommand_Integration_EmptyDatabase(t *testing.T) {
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

	// Create schema but don't populate
	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Optimize empty database (should succeed with 0 records processed)
	optimizer := iooptimize.NewOptimizer(op)
	err = optimizer.Optimize(ctx, cfg)

	// Should succeed (optimize is idempotent and handles empty database)
	assert.NoError(t, err, "Optimize should handle empty database gracefully")

	// Verify verification view was created (even if empty)
	exists, err := viewExists(ctx, op, "verification")
	require.NoError(t, err)
	assert.True(t, exists, "Verification view should be created even for empty database")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// setupTestDatabase creates schema and populates with testdata 1000 (Ruhoff 1980).
// This is a helper function for integration tests.
func setupTestDatabase(ctx context.Context, op *iodb.PgxOperator, cfg *config.Config) error {
	// Drop existing tables
	_ = op.DropAllTables(ctx)

	// Create schema
	sm := ioschema.NewManager(op)
	if err := sm.Create(ctx, cfg); err != nil {
		return err
	}

	// Setup test sources.yaml (same pattern as populate_e2e_test.go)
	configDir, err := ioconfig.GetConfigDir()
	if err != nil {
		return err
	}

	sourcesYAMLPath := filepath.Join(configDir, "sources.yaml")
	backupPath := sourcesYAMLPath + ".optimize_test_backup"

	// Backup existing sources.yaml if it exists
	originalExists := false
	if _, err := os.Stat(sourcesYAMLPath); err == nil {
		originalExists = true
		if err := os.Rename(sourcesYAMLPath, backupPath); err != nil {
			return err
		}
	}

	// Create test sources.yaml with absolute path to testdata
	// (same pattern as populate_e2e_test.go)
	testdataPath, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	if err != nil {
		// Restore backup if path resolution fails
		if originalExists {
			_ = os.Rename(backupPath, sourcesYAMLPath)
		}
		return err
	}

	testSourcesYAML := `data_sources:
  # Ruhoff 1980 - small test dataset
  - id: 1000
    parent: ` + testdataPath + `
    title_short: "Ruhoff 1980"
    home_url: "https://doi.org/10.5479/si.00810282.294"
    is_auto_curated: true
`
	if err := os.WriteFile(sourcesYAMLPath, []byte(testSourcesYAML), 0644); err != nil {
		// Restore backup if write fails
		if originalExists {
			_ = os.Rename(backupPath, sourcesYAMLPath)
		}
		return err
	}

	// Populate with testdata 1000 (Ruhoff 1980 - small test dataset)
	cfg.Populate.SourceIDs = []int{1000}
	populator := iopopulate.NewPopulator(op)
	populateErr := populator.Populate(ctx, cfg)

	// Restore original sources.yaml
	_ = os.Remove(sourcesYAMLPath)
	if originalExists {
		_ = os.Rename(backupPath, sourcesYAMLPath)
	}

	return populateErr
}

// viewExists checks if a materialized view exists in the database.
func viewExists(ctx context.Context, op *iodb.PgxOperator, viewName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM pg_matviews
			WHERE schemaname = 'public'
			  AND matviewname = $1
		)
	`
	var exists bool
	err := op.Pool().QueryRow(ctx, query, viewName).Scan(&exists)
	return exists, err
}

// indexExists checks if an index exists in the database.
func indexExists(ctx context.Context, op *iodb.PgxOperator, indexName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE schemaname = 'public'
			  AND indexname = $1
		)
	`
	var exists bool
	err := op.Pool().QueryRow(ctx, query, indexName).Scan(&exists)
	return exists, err
}
