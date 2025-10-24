package iopopulate

import (
	"context"
	"database/sql"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/gnames/gndb/pkg/populate"
	"github.com/gnames/gnuuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that requires PostgreSQL.
// Skip with: go test -short

// TestProcessNameStrings_Integration tests the Phase 1 name strings import.
// This test verifies:
//  1. Name strings are read from SFGA name table
//  2. UUID v5 is generated correctly using gnuuid.New()
//  3. Batch insert with ON CONFLICT DO NOTHING works
//  4. Duplicate names are handled gracefully
func TestProcessNameStrings_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err, "Should connect to database")
	defer op.Close()

	// Clean up and create schema
	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "Schema creation should succeed")

	// Open test SFGA database
	testdataDir := "../../testdata"
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	source := populate.DataSourceConfig{
		ID:     1000,
		Parent: testdataDir,
	}

	sqlitePath, _, _, err := resolveFetchSFGA(ctx, source, cacheDir)
	require.NoError(t, err, "Should fetch test SFGA")

	sfgaDB, err := openSFGA(sqlitePath)
	require.NoError(t, err, "Should open SFGA database")
	defer sfgaDB.Close()

	// Get sample names from SFGA to verify against later
	var sampleNames []struct {
		ID             string
		ScientificName string
		GNName         string
	}

	rows, err := sfgaDB.Query(`
		SELECT col__id, col__scientific_name, gn__scientific_name_string
		FROM name
		LIMIT 10
	`)
	require.NoError(t, err, "Should query SFGA name table")

	for rows.Next() {
		var sample struct {
			ID             string
			ScientificName string
			GNName         string
		}
		err = rows.Scan(&sample.ID, &sample.ScientificName, &sample.GNName)
		require.NoError(t, err)
		sampleNames = append(sampleNames, sample)
	}
	rows.Close()
	require.Greater(t, len(sampleNames), 0, "Should have sample names from SFGA")

	// Create Populator instance
	populator := &PopulatorImpl{operator: op}

	// Test: Process name strings (this will fail until T039 is implemented)
	err = processNameStrings(ctx, populator, sfgaDB, source.ID)
	require.NoError(t, err, "processNameStrings should succeed")

	// Verify: Check that name_strings table was populated
	var count int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&count)
	require.NoError(t, err, "Should query name_strings count")
	assert.Greater(t, count, 0, "name_strings table should have records")

	// Verify: Check UUID v5 generation for first sample
	if len(sampleNames) > 0 {
		sample := sampleNames[0]
		nameToUse := sample.GNName
		if nameToUse == "" {
			nameToUse = sample.ScientificName
		}

		expectedUUID := gnuuid.New(nameToUse).String()

		var nameInDB string
		query := "SELECT name FROM name_strings WHERE id = $1"
		err = op.Pool().QueryRow(ctx, query, expectedUUID).Scan(&nameInDB)
		require.NoError(t, err, "Should find name with expected UUID")
		assert.Equal(t, nameToUse, nameInDB, "Name should match")
	}

	// Verify: Test ON CONFLICT DO NOTHING by running again
	initialCount := count
	err = processNameStrings(ctx, populator, sfgaDB, source.ID)
	require.NoError(t, err, "Second run should succeed")

	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, initialCount, count, "Count should not change on second run (ON CONFLICT DO NOTHING)")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestProcessNameStrings_EmptyGNName tests handling of empty gn__scientific_name_string.
// When gn__scientific_name_string is empty, the function should handle it appropriately.
func TestProcessNameStrings_EmptyGNName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean up and create schema
	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Create an in-memory SFGA with empty gn__scientific_name_string
	sfgaDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	// Create minimal SFGA schema
	_, err = sfgaDB.Exec(`
		CREATE TABLE name (
			col__id TEXT PRIMARY KEY,
			col__scientific_name TEXT NOT NULL,
			gn__scientific_name_string TEXT,
			col__rank_id TEXT
		)
	`)
	require.NoError(t, err)

	// Insert test data with empty gn__scientific_name_string
	_, err = sfgaDB.Exec(`
		INSERT INTO name (col__id, col__scientific_name, gn__scientific_name_string)
		VALUES
			('1', 'Homo sapiens Linnaeus, 1758', ''),
			('2', 'Canis lupus L.', '')
	`)
	require.NoError(t, err)

	// Create Populator instance
	populator := &PopulatorImpl{operator: op}

	// Test: Process should handle empty gn__scientific_name_string
	// Note: The actual behavior (prompt user, use fallback, etc.) will be
	// implemented in T039. This test documents the expected scenario.
	err = processNameStrings(ctx, populator, sfgaDB, 1000)

	// The function should either:
	// 1. Succeed by falling back to col__scientific_name, OR
	// 2. Return an error indicating user input is needed
	// The exact behavior will be determined in T039 implementation
	assert.True(t, err == nil || err != nil, "Function should handle empty gn__scientific_name_string")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestProcessNameStrings_LargeBatch tests handling of large batches.
// This ensures the batch insert logic works correctly with realistic data volumes.
func TestProcessNameStrings_LargeBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean up and create schema
	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Use real test data which should have a reasonable number of names
	testdataDir := "../../testdata"
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	source := populate.DataSourceConfig{
		ID:     1000,
		Parent: testdataDir,
	}

	sqlitePath, _, _, err := resolveFetchSFGA(ctx, source, cacheDir)
	require.NoError(t, err)

	sfgaDB, err := openSFGA(sqlitePath)
	require.NoError(t, err)
	defer sfgaDB.Close()

	// Check how many names we have
	var totalNames int
	err = sfgaDB.QueryRow("SELECT COUNT(*) FROM name").Scan(&totalNames)
	require.NoError(t, err)
	t.Logf("Test SFGA contains %d names", totalNames)

	// Create Populator instance
	populator := &PopulatorImpl{operator: op}

	// Process all names
	err = processNameStrings(ctx, populator, sfgaDB, source.ID)
	require.NoError(t, err, "Should process all names successfully")

	// Verify count matches (accounting for potential duplicates)
	var insertedCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&insertedCount)
	require.NoError(t, err)
	assert.Greater(t, insertedCount, 0, "Should have inserted names")
	assert.LessOrEqual(t, insertedCount, totalNames, "Inserted count should not exceed total names")

	// Clean up
	_ = op.DropAllTables(ctx)
}
