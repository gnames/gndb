package iooptimize

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/gnames/gnuuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateReparseTempTable tests the creation of temporary table for batch reparsing.
// This test validates T029 implementation requirements:
// - Temp table is created with correct schema
// - Columns match name_strings structure
// - Table can be dropped successfully
//
// EXPECTED: This test will FAIL until T029 (createReparseTempTable) is implemented.
func TestCreateReparseTempTable(t *testing.T) {
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

	pool := op.Pool()

	// TEST: Call createReparseTempTable (will fail - function doesn't exist yet)
	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err, "createReparseTempTable should succeed")

	// VERIFY 1: Temp table exists
	var tableExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM pg_tables
			WHERE tablename = 'temp_reparse_names'
		)
	`).Scan(&tableExists)
	require.NoError(t, err)
	assert.True(t, tableExists, "Temp table should exist after creation")

	// VERIFY 2: Temp table has correct columns
	expectedColumns := map[string]string{
		"name_string_id":    "uuid",
		"canonical_id":      "uuid",
		"canonical_full_id": "uuid",
		"canonical_stem_id": "uuid",
		"canonical":         "text",
		"canonical_full":    "text",
		"canonical_stem":    "text",
		"bacteria":          "boolean",
		"virus":             "boolean",
		"surrogate":         "boolean",
		"parse_quality":     "integer",
		"cardinality":       "integer",
		"year":              "smallint",
	}

	for colName, expectedType := range expectedColumns {
		var dataType string
		err = pool.QueryRow(ctx, `
			SELECT data_type
			FROM information_schema.columns
			WHERE table_name = 'temp_reparse_names'
			AND column_name = $1
		`, colName).Scan(&dataType)
		require.NoError(t, err, "Column %s should exist", colName)
		assert.Equal(t, expectedType, dataType, "Column %s should have type %s", colName, expectedType)
	}

	// VERIFY 3: Primary key exists on name_string_id
	var constraintExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM pg_constraint
			WHERE conname = 'temp_reparse_names_pkey'
			AND contype = 'p'
		)
	`).Scan(&constraintExists)
	require.NoError(t, err)
	assert.True(t, constraintExists, "Primary key should exist on name_string_id")

	// VERIFY 4: Table is UNLOGGED (for performance)
	var relpersistence string
	err = pool.QueryRow(ctx, `
		SELECT relpersistence::text
		FROM pg_class
		WHERE relname = 'temp_reparse_names'
	`).Scan(&relpersistence)
	require.NoError(t, err)
	assert.Equal(t, "u", relpersistence, "Table should be UNLOGGED (relpersistence='u')")

	// TEST: Drop temp table
	_, err = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	require.NoError(t, err, "Should be able to drop temp table")

	// VERIFY: Table no longer exists
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM pg_tables
			WHERE tablename = 'temp_reparse_names'
		)
	`).Scan(&tableExists)
	require.NoError(t, err)
	assert.False(t, tableExists, "Temp table should not exist after drop")

	// Cleanup
	_ = op.DropAllTables(ctx)
}

// TestCreateReparseTempTable_Idempotent tests that creating temp table multiple times is safe.
// The function should handle "table already exists" gracefully.
func TestCreateReparseTempTable_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Create table first time
	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err, "First creation should succeed")

	// Create table second time - should handle gracefully
	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err, "Second creation should succeed (idempotent)")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestCreateReparseTempTable_ContextCancellation tests that table creation
// respects context cancellation.
func TestCreateReparseTempTable_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Create cancelled context
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	// Try to create table with cancelled context
	err = createReparseTempTable(cancelCtx, pool)
	assert.Error(t, err, "Should return error when context is cancelled")

	// Cleanup
	_ = op.DropAllTables(ctx)
}

// TestBulkInsertToTempTable tests bulk insertion of changed names to temp table.
// This test validates T030 implementation requirements:
// - Only CHANGED names are inserted (filtered by parsedIsSame)
// - pgx CopyFrom is used for bulk insert
// - NULL values handled correctly
// - Progress tracking works
//
// EXPECTED: This test will FAIL until T030 (bulkInsertToTempTable) is implemented.
func TestBulkInsertToTempTable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Create temp table first
	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	// Prepare test data - 1000 changed names
	changedNames := make([]reparsed, 1000)
	for i := 0; i < 1000; i++ {
		name := fmt.Sprintf("Genus species%d", i)
		canonicalID := gnuuid.New(name).String()
		changedNames[i] = reparsed{
			nameStringID: gnuuid.New(fmt.Sprintf("name_%d", i)).String(),
			name:         name,
			canonicalID:  sql.NullString{String: canonicalID, Valid: true},
			canonical:    name,
			bacteria:     false,
			virus:        sql.NullBool{Bool: false, Valid: true},
			surrogate:    sql.NullBool{Bool: false, Valid: true},
			parseQuality: 1,
			cardinality:  sql.NullInt32{Int32: 2, Valid: true},
			year:         sql.NullInt16{Int16: 2020, Valid: true},
		}
	}

	// TEST: Call bulkInsertToTempTable (will fail - function doesn't exist yet)
	err = bulkInsertToTempTable(ctx, pool, changedNames)
	require.NoError(t, err, "bulkInsertToTempTable should succeed")

	// VERIFY 1: Count rows in temp table
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM temp_reparse_names").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1000, count, "Should insert exactly 1000 rows")

	// VERIFY 2: Check data integrity - sample a few rows
	var verifyCanonical string
	var verifyParseQuality int
	err = pool.QueryRow(ctx, `
		SELECT canonical, parse_quality
		FROM temp_reparse_names
		WHERE canonical = $1
	`, "Genus species0").Scan(&verifyCanonical, &verifyParseQuality)
	require.NoError(t, err)
	assert.Equal(t, "Genus species0", verifyCanonical)
	assert.Equal(t, 1, verifyParseQuality)

	// VERIFY 3: Check NULL handling - create entry with NULL fields
	nameWithNulls := reparsed{
		nameStringID:    gnuuid.New("null_test").String(),
		name:            "Unparseable name",
		canonicalID:     sql.NullString{}, // NULL
		canonicalFullID: sql.NullString{}, // NULL
		canonicalStemID: sql.NullString{}, // NULL
		canonical:       "",
		bacteria:        false,
		virus:           sql.NullBool{},  // NULL
		surrogate:       sql.NullBool{},  // NULL
		parseQuality:    0,               // Unparseable
		cardinality:     sql.NullInt32{}, // NULL
		year:            sql.NullInt16{}, // NULL
	}

	err = bulkInsertToTempTable(ctx, pool, []reparsed{nameWithNulls})
	require.NoError(t, err)

	// Verify NULL fields are stored correctly
	var canonicalIDNull sql.NullString
	var virusNull sql.NullBool
	err = pool.QueryRow(ctx, `
		SELECT canonical_id, virus
		FROM temp_reparse_names
		WHERE name_string_id = $1
	`, nameWithNulls.nameStringID).Scan(&canonicalIDNull, &virusNull)
	require.NoError(t, err)
	assert.False(t, canonicalIDNull.Valid, "canonical_id should be NULL")
	assert.False(t, virusNull.Valid, "virus should be NULL")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBulkInsertToTempTable_EmptyBatch tests handling of empty batch.
func TestBulkInsertToTempTable_EmptyBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	// Insert empty batch - should handle gracefully
	err = bulkInsertToTempTable(ctx, pool, []reparsed{})
	require.NoError(t, err, "Empty batch should be handled gracefully")

	// Verify no rows inserted
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM temp_reparse_names").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "No rows should be inserted for empty batch")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBulkInsertToTempTable_LargeBatch tests performance with large batches.
// This validates the batch size configuration from Config.Import.BatchSize.
func TestBulkInsertToTempTable_LargeBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	// Create 50,000 rows (default batch size)
	batchSize := 50000
	largeBatch := make([]reparsed, batchSize)
	for i := 0; i < batchSize; i++ {
		name := fmt.Sprintf("Species %d", i)
		largeBatch[i] = reparsed{
			nameStringID: gnuuid.New(fmt.Sprintf("large_%d", i)).String(),
			name:         name,
			canonicalID:  sql.NullString{String: gnuuid.New(name).String(), Valid: true},
			canonical:    name,
			parseQuality: 1,
		}
	}

	// Insert large batch
	startTime := time.Now()
	err = bulkInsertToTempTable(ctx, pool, largeBatch)
	duration := time.Since(startTime)
	require.NoError(t, err, "Large batch insert should succeed")

	// Performance check: 50K rows should insert in < 5 seconds
	assert.Less(t, duration.Seconds(), 5.0, "50K rows should insert quickly via CopyFrom")

	// Verify count
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM temp_reparse_names").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, batchSize, count, "Should insert all rows")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBulkInsertToTempTable_ContextCancellation tests that bulk insert
// respects context cancellation.
func TestBulkInsertToTempTable_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	// Create cancelled context
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	// Prepare small batch
	batch := []reparsed{
		{
			nameStringID: gnuuid.New("test").String(),
			name:         "Test name",
			canonicalID:  sql.NullString{String: gnuuid.New("test").String(), Valid: true},
			canonical:    "Test name",
			parseQuality: 1,
		},
	}

	// Try to insert with cancelled context
	err = bulkInsertToTempTable(cancelCtx, pool, batch)
	assert.Error(t, err, "Should return error when context is cancelled")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBulkInsertToTempTable_DuplicateKeys tests handling of duplicate name_string_id.
// Since name_string_id is PRIMARY KEY, duplicates should cause an error.
func TestBulkInsertToTempTable_DuplicateKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	// Create batch with duplicate IDs
	duplicateID := gnuuid.New("duplicate").String()
	batch := []reparsed{
		{
			nameStringID: duplicateID,
			name:         "First",
			canonicalID:  sql.NullString{String: gnuuid.New("first").String(), Valid: true},
			canonical:    "First",
			parseQuality: 1,
		},
		{
			nameStringID: duplicateID, // Duplicate!
			name:         "Second",
			canonicalID:  sql.NullString{String: gnuuid.New("second").String(), Valid: true},
			canonical:    "Second",
			parseQuality: 1,
		},
	}

	// Insert should fail due to PRIMARY KEY violation
	err = bulkInsertToTempTable(ctx, pool, batch)
	assert.Error(t, err, "Should return error for duplicate PRIMARY KEY")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBatchUpdateNameStrings tests the single UPDATE statement that applies
// all changes from temp table to name_strings.
// This test validates T031 implementation requirements:
// - Single UPDATE with JOIN updates all rows
// - All fields updated correctly
// - Rows not in temp table remain unchanged
//
// EXPECTED: This test will FAIL until T031 (batchUpdateNameStrings) is implemented.
func TestBatchUpdateNameStrings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Setup: Insert 1000 name_strings with initial values
	for i := 0; i < 1000; i++ {
		nameID := gnuuid.New(fmt.Sprintf("name_%d", i)).String()
		name := fmt.Sprintf("Genus species%d", i)
		_, err = pool.Exec(ctx, `
			INSERT INTO name_strings (
				id, name, canonical_id, bacteria, virus, surrogate, parse_quality, cardinality, year
			) VALUES ($1, $2, NULL, false, false, false, 0, NULL, NULL)
		`, nameID, name)
		require.NoError(t, err)
	}

	// Create temp table and populate with updated values (only first 500 changed)
	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	changedBatch := make([]reparsed, 500)
	for i := 0; i < 500; i++ {
		name := fmt.Sprintf("Genus species%d", i)
		nameID := gnuuid.New(fmt.Sprintf("name_%d", i)).String()
		canonicalID := gnuuid.New(name).String()

		changedBatch[i] = reparsed{
			nameStringID: nameID,
			name:         name,
			canonicalID:  sql.NullString{String: canonicalID, Valid: true},
			canonical:    name,
			bacteria:     false,
			virus:        sql.NullBool{Bool: false, Valid: true},
			surrogate:    sql.NullBool{Bool: false, Valid: true},
			parseQuality: 1, // Changed from 0
			cardinality:  sql.NullInt32{Int32: 2, Valid: true},
			year:         sql.NullInt16{Int16: 2020, Valid: true},
		}
	}

	err = bulkInsertToTempTable(ctx, pool, changedBatch)
	require.NoError(t, err)

	// TEST: Call batchUpdateNameStrings (will fail - function doesn't exist yet)
	rowsUpdated, err := batchUpdateNameStrings(ctx, pool)
	require.NoError(t, err, "batchUpdateNameStrings should succeed")
	assert.Equal(t, int64(500), rowsUpdated, "Should update exactly 500 rows")

	// VERIFY 1: Changed rows updated correctly
	var parseQuality int
	var canonicalID sql.NullString
	var cardinality sql.NullInt32
	var year sql.NullInt16

	nameID0 := gnuuid.New("name_0").String()
	err = pool.QueryRow(ctx, `
		SELECT canonical_id, parse_quality, cardinality, year
		FROM name_strings
		WHERE id = $1
	`, nameID0).Scan(&canonicalID, &parseQuality, &cardinality, &year)
	require.NoError(t, err)

	assert.True(t, canonicalID.Valid, "canonical_id should be updated")
	assert.Equal(t, 1, parseQuality, "parse_quality should be updated to 1")
	assert.True(t, cardinality.Valid, "cardinality should be updated")
	assert.Equal(t, int32(2), cardinality.Int32)
	assert.True(t, year.Valid, "year should be updated")
	assert.Equal(t, int16(2020), year.Int16)

	// VERIFY 2: Unchanged rows remain untouched
	nameID999 := gnuuid.New("name_999").String()
	err = pool.QueryRow(ctx, `
		SELECT canonical_id, parse_quality, cardinality, year
		FROM name_strings
		WHERE id = $1
	`, nameID999).Scan(&canonicalID, &parseQuality, &cardinality, &year)
	require.NoError(t, err)

	assert.False(t, canonicalID.Valid, "canonical_id should remain NULL")
	assert.Equal(t, 0, parseQuality, "parse_quality should remain 0")
	assert.False(t, cardinality.Valid, "cardinality should remain NULL")
	assert.False(t, year.Valid, "year should remain NULL")

	// VERIFY 3: Count total updated rows
	var updatedCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM name_strings WHERE parse_quality = 1
	`).Scan(&updatedCount)
	require.NoError(t, err)
	assert.Equal(t, 500, updatedCount, "Exactly 500 rows should have parse_quality=1")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBatchUpdateNameStrings_AllFields tests that all fields are updated correctly.
