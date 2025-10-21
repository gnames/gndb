package iooptimize

import (
	"context"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/gnames/gnuuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateVerificationView_Integration tests that the verification materialized view
// is created correctly with proper structure, indexes, and data filtering.
//
// Test scenario:
// 1. Given: Database with name_strings and name_string_indices
// 2. When: Call createVerificationView()
// 3. Then:
//   - Existing verification view dropped (if exists)
//   - New verification materialized view created
//   - View contains expected columns
//   - 3 indexes created: canonical_id, name_string_id, year
//   - Query verification view returns expected records
//   - Filters applied: canonical_id NOT NULL, surrogate=false, bacteria filter, or virus=true
//
// Reference: gnidump createVerification() in db_views.go
func TestCreateVerificationView_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database operator
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err, "Failed to connect to test database")
	defer op.Close()

	// Clean slate: drop all tables and recreate schema
	err = op.DropAllTables(ctx)
	require.NoError(t, err)

	// Create schema
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "Schema creation should succeed")

	// Setup test data
	pool := op.Pool()

	// Create test name_strings
	homoID := gnuuid.New("Homo sapiens").String()
	plantaID := gnuuid.New("Plantae").String()
	surrogateID := gnuuid.New("12345").String() // Surrogate, should be filtered
	virusID := gnuuid.New("Tobacco mosaic virus").String()
	bacteriaID := gnuuid.New("Escherichia coli").String()

	homoCanonicalID := gnuuid.New("Homo sapiens").String()
	plantaCanonicalID := gnuuid.New("Plantae").String()
	virusCanonicalID := gnuuid.New("Tobacco mosaic virus").String()
	bacteriaCanonicalID := gnuuid.New("Escherichia coli").String()

	// Insert canonicals
	_, err = pool.Exec(ctx, `
		INSERT INTO canonicals (id, name) VALUES
		($1, 'Homo sapiens'),
		($2, 'Plantae'),
		($3, 'Tobacco mosaic virus'),
		($4, 'Escherichia coli')
	`, homoCanonicalID, plantaCanonicalID, virusCanonicalID, bacteriaCanonicalID)
	require.NoError(t, err)

	// Insert name_strings
	_, err = pool.Exec(ctx, `
		INSERT INTO name_strings (id, name, canonical_id, surrogate, virus, bacteria, parse_quality, cardinality, year) VALUES
		($1, 'Homo sapiens Linnaeus 1758', $2, false, false, false, 1, 2, 1758),
		($3, 'Plantae', $4, false, false, false, 1, 1, NULL),
		($5, '12345', NULL, true, false, false, 0, 0, NULL),
		($6, 'Tobacco mosaic virus', $7, false, true, false, 1, 0, NULL),
		($8, 'Escherichia coli (Migula 1895)', $9, false, false, true, 1, 2, 1895)
	`, homoID, homoCanonicalID, plantaID, plantaCanonicalID, surrogateID, virusID, virusCanonicalID, bacteriaID, bacteriaCanonicalID)
	require.NoError(t, err)

	// Insert name_string_indices
	_, err = pool.Exec(ctx, `
		INSERT INTO name_string_indices (data_source_id, record_id, name_string_id, local_id, outlink_id, code_id, rank, taxonomic_status, accepted_record_id, classification, classification_ranks, classification_ids) VALUES
		(1, 'homo-1', $1, 'local-1', 'outlink-1', 1, 'species', 'accepted', NULL, 'Animalia|Chordata|Mammalia|Primates|Hominidae|Homo', 'kingdom|phylum|class|order|family|genus', 'id1|id2|id3|id4|id5|id6'),
		(1, 'plantae-1', $2, 'local-2', 'outlink-2', 2, 'kingdom', 'accepted', NULL, '', '', ''),
		(1, 'surrogate-1', $3, 'local-3', 'outlink-3', 0, '', '', NULL, '', '', ''),
		(1, 'virus-1', $4, 'local-4', 'outlink-4', 4, '', 'accepted', NULL, '', '', ''),
		(1, 'bacteria-1', $5, 'local-5', 'outlink-5', 3, 'species', 'accepted', NULL, 'Bacteria|Proteobacteria|Gammaproteobacteria|Enterobacterales|Enterobacteriaceae|Escherichia', 'domain|phylum|class|order|family|genus', 'bid1|bid2|bid3|bid4|bid5|bid6')
	`, homoID, plantaID, surrogateID, virusID, bacteriaID)
	require.NoError(t, err)

	// Create optimizer
	optimizer := &OptimizerImpl{
		operator: op,
	}

	// ACTION: Create verification view
	err = createVerificationView(ctx, optimizer, cfg)
	require.NoError(t, err, "createVerificationView should succeed")

	// VERIFY 1: View exists
	var viewExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_matviews WHERE matviewname = 'verification'
		)
	`).Scan(&viewExists)
	require.NoError(t, err)
	assert.True(t, viewExists, "Verification view should exist")

	// VERIFY 2: View has expected columns
	// Note: information_schema.columns doesn't work for materialized views,
	// so we use pg_attribute instead
	var columnCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_attribute
		WHERE attrelid = 'verification'::regclass AND attnum > 0 AND NOT attisdropped
	`).Scan(&columnCount)
	require.NoError(t, err)
	assert.Equal(t, 21, columnCount, "Verification view should have 21 columns")

	// VERIFY 3: Check specific columns exist
	// Note: For materialized views, we need to use pg_attribute
	expectedColumns := []string{
		"data_source_id", "record_id", "name_string_id", "name",
		"name_id", "code_id", "year", "cardinality", "canonical_id",
		"virus", "bacteria", "parse_quality", "local_id", "outlink_id",
		"taxonomic_status", "accepted_record_id", "accepted_name_id",
		"accepted_name", "classification", "classification_ranks", "classification_ids",
	}
	for _, col := range expectedColumns {
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_attribute
				WHERE attrelid = 'verification'::regclass
				AND attname = $1
				AND attnum > 0
				AND NOT attisdropped
			)
		`, col).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Column %s should exist", col)
	}

	// VERIFY 4: Check indexes exist
	// Index on canonical_id
	var canonicalIdxExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE tablename = 'verification' AND indexname LIKE '%canonical_id%'
		)
	`).Scan(&canonicalIdxExists)
	require.NoError(t, err)
	assert.True(t, canonicalIdxExists, "Index on canonical_id should exist")

	// Index on name_string_id
	var nameStringIdxExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE tablename = 'verification' AND indexname LIKE '%name_string_id%'
		)
	`).Scan(&nameStringIdxExists)
	require.NoError(t, err)
	assert.True(t, nameStringIdxExists, "Index on name_string_id should exist")

	// Index on year
	var yearIdxExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE tablename = 'verification' AND indexname LIKE '%year%'
		)
	`).Scan(&yearIdxExists)
	require.NoError(t, err)
	assert.True(t, yearIdxExists, "Index on year should exist")

	// VERIFY 5: Check data filtering - should include Homo sapiens (has canonical_id, not surrogate)
	var homoExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM verification WHERE name_string_id = $1
		)
	`, homoID).Scan(&homoExists)
	require.NoError(t, err)
	assert.True(t, homoExists, "Homo sapiens should be in verification view")

	// VERIFY 6: Check data filtering - should include Plantae (has canonical_id, not surrogate)
	var plantaExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM verification WHERE name_string_id = $1
		)
	`, plantaID).Scan(&plantaExists)
	require.NoError(t, err)
	assert.True(t, plantaExists, "Plantae should be in verification view")

	// VERIFY 7: Check data filtering - should NOT include surrogate (canonical_id is NULL)
	var surrogateExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM verification WHERE name_string_id = $1
		)
	`, surrogateID).Scan(&surrogateExists)
	require.NoError(t, err)
	assert.False(t, surrogateExists, "Surrogate name should NOT be in verification view")

	// VERIFY 8: Check data filtering - should include virus (virus=true exception)
	var virusExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM verification WHERE name_string_id = $1
		)
	`, virusID).Scan(&virusExists)
	require.NoError(t, err)
	assert.True(t, virusExists, "Virus name should be in verification view (virus exception)")

	// VERIFY 9: Check data filtering - should include bacteria (bacteria=true but parse_quality=1 < 3)
	var bacteriaExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM verification WHERE name_string_id = $1
		)
	`, bacteriaID).Scan(&bacteriaExists)
	require.NoError(t, err)
	assert.True(t, bacteriaExists, "Bacteria with good parse quality should be in verification view")

	// VERIFY 10: Verify total count matches expectations (4 records: Homo, Plantae, Virus, Bacteria)
	var totalCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM verification").Scan(&totalCount)
	require.NoError(t, err)
	assert.Equal(t, 4, totalCount, "Verification view should contain exactly 4 records")

	// VERIFY 11: Check specific data fields are populated correctly
	var name string
	var year *int16
	var canonicalID string
	err = pool.QueryRow(ctx, `
		SELECT name, year, canonical_id FROM verification WHERE name_string_id = $1
	`, homoID).Scan(&name, &year, &canonicalID)
	require.NoError(t, err)
	assert.Equal(t, "Homo sapiens Linnaeus 1758", name)
	assert.NotNil(t, year)
	assert.Equal(t, int16(1758), *year)
	assert.Equal(t, homoCanonicalID, canonicalID)

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestCreateVerificationView_Idempotent tests that creating the view multiple times
// is safe and produces the same result.
func TestCreateVerificationView_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	err = op.DropAllTables(ctx)
	require.NoError(t, err)

	// Create schema
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Setup minimal test data
	nameID := gnuuid.New("Test name").String()
	canonicalID := gnuuid.New("Test name").String()

	_, err = pool.Exec(ctx, "INSERT INTO canonicals (id, name) VALUES ($1, 'Test name')", canonicalID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO name_strings (id, name, canonical_id, surrogate, virus, bacteria, parse_quality, cardinality)
		VALUES ($1, 'Test name Author', $2, false, false, false, 1, 2)
	`, nameID, canonicalID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO name_string_indices (data_source_id, record_id, name_string_id, local_id, outlink_id, code_id, rank, taxonomic_status, accepted_record_id, classification, classification_ranks, classification_ids)
		VALUES (1, 'test-1', $1, 'local-1', 'outlink-1', 1, 'species', 'accepted', NULL, '', '', '')
	`, nameID)
	require.NoError(t, err)

	optimizer := &OptimizerImpl{
		operator: op,
	}

	// Create view first time
	err = createVerificationView(ctx, optimizer, cfg)
	require.NoError(t, err)

	var countFirst int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM verification").Scan(&countFirst)
	require.NoError(t, err)

	// Create view second time (should drop and recreate)
	err = createVerificationView(ctx, optimizer, cfg)
	require.NoError(t, err)

	var countSecond int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM verification").Scan(&countSecond)
	require.NoError(t, err)

	assert.Equal(t, countFirst, countSecond, "Record count should be same after recreating view")
	assert.Equal(t, 1, countSecond, "Should have exactly 1 record")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestCreateVerificationView_EmptyDatabase tests graceful handling when database is empty.
func TestCreateVerificationView_EmptyDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	err = op.DropAllTables(ctx)
	require.NoError(t, err)

	// Create schema
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	optimizer := &OptimizerImpl{
		operator: op,
	}

	// Create view with empty database
	err = createVerificationView(ctx, optimizer, cfg)
	require.NoError(t, err, "Should handle empty database gracefully")

	pool := op.Pool()
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM verification").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Empty database should result in 0 records in view")

	// Clean up
	_ = op.DropAllTables(ctx)
}
