package populate

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/gnames/gndb/internal/io/database"
	"github.com/gnames/gndb/internal/io/schema"
	iotesting "github.com/gnames/gndb/internal/io/testing"
	"github.com/gnames/gndb/pkg/populate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that requires PostgreSQL.
// Skip with: go test -short

// TestUpdateDataSourceMetadata_NewDataSource tests creating a new data source
// record with metadata from SFGA and sources.yaml merged together.
//
// This test verifies:
//  1. New data source is created with ID from sources.yaml
//  2. Metadata from SFGA metadata table is used (title, description, doi)
//  3. Metadata from sources.yaml overrides/supplements SFGA data
//  4. Record counts are queried from name_string_indices and vernacular_string_indices
//  5. updated_at timestamp is set to current time
func TestUpdateDataSourceMetadata_NewDataSource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := database.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err, "Should connect to database")
	defer op.Close()

	// Clean up and create schema
	_ = op.DropAllTables(ctx)
	sm := schema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err, "Schema creation should succeed")

	// Create test SFGA with metadata
	sfgaDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE metadata (
			col__title TEXT,
			col__description TEXT,
			col__doi TEXT
		);

		INSERT INTO metadata (col__title, col__description, col__doi) VALUES (
			'Test Dataset - Full Title',
			'This is a comprehensive description of the test dataset with detailed information.',
			'10.1234/test.dataset.2024'
		);

		CREATE TABLE name (col__id TEXT PRIMARY KEY);
		INSERT INTO name (col__id) VALUES ('name-1'), ('name-2'), ('name-3');

		CREATE TABLE vernacular (col__taxon_id TEXT, col__name TEXT);
		INSERT INTO vernacular (col__taxon_id, col__name) VALUES
			('taxon-1', 'Common name 1'),
			('taxon-1', 'Common name 2');
	`)
	require.NoError(t, err, "Should create test SFGA")

	// Insert some name_string_indices and vernacular_string_indices for counting
	populator := &PopulatorImpl{operator: op}
	sourceID := 9999

	// Insert test name indices
	_, err = op.Pool().Exec(ctx, `
		INSERT INTO name_string_indices (data_source_id, record_id, name_string_id, code_id)
		VALUES
			($1, 'rec-1', '00000000-0000-0000-0000-000000000001', 0),
			($1, 'rec-2', '00000000-0000-0000-0000-000000000002', 0),
			($1, 'rec-3', '00000000-0000-0000-0000-000000000003', 0)
	`, sourceID)
	require.NoError(t, err)

	// Insert test vernacular indices
	_, err = op.Pool().Exec(ctx, `
		INSERT INTO vernacular_string_indices (data_source_id, record_id, vernacular_string_id, language)
		VALUES
			($1, 'taxon-1', '00000000-0000-0000-0000-000000000011', 'English'),
			($1, 'taxon-1', '00000000-0000-0000-0000-000000000012', 'English')
	`, sourceID)
	require.NoError(t, err)

	// Create source config with metadata from sources.yaml
	source := populate.DataSourceConfig{
		ID:                sourceID,
		TitleShort:        "Test DS",
		HomeURL:           "https://example.org/test",
		IsCurated:         true,
		HasClassification: true,
	}

	// Test: Update data source metadata (this will fail until T047 is implemented)
	beforeUpdate := time.Now()
	err = updateDataSourceMetadata(ctx, populator, source, sfgaDB)
	require.NoError(t, err, "updateDataSourceMetadata should succeed")
	afterUpdate := time.Now()

	// Verify: Data source record was created
	var dsRecord struct {
		ID              int
		Title           string
		TitleShort      string
		Description     string
		DOI             string
		WebsiteURL      string
		IsCurated       bool
		HasTaxonData    bool
		RecordCount     int
		VernRecordCount int
		UpdatedAt       time.Time
	}

	err = op.Pool().QueryRow(ctx, `
		SELECT id, title, title_short, description, doi, website_url,
		       is_curated, has_taxon_data, record_count, vern_record_count, updated_at
		FROM data_sources
		WHERE id = $1
	`, sourceID).Scan(
		&dsRecord.ID,
		&dsRecord.Title,
		&dsRecord.TitleShort,
		&dsRecord.Description,
		&dsRecord.DOI,
		&dsRecord.WebsiteURL,
		&dsRecord.IsCurated,
		&dsRecord.HasTaxonData,
		&dsRecord.RecordCount,
		&dsRecord.VernRecordCount,
		&dsRecord.UpdatedAt,
	)
	require.NoError(t, err, "Should find data source record")

	// Verify: Basic fields
	assert.Equal(t, sourceID, dsRecord.ID, "ID should match")
	assert.Equal(t, "Test Dataset - Full Title", dsRecord.Title, "Title from SFGA")
	assert.Equal(t, "Test DS", dsRecord.TitleShort, "TitleShort from sources.yaml")
	assert.Equal(t, "This is a comprehensive description of the test dataset with detailed information.", dsRecord.Description, "Description from SFGA")
	assert.Equal(t, "10.1234/test.dataset.2024", dsRecord.DOI, "DOI from SFGA")
	assert.Equal(t, "https://example.org/test", dsRecord.WebsiteURL, "WebsiteURL from sources.yaml")

	// Verify: Boolean flags from sources.yaml
	assert.True(t, dsRecord.IsCurated, "IsCurated from sources.yaml")
	assert.True(t, dsRecord.HasTaxonData, "HasTaxonData from sources.yaml (HasClassification)")

	// Verify: Counts queried from database
	assert.Equal(t, 3, dsRecord.RecordCount, "RecordCount should be 3 (name indices)")
	assert.Equal(t, 2, dsRecord.VernRecordCount, "VernRecordCount should be 2 (vernacular indices)")

	// Verify: UpdatedAt timestamp is recent
	assert.True(t, dsRecord.UpdatedAt.After(beforeUpdate) || dsRecord.UpdatedAt.Equal(beforeUpdate),
		"UpdatedAt should be after or equal to beforeUpdate")
	assert.True(t, dsRecord.UpdatedAt.Before(afterUpdate) || dsRecord.UpdatedAt.Equal(afterUpdate),
		"UpdatedAt should be before or equal to afterUpdate")

	t.Logf("Data source created: ID=%d, Title=%s, RecordCount=%d, VernRecordCount=%d",
		dsRecord.ID, dsRecord.TitleShort, dsRecord.RecordCount, dsRecord.VernRecordCount)

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestUpdateDataSourceMetadata_ExistingDataSource tests updating an existing
// data source record (idempotency).
//
// This test verifies:
//  1. Existing data source is deleted before re-creating
//  2. Updated metadata replaces old metadata
//  3. Counts are re-queried and updated
//  4. updated_at timestamp is refreshed
func TestUpdateDataSourceMetadata_ExistingDataSource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := database.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := schema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Create test SFGA
	sfgaDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE metadata (
			col__title TEXT,
			col__description TEXT,
			col__doi TEXT
		);

		INSERT INTO metadata (col__title, col__description, col__doi) VALUES (
			'Updated Dataset Title',
			'Updated description',
			'10.5678/updated.2024'
		);
	`)
	require.NoError(t, err)

	populator := &PopulatorImpl{operator: op}
	sourceID := 9999

	// Insert initial data source record with old data
	oldTime := time.Now().Add(-24 * time.Hour) // 1 day ago
	_, err = op.Pool().Exec(ctx, `
		INSERT INTO data_sources (id, title, title_short, record_count, vern_record_count, updated_at)
		VALUES ($1, 'Old Title', 'Old DS', 100, 50, $2)
	`, sourceID, oldTime)
	require.NoError(t, err)

	// Insert some indices for counting
	_, err = op.Pool().Exec(ctx, `
		INSERT INTO name_string_indices (data_source_id, record_id, name_string_id, code_id)
		VALUES ($1, 'new-1', '00000000-0000-0000-0000-000000000001', 0)
	`, sourceID)
	require.NoError(t, err)

	source := populate.DataSourceConfig{
		ID:         sourceID,
		TitleShort: "Updated DS",
		HomeURL:    "https://example.org/updated",
	}

	// First update
	err = updateDataSourceMetadata(ctx, populator, source, sfgaDB)
	require.NoError(t, err)

	// Verify first update
	var firstUpdate struct {
		Title       string
		RecordCount int
		UpdatedAt   time.Time
	}
	err = op.Pool().QueryRow(ctx, `
		SELECT title, record_count, updated_at
		FROM data_sources WHERE id = $1
	`, sourceID).Scan(&firstUpdate.Title, &firstUpdate.RecordCount, &firstUpdate.UpdatedAt)
	require.NoError(t, err)

	assert.Equal(t, "Updated Dataset Title", firstUpdate.Title, "Title should be updated")
	assert.Equal(t, 1, firstUpdate.RecordCount, "RecordCount should be re-queried")
	assert.True(t, firstUpdate.UpdatedAt.After(oldTime), "UpdatedAt should be refreshed")

	// Insert more indices
	_, err = op.Pool().Exec(ctx, `
		INSERT INTO name_string_indices (data_source_id, record_id, name_string_id, code_id)
		VALUES ($1, 'new-2', '00000000-0000-0000-0000-000000000002', 0)
	`, sourceID)
	require.NoError(t, err)

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	// Second update (idempotency test)
	err = updateDataSourceMetadata(ctx, populator, source, sfgaDB)
	require.NoError(t, err)

	// Verify second update
	var secondUpdate struct {
		Title       string
		RecordCount int
		UpdatedAt   time.Time
	}
	err = op.Pool().QueryRow(ctx, `
		SELECT title, record_count, updated_at
		FROM data_sources WHERE id = $1
	`, sourceID).Scan(&secondUpdate.Title, &secondUpdate.RecordCount, &secondUpdate.UpdatedAt)
	require.NoError(t, err)

	assert.Equal(t, "Updated Dataset Title", secondUpdate.Title, "Title should remain updated")
	assert.Equal(t, 2, secondUpdate.RecordCount, "RecordCount should be re-queried again (now 2)")
	assert.True(t, secondUpdate.UpdatedAt.After(firstUpdate.UpdatedAt), "UpdatedAt should be refreshed on second update")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestUpdateDataSourceMetadata_EmptyMetadata tests handling when SFGA