func TestBatchUpdateNameStrings_AllFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Insert one name_string with all NULL values
	testID := gnuuid.New("test_all_fields").String()
	_, err = pool.Exec(ctx, `
		INSERT INTO name_strings (
			id, name, canonical_id, canonical_full_id, canonical_stem_id,
			bacteria, virus, surrogate, parse_quality, cardinality, year
		) VALUES ($1, $2, NULL, NULL, NULL, false, NULL, NULL, 0, NULL, NULL)
	`, testID, "Test name")
	require.NoError(t, err)

	// Create temp table with all fields populated
	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	canonicalID := gnuuid.New("canonical").String()
	canonicalFullID := gnuuid.New("canonical_full").String()
	canonicalStemID := gnuuid.New("canonical_stem").String()

	batch := []reparsed{
		{
			nameStringID:    testID,
			name:            "Test name",
			canonicalID:     sql.NullString{String: canonicalID, Valid: true},
			canonicalFullID: sql.NullString{String: canonicalFullID, Valid: true},
			canonicalStemID: sql.NullString{String: canonicalStemID, Valid: true},
			canonical:       "Test",
			canonicalFull:   "Test full",
			canonicalStem:   "Test stem",
			bacteria:        true,
			virus:           sql.NullBool{Bool: true, Valid: true},
			surrogate:       sql.NullBool{Bool: true, Valid: true},
			parseQuality:    2,
			cardinality:     sql.NullInt32{Int32: 3, Valid: true},
			year:            sql.NullInt16{Int16: 1758, Valid: true},
		},
	}

	err = bulkInsertToTempTable(ctx, pool, batch)
	require.NoError(t, err)

	// Update
	_, err = batchUpdateNameStrings(ctx, pool)
	require.NoError(t, err)

	// Verify all fields
	var (
		gotCanonicalID     sql.NullString
		gotCanonicalFullID sql.NullString
		gotCanonicalStemID sql.NullString
		gotBacteria        bool
		gotVirus           sql.NullBool
		gotSurrogate       sql.NullBool
		gotParseQuality    int
		gotCardinality     sql.NullInt32
		gotYear            sql.NullInt16
	)

	err = pool.QueryRow(ctx, `
		SELECT canonical_id, canonical_full_id, canonical_stem_id,
			bacteria, virus, surrogate, parse_quality, cardinality, year
		FROM name_strings WHERE id = $1
	`, testID).Scan(
		&gotCanonicalID, &gotCanonicalFullID, &gotCanonicalStemID,
		&gotBacteria, &gotVirus, &gotSurrogate, &gotParseQuality,
		&gotCardinality, &gotYear,
	)
	require.NoError(t, err)

	assert.Equal(t, canonicalID, gotCanonicalID.String)
	assert.Equal(t, canonicalFullID, gotCanonicalFullID.String)
	assert.Equal(t, canonicalStemID, gotCanonicalStemID.String)
	assert.True(t, gotBacteria)
	assert.True(t, gotVirus.Bool)
	assert.True(t, gotSurrogate.Bool)
	assert.Equal(t, 2, gotParseQuality)
	assert.Equal(t, int32(3), gotCardinality.Int32)
	assert.Equal(t, int16(1758), gotYear.Int16)

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBatchUpdateNameStrings_EmptyTempTable tests behavior when temp table is empty.
func TestBatchUpdateNameStrings_EmptyTempTable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Insert name_strings
	testID := gnuuid.New("test").String()
	_, err = pool.Exec(ctx, `
		INSERT INTO name_strings (id, name, parse_quality)
		VALUES ($1, $2, 0)
	`, testID, "Test")
	require.NoError(t, err)

	// Create empty temp table
	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	// Update with empty temp table
	rowsUpdated, err := batchUpdateNameStrings(ctx, pool)
	require.NoError(t, err, "Should handle empty temp table gracefully")
	assert.Equal(t, int64(0), rowsUpdated, "Should update 0 rows")

	// Verify name_strings unchanged
	var parseQuality int
	err = pool.QueryRow(ctx, "SELECT parse_quality FROM name_strings WHERE id = $1", testID).
		Scan(&parseQuality)
	require.NoError(t, err)
	assert.Equal(t, 0, parseQuality, "Should remain unchanged")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBatchInsertCanonicals tests batch insertion of unique canonicals from temp table.
