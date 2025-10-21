package iooptimize

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/parserpool"
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
	timeStart := time.Now().UnixNano()

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

		// Progress tracking: log every 100,000 names
		if count%100_000 == 0 {
			timeSpent := float64(time.Now().UnixNano()-timeStart) / 1_000_000_000
			speed := int64(float64(count) / timeSpent)
			fmt.Fprintf(os.Stderr, "\r%s", strings.Repeat(" ", 40))
			fmt.Fprintf(os.Stderr, "\rLoaded %s names, %s names/sec",
				humanize.Comma(int64(count)), humanize.Comma(speed))
		}
	}

	// Check for errors from iteration
	if err := rows.Err(); err != nil {
		return NewReparseIterationError(err)
	}

	// Clear progress line
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 40))
	return nil
}

// workerReparse is a concurrent worker that parses names using gnparser.
// It receives names from chIn, parses them, generates UUIDs for canonical forms,
// stores results in cache, and sends updated records to chOut.
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

		// Send updated record to save channel
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

// saveReparsedNames receives reparsed name data from the output channel
// and saves it back to the database using updateNameString.
// Progress is tracked and logged to stderr.
//
// Reference: gnidump saveReparse() in db_reparse.go
func saveReparsedNames(
	ctx context.Context,
	optimizer *OptimizerImpl,
	chOut <-chan reparsed,
) error {
	var count int
	timeStart := time.Now().UnixNano()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r, ok := <-chOut:
			if !ok {
				// Channel closed, we're done
				fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 40))
				return nil
			}

			// Update the name_string record in database
			err := updateNameString(ctx, optimizer, r)
			if err != nil {
				return err
			}

			count++

			// Progress tracking: log every 100,000 updates
			if count%100_000 == 0 {
				timeSpent := float64(time.Now().UnixNano()-timeStart) / 1_000_000_000
				speed := int64(float64(count) / timeSpent)
				fmt.Fprintf(os.Stderr, "\r%s", strings.Repeat(" ", 40))
				fmt.Fprintf(os.Stderr, "\rSaved %s names, %s names/sec",
					humanize.Comma(int64(count)), humanize.Comma(speed))
			}
		}
	}
}

// updateNameString updates a single name_string record and inserts canonical records.
// All operations are performed within a transaction for atomicity.
//
// Reference: gnidump updateNameString() in db_reparse.go
func updateNameString(ctx context.Context, optimizer *OptimizerImpl, r reparsed) error {
	pool := optimizer.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return NewReparseTransactionError(err)
	}
	defer func() {
		_ = tx.Rollback(
			ctx,
		) // Rollback in case of any error; ignore error as commit handles success
	}()

	// Update name_strings table with new canonical IDs, flags, year, and cardinality
	_, err = tx.Exec(ctx, `
		UPDATE name_strings
		SET
			canonical_id = $1, canonical_full_id = $2, canonical_stem_id = $3,
			bacteria = $4, virus = $5, surrogate = $6, parse_quality = $7,
			cardinality = $8, year = $9
		WHERE id = $10`,
		r.canonicalID, r.canonicalFullID, r.canonicalStemID,
		r.bacteria, r.virus, r.surrogate, r.parseQuality,
		r.cardinality, r.year, r.nameStringID,
	)
	if err != nil {
		return NewReparseUpdateError(err)
	}

	// If parse quality is 0, the name is unparseable - no canonicals to insert
	if r.parseQuality == 0 {
		return tx.Commit(ctx)
	}

	// Insert canonical form (ON CONFLICT DO NOTHING for idempotency)
	if r.canonical != "" {
		_, err = tx.Exec(ctx, `
			INSERT INTO canonicals (id, name)
			VALUES ($1, $2)
			ON CONFLICT (id) DO NOTHING`,
			r.canonicalID, r.canonical)
		if err != nil {
			return NewReparseInsertError("canonicals", err)
		}
	}

	// Insert canonical stem (ON CONFLICT DO NOTHING for idempotency)
	if r.canonicalStem != "" {
		_, err = tx.Exec(ctx, `
			INSERT INTO canonical_stems (id, name)
			VALUES ($1, $2)
			ON CONFLICT (id) DO NOTHING`,
			r.canonicalStemID, r.canonicalStem)
		if err != nil {
			return NewReparseInsertError("canonical_stems", err)
		}
	}

	// Insert canonical full (only if different from simple canonical)
	if r.canonicalFull != "" {
		_, err = tx.Exec(ctx, `
			INSERT INTO canonical_fulls (id, name)
			VALUES ($1, $2)
			ON CONFLICT (id) DO NOTHING`,
			r.canonicalFullID, r.canonicalFull)
		if err != nil {
			return NewReparseInsertError("canonical_fulls", err)
		}
	}

	// Commit the transaction if all operations were successful
	return tx.Commit(ctx)
}

// reparseNames orchestrates the name reparsing workflow.
// It coordinates three pipeline stages using concurrent goroutines:
// 1. loadNamesForReparse - reads all name_strings from database
// 2. workerReparse - N concurrent workers parse names using gnparser
// 3. saveReparsedNames - saves parsed results back to database
//
// Reference: gnidump reparse() in db_reparse.go
func reparseNames(ctx context.Context, optimizer *OptimizerImpl, cfg *config.Config) error {
	// Create channels for pipeline communication
	chIn := make(chan reparsed)
	chOut := make(chan reparsed)

	// Create errgroup for coordinated error handling and context cancellation
	g, ctx := errgroup.WithContext(ctx)

	// Stage 1: Load all name_strings from database
	// Goroutine closes chIn when loading is complete
	g.Go(func() error {
		defer close(chIn)
		return loadNamesForReparse(ctx, optimizer, chIn)
	})

	// Stage 2: Launch N concurrent workers to parse names
	// Use Config.JobsNumber for worker count (gnidump uses hardcoded 50)
	workerCount := cfg.JobsNumber
	if workerCount <= 0 {
		workerCount = 1 // Minimum 1 worker
	}

	// Create parser pool for workers to share
	parserPool := parserpool.NewPool(workerCount)
	defer parserPool.Close()

	// Use WaitGroup to track when all workers complete
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			return workerReparse(ctx, chIn, chOut)
		})
	}

	// Stage 3: Save reparsed results to database
	g.Go(func() error {
		return saveReparsedNames(ctx, optimizer, chOut)
	})

	// Goroutine to close chOut after all workers finish
	// This signals saveReparsedNames that no more data is coming
	go func() {
		wg.Wait()
		close(chOut)
	}()

	// Wait for all goroutines to complete
	// If any goroutine returns an error, context is cancelled and all others stop
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
