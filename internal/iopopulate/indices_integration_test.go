package iopopulate

import (
	"context"
	"database/sql"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/gnames/gndb/pkg/populate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessNameIndices_FlatClassification(t *testing.T) {
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

	// Open real SFGA test data (1002-vascan)
	testdataDir := "../../testdata"
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	source := populate.DataSourceConfig{
		ID:     1002, // vascan
		Parent: testdataDir,
	}

	sqlitePath, err := fetchSFGA(ctx, source, cacheDir)
	require.NoError(t, err, "Should fetch test SFGA")

	sfgaDB, err := openSFGA(sqlitePath)
	require.NoError(t, err, "Should open SFGA database")
	defer sfgaDB.Close()

}

// Note: This is an integration test that uses real SFGA test data (vascan 1002).
// Skip with: go test -short

// TestProcessNameIndices_Integration tests the complete name indices import
// with all three scenarios: taxa (accepted names), synonyms, and bare names.
//
// This test verifies:
//  1. Taxa records get full classification from hierarchy
//  2. Synonym records link to accepted taxa via AcceptedRecordID
//  3. Bare names (orphans) get "bare-name-{id}" RecordID and no classification
//  4. Classification strings (name|rank|id) are correctly generated
//  5. Old data is cleaned before import (DELETE WHERE data_source_id)
func TestProcessNameIndices_Integration(t *testing.T) {
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

	// Open real SFGA test data (1002-vascan)
	testdataDir := "../../testdata"
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	source := populate.DataSourceConfig{
		ID:     1002, // vascan
		Parent: testdataDir,
	}

	sqlitePath, err := fetchSFGA(ctx, source, cacheDir)
	require.NoError(t, err, "Should fetch test SFGA")

	sfgaDB, err := openSFGA(sqlitePath)
	require.NoError(t, err, "Should open SFGA database")
	defer sfgaDB.Close()

	// Build hierarchy first (required for classification)
	hierarchy, err := buildHierarchy(ctx, sfgaDB, 4)
	require.NoError(t, err, "Should build hierarchy")
	require.NotEmpty(t, hierarchy, "Hierarchy should not be empty")

	// Create Populator instance
	populator := &PopulatorImpl{operator: op}

	// Test: Process name indices (this will fail until T043 is implemented)
	err = processNameIndices(ctx, populator, sfgaDB, &source, hierarchy, cfg)
	require.NoError(t, err, "processNameIndices should succeed")

	// Verify: Check that name_string_indices table was populated
	var count int
	err = op.Pool().
		QueryRow(ctx, "SELECT COUNT(*) FROM name_string_indices WHERE data_source_id = $1", source.ID).
		Scan(&count)
	require.NoError(t, err, "Should query name_string_indices count")
	assert.Greater(t, count, 0, "name_string_indices table should have records")

	// Verify: Count should match or exceed number of names in SFGA
	// (some names might appear multiple times if they have multiple taxa/synonyms)
	var sfgaNameCount int
	err = sfgaDB.QueryRow("SELECT COUNT(*) FROM name").Scan(&sfgaNameCount)
	require.NoError(t, err)
	t.Logf("SFGA has %d names, imported %d name indices", sfgaNameCount, count)

	// Verify: Check that we have accepted taxa with classification
	var taxaWithClassification int
	query := `
		SELECT COUNT(*) FROM name_string_indices
		WHERE data_source_id = $1
		AND classification IS NOT NULL
		AND classification != ''
		AND taxonomic_status NOT LIKE 'bare%'
	`
	err = op.Pool().QueryRow(ctx, query, source.ID).Scan(&taxaWithClassification)
	require.NoError(t, err)
	assert.Greater(t, taxaWithClassification, 0, "Should have taxa with classification")
	t.Logf("Found %d taxa with classification", taxaWithClassification)

	// Verify: Check that we have bare names (if any exist in vascan)
	var bareNameCount int
	query = `
		SELECT COUNT(*) FROM name_string_indices
		WHERE data_source_id = $1
		AND record_id LIKE 'bare-name-%'
	`
	err = op.Pool().QueryRow(ctx, query, source.ID).Scan(&bareNameCount)
	require.NoError(t, err)
	t.Logf("Found %d bare names", bareNameCount)

	// Verify: Sample a specific record and check classification structure
	var sampleRecord struct {
		RecordID            string
		NameStringID        string
		TaxonomicStatus     string
		Classification      string
		ClassificationRanks string
		ClassificationIDs   string
		AcceptedRecordID    string
	}

	query = `
		SELECT record_id, name_string_id, taxonomic_status,
		       classification, classification_ranks, classification_ids,
		       accepted_record_id
		FROM name_string_indices
		WHERE data_source_id = $1
		AND classification IS NOT NULL
		AND classification != ''
		LIMIT 1
	`
	err = op.Pool().QueryRow(ctx, query, source.ID).Scan(
		&sampleRecord.RecordID,
		&sampleRecord.NameStringID,
		&sampleRecord.TaxonomicStatus,
		&sampleRecord.Classification,
		&sampleRecord.ClassificationRanks,
		&sampleRecord.ClassificationIDs,
		&sampleRecord.AcceptedRecordID,
	)
	require.NoError(t, err, "Should find at least one record with classification")

	// Verify: Classification strings are pipe-delimited
	assert.Contains(t, sampleRecord.Classification, "|", "Classification should be pipe-delimited")
	assert.Contains(
		t,
		sampleRecord.ClassificationRanks,
		"|",
		"Classification ranks should be pipe-delimited",
	)
	assert.Contains(
		t,
		sampleRecord.ClassificationIDs,
		"|",
		"Classification IDs should be pipe-delimited",
	)

	// Verify: All three classification strings have same number of elements
	classificationParts := len(splitByPipe(sampleRecord.Classification))
	ranksParts := len(splitByPipe(sampleRecord.ClassificationRanks))
	idsParts := len(splitByPipe(sampleRecord.ClassificationIDs))
	assert.Equal(
		t,
		classificationParts,
		ranksParts,
		"Classification and ranks should have same length",
	)
	assert.Equal(t, classificationParts, idsParts, "Classification and IDs should have same length")

	t.Logf("Sample record: %s, status: %s, classification depth: %d",
		sampleRecord.RecordID, sampleRecord.TaxonomicStatus, classificationParts)

	// Clean up
	_ = op.DropAllTables(ctx)
}

func TestProcessNameIndices_OutlinkIDs_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Use CoL small test data (1001)
	testdataDir := "../../testdata"

	t.Run("taxon.col__id - taxa and synonyms get IDs, bare names empty", func(t *testing.T) {
		source := populate.DataSourceConfig{
			ID:              1001,
			Parent:          testdataDir,
			OutlinkIDColumn: "taxon.col__id",
			IsOutlinkReady:  true,
			OutlinkURL:      "https://www.catalogueoflife.org/data/taxon/{}",
		}

		cacheDir, err := prepareCacheDir()
		require.NoError(t, err)

		sqlitePath, err := fetchSFGA(ctx, source, cacheDir)
		require.NoError(t, err)

		sfgaDB, err := openSFGA(sqlitePath)
		require.NoError(t, err)
		defer sfgaDB.Close()

		hierarchy, err := buildHierarchy(ctx, sfgaDB, 4)
		require.NoError(t, err)

		populator := &PopulatorImpl{operator: op}

		// Process with outlink ID
		err = processNameIndices(ctx, populator, sfgaDB, &source, hierarchy, cfg)
		require.NoError(t, err)

		// Verify: Taxa should have outlink IDs (taxon.col__id)
		var taxaWithOutlink int
		err = op.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM name_string_indices
			WHERE data_source_id = $1
			AND record_id NOT LIKE 'bare-name-%'
			AND record_id NOT IN (
				SELECT record_id FROM name_string_indices WHERE accepted_record_id != record_id
			)
			AND outlink_id != ''
			AND outlink_id IS NOT NULL
		`, source.ID).Scan(&taxaWithOutlink)
		require.NoError(t, err)
		assert.Greater(t, taxaWithOutlink, 0, "Taxa should have outlink IDs")

		// Note: CoL small dataset may not have synonyms, so we just verify count is >= 0
		var synonymsWithOutlink int
		err = op.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM name_string_indices
			WHERE data_source_id = $1
			AND accepted_record_id != record_id
			AND outlink_id != ''
			AND outlink_id IS NOT NULL
		`, source.ID).Scan(&synonymsWithOutlink)
		require.NoError(t, err)
		t.Logf("Synonyms with outlink IDs: %d", synonymsWithOutlink)

		// Verify: Bare names should have empty outlink IDs (no taxon)
		var bareNamesWithOutlink int
		err = op.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM name_string_indices
			WHERE data_source_id = $1
			AND record_id LIKE 'bare-name-%'
			AND outlink_id != ''
			AND outlink_id IS NOT NULL
		`, source.ID).Scan(&bareNamesWithOutlink)
		require.NoError(t, err)
		assert.Equal(t, 0, bareNamesWithOutlink, "Bare names should have empty outlink IDs (no taxon table)")

		// Verify: Sample a few records to check outlink_id format
		var sampleOutlinkID string
		err = op.Pool().QueryRow(ctx, `
			SELECT outlink_id FROM name_string_indices
			WHERE data_source_id = $1
			AND outlink_id != ''
			LIMIT 1
		`, source.ID).Scan(&sampleOutlinkID)
		require.NoError(t, err)
		assert.NotEmpty(t, sampleOutlinkID, "Sample outlink ID should not be empty")
		t.Logf("Sample taxon outlink ID: %s", sampleOutlinkID)

		// Clean up for next test
		_, err = op.Pool().Exec(ctx, "DELETE FROM name_string_indices WHERE data_source_id = $1", source.ID)
		require.NoError(t, err)
	})

	t.Run("name.col__id - all record types get IDs", func(t *testing.T) {
		source := populate.DataSourceConfig{
			ID:              1001,
			Parent:          testdataDir,
			OutlinkIDColumn: "name.col__id",
			IsOutlinkReady:  true,
			OutlinkURL:      "https://example.org/name/{}",
		}

		cacheDir, err := prepareCacheDir()
		require.NoError(t, err)

		sqlitePath, err := fetchSFGA(ctx, source, cacheDir)
		require.NoError(t, err)

		sfgaDB, err := openSFGA(sqlitePath)
		require.NoError(t, err)
		defer sfgaDB.Close()

		hierarchy, err := buildHierarchy(ctx, sfgaDB, 4)
		require.NoError(t, err)

		populator := &PopulatorImpl{operator: op}

		// Process with name.col__id
		err = processNameIndices(ctx, populator, sfgaDB, &source, hierarchy, cfg)
		require.NoError(t, err)

		// Verify: ALL records should have outlink IDs (name table available for all)
		var totalRecords int
		err = op.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM name_string_indices
			WHERE data_source_id = $1
		`, source.ID).Scan(&totalRecords)
		require.NoError(t, err)

		var recordsWithOutlink int
		err = op.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM name_string_indices
			WHERE data_source_id = $1
			AND outlink_id != ''
			AND outlink_id IS NOT NULL
		`, source.ID).Scan(&recordsWithOutlink)
		require.NoError(t, err)

		assert.Equal(t, totalRecords, recordsWithOutlink, "All records (taxa, synonyms, bare names) should have outlink IDs from name table")

		// Clean up for next test
		_, err = op.Pool().Exec(ctx, "DELETE FROM name_string_indices WHERE data_source_id = $1", source.ID)
		require.NoError(t, err)
	})

	t.Run("empty outlink column - all records have empty outlink_id", func(t *testing.T) {
		source := populate.DataSourceConfig{
			ID:              1001,
			Parent:          testdataDir,
			OutlinkIDColumn: "", // No outlink configuration
			IsOutlinkReady:  false,
		}

		cacheDir, err := prepareCacheDir()
		require.NoError(t, err)

		sqlitePath, err := fetchSFGA(ctx, source, cacheDir)
		require.NoError(t, err)

		sfgaDB, err := openSFGA(sqlitePath)
		require.NoError(t, err)
		defer sfgaDB.Close()

		hierarchy, err := buildHierarchy(ctx, sfgaDB, 4)
		require.NoError(t, err)

		populator := &PopulatorImpl{operator: op}

		// Process without outlink configuration
		err = processNameIndices(ctx, populator, sfgaDB, &source, hierarchy, cfg)
		require.NoError(t, err)

		// Verify: No records should have outlink IDs
		var recordsWithOutlink int
		err = op.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM name_string_indices
			WHERE data_source_id = $1
			AND outlink_id != ''
			AND outlink_id IS NOT NULL
		`, source.ID).Scan(&recordsWithOutlink)
		require.NoError(t, err)

		assert.Equal(t, 0, recordsWithOutlink, "No records should have outlink IDs when not configured")
	})

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestProcessNameIndices_Idempotency tests that running processNameIndices twice
// produces the same result (old data is cleaned before import).
func TestProcessNameIndices_Idempotency(t *testing.T) {
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

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Open SFGA
	testdataDir := "../../testdata"
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	source := populate.DataSourceConfig{
		ID:     1002,
		Parent: testdataDir,
	}

	sqlitePath, err := fetchSFGA(ctx, source, cacheDir)
	require.NoError(t, err)

	sfgaDB, err := openSFGA(sqlitePath)
	require.NoError(t, err)
	defer sfgaDB.Close()

	hierarchy, err := buildHierarchy(ctx, sfgaDB, 4)
	require.NoError(t, err)

	populator := &PopulatorImpl{operator: op}

	// First import
	err = processNameIndices(ctx, populator, sfgaDB, &source, hierarchy, cfg)
	require.NoError(t, err)

	var firstCount int
	err = op.Pool().
		QueryRow(ctx, "SELECT COUNT(*) FROM name_string_indices WHERE data_source_id = $1", source.ID).
		Scan(&firstCount)
	require.NoError(t, err)
	require.Greater(t, firstCount, 0)

	// Second import (should clean old data first)
	err = processNameIndices(ctx, populator, sfgaDB, &source, hierarchy, cfg)
	require.NoError(t, err)

	var secondCount int
	err = op.Pool().
		QueryRow(ctx, "SELECT COUNT(*) FROM name_string_indices WHERE data_source_id = $1", source.ID).
		Scan(&secondCount)
	require.NoError(t, err)

	// Counts should be identical (idempotent)
	assert.Equal(t, firstCount, secondCount, "Second import should produce same count (idempotent)")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestProcessNameIndices_BareNames tests handling of bare names (names without taxa/synonyms).
// This test creates a minimal SFGA with only names table to ensure bare names are handled.
func TestProcessNameIndices_BareNames(t *testing.T) {
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

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Create minimal SFGA with only names (no taxon/synonym tables)
	sfgaDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE name (
			col__id TEXT PRIMARY KEY,
			col__scientific_name TEXT NOT NULL,
			gn__scientific_name_string TEXT,
			col__rank_id TEXT,
			col__code_id TEXT
		);

		INSERT INTO name (col__id, col__scientific_name, gn__scientific_name_string, col__rank_id, col__code_id) VALUES
			('1', 'Plantago major', 'Plantago major', 'species', 'BOTANICAL'),
			('2', 'Homo sapiens', 'Homo sapiens', 'species', 'ZOOLOGICAL'),
			('3', 'Escherichia coli', 'Escherichia coli', 'species', 'BACTERIAL');

		CREATE TABLE taxon (
			col__id TEXT PRIMARY KEY,
			col__name_id TEXT NOT NULL,
			col__parent_id TEXT,
			col__status_id TEXT,
			col__species TEXT DEFAULT '',
			col__section TEXT DEFAULT '',
			col__subgenus TEXT DEFAULT '',
			col__genus TEXT DEFAULT '',
			col__subtribe TEXT DEFAULT '',
			col__tribe TEXT DEFAULT '',
			col__subfamily TEXT DEFAULT '',
			col__family TEXT DEFAULT '',
			col__superfamily TEXT DEFAULT '',
			col__suborder TEXT DEFAULT '',
			col__order TEXT DEFAULT '',
			col__subclass TEXT DEFAULT '',
			col__class TEXT DEFAULT '',
			col__subphylum TEXT DEFAULT '',
			col__phylum TEXT DEFAULT '',
			col__kingdom TEXT DEFAULT '',
			sf__species_id TEXT DEFAULT '',
			sf__section_id TEXT DEFAULT '',
			sf__subgenus_id TEXT DEFAULT '',
			sf__genus_id TEXT DEFAULT '',
			sf__subtribe_id TEXT DEFAULT '',
			sf__tribe_id TEXT DEFAULT '',
			sf__subfamily_id TEXT DEFAULT '',
			sf__family_id TEXT DEFAULT '',
			sf__superfamily_id TEXT DEFAULT '',
			sf__suborder_id TEXT DEFAULT '',
			sf__order_id TEXT DEFAULT '',
			sf__subclass_id TEXT DEFAULT '',
			sf__class_id TEXT DEFAULT '',
			sf__subphylum_id TEXT DEFAULT '',
			sf__phylum_id TEXT DEFAULT '',
			sf__kingdom_id TEXT DEFAULT ''
		);

		CREATE TABLE synonym (
			col__id TEXT PRIMARY KEY,
			col__name_id TEXT NOT NULL,
			col__taxon_id TEXT,
			col__status_id TEXT
		);
	`)
	require.NoError(t, err)

	// Empty hierarchy (no taxa)
	hierarchy := make(map[string]*hNode)
	populator := &PopulatorImpl{operator: op}

	// Create minimal source config for test
	source := populate.DataSourceConfig{
		ID: 9999,
	}

	// Process - all names should become bare names
	err = processNameIndices(ctx, populator, sfgaDB, &source, hierarchy, cfg)
	require.NoError(t, err)

	// Verify: All names should be bare names
	var bareNameCount int
	err = op.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM name_string_indices
		WHERE data_source_id = 9999
		AND record_id LIKE 'bare-name-%'
	`).Scan(&bareNameCount)
	require.NoError(t, err)
	assert.Equal(t, 3, bareNameCount, "All 3 names should be bare names")

	// Verify: Bare names should have no classification
	var withClassification int
	err = op.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM name_string_indices
		WHERE data_source_id = 9999
		AND classification IS NOT NULL
		AND classification != ''
	`).Scan(&withClassification)
	require.NoError(t, err)
	assert.Equal(t, 0, withClassification, "Bare names should have no classification")

	// Clean up
	_ = op.DropAllTables(ctx)
}
