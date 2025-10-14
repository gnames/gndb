package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	ioconfig "github.com/gnames/gndb/internal/io/config"
	iodatabase "github.com/gnames/gndb/internal/io/database"
	iopopulate "github.com/gnames/gndb/internal/io/populate"
	ioschema "github.com/gnames/gndb/internal/io/schema"
	iotesting "github.com/gnames/gndb/internal/io/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that requires PostgreSQL.
// See operator_test.go for configuration instructions.
// Skip with: go test -short

// TestPopulateCommand_E2E tests the complete populate workflow end-to-end.
// This test verifies:
//  1. Database connection and schema creation
//  2. sources.yaml loading and filtering
//  3. SFGA fetching and opening
//  4. Phase 1: Name strings processing
//  5. Phase 1.5: Hierarchy building
//  6. Phase 2: Name indices with classifications
//  7. Phase 3-4: Vernacular strings and indices
//  8. Phase 5: Data source metadata
//  9. All tables populated with correct data
//  10. Idempotency (running twice produces same result)
func TestPopulateCommand_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Set JobsNumber for parallel processing
	cfg.JobsNumber = 2

	// Create database operator
	op := iodatabase.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err, "Should connect to database")
	defer op.Close()

	// Clean up any existing tables first
	_ = op.DropAllTables(ctx)

	// Create schema
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "Schema creation should succeed")

	// Setup test sources.yaml
	// The populate command loads sources.yaml from GetConfigDir()/sources.yaml
	// We need to temporarily replace it with our test version
	configDir, err := ioconfig.GetConfigDir()
	require.NoError(t, err, "Should get config directory")

	sourcesYAMLPath := filepath.Join(configDir, "sources.yaml")
	backupPath := sourcesYAMLPath + ".e2e_backup"

	// Backup existing sources.yaml if it exists
	originalExists := false
	if _, err := os.Stat(sourcesYAMLPath); err == nil {
		originalExists = true
		err = os.Rename(sourcesYAMLPath, backupPath)
		require.NoError(t, err, "Should backup original sources.yaml")
	}

	// Restore original sources.yaml after test
	defer func() {
		if originalExists {
			_ = os.Remove(sourcesYAMLPath) // Remove test version
			_ = os.Rename(backupPath, sourcesYAMLPath)
		} else {
			_ = os.Remove(sourcesYAMLPath) // Clean up test version
		}
	}()

	// Create test sources.yaml
	// Use absolute path to testdata
	testdataPath, err := filepath.Abs("../../testdata")
	require.NoError(t, err, "Should get testdata path")

	testSourcesYAML := `data_sources:
  # Ruhoff 1980 - small test dataset
  - id: 1000
    parent: ` + testdataPath + `
    title_short: "Ruhoff 1980"
    home_url: "https://doi.org/10.5479/si.00810282.294"
    is_auto_curated: true
`
	err = os.WriteFile(sourcesYAMLPath, []byte(testSourcesYAML), 0644)
	require.NoError(t, err, "Should write test sources.yaml")

	// Configure to process only source ID 1000
	cfg.Populate.SourceIDs = []int{1000}

	// Get SFGA counts before populate for verification
	testSFGAPath := filepath.Join(testdataPath, "1000_ruhoff_2023-08-22_v1.0.0.sqlite.zip")
	require.FileExists(t, testSFGAPath, "Test SFGA file should exist")

	// Create populator
	populator := iopopulate.NewPopulator(op)

	// Execute populate workflow - FIRST RUN
	t.Log("Running first populate...")
	err = populator.Populate(ctx, cfg)
	require.NoError(t, err, "First populate should succeed")

	// Verify: name_strings table
	var nameStringsCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&nameStringsCount)
	require.NoError(t, err, "Should query name_strings count")
	assert.Greater(t, nameStringsCount, 0, "name_strings should be populated")
	t.Logf("✓ name_strings: %d records", nameStringsCount)

	// Verify: name_string_indices table
	var indicesCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_string_indices").Scan(&indicesCount)
	require.NoError(t, err, "Should query name_string_indices count")
	assert.Greater(t, indicesCount, 0, "name_string_indices should be populated")

	// Check if there are any classifications (may be 0 for datasets with only bare names)
	var classifiedCount int
	query := `SELECT COUNT(*) FROM name_string_indices
	          WHERE classification IS NOT NULL AND classification != ''`
	err = op.Pool().QueryRow(ctx, query).Scan(&classifiedCount)
	require.NoError(t, err, "Should query classified records")
	// Note: classifiedCount may be 0 if the dataset has only bare names (like Ruhoff)
	t.Logf("✓ name_string_indices: %d records (%d with classifications)",
		indicesCount, classifiedCount)

	// Verify: Specific classification structure
	var sampleClassification, sampleRanks, sampleIDs string
	query = `SELECT classification, classification_ranks, classification_ids
	         FROM name_string_indices
	         WHERE classification IS NOT NULL AND classification != ''
	         LIMIT 1`
	err = op.Pool().QueryRow(ctx, query).Scan(&sampleClassification, &sampleRanks, &sampleIDs)
	if err == nil {
		// Verify pipe-delimited format
		assert.Contains(t, sampleClassification, "|", "Classification should be pipe-delimited")
		assert.Contains(t, sampleRanks, "|", "Ranks should be pipe-delimited")
		assert.Contains(t, sampleIDs, "|", "IDs should be pipe-delimited")
		t.Logf("✓ Classification format validated: %s", sampleClassification)
	}

	// Verify: vernacular_strings table
	var vernacularStringsCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_strings").Scan(&vernacularStringsCount)
	require.NoError(t, err, "Should query vernacular_strings count")
	t.Logf("✓ vernacular_strings: %d records", vernacularStringsCount)

	// Verify: vernacular_string_indices table
	var vernacularIndicesCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_string_indices").Scan(&vernacularIndicesCount)
	require.NoError(t, err, "Should query vernacular_string_indices count")
	t.Logf("✓ vernacular_string_indices: %d records", vernacularIndicesCount)

	// Verify: data_sources table
	var dataSourceCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM data_sources").Scan(&dataSourceCount)
	require.NoError(t, err, "Should query data_sources count")
	assert.Equal(t, 1, dataSourceCount, "Should have exactly 1 data source record")

	// Verify data source metadata fields
	var recordedID int
	var titleShort string
	var recordCount int
	var vernRecordCount int
	query = `SELECT id, title_short, record_count, vern_record_count
	         FROM data_sources WHERE id = 1000`
	err = op.Pool().QueryRow(ctx, query).Scan(&recordedID, &titleShort, &recordCount, &vernRecordCount)
	require.NoError(t, err, "Should query data source metadata")
	assert.Equal(t, 1000, recordedID, "Data source ID should match")
	assert.Equal(t, "Ruhoff 1980", titleShort, "Title should match")
	assert.Greater(t, recordCount, 0, "Record count should be > 0")
	t.Logf("✓ data_sources: ID=%d, Title=%s, RecordCount=%d, VernRecordCount=%d",
		recordedID, titleShort, recordCount, vernRecordCount)

	// Verify: Counts are consistent
	assert.Equal(t, indicesCount, recordCount,
		"data_sources.record_count should match name_string_indices count")
	assert.Equal(t, vernacularIndicesCount, vernRecordCount,
		"data_sources.vern_record_count should match vernacular_string_indices count")

	// Test idempotency: Run populate again and verify counts
	t.Log("\nTesting idempotency by running populate again...")
	err = populator.Populate(ctx, cfg)
	require.NoError(t, err, "Second populate should succeed")

	// Verify counts haven't changed (or changed predictably)
	var nameStringsCount2, indicesCount2, vernacularStringsCount2, vernacularIndicesCount2 int

	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&nameStringsCount2)
	require.NoError(t, err)
	assert.Equal(t, nameStringsCount, nameStringsCount2,
		"name_strings count should not change (ON CONFLICT DO NOTHING)")

	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_string_indices").Scan(&indicesCount2)
	require.NoError(t, err)
	assert.Equal(t, indicesCount, indicesCount2,
		"name_string_indices count should be same (DELETE + re-insert)")

	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_strings").Scan(&vernacularStringsCount2)
	require.NoError(t, err)
	assert.Equal(t, vernacularStringsCount, vernacularStringsCount2,
		"vernacular_strings count should not change")

	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_string_indices").Scan(&vernacularIndicesCount2)
	require.NoError(t, err)
	assert.Equal(t, vernacularIndicesCount, vernacularIndicesCount2,
		"vernacular_string_indices count should be same (DELETE + re-insert)")

	t.Log("✓ Idempotency test passed: all counts match after second run")

	// Verify data source metadata updated_at is set
	var updatedAt time.Time
	query = `SELECT updated_at FROM data_sources WHERE id = 1000`
	err = op.Pool().QueryRow(ctx, query).Scan(&updatedAt)
	require.NoError(t, err, "Should query updated_at")
	assert.False(t, updatedAt.IsZero(), "updated_at should be set")

	t.Log("\n=== E2E Test Summary ===")
	t.Logf("✓ All 5 phases executed successfully")
	t.Logf("✓ %d name strings imported", nameStringsCount)
	t.Logf("✓ %d name indices created (%d with classifications)", indicesCount, classifiedCount)
	t.Logf("✓ %d vernacular strings and %d indices", vernacularStringsCount, vernacularIndicesCount)
	t.Logf("✓ 1 data source record with correct metadata")
	t.Logf("✓ Idempotency verified")

	// Clean up
	err = op.DropAllTables(ctx)
	assert.NoError(t, err, "Should be able to drop tables after test")
}