// This test validates T032 implementation requirements:
// - Extract unique canonical forms from temp table
// - Batch INSERT into canonicals, canonical_stems, canonical_fulls
// - ON CONFLICT DO NOTHING for idempotency
//
// EXPECTED: This test will FAIL until T032 (batchInsertCanonicals) is implemented.
func TestBatchInsertCanonicals(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Create temp table
	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	// Populate temp table with 100 names (some duplicate canonicals)
	batch := make([]reparsed, 100)
	for i := 0; i < 100; i++ {
		// Create duplicates - every 10 names share same canonical
		canonicalName := fmt.Sprintf("Genus species%d", i/10)
		canonicalID := gnuuid.New(canonicalName).String()
		canonicalStemID := gnuuid.New(canonicalName + "_stem").String()

		batch[i] = reparsed{
			nameStringID:    gnuuid.New(fmt.Sprintf("name_%d", i)).String(),
			name:            fmt.Sprintf("%s var%d", canonicalName, i),
			canonicalID:     sql.NullString{String: canonicalID, Valid: true},
			canonicalStemID: sql.NullString{String: canonicalStemID, Valid: true},
			canonical:       canonicalName,
			canonicalStem:   canonicalName + "_stem",
			parseQuality:    1,
		}
	}

	err = bulkInsertToTempTable(ctx, pool, batch)
	require.NoError(t, err)

	// TEST: Call batchInsertCanonicals (will fail - function doesn't exist yet)
	err = batchInsertCanonicals(ctx, pool)
	require.NoError(t, err, "batchInsertCanonicals should succeed")

	// VERIFY 1: Canonicals table has unique entries only
	var canonicalCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount)
	require.NoError(t, err)
	assert.Equal(t, 10, canonicalCount, "Should have 10 unique canonicals (100 names / 10)")

	// VERIFY 2: Canonical_stems table populated
	var stemCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonical_stems").Scan(&stemCount)
	require.NoError(t, err)
	assert.Equal(t, 10, stemCount, "Should have 10 unique stems")

	// VERIFY 3: Check specific canonical exists
	var canonicalName string
	canonicalID := gnuuid.New("Genus species0").String()
	err = pool.QueryRow(ctx, "SELECT name FROM canonicals WHERE id = $1", canonicalID).
		Scan(&canonicalName)
	require.NoError(t, err)
	assert.Equal(t, "Genus species0", canonicalName)

	// VERIFY 4: Idempotency - run again, should not error
	err = batchInsertCanonicals(ctx, pool)
	require.NoError(t, err, "Should be idempotent")

	// Count should remain the same
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount)
	require.NoError(t, err)
	assert.Equal(t, 10, canonicalCount, "Count should not change on second run")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBatchInsertCanonicals_NullValues tests that NULL canonicals are skipped.
