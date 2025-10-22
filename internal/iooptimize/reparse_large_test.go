//go:build large_scale
// +build large_scale

package iooptimize

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/internal/iotesting"
	"github.com/gnames/gnuuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReparseNames_LargeScale tests the batch reparse workflow with 100K rows
// to validate scalability for 100M row databases.
//
// This test simulates real-world scenarios:
// - First optimization: 100% of rows need parsing (worst case)
// - Partial update: 50% of rows changed
// - Re-optimization: 1-10% of rows changed (best case)
//
// Performance targets (for 100K rows):
// - Time: < 10 minutes
// - Memory: < 2GB
// - Linear scaling: If 10K rows = 1 min, 100K rows should = ~10 min (not 100 min)
//
// Build with: go test -tags=large_scale -timeout=30m -v ./internal/iooptimize
//
// EXPECTED: This test will FAIL until T029-T033 (batch implementation) is complete.
func TestReparseNames_LargeScale(t *testing.T) {
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

	// Test parameters
	const totalRows = 100000 // 100K rows (scaled test for 100M)
	t.Logf("Creating test database with %d rows...", totalRows)

	// Scenario 1: First optimization (100% need parsing)
	t.Run("FirstOptimization_100Percent", func(t *testing.T) {
		// Track memory before
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		// Insert 100K name_strings with parse_quality=0 (never parsed)
		startInsert := time.Now()
		insertTestNamesLarge(t, ctx, op, totalRows, 0) // 0% pre-parsed
		insertDuration := time.Since(startInsert)
		t.Logf("Inserted %d rows in %v", totalRows, insertDuration)

		// Create optimizer
		optimizer := &OptimizerImpl{operator: op}

		// Run batch reparse workflow
		startReparse := time.Now()
		err = reparseNames(ctx, optimizer, cfg)
		reparseDuration := time.Since(startReparse)
		require.NoError(t, err, "Batch reparse should succeed")

		t.Logf("Reparsed %d rows in %v", totalRows, reparseDuration)

		// Track memory after
		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)
		memUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / 1024 / 1024

		// VERIFY 1: All rows updated
		var updatedCount int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM name_strings WHERE parse_quality > 0
		`).Scan(&updatedCount)
		require.NoError(t, err)
		assert.Equal(t, totalRows, updatedCount, "All rows should be parsed")

		// VERIFY 2: Performance - should complete in < 10 minutes
		assert.Less(t, reparseDuration.Minutes(), 10.0,
			"100K rows should complete in < 10 minutes (got %v)", reparseDuration)

		// VERIFY 3: Memory usage - should stay under 2GB
		assert.Less(t, memUsedMB, 2048.0,
			"Memory usage should be < 2GB (got %.2f MB)", memUsedMB)

		// VERIFY 4: Throughput
		rowsPerSec := float64(totalRows) / reparseDuration.Seconds()
		t.Logf("Throughput: %.0f rows/sec", rowsPerSec)
		assert.Greater(t, rowsPerSec, 100.0,
			"Should process > 100 rows/sec (got %.0f)", rowsPerSec)

		t.Logf("Memory used: %.2f MB", memUsedMB)
		t.Logf("Performance: %.0f rows/sec", rowsPerSec)
	})

	// Scenario 2: Partial update (50% changed)
	t.Run("PartialUpdate_50Percent", func(t *testing.T) {
		// Clear and re-insert with 50% already parsed
		_, _ = pool.Exec(ctx, "TRUNCATE name_strings CASCADE")
		insertTestNamesLarge(t, ctx, op, totalRows, 50) // 50% pre-parsed

		optimizer := &OptimizerImpl{operator: op}

		startReparse := time.Now()
		err = reparseNames(ctx, optimizer, cfg)
		reparseDuration := time.Since(startReparse)
		require.NoError(t, err)

		t.Logf("Reparsed %d rows (50%% changed) in %v", totalRows, reparseDuration)

		// With filter-then-batch, should be faster since only 50% go to temp table
		// Expected: ~50-70% of full reparse time (not 100%)
		rowsPerSec := float64(totalRows) / reparseDuration.Seconds()
		t.Logf("Throughput: %.0f rows/sec", rowsPerSec)

		// Should still be fast
		assert.Less(t, reparseDuration.Minutes(), 8.0,
			"50%% changed should complete in < 8 minutes (got %v)", reparseDuration)
	})

	// Scenario 3: Re-optimization (10% changed)
	t.Run("ReOptimization_10Percent", func(t *testing.T) {
		// Clear and re-insert with 90% already parsed
		_, _ = pool.Exec(ctx, "TRUNCATE name_strings CASCADE")
		insertTestNamesLarge(t, ctx, op, totalRows, 90) // 90% pre-parsed

		optimizer := &OptimizerImpl{operator: op}

		startReparse := time.Now()
		err = reparseNames(ctx, optimizer, cfg)
		reparseDuration := time.Since(startReparse)
		require.NoError(t, err)

		t.Logf("Reparsed %d rows (10%% changed) in %v", totalRows, reparseDuration)

		// With filter-then-batch, should be much faster
		// Expected: ~20-30% of full reparse time
		rowsPerSec := float64(totalRows) / reparseDuration.Seconds()
		t.Logf("Throughput: %.0f rows/sec", rowsPerSec)

		assert.Less(t, reparseDuration.Minutes(), 5.0,
			"10%% changed should complete in < 5 minutes (got %v)", reparseDuration)
	})

	// Cleanup
	_ = op.DropAllTables(ctx)
}

// TestReparseNames_LinearScaling validates that performance scales linearly.
// Tests with 10K, 50K, 100K rows to ensure O(n) complexity.
func TestReparseNames_LinearScaling(t *testing.T) {
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

	sizes := []int{10000, 50000, 100000}
	durations := make([]time.Duration, len(sizes))

	for i, size := range sizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			// Clear database
			_, _ = pool.Exec(ctx, "TRUNCATE name_strings CASCADE")

			// Insert test data
			insertTestNamesLarge(t, ctx, op, size, 0)

			optimizer := &OptimizerImpl{operator: op}

			// Time the reparse
			start := time.Now()
			err = reparseNames(ctx, optimizer, cfg)
			duration := time.Since(start)
			require.NoError(t, err)

			durations[i] = duration
			rowsPerSec := float64(size) / duration.Seconds()

			t.Logf("Size: %d rows, Duration: %v, Throughput: %.0f rows/sec",
				size, duration, rowsPerSec)
		})
	}

	// Verify linear scaling
	// If 10K takes T seconds, 50K should take ~5T (not 25T)
	// Allow 2x margin for overhead
	ratio_50k_10k := durations[1].Seconds() / durations[0].Seconds()
	expectedRatio := 5.0
	t.Logf("50K/10K ratio: %.2fx (expected ~5x)", ratio_50k_10k)
	assert.Less(t, ratio_50k_10k, expectedRatio*2,
		"50K should scale linearly from 10K (got %.2fx, expected ~%.0fx)",
		ratio_50k_10k, expectedRatio)

	ratio_100k_10k := durations[2].Seconds() / durations[0].Seconds()
	expectedRatio = 10.0
	t.Logf("100K/10K ratio: %.2fx (expected ~10x)", ratio_100k_10k)
	assert.Less(t, ratio_100k_10k, expectedRatio*2,
		"100K should scale linearly from 10K (got %.2fx, expected ~%.0fx)",
		ratio_100k_10k, expectedRatio)

	_ = op.DropAllTables(ctx)
}

// TestReparseNames_MemoryProfile tests memory usage across different scenarios.
func TestReparseNames_MemoryProfile(t *testing.T) {
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

	testCases := []struct {
		name        string
		totalRows   int
		changedPct  int
		maxMemoryMB float64
	}{
		{"10K_100Pct", 10000, 0, 200},    // 10K all changed = ~200MB
		{"50K_100Pct", 50000, 0, 1000},   // 50K all changed = ~1GB
		{"100K_50Pct", 100000, 50, 1000}, // 100K 50% changed = ~1GB
		{"100K_10Pct", 100000, 90, 500},  // 100K 10% changed = ~500MB
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear
			_, _ = pool.Exec(ctx, "TRUNCATE name_strings CASCADE")

			// Force GC before measurement
			runtime.GC()
			var memBefore runtime.MemStats
			runtime.ReadMemStats(&memBefore)

			// Insert data
			insertTestNamesLarge(t, ctx, op, tc.totalRows, tc.changedPct)

			optimizer := &OptimizerImpl{operator: op}

			// Reparse
			err = reparseNames(ctx, optimizer, cfg)
			require.NoError(t, err)

			// Measure memory
			runtime.GC()
			var memAfter runtime.MemStats
			runtime.ReadMemStats(&memAfter)

			memUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / 1024 / 1024

			t.Logf("Memory used: %.2f MB (max: %.0f MB)", memUsedMB, tc.maxMemoryMB)

			assert.Less(t, memUsedMB, tc.maxMemoryMB,
				"Memory should be < %.0f MB (got %.2f MB)",
				tc.maxMemoryMB, memUsedMB)
		})
	}

	_ = op.DropAllTables(ctx)
}

// insertTestNamesLarge inserts N name_strings with optional pre-parsing.
// preParsedPercent: 0-100, percentage of rows to pre-parse (simulate already optimized)
func insertTestNamesLarge(
	t *testing.T,
	ctx context.Context,
	op *iodb.PgxOperator,
	count int,
	preParsedPercent int,
) {
	t.Helper()

	// Template names for variety
	templates := []string{
		"Homo sapiens",
		"Homo sapiens Linnaeus 1758",
		"Canis lupus familiaris",
		"Felis catus (Linnaeus, 1758)",
		"Escherichia coli",
		"Mus musculus domesticus",
		"Arabidopsis thaliana",
		"Drosophila melanogaster",
		"Saccharomyces cerevisiae",
		"Zea mays L.",
	}

	batchSize := 1000
	pool := op.Pool()

	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}

		// Use transaction for batch insert
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)

		for j := i; j < end; j++ {
			template := templates[j%len(templates)]
			name := fmt.Sprintf("%s var%d", template, j)
			nameID := gnuuid.New(name).String()

			// Determine if this row should be pre-parsed
			isPreparsed := (j * 100 / count) < preParsedPercent

			var canonicalID sql.NullString
			var parseQuality int

			if isPreparsed {
				// Pre-parsed: set canonical_id and parse_quality
				canonicalID = sql.NullString{
					String: gnuuid.New(template).String(),
					Valid:  true,
				}
				parseQuality = 1
			} else {
				// Not parsed: NULL canonical_id, parse_quality=0
				canonicalID = sql.NullString{}
				parseQuality = 0
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO name_strings (
					id, name, canonical_id, bacteria, virus, surrogate, parse_quality, cardinality, year
				) VALUES ($1, $2, $3, false, false, false, $4, NULL, NULL)
			`, nameID, name, canonicalID, parseQuality)
			require.NoError(t, err)
		}

		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Progress
		if (i+batchSize)%10000 == 0 {
			t.Logf("Inserted %d/%d rows", i+batchSize, count)
		}
	}

	t.Logf("Inserted %d rows total (%d%% pre-parsed)", count, preParsedPercent)
}
