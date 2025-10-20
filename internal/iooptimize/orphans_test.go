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

// Note: This is an integration test that requires PostgreSQL.
// Skip with: go test -short

// TestRemoveOrphans_Integration tests the Step 3 orphan removal workflow.
// This test verifies:
//  1. Orphaned name_strings (not in name_string_indices) are deleted
//  2. Orphaned canonicals (not referenced by name_strings) are deleted
//  3. Orphaned canonical_fulls (not referenced by name_strings) are deleted
//  4. Orphaned canonical_stems (not referenced by name_strings) are deleted
//  5. Referenced records remain intact
//
// Reference: gnidump removeOrphans() in db_views.go
func TestRemoveOrphans_Integration(t *testing.T) {
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

	// Setup test data source
	_, err = pool.Exec(ctx, `
		INSERT INTO data_sources (id, title, description, is_outlink_ready)
		VALUES (999, 'Test Source', 'Test orphan data', false)
	`)
	require.NoError(t, err)

	// Scenario: Create 4 name_strings, but only 2 are referenced by name_string_indices
	nameID1 := gnuuid.New("Homo sapiens").String()
	nameID2 := gnuuid.New("Mus musculus").String()
	nameID3 := gnuuid.New("Felis catus").String() // orphan - not in indices
	nameID4 := gnuuid.New("Canis lupus").String() // orphan - not in indices

	canonicalID1 := gnuuid.New("Homo sapiens").String()
	canonicalID2 := gnuuid.New("Mus musculus").String()
	canonicalID3 := gnuuid.New("Felis catus").String()         // orphan - referenced by orphan name
	canonicalOrphan := gnuuid.New("Orphan canonical").String() // orphan - not referenced at all

	canonicalFullID1 := gnuuid.New("Homo sapiens Linnaeus").String()
	canonicalFullOrphan := gnuuid.New("Orphan full").String() // orphan - not referenced

	canonicalStemID1 := gnuuid.New("Hom sapien").String()
	canonicalStemOrphan := gnuuid.New("Orphan stem").String() // orphan - not referenced

	// Insert canonicals (some will become orphans)
	canonicals := []struct {
		id   string
		name string
	}{
		{canonicalID1, "Homo sapiens"},
		{canonicalID2, "Mus musculus"},
		{canonicalID3, "Felis catus"},
		{canonicalOrphan, "Orphan canonical"},
	}
	for _, c := range canonicals {
		_, err = pool.Exec(ctx, "INSERT INTO canonicals (id, name) VALUES ($1, $2)", c.id, c.name)
		require.NoError(t, err)
	}

	// Insert canonical_fulls (some will become orphans)
	fullCanonicals := []struct {
		id   string
		name string
	}{
		{canonicalFullID1, "Homo sapiens Linnaeus"},
		{canonicalFullOrphan, "Orphan full"},
	}
	for _, cf := range fullCanonicals {
		_, err = pool.Exec(ctx, "INSERT INTO canonical_fulls (id, name) VALUES ($1, $2)", cf.id, cf.name)
		require.NoError(t, err)
	}

	// Insert canonical_stems (some will become orphans)
	stemCanonicals := []struct {
		id   string
		name string
	}{
		{canonicalStemID1, "Hom sapien"},
		{canonicalStemOrphan, "Orphan stem"},
	}
	for _, cs := range stemCanonicals {
		_, err = pool.Exec(ctx, "INSERT INTO canonical_stems (id, name) VALUES ($1, $2)", cs.id, cs.name)
		require.NoError(t, err)
	}

	// Insert name_strings (some will become orphans)
	nameStrings := []struct {
		id              string
		name            string
		canonicalID     string
		canonicalFullID *string
		canonicalStemID *string
	}{
		{nameID1, "Homo sapiens", canonicalID1, &canonicalFullID1, &canonicalStemID1}, // referenced
		{nameID2, "Mus musculus", canonicalID2, nil, nil},                             // referenced
		{nameID3, "Felis catus", canonicalID3, nil, nil},                              // orphan name
		{nameID4, "Canis lupus", "", nil, nil},                                        // orphan name with no canonical
	}

	for _, ns := range nameStrings {
		var canonicalIDVal interface{} = nil
		if ns.canonicalID != "" {
			canonicalIDVal = ns.canonicalID
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO name_strings
			(id, name, canonical_id, canonical_full_id, canonical_stem_id, virus, bacteria, surrogate, parse_quality)
			VALUES ($1, $2, $3, $4, $5, false, false, false, 1)
		`, ns.id, ns.name, canonicalIDVal, ns.canonicalFullID, ns.canonicalStemID)
		require.NoError(t, err)
	}

	// Insert name_string_indices - only for nameID1 and nameID2 (others become orphans)
	indices := []struct {
		nameStringID string
		recordID     string
	}{
		{nameID1, "rec_1"},
		{nameID2, "rec_2"},
		// nameID3 and nameID4 are NOT in indices -> they are orphans
	}

	for _, idx := range indices {
		_, err = pool.Exec(ctx, `
			INSERT INTO name_string_indices
			(data_source_id, record_id, name_string_id, local_id)
			VALUES (999, $1, $2, $1)
		`, idx.recordID, idx.nameStringID)
		require.NoError(t, err)
	}

	// Verify initial state
	var nameCount, canonicalCount, fullCount, stemCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&nameCount)
	require.NoError(t, err)
	assert.Equal(t, 4, nameCount, "Should have 4 name_strings initially")

	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount)
	require.NoError(t, err)
	assert.Equal(t, 4, canonicalCount, "Should have 4 canonicals initially")

	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonical_fulls").Scan(&fullCount)
	require.NoError(t, err)
	assert.Equal(t, 2, fullCount, "Should have 2 canonical_fulls initially")

	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonical_stems").Scan(&stemCount)
	require.NoError(t, err)
	assert.Equal(t, 2, stemCount, "Should have 2 canonical_stems initially")

	// Create optimizer
	optimizer := &OptimizerImpl{
		operator: op,
	}

	// TEST: Call removeOrphans (this will fail until T017-T021 are implemented)
	err = removeOrphans(ctx, optimizer, cfg)
	require.NoError(t, err, "removeOrphans should succeed")

	// VERIFY 1: Orphaned name_strings should be deleted
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&nameCount)
	require.NoError(t, err)
	assert.Equal(t, 2, nameCount, "Should have 2 name_strings after orphan removal (nameID1 and nameID2)")

	// Verify the right names remain
	var exists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM name_strings WHERE id = $1)", nameID1).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "nameID1 (Homo sapiens) should still exist")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM name_strings WHERE id = $1)", nameID2).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "nameID2 (Mus musculus) should still exist")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM name_strings WHERE id = $1)", nameID3).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "nameID3 (Felis catus) should be deleted as orphan")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM name_strings WHERE id = $1)", nameID4).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "nameID4 (Canis lupus) should be deleted as orphan")

	// VERIFY 2: Orphaned canonicals should be deleted
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount)
	require.NoError(t, err)
	assert.Equal(t, 2, canonicalCount, "Should have 2 canonicals after orphan removal")

	// Verify the right canonicals remain
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM canonicals WHERE id = $1)", canonicalID1).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "canonicalID1 should still exist (referenced by nameID1)")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM canonicals WHERE id = $1)", canonicalID2).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "canonicalID2 should still exist (referenced by nameID2)")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM canonicals WHERE id = $1)", canonicalID3).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "canonicalID3 should be deleted (referenced by orphan name)")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM canonicals WHERE id = $1)", canonicalOrphan).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "canonicalOrphan should be deleted (not referenced)")

	// VERIFY 3: Orphaned canonical_fulls should be deleted
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonical_fulls").Scan(&fullCount)
	require.NoError(t, err)
	assert.Equal(t, 1, fullCount, "Should have 1 canonical_full after orphan removal")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM canonical_fulls WHERE id = $1)", canonicalFullID1).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "canonicalFullID1 should still exist (referenced by nameID1)")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM canonical_fulls WHERE id = $1)", canonicalFullOrphan).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "canonicalFullOrphan should be deleted (not referenced)")

	// VERIFY 4: Orphaned canonical_stems should be deleted
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonical_stems").Scan(&stemCount)
	require.NoError(t, err)
	assert.Equal(t, 1, stemCount, "Should have 1 canonical_stem after orphan removal")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM canonical_stems WHERE id = $1)", canonicalStemID1).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "canonicalStemID1 should still exist (referenced by nameID1)")

	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM canonical_stems WHERE id = $1)", canonicalStemOrphan).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "canonicalStemOrphan should be deleted (not referenced)")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestRemoveOrphans_Idempotent tests that orphan removal can be run multiple times safely.
// Running removeOrphans twice should not cause errors or affect remaining data.
func TestRemoveOrphans_Idempotent(t *testing.T) {
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

	// Setup test data source
	_, err = pool.Exec(ctx, `
		INSERT INTO data_sources (id, title, description, is_outlink_ready)
		VALUES (999, 'Test Source', 'Test orphan data', false)
	`)
	require.NoError(t, err)

	// Create minimal test data
	nameID := gnuuid.New("Homo sapiens").String()
	canonicalID := gnuuid.New("Homo sapiens").String()

	_, err = pool.Exec(ctx, "INSERT INTO canonicals (id, name) VALUES ($1, $2)", canonicalID, "Homo sapiens")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO name_strings
		(id, name, canonical_id, virus, bacteria, surrogate, parse_quality)
		VALUES ($1, $2, $3, false, false, false, 1)
	`, nameID, "Homo sapiens", canonicalID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO name_string_indices
		(data_source_id, record_id, name_string_id, local_id)
		VALUES (999, 'rec_1', $1, 'rec_1')
	`, nameID)
	require.NoError(t, err)

	optimizer := &OptimizerImpl{
		operator: op,
	}

	// First run
	err = removeOrphans(ctx, optimizer, cfg)
	require.NoError(t, err, "First removeOrphans should succeed")

	// Get counts after first run
	var nameCount1, canonicalCount1 int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&nameCount1)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount1)
	require.NoError(t, err)

	// Second run (idempotent test)
	err = removeOrphans(ctx, optimizer, cfg)
	require.NoError(t, err, "Second removeOrphans should succeed")

	// Get counts after second run
	var nameCount2, canonicalCount2 int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&nameCount2)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount2)
	require.NoError(t, err)

	// Counts should remain the same (no data loss)
	assert.Equal(t, nameCount1, nameCount2, "name_strings count should not change on second run")
	assert.Equal(t, canonicalCount1, canonicalCount2, "canonicals count should not change on second run")
	assert.Equal(t, 1, nameCount2, "Should still have 1 name_string")
	assert.Equal(t, 1, canonicalCount2, "Should still have 1 canonical")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestRemoveOrphans_EmptyDatabase tests that removeOrphans handles empty database gracefully.
func TestRemoveOrphans_EmptyDatabase(t *testing.T) {
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

	optimizer := &OptimizerImpl{
		operator: op,
	}

	// Call removeOrphans on empty database
	err = removeOrphans(ctx, optimizer, cfg)
	require.NoError(t, err, "removeOrphans should succeed on empty database")

	// Verify counts are still zero
	var nameCount, canonicalCount int
	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&nameCount)
	require.NoError(t, err)
	assert.Equal(t, 0, nameCount, "name_strings should remain empty")

	err = op.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount)
	require.NoError(t, err)
	assert.Equal(t, 0, canonicalCount, "canonicals should remain empty")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestRemoveOrphans_CascadeOrder tests that orphans are removed in correct order.
// Names must be removed before canonicals, otherwise we'd create more orphans.
func TestRemoveOrphans_CascadeOrder(t *testing.T) {
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

	// Setup test data source
	_, err = pool.Exec(ctx, `
		INSERT INTO data_sources (id, title, description, is_outlink_ready)
		VALUES (999, 'Test Source', 'Test cascade', false)
	`)
	require.NoError(t, err)

	// Create orphan name that references canonical
	// After name removal, canonical should also be removed
	nameID := gnuuid.New("Orphan name").String()
	canonicalID := gnuuid.New("Orphan canonical").String()

	_, err = pool.Exec(ctx, "INSERT INTO canonicals (id, name) VALUES ($1, $2)", canonicalID, "Orphan canonical")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO name_strings
		(id, name, canonical_id, virus, bacteria, surrogate, parse_quality)
		VALUES ($1, $2, $3, false, false, false, 1)
	`, nameID, "Orphan name", canonicalID)
	require.NoError(t, err)

	// Do NOT insert into name_string_indices - name is orphan

	optimizer := &OptimizerImpl{
		operator: op,
	}

	// Remove orphans
	err = removeOrphans(ctx, optimizer, cfg)
	require.NoError(t, err)

	// Both name and canonical should be gone
	var nameCount, canonicalCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM name_strings").Scan(&nameCount)
	require.NoError(t, err)
	assert.Equal(t, 0, nameCount, "Orphan name_string should be removed")

	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM canonicals").Scan(&canonicalCount)
	require.NoError(t, err)
	assert.Equal(t, 0, canonicalCount, "Canonical should be removed after orphan name is gone")

	// Clean up
	_ = op.DropAllTables(ctx)
}