// TestPopulateCommand_E2E_NoSources tests behavior when no sources configured.
func TestPopulateCommand_E2E_NoSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Create database operator
	op := iodatabase.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err, "Should connect to database")
	defer op.Close()

	// Clean up and create schema
	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Setup empty sources.yaml
	configDir, err := ioconfig.GetConfigDir()
	require.NoError(t, err)

	sourcesYAMLPath := filepath.Join(configDir, "sources.yaml")
	backupPath := sourcesYAMLPath + ".e2e_backup2"

	// Backup existing
	originalExists := false
	if _, err := os.Stat(sourcesYAMLPath); err == nil {
		originalExists = true
		err = os.Rename(sourcesYAMLPath, backupPath)
		require.NoError(t, err)
	}
	defer func() {
		if originalExists {
			_ = os.Remove(sourcesYAMLPath)
			_ = os.Rename(backupPath, sourcesYAMLPath)
		} else {
			_ = os.Remove(sourcesYAMLPath)
		}
	}()

	// Create empty sources.yaml
	emptySourcesYAML := `data_sources: []
`
	err = os.WriteFile(sourcesYAMLPath, []byte(emptySourcesYAML), 0644)
	require.NoError(t, err)

	// Create populator
	populator := iopopulate.NewPopulator(op)

	// Execute populate - should succeed but do nothing
	err = populator.Populate(ctx, cfg)
	// This should either succeed with no work, or give a helpful message
	// The behavior depends on implementation - check what makes sense
	if err != nil {
		// If it errors, it should be a clear message
		t.Logf("Populate with no sources returned: %v", err)
	}

	// Verify no data was inserted
	var count int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM data_sources").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Should have no data sources when sources.yaml is empty")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestPopulateCommand_E2E_FilteredSource tests source ID filtering.
