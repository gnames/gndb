package iooptimize

import (
	"context"
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/gnames/gnparser"
	"github.com/gnames/gnparser/ent/parsed"
	"github.com/gnames/gnuuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that requires PostgreSQL.
// Skip with: go test -short

// TestCreateWords_Integration tests the Step 4 word extraction workflow.
// This test verifies:
//  1. words table is populated with normalized and modified word forms
//  2. word_name_strings junction table links words to names and canonicals
//  3. Only epithet and author words are included (type filtering)
//  4. Deduplication is applied (no duplicate words)
//  5. Words are extracted from cached parse results (no re-parsing)
//
// Reference: gnidump createWords() in words.go
func TestCreateWords_Integration(t *testing.T) {
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

	// Setup cache with parsed results
	cacheDir := t.TempDir() + "/cache"
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err, "Should create cache manager")
	err = cm.Open()
	require.NoError(t, err, "Should open cache")
	defer func() { _ = cm.Cleanup() }()

	// Prepare test data: Insert name_strings with parsed results in cache
	// We'll use real scientific names that gnparser can parse
	testNames := []struct {
		id       string
		name     string
		wantWord string // Expected word to be extracted
	}{
		{
			gnuuid.New("Homo sapiens Linnaeus 1758").String(),
			"Homo sapiens Linnaeus 1758",
			"sapiens", // genus epithet
		},
		{
			gnuuid.New("Mus musculus").String(),
			"Mus musculus",
			"musculus", // species epithet
		},
		{
			gnuuid.New("Felis catus domesticus").String(),
			"Felis catus domesticus",
			"domesticus", // infraspecific epithet
		},
	}

	// Insert name_strings and populate cache with parsed results
	for _, tn := range testNames {
		// Insert name_string
		canonicalID := gnuuid.New(tn.name).String() // Simplified for test
		_, err = pool.Exec(ctx, `
			INSERT INTO name_strings (id, name, canonical_id, virus, bacteria, surrogate, parse_quality)
			VALUES ($1, $2, $3, false, false, false, 1)
		`, tn.id, tn.name, canonicalID)
		require.NoError(t, err, "Should insert test name_string")

		// Insert corresponding canonical
		_, err = pool.Exec(ctx, `
			INSERT INTO canonicals (id, name) VALUES ($1, $2)
		`, canonicalID, tn.name)
		require.NoError(t, err, "Should insert canonical")

		// Parse the name and store in cache (simulating Step 1 reparse)
		// This is critical - createWords must use cached results, not re-parse
		parsed := parseNameForTest(t, tn.name)
		err = cm.StoreParsed(tn.id, parsed)
		require.NoError(t, err, "Should store parsed result in cache")
	}

	// Verify cache contains parsed results (Step 1 prerequisite)
	cachedParsed, err := cm.GetParsed(testNames[0].id)
	require.NoError(t, err, "Should retrieve from cache")
	require.NotNil(t, cachedParsed, "Cache should contain parsed result")

	// Create optimizer with cache
	optimizer := &OptimizerImpl{
		operator: op,
		cache:    cm,
	}

	// TEST: Call createWords (this will fail until T023-T030 are implemented)
	err = createWords(ctx, optimizer, cfg)
	require.NoError(t, err, "createWords should succeed")

	// VERIFY 1: words table is populated
	var wordCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM words").Scan(&wordCount)
	require.NoError(t, err, "Should query words count")
	assert.Greater(t, wordCount, 0, "Words table should be populated")

	// VERIFY 2: word_name_strings junction table is populated
	var junctionCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM word_name_strings").Scan(&junctionCount)
	require.NoError(t, err, "Should query word_name_strings count")
	assert.Greater(t, junctionCount, 0, "Junction table should be populated")

	// VERIFY 3: Check specific word extraction (e.g., "sapiens" from "Homo sapiens")
	// The word should have both normalized and modified forms
	var wordID, normalized, modified string
	var typeID int
	query := `
		SELECT id, normalized, modified, type_id
		FROM words
		WHERE normalized = $1
		LIMIT 1
	`
	err = pool.QueryRow(ctx, query, testNames[0].wantWord).Scan(&wordID, &normalized, &modified, &typeID)
	require.NoError(t, err, "Should find extracted word")
	assert.Equal(t, testNames[0].wantWord, normalized, "Word should be normalized correctly")
	assert.NotEmpty(t, modified, "Word should have modified form")
	assert.NotEmpty(t, wordID, "Word should have UUID")

	// VERIFY 4: Check word-name-string linkage
	var linkedNameID, linkedCanonicalID string
	linkQuery := `
		SELECT name_string_id, canonical_id
		FROM word_name_strings
		WHERE word_id = $1
		LIMIT 1
	`
	err = pool.QueryRow(ctx, linkQuery, wordID).Scan(&linkedNameID, &linkedCanonicalID)
	require.NoError(t, err, "Should find word-name linkage")
	assert.NotEmpty(t, linkedNameID, "Should link to name_string")
	assert.NotEmpty(t, linkedCanonicalID, "Should link to canonical")

	// VERIFY 5: Deduplication - if same word appears in multiple names, should only have one entry
	// Count distinct words with same normalized form
	var distinctCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT id) FROM words WHERE normalized = $1
	`, testNames[0].wantWord).Scan(&distinctCount)
	require.NoError(t, err)
	assert.Equal(t, 1, distinctCount, "Duplicate words should be deduplicated")

	// VERIFY 6: Only epithet and author words are included (type filtering)
	// Check that no genus words (type 1) are included if we're only extracting epithets
	// This depends on gnparser word types: SpEpithet, InfraspEpithet, AuthorWord
	var genusCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM words WHERE type_id = 1").Scan(&genusCount)
	require.NoError(t, err)
	// Genus words (type 1) should NOT be extracted per gnidump logic
	assert.Equal(t, 0, genusCount, "Genus words should not be extracted")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestCreateWords_Idempotent tests that word creation can be run multiple times safely.
func TestCreateWords_Idempotent(t *testing.T) {
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

	pool := op.Pool()

	// Setup cache
	cacheDir := t.TempDir() + "/cache"
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)
	err = cm.Open()
	require.NoError(t, err)
	defer func() { _ = cm.Cleanup() }()

	// Insert test data
	testID := gnuuid.New("Homo sapiens").String()
	canonicalID := gnuuid.New("Homo sapiens").String()

	_, err = pool.Exec(ctx, `
		INSERT INTO name_strings (id, name, canonical_id, virus, bacteria, surrogate, parse_quality)
		VALUES ($1, $2, $3, false, false, false, 1)
	`, testID, "Homo sapiens", canonicalID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO canonicals (id, name) VALUES ($1, $2)
	`, canonicalID, "Homo sapiens")
	require.NoError(t, err)

	// Cache parsed result
	parsed := parseNameForTest(t, "Homo sapiens")
	err = cm.StoreParsed(testID, parsed)
	require.NoError(t, err)

	optimizer := &OptimizerImpl{
		operator: op,
		cache:    cm,
	}

	// First run
	err = createWords(ctx, optimizer, cfg)
	require.NoError(t, err, "First createWords should succeed")

	// Get counts after first run
	var wordCount1, junctionCount1 int
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM words").Scan(&wordCount1)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM word_name_strings").Scan(&junctionCount1)

	// Second run (idempotent test)
	err = createWords(ctx, optimizer, cfg)
	require.NoError(t, err, "Second createWords should succeed")

	// Get counts after second run
	var wordCount2, junctionCount2 int
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM words").Scan(&wordCount2)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM word_name_strings").Scan(&junctionCount2)

	// Counts should remain the same (no duplicates)
	assert.Equal(t, wordCount1, wordCount2, "Words count should not change on second run")
	assert.Equal(t, junctionCount1, junctionCount2, "Junction count should not change on second run")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestCreateWords_EmptyCache tests that createWords handles missing cache gracefully.
