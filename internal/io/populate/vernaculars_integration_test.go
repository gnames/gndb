package populate

import (
	"context"
	"database/sql"
	"testing"

	iodatabase "github.com/gnames/gndb/internal/io/database"
	ioschema "github.com/gnames/gndb/internal/io/schema"
	iotesting "github.com/gnames/gndb/internal/io/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that requires PostgreSQL.
// Skip with: go test -short

// TestProcessVernaculars_Integration tests the complete vernacular import
// with both vernacular strings and indices.
//
// This test verifies:
//  1. Vernacular strings are inserted uniquely (ON CONFLICT DO NOTHING)
//  2. Vernacular indices link to vernacular strings via UUID
//  3. Vernacular indices include language, locality, country metadata
//  4. Old data is cleaned before import (DELETE WHERE data_source_id)
//  5. Counts are correct
func TestProcessVernaculars_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := iodatabase.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err, "Should connect to database")
	defer op.Close()

	// Clean up and create schema
	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "Schema creation should succeed")

	// Create test SFGA with vernacular data
	sfgaDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE vernacular (
			col__taxon_id TEXT,
			col__name TEXT,
			col__language TEXT,
			col__area TEXT,
			col__country TEXT
		);

		INSERT INTO vernacular (col__taxon_id, col__name, col__language, col__area, col__country) VALUES
			('taxon-1', 'Common plantain', 'English', 'North America', 'USA'),
			('taxon-1', 'Common plantain', 'English', 'Europe', 'UK'),
			('taxon-1', 'Llantén común', 'Spanish', 'Latin America', 'MX'),
			('taxon-2', 'Human', 'English', '', ''),
			('taxon-2', 'Ser humano', 'Spanish', '', ''),
			('taxon-3', 'E. coli', 'English', '', '');
	`)
	require.NoError(t, err, "Should create test SFGA")

	// Create Populator instance
	populator := &PopulatorImpl{operator: op}

	// Test: Process vernaculars (this will fail until T045 is implemented)
	sourceID := 9999
	err = processVernaculars(ctx, populator, sfgaDB, sourceID)
	require.NoError(t, err, "processVernaculars should succeed")

	// Verify: Check vernacular_strings table
	var vernStringCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_strings").Scan(&vernStringCount)
	require.NoError(t, err, "Should query vernacular_strings count")

	// We have 5 unique vernacular names (Common plantain appears twice but should be deduplicated)
	assert.Equal(t, 5, vernStringCount, "Should have 5 unique vernacular strings")

	// Verify: Check vernacular_string_indices table
	var vernIndexCount int
	err = op.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM vernacular_string_indices
		WHERE data_source_id = $1
	`, sourceID).Scan(&vernIndexCount)
	require.NoError(t, err, "Should query vernacular_string_indices count")

	// We have 6 vernacular records (including the duplicate "Common plantain")
	assert.Equal(t, 6, vernIndexCount, "Should have 6 vernacular indices")

	// Verify: Check that vernacular strings link to indices
	var linkedCount int
	err = op.Pool().QueryRow(ctx, `
		SELECT COUNT(*)
		FROM vernacular_string_indices vsi
		JOIN vernacular_strings vs ON vs.id = vsi.vernacular_string_id
		WHERE vsi.data_source_id = $1
	`, sourceID).Scan(&linkedCount)
	require.NoError(t, err, "Should query linked vernaculars")
	assert.Equal(t, 6, linkedCount, "All vernacular indices should link to vernacular strings")

	// Verify: Check language metadata
	var englishCount int
	err = op.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM vernacular_string_indices
		WHERE data_source_id = $1 AND language = 'English'
	`, sourceID).Scan(&englishCount)
	require.NoError(t, err)
	assert.Equal(t, 4, englishCount, "Should have 4 English vernaculars (Common plantain x2, Human, E. coli)")

	var spanishCount int
	err = op.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM vernacular_string_indices
		WHERE data_source_id = $1 AND language = 'Spanish'
	`, sourceID).Scan(&spanishCount)
	require.NoError(t, err)
	assert.Equal(t, 2, spanishCount, "Should have 2 Spanish vernaculars")

	// Verify: Sample a specific record
	var sampleRecord struct {
		RecordID           string
		VernacularStringID string
		Language           string
		Locality           string
		CountryCode        string
	}

	err = op.Pool().QueryRow(ctx, `
		SELECT record_id, vernacular_string_id, language, locality, country_code
		FROM vernacular_string_indices
		WHERE data_source_id = $1 AND language = 'Spanish'
		LIMIT 1
	`, sourceID).Scan(
		&sampleRecord.RecordID,
		&sampleRecord.VernacularStringID,
		&sampleRecord.Language,
		&sampleRecord.Locality,
		&sampleRecord.CountryCode,
	)
	require.NoError(t, err, "Should find Spanish vernacular record")

	assert.NotEmpty(t, sampleRecord.RecordID, "RecordID should not be empty")
	assert.NotEmpty(t, sampleRecord.VernacularStringID, "VernacularStringID should be valid UUID")
	assert.Equal(t, "Spanish", sampleRecord.Language)

	t.Logf("Sample Spanish record: %s, locality: %s, country: %s",
		sampleRecord.RecordID, sampleRecord.Locality, sampleRecord.CountryCode)

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestProcessVernaculars_Idempotency tests that running processVernaculars twice
// produces the same result (old data is cleaned before import).
func TestProcessVernaculars_Idempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := iodatabase.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Create test SFGA
	sfgaDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE vernacular (
			col__taxon_id TEXT,
			col__name TEXT,
			col__language TEXT,
			col__area TEXT,
			col__country TEXT
		);

		INSERT INTO vernacular (col__taxon_id, col__name, col__language, col__area, col__country) VALUES
			('taxon-1', 'Common plantain', 'English', 'North America', 'USA'),
			('taxon-2', 'Human', 'English', '', '');
	`)
	require.NoError(t, err)

	populator := &PopulatorImpl{operator: op}
	sourceID := 9999

	// First import
	err = processVernaculars(ctx, populator, sfgaDB, sourceID)
	require.NoError(t, err)

	var firstStringCount, firstIndexCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_strings").Scan(&firstStringCount)
	require.NoError(t, err)
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_string_indices WHERE data_source_id = $1", sourceID).Scan(&firstIndexCount)
	require.NoError(t, err)

	require.Greater(t, firstStringCount, 0, "First import should insert vernacular strings")
	require.Greater(t, firstIndexCount, 0, "First import should insert vernacular indices")

	// Second import (should clean old data first)
	err = processVernaculars(ctx, populator, sfgaDB, sourceID)
	require.NoError(t, err)

	var secondStringCount, secondIndexCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_strings").Scan(&secondStringCount)
	require.NoError(t, err)
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_string_indices WHERE data_source_id = $1", sourceID).Scan(&secondIndexCount)
	require.NoError(t, err)

	// Counts should be identical (idempotent)
	assert.Equal(t, firstStringCount, secondStringCount, "Second import should have same vernacular string count (idempotent)")
	assert.Equal(t, firstIndexCount, secondIndexCount, "Second import should have same vernacular index count (idempotent)")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestProcessVernaculars_EmptyTable tests handling when SFGA has no vernaculars.
func TestProcessVernaculars_EmptyTable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := iodatabase.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Create empty SFGA
	sfgaDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE vernacular (
			col__taxon_id TEXT,
			col__name TEXT,
			col__language TEXT,
			col__area TEXT,
			col__country TEXT
		);
	`)
	require.NoError(t, err)

	populator := &PopulatorImpl{operator: op}
	sourceID := 9999

	// Process empty vernaculars (should not error)
	err = processVernaculars(ctx, populator, sfgaDB, sourceID)
	require.NoError(t, err, "Processing empty vernaculars should succeed")

	// Verify: No data inserted
	var stringCount, indexCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_strings").Scan(&stringCount)
	require.NoError(t, err)
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM vernacular_string_indices WHERE data_source_id = $1", sourceID).Scan(&indexCount)
	require.NoError(t, err)

	assert.Equal(t, 0, stringCount, "Should have no vernacular strings")
	assert.Equal(t, 0, indexCount, "Should have no vernacular indices")

	// Clean up
	_ = op.DropAllTables(ctx)
}