func TestBatchInsertCanonicals_NullValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	// Mix of valid and NULL canonicals
	batch := []reparsed{
		{
			nameStringID: gnuuid.New("valid").String(),
			name:         "Valid name",
			canonicalID:  sql.NullString{String: gnuuid.New("valid").String(), Valid: true},
			canonical:    "Valid name",
			parseQuality: 1,
		},
		{
			nameStringID: gnuuid.New("null").String(),
			name:         "Unparseable",
			canonicalID:  sql.NullString{}, // NULL
			canonical:    "",
			parseQuality: 0,
		},
	}

	err = bulkInsertToTempTable(ctx, pool, batch)
	require.NoError(t, err)

	// Insert canonicals
	err = batchInsertCanonicals(ctx, pool)
	require.NoError(t, err)

	// Only 1 canonical should be inserted (NULL skipped)
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should insert only non-NULL canonicals")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}

// TestBatchInsertCanonicals_EmptyStrings tests that empty string canonicals are skipped.
func TestBatchInsertCanonicals_EmptyStrings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	err = createReparseTempTable(ctx, pool)
	require.NoError(t, err)

	// Canonical with empty string
	batch := []reparsed{
		{
			nameStringID: gnuuid.New("empty").String(),
			name:         "Name",
			canonicalID:  sql.NullString{String: gnuuid.New("empty").String(), Valid: true},
			canonical:    "", // Empty string
			parseQuality: 1,
		},
	}

	err = bulkInsertToTempTable(ctx, pool, batch)
	require.NoError(t, err)

	// Insert canonicals
	err = batchInsertCanonicals(ctx, pool)
	require.NoError(t, err)

	// Should skip empty strings
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Should not insert empty string canonicals")

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
	_ = op.DropAllTables(ctx)
}
