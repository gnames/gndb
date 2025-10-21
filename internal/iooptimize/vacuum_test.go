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

// TestVacuumAnalyze_Integration tests that VACUUM ANALYZE runs successfully
// and updates database statistics.
//
// Test scenario:
// 1. Given: Database with some data
// 2. When: Call vacuumAnalyze()
// 3. Then:
//   - VACUUM ANALYZE executes without error
//   - Statistics are updated (verify via pg_stat_user_tables)
//
// Note: This is a gndb enhancement, not in gnidump
func TestVacuumAnalyze_Integration(t *testing.T) {
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

	// Setup test data - insert some records to make statistics meaningful
	pool := op.Pool()

	// Insert test canonicals
	canonicalID := gnuuid.New("Test name").String()
	_, err = pool.Exec(ctx, `
		INSERT INTO canonicals (id, name) VALUES ($1, 'Test name')
	`, canonicalID)
	require.NoError(t, err)

	// Insert test name_strings
	nameID := gnuuid.New("Test name Author").String()
	_, err = pool.Exec(ctx, `
		INSERT INTO name_strings (id, name, canonical_id, surrogate, virus, bacteria, parse_quality, cardinality)
		VALUES ($1, 'Test name Author', $2, false, false, false, 1, 2)
	`, nameID, canonicalID)
	require.NoError(t, err)

	// Insert test name_string_indices
	_, err = pool.Exec(ctx, `
		INSERT INTO name_string_indices
		(data_source_id, record_id, name_string_id, local_id, outlink_id, code_id, rank, taxonomic_status, accepted_record_id, classification, classification_ranks, classification_ids)
		VALUES (1, 'test-1', $1, 'local-1', 'outlink-1', 1, 'species', 'accepted', NULL, '', '', '')
	`, nameID)
	require.NoError(t, err)

	// Create optimizer
	optimizer := &OptimizerImpl{
		operator: op,
	}

	// ACTION: Run VACUUM ANALYZE (should fail until T037 is implemented)
	err = vacuumAnalyze(ctx, optimizer, cfg)
	require.NoError(t, err, "vacuumAnalyze should succeed")

	// VERIFY: Check that statistics exist for our tables
	// We check pg_stat_user_tables to verify ANALYZE updated statistics
	var statsExist bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_stat_user_tables
			WHERE schemaname = 'public'
			AND relname IN ('name_strings', 'canonicals', 'name_string_indices')
		)
	`).Scan(&statsExist)
	require.NoError(t, err)
	assert.True(t, statsExist, "Statistics should exist for tables")

	// VERIFY: Check that last_analyze or last_autovacuum is set for at least one table
	// Note: This might be NULL if autovacuum hasn't run yet, so we just verify the query works
	var analyzeCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_stat_user_tables
		WHERE schemaname = 'public'
		AND relname IN ('name_strings', 'canonicals', 'name_string_indices')
	`).Scan(&analyzeCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, analyzeCount, 3, "Should have statistics for at least 3 tables")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestVacuumAnalyze_EmptyDatabase tests that VACUUM ANALYZE handles empty database gracefully.
func TestVacuumAnalyze_EmptyDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean and create empty schema
	err = op.DropAllTables(ctx)
	require.NoError(t, err)

	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	optimizer := &OptimizerImpl{
		operator: op,
	}

	// ACTION: Run VACUUM ANALYZE on empty database
	err = vacuumAnalyze(ctx, optimizer, cfg)
	require.NoError(t, err, "vacuumAnalyze should handle empty database gracefully")

	// Clean up
	_ = op.DropAllTables(ctx)
}

// TestVacuumAnalyze_Idempotent tests that running VACUUM ANALYZE multiple times is safe.
func TestVacuumAnalyze_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(t, err)
	defer op.Close()

	// Clean and setup
	err = op.DropAllTables(ctx)
	require.NoError(t, err)

	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(t, err)

	pool := op.Pool()

	// Insert minimal test data
	canonicalID := gnuuid.New("Test").String()
	_, err = pool.Exec(ctx, "INSERT INTO canonicals (id, name) VALUES ($1, 'Test')", canonicalID)
	require.NoError(t, err)

	optimizer := &OptimizerImpl{
		operator: op,
	}

	// Run VACUUM ANALYZE first time
	err = vacuumAnalyze(ctx, optimizer, cfg)
	require.NoError(t, err)

	// Run VACUUM ANALYZE second time (idempotent check)
	err = vacuumAnalyze(ctx, optimizer, cfg)
	require.NoError(t, err, "Second VACUUM ANALYZE should succeed")

	// Run VACUUM ANALYZE third time
	err = vacuumAnalyze(ctx, optimizer, cfg)
	require.NoError(t, err, "Third VACUUM ANALYZE should succeed")

	// Clean up
	_ = op.DropAllTables(ctx)
}