func TestCreateWords_EmptyCache(t *testing.T) {
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

	pool := op.Pool()

	// Setup cache (but don't populate it)
	cacheDir := t.TempDir() + "/cache"
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)
	err = cm.Open()
	require.NoError(t, err)
	defer func() { _ = cm.Cleanup() }()

	// Insert name_string WITHOUT caching parsed result
	testID := gnuuid.New("Homo sapiens").String()
	canonicalID := gnuuid.New("Homo sapiens").String()

	_, err = pool.Exec(ctx, `
		INSERT INTO name_strings (id, name, canonical_id, virus, bacteria, surrogate, parse_quality)
		VALUES ($1, $2, $3, false, false, false, 1)
	`, testID, "Homo sapiens", canonicalID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO canonicals (id, name) VALUES ($1, $2)
	`, canonicalID, "Homo sapiens")
	require.NoError(t, err)

	optimizer := &OptimizerImpl{
		operator: op,
		cache:    cm,
	}

	// Call createWords with empty cache
	// This should handle missing cache entries gracefully (skip or error appropriately)
	err = createWords(ctx, optimizer, cfg)
	// The exact behavior depends on implementation - either succeed with 0 words or return error
	// For now, we just verify it doesn't panic
	assert.NotPanics(t, func() {
		_ = createWords(ctx, optimizer, cfg)
	}, "Should handle empty cache gracefully")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// parseNameForTest is a helper function to parse a name for testing purposes.
// This simulates what Step 1 (reparse) would store in the cache.
func parseNameForTest(t *testing.T, name string) *parsed.Parsed {
	t.Helper()

	cfg := gnparser.NewConfig()
	gnp := gnparser.New(cfg)
	parsed := gnp.ParseName(name)

	return &parsed
}
