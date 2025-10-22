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
	"github.com/stretchr/testify/require"
)

// BenchmarkReparseRowByRow benchmarks the current row-by-row update approach.
// This represents the CURRENT implementation (T013) that updates name_strings
// one row at a time using individual transactions.
//
// Run with: go test -bench=BenchmarkReparseRowByRow -benchmem -run=^$
func BenchmarkReparseRowByRow(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	// Setup database once for all benchmark runs
	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(b, err, "Should connect to database")
	defer op.Close()

	// Test with different dataset sizes
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.StopTimer()

			// Setup: Clean database and create schema
			_ = op.DropAllTables(ctx)
			sm := ioschema.NewManager(op)
			err = sm.Create(ctx, cfg)
			require.NoError(b, err)

			// Insert test data
			insertTestNames(b, ctx, op, size)

			// Create optimizer
			optimizer := &OptimizerImpl{operator: op}

			b.ReportAllocs()
			b.StartTimer()

			// Benchmark the row-by-row reparse workflow
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// Reset parse_quality to 0 to force reparsing
				_, err = op.Pool().Exec(ctx, "UPDATE name_strings SET canonical_id = NULL, parse_quality = 0")
				require.NoError(b, err)
				b.StartTimer()

				err = reparseNames(ctx, optimizer, cfg)
				require.NoError(b, err, "reparseNames should succeed")
			}

			b.StopTimer()
			// Report rows processed per second
			b.ReportMetric(float64(size*b.N)/b.Elapsed().Seconds(), "rows/sec")
		})
	}
}

// insertTestNames inserts N test name_strings into the database.
// Names are realistic scientific names with varying complexity.
func insertTestNames(b *testing.B, ctx context.Context, op *iodb.PgxOperator, count int) {
	// Template names with varying characteristics
	templates := []string{
		"Homo sapiens",
		"Homo sapiens Linnaeus 1758",
		"Canis lupus familiaris",
		"Felis catus (Linnaeus, 1758)",
		"Escherichia coli",
		"Tobacco mosaic virus",
		"Quercus robur L.",
		"Pinus sylvestris var. mongolica",
		"Taraxacum officinale F.H.Wigg.",
		"Mus musculus domesticus",
	}

	// Batch insert for performance
	batchSize := 1000
	batch := make([]struct {
		id   string
		name string
	}, 0, batchSize)

	for i := 0; i < count; i++ {
		template := templates[i%len(templates)]
		name := fmt.Sprintf("%s var%d", template, i/len(templates))
		id := gnuuid.New(name).String()

		batch = append(batch, struct {
			id   string
			name string
		}{id, name})

		if len(batch) == batchSize || i == count-1 {
			for _, item := range batch {
				query := `
					INSERT INTO name_strings (
						id, name, cardinality, canonical_id, canonical_full_id,
						canonical_stem_id, virus, bacteria, surrogate, parse_quality, year
					)
					VALUES ($1, $2, NULL, NULL, NULL, NULL, false, false, false, 0, NULL)
				`
				_, err := op.Pool().Exec(ctx, query, item.id, item.name)
				require.NoError(b, err, "Should insert test name_string")
			}
			batch = batch[:0]
		}
	}
}

// BenchmarkReparseBatch benchmarks the PLANNED batch update approach.
// This represents the NEW implementation (T029-T033) that will use:
// 1. Temporary table for parsed results
// 2. Bulk insert via pgx CopyFrom
// 3. Single batch UPDATE statement
//
// EXPECTED: This benchmark will FAIL until T029-T033 are implemented.
// Run with: go test -bench=BenchmarkReparseBatch -benchmem -run=^$
func BenchmarkReparseBatch(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(b, err)
	defer op.Close()

	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.StopTimer()

			_ = op.DropAllTables(ctx)
			sm := ioschema.NewManager(op)
			err = sm.Create(ctx, cfg)
			require.NoError(b, err)

			insertTestNames(b, ctx, op, size)
			optimizer := &OptimizerImpl{operator: op}

			b.ReportAllocs()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				_, err = op.Pool().Exec(ctx, "UPDATE name_strings SET canonical_id = NULL, parse_quality = 0")
				require.NoError(b, err)
				b.StartTimer()

				// TODO T033: Replace with reparseNamesBatch() when implemented
				err = reparseNames(ctx, optimizer, cfg)
				require.NoError(b, err, "batch reparse should succeed")
			}

			b.StopTimer()
			b.ReportMetric(float64(size*b.N)/b.Elapsed().Seconds(), "rows/sec")
		})
	}
}

// BenchmarkReparseScaling tests performance scaling from 1K to 100K rows.
// Run with: go test -bench=BenchmarkReparseScaling -benchmem -run=^$ -timeout=30m
func BenchmarkReparseScaling(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(b, err)
	defer op.Close()

	sizes := []int{1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.StopTimer()

			_ = op.DropAllTables(ctx)
			sm := ioschema.NewManager(op)
			err = sm.Create(ctx, cfg)
			require.NoError(b, err)

			insertTestNames(b, ctx, op, size)
			optimizer := &OptimizerImpl{operator: op}

			b.ReportAllocs()
			b.StartTimer()

			err = reparseNames(ctx, optimizer, cfg)
			require.NoError(b, err)

			b.StopTimer()

			rowsPerSec := float64(size) / b.Elapsed().Seconds()
			b.ReportMetric(rowsPerSec, "rows/sec")
			b.ReportMetric(b.Elapsed().Seconds(), "total_sec")
		})
	}
}