// metadata table has NULL or empty fields.
//
// This test verifies:
//  1. NULL/empty SFGA fields are handled gracefully
//  2. Sources.yaml data still populates correctly
//  3. No errors occur with minimal metadata
func TestUpdateDataSourceMetadata_EmptyMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database
	op := database.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := schema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	// Create test SFGA with empty/null metadata
	sfgaDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE metadata (
			col__title TEXT,
			col__description TEXT,
			col__doi TEXT
		);

		INSERT INTO metadata (col__title, col__description, col__doi) VALUES (
			'', '', ''
		);
	`)
	require.NoError(t, err)

	populator := &PopulatorImpl{operator: op}
	sourceID := 9999

	source := populate.DataSourceConfig{
		ID:         sourceID,
		TitleShort: "Minimal DS",
		HomeURL:    "https://example.org/minimal",
	}

	// Test: Should handle empty metadata gracefully
	err = updateDataSourceMetadata(ctx, populator, source, sfgaDB)
	require.NoError(t, err, "Should handle empty SFGA metadata gracefully")

	// Verify: Data source record was created with sources.yaml data
	var dsRecord struct {
		ID         int
		Title      string
		TitleShort string
		WebsiteURL string
	}

	err = op.Pool().QueryRow(ctx, `
		SELECT id, title, title_short, website_url
		FROM data_sources WHERE id = $1
	`, sourceID).Scan(&dsRecord.ID, &dsRecord.Title, &dsRecord.TitleShort, &dsRecord.WebsiteURL)
	require.NoError(t, err, "Should find data source record")

	assert.Equal(t, sourceID, dsRecord.ID)
	assert.Equal(t, "", dsRecord.Title, "Title should be empty string from SFGA")
	assert.Equal(t, "Minimal DS", dsRecord.TitleShort, "TitleShort from sources.yaml should work")
	assert.Equal(t, "https://example.org/minimal", dsRecord.WebsiteURL, "WebsiteURL from sources.yaml should work")

	// Clean up
	_ = op.DropAllTables(ctx)
}
