package iooptimize

import (
	"context"
	"database/sql"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/gnames/gnuuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that requires PostgreSQL.
// Skip with: go test -short

// TestFixVernacularLanguages_Integration tests the Step 2 vernacular language fix workflow.
// This test verifies:
//  1. language_orig field is populated with original language value
//  2. lang_code is converted to 3-letter ISO code (e.g., "en" → "eng")
//  3. All lang_code values are lowercase
//  4. language field is normalized
//
// Reference: gnidump fixVernLang() in db_vern.go
func TestFixVernacularLanguages_Integration(t *testing.T) {
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

	// Insert test data source
	_, err = pool.Exec(ctx, `
		INSERT INTO data_sources (id, title, description, is_outlink_ready)
		VALUES (999, 'Test Source', 'Test vernacular data', false)
	`)
	require.NoError(t, err)

	// Insert vernacular_string_indices with various language codes
	// Schema: data_source_id, record_id, vernacular_string_id, language, lang_code
	testData := []struct {
		recordID     string
		language     string
		langCode     sql.NullString
		expectedCode string
		expectedLang string
		description  string
	}{
		{"rec_1", "en", sql.NullString{}, "eng", "English", "2-letter code, no lang_code"},
		{"rec_2", "de", sql.NullString{}, "deu", "German", "2-letter code, no lang_code"},
		{"rec_3", "fra", sql.NullString{}, "fra", "French", "3-letter code, no lang_code"},
		{"rec_4", "ES", sql.NullString{}, "spa", "Spanish", "Uppercase 2-letter"},
		{"rec_5", "Italian", sql.NullString{}, "ita", "Italian", "Language name"},
		{"rec_6", "por", newNullStr("POR"), "por", "Portuguese", "Uppercase lang_code"},
		{"rec_7", "rus", newNullStr("rus"), "rus", "Russian", "Already correct"},
	}

	for _, td := range testData {
		vernUUID := gnuuid.New(td.recordID).String()
		_, err = pool.Exec(ctx, `
			INSERT INTO vernacular_string_indices
			(data_source_id, record_id, vernacular_string_id, language, lang_code)
			VALUES ($1, $2, $3, $4, $5)
		`, 999, td.recordID, vernUUID, td.language, td.langCode)
		require.NoError(t, err, "Should insert test data: %s", td.description)
	}

	// Create optimizer (no cache needed for vernacular fix)
	optimizer := &OptimizerImpl{
		operator: op,
	}

	// TEST: Call fixVernacularLanguages (this will fail until T010-T015 are implemented)
	err = fixVernacularLanguages(ctx, optimizer, cfg)
	require.NoError(t, err, "fixVernacularLanguages should succeed")

	// Verify results
	rows, err := pool.Query(ctx, `
		SELECT record_id, language, language_orig, lang_code
		FROM vernacular_string_indices
		ORDER BY record_id
	`)
	require.NoError(t, err)
	defer rows.Close()

	var results []struct {
		recordID     string
		language     string
		languageOrig sql.NullString
		langCode     sql.NullString
	}

	for rows.Next() {
		var r struct {
			recordID     string
			language     string
			languageOrig sql.NullString
			langCode     sql.NullString
		}
		err = rows.Scan(&r.recordID, &r.language, &r.languageOrig, &r.langCode)
		require.NoError(t, err)
		results = append(results, r)
	}

	require.Len(t, results, 7, "Should have 7 vernacular records")

	// VERIFY 1: language_orig should be populated
	t.Run("language_orig populated", func(t *testing.T) {
		for i, r := range results {
			assert.True(t, r.languageOrig.Valid, "language_orig should be populated for %s", testData[i].description)
			assert.Equal(t, testData[i].language, r.languageOrig.String, "language_orig should preserve original for %s", testData[i].description)
		}
	})

	// VERIFY 2: lang_code converted to 3-letter lowercase
	t.Run("lang_code normalized", func(t *testing.T) {
		for i, r := range results {
			assert.True(t, r.langCode.Valid, "lang_code should be set for %s", testData[i].description)
			assert.Equal(t, testData[i].expectedCode, r.langCode.String, "lang_code mismatch for %s", testData[i].description)
		}
	})

	// VERIFY 3: language field normalized to full name
	t.Run("language normalized", func(t *testing.T) {
		for i, r := range results {
			assert.Equal(t, testData[i].expectedLang, r.language, "language should be normalized for %s", testData[i].description)
		}
	})
}

// TestFixVernacularLanguages_Idempotent verifies that running the fix multiple
// times produces the same result (safe reruns).
func TestFixVernacularLanguages_Idempotent(t *testing.T) {
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

	// Clean and setup
	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Insert test data source
	_, err = pool.Exec(ctx, `
		INSERT INTO data_sources (id, title, description, is_outlink_ready)
		VALUES (999, 'Test Source', 'Test vernacular data', false)
	`)
	require.NoError(t, err)

	// Insert test vernacular record
	vernUUID := gnuuid.New("rec_1").String()
	_, err = pool.Exec(ctx, `
		INSERT INTO vernacular_string_indices
		(data_source_id, record_id, vernacular_string_id, language, lang_code)
		VALUES (999, 'rec_1', $1, 'en', NULL)
	`, vernUUID)
	require.NoError(t, err)

	optimizer := &OptimizerImpl{
		operator: op,
	}

	// Run first time
	err = fixVernacularLanguages(ctx, optimizer, cfg)
	require.NoError(t, err)

	// Get results after first run
	var firstLang, firstLangOrig, firstCode string
	err = pool.QueryRow(ctx, `
		SELECT language, language_orig, lang_code
		FROM vernacular_string_indices WHERE record_id = 'rec_1'
	`).Scan(&firstLang, &firstLangOrig, &firstCode)
	require.NoError(t, err)

	// Run second time (idempotent check)
	err = fixVernacularLanguages(ctx, optimizer, cfg)
	require.NoError(t, err)

	// Get results after second run
	var secondLang, secondLangOrig, secondCode string
	err = pool.QueryRow(ctx, `
		SELECT language, language_orig, lang_code
		FROM vernacular_string_indices WHERE record_id = 'rec_1'
	`).Scan(&secondLang, &secondLangOrig, &secondCode)
	require.NoError(t, err)

	// Verify idempotent behavior
	assert.Equal(t, firstLang, secondLang, "language should not change on second run")
	assert.Equal(t, firstLangOrig, secondLangOrig, "language_orig should not change on second run")
	assert.Equal(t, firstCode, secondCode, "lang_code should not change on second run")
	assert.Equal(t, "eng", secondCode, "lang_code should be 'eng'")
	assert.Equal(t, "English", secondLang, "language should be 'English'")
}