// BenchmarkReparseMemory measures memory usage during reparsing.
// Run with: go test -bench=BenchmarkReparseMemory -benchmem -run=^$
func BenchmarkReparseMemory(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(b, err)
	defer op.Close()

	size := 10000

	b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
		b.StopTimer()

		_ = op.DropAllTables(ctx)
		sm := ioschema.NewManager(op)
		err = sm.Create(ctx, cfg)
		require.NoError(b, err)

		insertTestNames(b, ctx, op, size)
		optimizer := &OptimizerImpl{operator: op}

		b.ReportAllocs()
		b.StartTimer()

		for i := 0; i < b.N; i++ {
			b.StopTimer()
			_, err = op.Pool().Exec(ctx, "UPDATE name_strings SET canonical_id = NULL, parse_quality = 0")
			require.NoError(b, err)
			b.StartTimer()

			err = reparseNames(ctx, optimizer, cfg)
			require.NoError(b, err)
		}
	})
}

// BenchmarkReparseConcurrency tests different worker counts.
// Run with: go test -bench=BenchmarkReparseConcurrency -benchmem -run=^$
func BenchmarkReparseConcurrency(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(b, err)
	defer op.Close()

	size := 5000
	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("workers_%d", workers), func(b *testing.B) {
			b.StopTimer()

			_ = op.DropAllTables(ctx)
			sm := ioschema.NewManager(op)
			err = sm.Create(ctx, cfg)
			require.NoError(b, err)

			insertTestNames(b, ctx, op, size)

			testCfg := *cfg
			testCfg.JobsNumber = workers

			optimizer := &OptimizerImpl{operator: op}

			b.ReportAllocs()
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				_, err = op.Pool().Exec(ctx, "UPDATE name_strings SET canonical_id = NULL, parse_quality = 0")
				require.NoError(b, err)
				b.StartTimer()

				err = reparseNames(ctx, optimizer, &testCfg)
				require.NoError(b, err)
			}

			b.StopTimer()
			b.ReportMetric(float64(size*b.N)/b.Elapsed().Seconds(), "rows/sec")
		})
	}
}

// BenchmarkTempTableOperations benchmarks temporary table creation and operations.
// Run with: go test -bench=BenchmarkTempTableOperations -benchmem -run=^$
func BenchmarkTempTableOperations(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()
	cfg := iotesting.GetTestConfig()

	op := iodb.NewPgxOperator()
	err := op.Connect(ctx, &cfg.Database)
	require.NoError(b, err)
	defer op.Close()

	_ = op.DropAllTables(ctx)
	sm := ioschema.NewManager(op)
	err = sm.Create(ctx, cfg)
	require.NoError(b, err)

	b.Run("create_drop_temp_table", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err = op.Pool().Exec(ctx, `
				CREATE UNLOGGED TABLE IF NOT EXISTS temp_reparse_names (
					name_string_id UUID PRIMARY KEY,
					canonical_id UUID,
					canonical_full_id UUID,
					canonical_stem_id UUID,
					canonical TEXT,
					canonical_full TEXT,
					canonical_stem TEXT,
					bacteria BOOLEAN,
					virus BOOLEAN,
					surrogate BOOLEAN,
					parse_quality INTEGER,
					cardinality INTEGER,
					year SMALLINT
				)
			`)
			require.NoError(b, err)

			_, err = op.Pool().Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
			require.NoError(b, err)
		}
	})

	b.Run("insert_to_temp_table", func(b *testing.B) {
		_, err = op.Pool().Exec(ctx, `
			CREATE UNLOGGED TABLE IF NOT EXISTS temp_reparse_names (
				name_string_id UUID PRIMARY KEY,
				canonical_id UUID,
				canonical TEXT,
				parse_quality INTEGER
			)
		`)
		require.NoError(b, err)

		testData := make([]reparsed, 1000)
		for i := 0; i < 1000; i++ {
			name := fmt.Sprintf("Genus species%d", i)
			testData[i] = reparsed{
				nameStringID: gnuuid.New(name).String(),
				name:         name,
				canonicalID:  sql.NullString{String: gnuuid.New(name).String(), Valid: true},
				canonical:    name,
				parseQuality: 1,
			}
		}

		b.ReportAllocs()
		b.StartTimer()

		for i := 0; i < b.N; i++ {
			b.StopTimer()
			_, err = op.Pool().Exec(ctx, "TRUNCATE temp_reparse_names")
			require.NoError(b, err)
			b.StartTimer()

			for _, r := range testData {
				_, err = op.Pool().Exec(ctx, `
					INSERT INTO temp_reparse_names (
						name_string_id, canonical_id, canonical, parse_quality
					) VALUES ($1, $2, $3, $4)
				`, r.nameStringID, r.canonicalID, r.canonical, r.parseQuality)
				require.NoError(b, err)
			}
		}

		_, err = op.Pool().Exec(ctx, "DROP TABLE IF EXISTS temp_reparse_names")
		require.NoError(b, err)
	})
}
