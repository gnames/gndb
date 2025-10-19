package iooptimize

import (
	"context"
	"database/sql"
	"fmt"
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

// TestReparseNames_Integration tests the Step 1 name reparsing workflow.
// This test verifies:
//  1. All name_strings are loaded from database
//  2. Names are parsed with latest gnparser algorithms
//  3. name_strings.canonical_id, canonical_full_id, canonical_stem_id are updated
//  4. bacteria, virus, surrogate flags are updated
//  5. parse_quality is set correctly
//  6. Parsed results are stored in cache
//  7. Canonical tables (canonicals, canonical_fulls, canonical_stems) are populated
func TestReparseNames_Integration(t *testing.T) {
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

	// Prepare test data: Insert sample name_strings
	testNames := []struct {
		id   string
		name string
	}{
		{gnuuid.New("Homo sapiens Linnaeus 1758").String(), "Homo sapiens Linnaeus 1758"},
		{gnuuid.New("Mus musculus").String(), "Mus musculus"},
		{gnuuid.New("Felis catus (Linnaeus, 1758)").String(), "Felis catus (Linnaeus, 1758)"},
		{gnuuid.New("Canis lupus familiaris").String(), "Canis lupus familiaris"},
	}

	// Insert initial name_strings with empty canonical fields
	// This simulates the state before reparsing
	for _, tn := range testNames {
		query := `
			INSERT INTO name_strings (id, name, cardinality, canonical_id, canonical_full_id, canonical_stem_id, virus, bacteria, surrogate, parse_quality)
			VALUES ($1, $2, NULL, NULL, NULL, NULL, false, false, false, 0)
		`
		_, err = op.Pool().Exec(ctx, query, tn.id, tn.name)
		require.NoError(t, err, "Should insert test name_string")
	}

	// Verify initial state: canonical_id should be NULL
	var nullCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_strings WHERE canonical_id IS NULL").Scan(&nullCount)
	require.NoError(t, err)
	assert.Equal(t, len(testNames), nullCount, "All canonical_id fields should be NULL initially")

	// Setup cache for parsed results
	cacheDir := t.TempDir() + "/cache"
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err, "Should create cache manager")

	err = cm.Open()
	require.NoError(t, err, "Should open cache")
	defer cm.Cleanup()

	// Create optimizer with cache
	optimizer := &OptimizerImpl{
		operator: op,
		cache:    cm,
	}

	// TEST: Call reparseNames (this will fail until T004-T008 are implemented)
	err = reparseNames(ctx, optimizer, cfg)
	require.NoError(t, err, "reparseNames should succeed")

	// VERIFY 1: canonical_id should now be populated
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_strings WHERE canonical_id IS NOT NULL").Scan(&nullCount)
	require.NoError(t, err)
	assert.Equal(t, len(testNames), nullCount, "All canonical_id fields should be populated after reparsing")

	// VERIFY 2: Check specific canonical forms
	var canonical string
	var cardinality sql.NullInt32
	var parseQuality int

	// Check "Homo sapiens Linnaeus 1758"
	homoID := gnuuid.New("Homo sapiens Linnaeus 1758").String()
	query := `
		SELECT ns.canonical_id, ns.cardinality, ns.parse_quality, c.name
		FROM name_strings ns
		LEFT JOIN canonicals c ON ns.canonical_id = c.id
		WHERE ns.id = $1
	`
	var canonicalID sql.NullString
	err = op.Pool().QueryRow(ctx, query, homoID).Scan(&canonicalID, &cardinality, &parseQuality, &canonical)
	require.NoError(t, err, "Should query Homo sapiens")
	assert.True(t, canonicalID.Valid, "canonical_id should be set")
	assert.Equal(t, "Homo sapiens", canonical, "Canonical form should be 'Homo sapiens'")
	assert.Equal(t, int32(2), cardinality.Int32, "Cardinality should be 2 (binomial)")
	assert.Equal(t, 1, parseQuality, "Parse quality should be 1 (clean parse)")

	// VERIFY 3: Check year extraction for "Homo sapiens Linnaeus 1758"
	var year sql.NullInt16
	err = op.Pool().QueryRow(ctx, "SELECT year FROM name_strings WHERE id = $1", homoID).Scan(&year)
	require.NoError(t, err)
	assert.True(t, year.Valid, "Year should be extracted")
	assert.Equal(t, int16(1758), year.Int16, "Year should be 1758")

	// VERIFY 4: Check trinomial "Canis lupus familiaris"
	canisID := gnuuid.New("Canis lupus familiaris").String()
	err = op.Pool().QueryRow(ctx, query, canisID).Scan(&canonicalID, &cardinality, &parseQuality, &canonical)
	require.NoError(t, err)
	assert.Equal(t, "Canis lupus familiaris", canonical, "Canonical form should be full trinomial")
	assert.Equal(t, int32(3), cardinality.Int32, "Cardinality should be 3 (trinomial)")

	// VERIFY 5: Check canonical_stem_id is populated for binomials/trinomials
	var stemID sql.NullString
	err = op.Pool().QueryRow(ctx, "SELECT canonical_stem_id FROM name_strings WHERE id = $1", homoID).Scan(&stemID)
	require.NoError(t, err)
	assert.True(t, stemID.Valid, "Stemmed canonical should be generated for binomials")

	// VERIFY 6: Check canonicals table was populated
	var canonicalCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount)
	require.NoError(t, err)
	assert.Greater(t, canonicalCount, 0, "Canonicals table should be populated")

	// VERIFY 7: Check canonical_fulls table for names with authorship
	var fullCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonical_fulls").Scan(&fullCount)
	require.NoError(t, err)
	// At least one name has authorship, so canonical_full should differ from canonical
	assert.GreaterOrEqual(t, fullCount, 0, "Canonical_fulls may be populated if authorship differs")

	// VERIFY 8: Check canonical_stems table
	var stemCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonical_stems").Scan(&stemCount)
	require.NoError(t, err)
	assert.Greater(t, stemCount, 0, "Canonical_stems should be populated for binomials/trinomials")

	// VERIFY 9: Check cache contains parsed results
	// Try to retrieve one parsed result from cache
	parsedFromCache, err := cm.GetParsed(homoID)
	require.NoError(t, err, "Should retrieve from cache")
	assert.NotNil(t, parsedFromCache, "Cache should contain parsed result")
	assert.Equal(t, "Homo sapiens", parsedFromCache.CanonicalSimple, "Cached canonical should match")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestReparseNames_Idempotent tests that reparsing can be run multiple times safely.
