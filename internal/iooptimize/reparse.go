package iooptimize

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"github.com/dustin/go-humanize"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gnlib"
	"github.com/gnames/gnparser"
	"github.com/gnames/gnparser/ent/parsed"
	"github.com/gnames/gnuuid"
	"golang.org/x/sync/errgroup"
)

// reparsed holds the data for a name_string being reparsed.
// This structure mirrors gnidump's reparsed struct for compatibility.
type reparsed struct {
	nameStringID                                  string
	name                                          string
	canonicalID, canonicalFullID, canonicalStemID sql.NullString
	canonical, canonicalFull, canonicalStem       string
	bacteria                                      bool
	surrogate, virus                              sql.NullBool
	parseQuality                                  int
	cardinality                                   sql.NullInt32
	year                                          sql.NullInt16
}

// loadNamesForReparse loads all name_strings from database for reparsing.
// It queries the database and sends each name_string to the input channel.
// Progress is logged every 100,000 names.
//
// Reference: gnidump loadReparse() in db_reparse.go
func loadNamesForReparse(
	ctx context.Context,
	optimizer *OptimizerImpl,
	chIn chan<- reparsed,
) error {
	pool := optimizer.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	// Count total name_strings for progress bar
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM name_strings`
	err := pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return NewReparseQueryError(err)
	}

	q := `
SELECT
	id, name, canonical_id, canonical_full_id, canonical_stem_id, bacteria,
	virus, surrogate, parse_quality
FROM name_strings
`
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return NewReparseQueryError(err)
	}
	defer rows.Close()

	var count int
	bar := pb.Full.Start(totalCount)
	bar.Set("prefix", "Processing names: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	// Update progress bar in batches to reduce overhead
	const updateInterval = 10000 // Update every 10k names

	for rows.Next() {
		count++
		res := reparsed{}
		err = rows.Scan(
			&res.nameStringID, &res.name, &res.canonicalID,
			&res.canonicalFullID, &res.canonicalStemID,
			&res.bacteria, &res.virus, &res.surrogate,
			&res.parseQuality,
		)
		if err != nil {
			return NewReparseScanError(err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chIn <- res
		}

		// Update progress bar every updateInterval items
		if count%updateInterval == 0 {
			bar.Add(updateInterval)
		}
	}

	// Add any remaining items to reach the total
	remainder := count % updateInterval
	if remainder > 0 {
		bar.Add(remainder)
	}

	// Check for errors from iteration
	if err := rows.Err(); err != nil {
		return NewReparseIterationError(err)
	}

	return nil
}

// workerReparse is a concurrent worker that parses names using gnparser.
// It receives names from chIn, parses them, generates UUIDs for canonical forms,
// and sends ONLY CHANGED records to chOut (filter-then-batch optimization).
//
// Reference: gnidump workerReparse() in db_reparse.go
func workerReparse(
	ctx context.Context,
	chIn <-chan reparsed,
	chOut chan<- reparsed,
) error {
	prsCfg := gnparser.NewConfig()
	prs := gnparser.New(prsCfg)
	// Process each name from the input channel
	for r := range chIn {
		select {
		case <-ctx.Done():
			// Drain the channel on cancellation
			for range chIn {
			}
			return ctx.Err()
		default:
		}

		parsed := prs.ParseName(r.name)

		// Skip if both old and new parse quality are 0 AND it's not a virus
		// Virus names need to be processed even if unparsed to set the virus flag
		if parsed.ParseQuality+r.parseQuality == 0 && !parsed.Virus {
			continue
		}

		// Handle unparsed names
		// Note: Virus names are often unparsed, but we still need to set the virus flag
		if !parsed.Parsed {
			// Check if virus flag changed - only send if different
			virusChanged := parsed.Virus != r.virus.Bool || (parsed.Virus && !r.virus.Valid)
			if !virusChanged {
				continue
			}

			updated := reparsed{
				nameStringID:    r.nameStringID,
				name:            r.name,
				canonicalID:     newNullStr(""),
				canonicalFullID: newNullStr(""),
				canonicalStemID: newNullStr(""),
				canonical:       "",
				canonicalFull:   "",
				canonicalStem:   "",
				bacteria:        false,
				// Virus flag can be set even if not parsed
				virus:        sql.NullBool{Bool: parsed.Virus, Valid: true},
				surrogate:    sql.NullBool{},
				parseQuality: parsed.ParseQuality,
				cardinality:  sql.NullInt32{},
				year:         sql.NullInt16{},
			}
			chOut <- updated
			continue
		}

		// Generate UUID v5 for canonical forms using gnuuid.New()
		var canonicalID, canonicalFullID, canonicalStemID string
		canonicalID = gnuuid.New(parsed.Canonical.Simple).String()

		// Check if parsing improved - skip if same as before
		// CRITICAL: This filter ensures only CHANGED names are sent to batch processing
		if parsedIsSame(r, parsed, canonicalID) {
			continue
		}

		// Handle canonical full (if different from simple)
		if parsed.Canonical.Simple != parsed.Canonical.Full {
			canonicalFullID = gnuuid.New(parsed.Canonical.Full).String()
		} else {
			parsed.Canonical.Full = "" // Clear if same as simple
		}

		// Generate stemmed canonical UUID
		if parsed.Canonical.Stemmed != "" {
			canonicalStemID = gnuuid.New(parsed.Canonical.Stemmed).String()
		}

		// Extract year from parsed data (from Authorship.Year string field)
		var year sql.NullInt16
		if parsed.Authorship != nil && parsed.Authorship.Year != "" {
			// Year is a string, parse it to int
			// Remove parentheses if present (indicates approximate year)
			yearStr := strings.Trim(parsed.Authorship.Year, "()")
			var yInt int
			if _, err := fmt.Sscanf(yearStr, "%d", &yInt); err == nil {
				year = sql.NullInt16{Int16: int16(yInt), Valid: true}
			}
		}

		// Extract cardinality from parsed data
		var cardinality sql.NullInt32
		if parsed.Cardinality > 0 {
			cardinality = sql.NullInt32{Int32: int32(parsed.Cardinality), Valid: true}
		}

		// Convert bacteria to boolean (if parser gives 0, make it false)
		bacteriaBool := false
		if parsed.Bacteria != nil {
			bacteriaBool = parsed.Bacteria.Bool()
		}

		// Send updated record to save channel (ONLY changed names reach here)
		updated := reparsed{
			nameStringID:    r.nameStringID,
			name:            r.name,
			canonicalID:     newNullStr(canonicalID),
			canonicalFullID: newNullStr(canonicalFullID),
			canonicalStemID: newNullStr(canonicalStemID),
			canonical:       parsed.Canonical.Simple,
			canonicalFull:   parsed.Canonical.Full,
			canonicalStem:   parsed.Canonical.Stemmed,
			bacteria:        bacteriaBool,
			virus:           sql.NullBool{Bool: parsed.Virus, Valid: true},
			surrogate:       sql.NullBool{Bool: parsed.Surrogate != nil, Valid: true},
			parseQuality:    parsed.ParseQuality,
			cardinality:     cardinality,
			year:            year,
		}
		chOut <- updated
	}

	return nil
}

// parsedIsSame checks if the newly parsed result is the same as the existing one.
// This optimization avoids unnecessary database updates.
func parsedIsSame(r reparsed, parsed parsed.Parsed, canonicalID string) bool {
	if r.canonicalID.String != canonicalID {
		return false
	}
	// if parsed as Surrogate, but it is not Surrogate in database
	isNewSurrogate := parsed.Surrogate != nil
	if (isNewSurrogate != r.surrogate.Bool) || (isNewSurrogate && !r.surrogate.Valid) {
		return false
	}

	if r.bacteria != (parsed.Bacteria != nil && parsed.Bacteria.Bool()) {
		return false
	}

	if (parsed.Virus != r.virus.Bool) || (parsed.Virus && !r.virus.Valid) {
		return false
	}

	return true
}

// newNullStr creates a sql.NullString from a string.
// Returns an invalid NullString if the input is empty.
func newNullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// saveBatchedNames receives ONLY changed names from workers and batches them
// for bulk insertion into the temporary table using bulkInsertToTempTable().
func saveBatchedNames(
	ctx context.Context,
	optimizer *OptimizerImpl,
	chOut <-chan reparsed,
	batchSize int,
) error {
	pool := optimizer.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	batch := make([]reparsed, 0, batchSize)
	var totalCount int

	// flushBatch inserts the current batch into the temp table
	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}

		err := bulkInsertToTempTable(ctx, pool, batch)
		if err != nil {
			return err
		}

		totalCount += len(batch)
		batch = batch[:0] // Reset batch slice

		return nil
	}

	// Collect names into batches
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r, ok := <-chOut:
			if !ok {
				// Channel closed, flush remaining batch
				return flushBatch()
			}

			// Add to batch
			batch = append(batch, r)

			// Flush when batch is full
			if len(batch) >= batchSize {
				if err := flushBatch(); err != nil {
					return err
				}
			}
		}
	}
}

// reparseNames orchestrates the name reparsing workflow using filter-then-batch strategy.
// It coordinates four pipeline stages using concurrent goroutines:
// 1. loadNamesForReparse - reads all name_strings from database
// 2. workerReparse - N concurrent workers parse names, filter via parsedIsSame()
// 3. saveBatchedNames - batches ONLY changed names into temp table
// 4. Batch operations - UPDATE name_strings, INSERT canonicals (once at end)
//
// Reference: gnidump reparse() in db_reparse.go + filter-then-batch optimization
func reparseNames(ctx context.Context, optimizer *OptimizerImpl, cfg *config.Config) error {
	pool := optimizer.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	// Step 1: Create temporary table for batch operations
	slog.Debug("Creating temporary table for name processing")
	err := createReparseTempTable(ctx, pool)
	if err != nil {
		return fmt.Errorf("failed to create temp table: %w", err)
	}

	// Ensure temp table is dropped on exit (success or failure)
	defer func() {
		dropCtx := context.Background() // Use background context to ensure cleanup
		_, _ = pool.Exec(dropCtx, "DROP TABLE IF EXISTS temp_reparse_names")
	}()

	// Create channels for pipeline communication
	chIn := make(chan reparsed)
	chOut := make(chan reparsed)

	// Create errgroup for coordinated error handling and context cancellation
	// IMPORTANT: Save original ctx for batch operations after g.Wait()
	// The errgroup context gets canceled when goroutines complete
	g, gCtx := errgroup.WithContext(ctx)

	// Stage 1: Load all name_strings from database
	// Goroutine closes chIn when loading is complete
	g.Go(func() error {
		defer close(chIn)
		return loadNamesForReparse(gCtx, optimizer, chIn)
	})

	// Stage 2: Launch N concurrent workers to parse names
	// Workers filter via parsedIsSame() and send ONLY changed names to chOut
	workerCount := 50

	// Use WaitGroup to track when all workers complete
	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			return workerReparse(gCtx, chIn, chOut)
		})
	}

	// Stage 3: Batch changed names into temporary table
	// Use Optimization.ReparseBatchSize for batch operations (default 50000)
	batchSize := cfg.Optimization.ReparseBatchSize
	if batchSize <= 0 {
		batchSize = 50000 // Fallback default
	}

	g.Go(func() error {
		return saveBatchedNames(gCtx, optimizer, chOut, batchSize)
	})

	// Goroutine to close chOut after all workers finish
	// This signals saveBatchedNames that no more data is coming
	go func() {
		wg.Wait()
		close(chOut)
	}()

	// Wait for all goroutines to complete (loading, parsing, batching)
	// If any goroutine returns an error, context is cancelled and all others stop
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	// Stage 4: Execute batch operations on temp table
	slog.Debug("Executing batch UPDATE on name_strings")
	rowsUpdated, err := batchUpdateNameStrings(ctx, pool)
	if err != nil {
		return fmt.Errorf("failed to batch update name_strings: %w", err)
	}
	msg := "<em>Parsing was identical to the previous one</em>"
	if rowsUpdated > 0 {
		msg = fmt.Sprintf("<em>Updated %s name_strings</em>", humanize.Comma(rowsUpdated))
	}
	fmt.Println(gnlib.FormatMessage(msg, nil))

	slog.Debug("Inserting unique canonicals")
	err = batchInsertCanonicals(ctx, pool)
	if err != nil {
		return fmt.Errorf("failed to batch insert canonicals: %w", err)
	}
	slog.Debug("Canonical forms inserted successfully")

	return nil
}