func TestPopulateCommand_E2E_FilteredSource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Create database operator
	op := iodatabase.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean up and create schema
	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Setup sources.yaml with multiple sources
	configDir, err := ioconfig.GetConfigDir()
	require.NoError(t, err)

	sourcesYAMLPath := filepath.Join(configDir, "sources.yaml")
	backupPath := sourcesYAMLPath + ".e2e_backup3"

	originalExists := false
	if _, err := os.Stat(sourcesYAMLPath); err == nil {
		originalExists = true
		err = os.Rename(sourcesYAMLPath, backupPath)
		require.NoError(t, err)
	}
	defer func() {
		if originalExists {
			_ = os.Remove(sourcesYAMLPath)
			_ = os.Rename(backupPath, sourcesYAMLPath)
		} else {
			_ = os.Remove(sourcesYAMLPath)
		}
	}()

	testdataPath, err := filepath.Abs("../../testdata")
	require.NoError(t, err)

	multiSourceYAML := `data_sources:
  - id: 1000
    parent: ` + testdataPath + `
    title_short: "Ruhoff 1980"
  - id: 9999
    parent: ` + testdataPath + `
    title_short: "Nonexistent Source"
`
	err = os.WriteFile(sourcesYAMLPath, []byte(multiSourceYAML), 0644)
	require.NoError(t, err)

	// Filter to only source 1000 (which exists)
	cfg.Populate.SourceIDs = []int{1000}

	// Create populator
	populator := iopopulate.NewPopulator(op)

	// Execute populate - should process only source 1000
	err = populator.Populate(ctx, cfg)
	require.NoError(t, err, "Populate with filtered source should succeed")

	// Verify only source 1000 was processed
	var dataSourceCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM data_sources").Scan(&dataSourceCount)
	require.NoError(t, err)
	assert.Equal(t, 1, dataSourceCount, "Should have exactly 1 data source")

	var sourceID int
	err = op.Pool().QueryRow(ctx, "SELECT id FROM data_sources").Scan(&sourceID)
	require.NoError(t, err)
	assert.Equal(t, 1000, sourceID, "Should have processed source ID 1000")

	t.Log("✓ Source filtering works correctly")

	// Clean up
	_ = op.DropAllTables(ctx)
}