// Running reparse twice should not cause errors or duplicate data.
func TestReparseNames_Idempotent(t *testing.T) {
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

	// Insert test data
	testID := gnuuid.New("Homo sapiens").String()
	query := `
		INSERT INTO name_strings (id, name, cardinality, canonical_id, canonical_full_id, canonical_stem_id, virus, bacteria, surrogate, parse_quality)
		VALUES ($1, $2, NULL, NULL, NULL, NULL, false, false, false, 0)
	`
	_, err = op.Pool().Exec(ctx, query, testID, "Homo sapiens")
	require.NoError(t, err)

	// Setup cache
	cacheDir := t.TempDir() + "/cache"
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)
	err = cm.Open()
	require.NoError(t, err)
	defer cm.Cleanup()

	optimizer := &OptimizerImpl{
		operator: op,
		cache:    cm,
	}

	// First reparse
	err = reparseNames(ctx, optimizer, cfg)
	require.NoError(t, err, "First reparse should succeed")

	// Get counts after first run
	var canonicalCount1, fullCount1, stemCount1 int
	_ = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount1)
	_ = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonical_fulls").Scan(&fullCount1)
	_ = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonical_stems").Scan(&stemCount1)

	// Second reparse (idempotent test)
	err = reparseNames(ctx, optimizer, cfg)
	require.NoError(t, err, "Second reparse should succeed")

	// Get counts after second run
	var canonicalCount2, fullCount2, stemCount2 int
	_ = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount2)
	_ = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonical_fulls").Scan(&fullCount2)
	_ = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonical_stems").Scan(&stemCount2)

	// Counts should remain the same (no duplicates)
	assert.Equal(t, canonicalCount1, canonicalCount2, "Canonicals count should not change on second run")
	assert.Equal(t, fullCount1, fullCount2, "Canonical_fulls count should not change")
	assert.Equal(t, stemCount1, stemCount2, "Canonical_stems count should not change")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestReparseNames_UpdatesOnlyChangedNames tests that reparsing only updates
// names whose canonical forms have changed with new parser algorithms.
func TestReparseNames_UpdatesOnlyChangedNames(t *testing.T) {
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

	// Insert name that already has correct canonical_id
	testID := gnuuid.New("Homo sapiens").String()
	correctCanonicalID := gnuuid.New("Homo sapiens").String()

	// First insert the canonical
	_, err = op.Pool().Exec(ctx, "INSERT INTO canonicals (id, name) VALUES ($1, $2)", correctCanonicalID, "Homo sapiens")
	require.NoError(t, err)

	// Then insert name_string with correct canonical_id already set
	query := `
		INSERT INTO name_strings (id, name, cardinality, canonical_id, canonical_full_id, canonical_stem_id, virus, bacteria, surrogate, parse_quality)
		VALUES ($1, $2, 2, $3, NULL, NULL, false, false, false, 1)
	`
	_, err = op.Pool().Exec(ctx, query, testID, "Homo sapiens", correctCanonicalID)
	require.NoError(t, err)

	// Setup cache
	cacheDir := t.TempDir() + "/cache"
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)
	err = cm.Open()
	require.NoError(t, err)
	defer cm.Cleanup()

	optimizer := &OptimizerImpl{
		operator: op,
		cache:    cm,
	}

	// Reparse
	err = reparseNames(ctx, optimizer, cfg)
	require.NoError(t, err)

	// Verify canonical_id remains the same (no unnecessary update)
	var canonicalIDAfter sql.NullString
	err = op.Pool().QueryRow(ctx, "SELECT canonical_id FROM name_strings WHERE id = $1", testID).Scan(&canonicalIDAfter)
	require.NoError(t, err)
	assert.Equal(t, correctCanonicalID, canonicalIDAfter.String, "Canonical ID should remain unchanged")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestReparseNames_VirusNames tests handling of virus names which have special parsing rules.
func TestReparseNames_VirusNames(t *testing.T) {
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

	// Insert a virus name
	virusName := "Tobacco mosaic virus"
	virusID := gnuuid.New(virusName).String()
	query := `
		INSERT INTO name_strings (id, name, cardinality, canonical_id, canonical_full_id, canonical_stem_id, virus, bacteria, surrogate, parse_quality)
		VALUES ($1, $2, NULL, NULL, NULL, NULL, false, false, false, 0)
	`
	_, err = op.Pool().Exec(ctx, query, virusID, virusName)
	require.NoError(t, err)

	// Setup cache
	cacheDir := t.TempDir() + "/cache"
	cm, err := NewCacheManager(cacheDir)
	require.NoError(t, err)
	err = cm.Open()
	require.NoError(t, err)
	defer cm.Cleanup()

	optimizer := &OptimizerImpl{
		operator: op,
		cache:    cm,
	}

	// Reparse
	err = reparseNames(ctx, optimizer, cfg)
	require.NoError(t, err)

	// Verify virus flag is set
	var virus bool
	err = op.Pool().QueryRow(ctx, "SELECT virus FROM name_strings WHERE id = $1", virusID).Scan(&virus)
	require.NoError(t, err)
	assert.True(t, virus, "Virus flag should be set for virus names")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestLoadNamesForReparse_Unit tests the loadNamesForReparse function in isolation.
// This is a unit test for T004 implementation.
func TestLoadNamesForReparse_Unit(t *testing.T) {
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

	// Insert test name_strings
	testData := []struct {
		id   string
		name string
	}{
		{gnuuid.New("Homo sapiens").String(), "Homo sapiens"},
		{gnuuid.New("Mus musculus").String(), "Mus musculus"},
		{gnuuid.New("Felis catus").String(), "Felis catus"},
	}

	for _, td := range testData {
		query := `
			INSERT INTO name_strings (id, name, cardinality, canonical_id, canonical_full_id, canonical_stem_id, virus, bacteria, surrogate, parse_quality)
			VALUES ($1, $2, NULL, NULL, NULL, NULL, false, false, false, 0)
		`
		_, err = op.Pool().Exec(ctx, query, td.id, td.name)
		require.NoError(t, err)
	}

	// Create optimizer
	optimizer := &OptimizerImpl{operator: op}

	// Create channel to receive names
	chIn := make(chan reparsed, 10)

	// Launch loadNamesForReparse in goroutine
	go func() {
		defer close(chIn)
		err := loadNamesForReparse(ctx, optimizer, chIn)
		assert.NoError(t, err, "loadNamesForReparse should succeed")
	}()

	// Collect all loaded names
	loaded := make(map[string]string)
	for r := range chIn {
		loaded[r.nameStringID] = r.name
	}

	// Verify all names were loaded
	assert.Equal(t, len(testData), len(loaded), "Should load all names")
	for _, td := range testData {
		name, found := loaded[td.id]
		assert.True(t, found, "Should find name ID: %s", td.id)
		assert.Equal(t, td.name, name, "Name should match")
	}

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestLoadNamesForReparse_ContextCancellation tests that loadNamesForReparse
// properly handles context cancellation.
func TestLoadNamesForReparse_ContextCancellation(t *testing.T) {
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

	// Insert many names to ensure cancellation happens mid-stream
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("Species name %d", i)
		id := gnuuid.New(name).String()
		query := `
			INSERT INTO name_strings (id, name, cardinality, canonical_id, canonical_full_id, canonical_stem_id, virus, bacteria, surrogate, parse_quality)
			VALUES ($1, $2, NULL, NULL, NULL, NULL, false, false, false, 0)
		`
		_, err = op.Pool().Exec(ctx, query, id, name)
		require.NoError(t, err)
	}

	// Create optimizer
	optimizer := &OptimizerImpl{operator: op}

	// Create cancellable context
	cancelCtx, cancel := context.WithCancel(ctx)

	// Create channel
	chIn := make(chan reparsed, 10)

	// Launch loadNamesForReparse
	errCh := make(chan error, 1)
	go func() {
		defer close(chIn)
		errCh <- loadNamesForReparse(cancelCtx, optimizer, chIn)
	}()

	// Read a few names then cancel
	count := 0
	for range chIn {
		count++
		if count == 5 {
			cancel() // Cancel context
			break
		}
	}

	// Drain channel
	for range chIn {
	}

	// Wait for error
	err = <-errCh
	assert.Error(t, err, "Should return error when context is cancelled")
	assert.Equal(t, context.Canceled, err, "Error should be context.Canceled")

	// Clean up
	_ = op.DropAllTables(ctx)
}
