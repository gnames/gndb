package iooptimize

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/gnames/gnparser"
	"github.com/gnames/gnparser/ent/parsed"
	"github.com/gnames/gnuuid"
	"golang.org/x/sync/errgroup"
)

// reparsed holds the data for a name_string being reparsed.
type reparsed struct {
	nameStringID    string
	name            string
	canonicalID     sql.NullString
	canonicalFullID sql.NullString
	canonicalStemID sql.NullString
	canonical       string
	canonicalFull   string
	canonicalStem   string
	bacteria        bool
	surrogate       sql.NullBool
	virus           sql.NullBool
	parseQuality    int
	cardinality     sql.NullInt32
	year            sql.NullInt16
}

// loadNamesForReparse loads all name_strings from database for
// reparsing.
func loadNamesForReparse(
	ctx context.Context,
	opt *optimizer,
	chIn chan<- reparsed,
) error {
	pool := opt.operator.Pool()
	if pool == nil {
		return &gn.Error{
			Code: errcode.OptimizerReparseError,
			Msg:  "Database connection lost",
			Err:  fmt.Errorf("pool is nil"),
		}
	}

	// Count total for progress
	var totalCount int
	q := `SELECT COUNT(*) FROM name_strings`
	err := pool.QueryRow(ctx, q).Scan(&totalCount)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerReparseError,
			Msg:  "Failed to count name strings",
			Err:  fmt.Errorf("count query: %w", err),
		}
	}

	q = `
SELECT id, name, canonical_id, canonical_full_id,
       canonical_stem_id, bacteria, virus, surrogate,
       parse_quality
FROM name_strings`

	rows, err := pool.Query(ctx, q)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerReparseError,
			Msg:  "Failed to query name strings",
			Err:  fmt.Errorf("query: %w", err),
		}
	}
	defer rows.Close()

	bar := newProgressBar(totalCount, "Loading names: ")
	defer bar.Finish()

	count := 0
	for rows.Next() {
		var r reparsed
		err = rows.Scan(
			&r.nameStringID, &r.name, &r.canonicalID,
			&r.canonicalFullID, &r.canonicalStemID,
			&r.bacteria, &r.virus, &r.surrogate,
			&r.parseQuality,
		)
		if err != nil {
			return &gn.Error{
				Code: errcode.OptimizerReparseError,
				Msg:  "Failed to scan name string record",
				Err:  fmt.Errorf("scan: %w", err),
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case chIn <- r:
			count++
			if count%1000 == 0 {
				bar.Add(1000)
			}
		}
	}

	// Add remainder
	if count%1000 > 0 {
		bar.Add(count % 1000)
	}

	return rows.Err()
}

// workerReparse parses names and sends only changed records.
func workerReparse(
	ctx context.Context,
	chIn <-chan reparsed,
	chOut chan<- reparsed,
) error {
	prsCfg := gnparser.NewConfig()
	prs := gnparser.New(prsCfg)

	for r := range chIn {
		select {
		case <-ctx.Done():
			for range chIn {
			}
			return ctx.Err()
		default:
		}

		parsed := prs.ParseName(r.name)

		// Skip if both old and new are unparsed and not virus
		if parsed.ParseQuality+r.parseQuality == 0 &&
			!parsed.Virus {
			continue
		}

		// Handle unparsed names
		if !parsed.Parsed {
			virusChanged := parsed.Virus != r.virus.Bool ||
				(parsed.Virus && !r.virus.Valid)
			if !virusChanged {
				continue
			}

			updated := reparsed{
				nameStringID: r.nameStringID,
				virus: sql.NullBool{
					Bool:  parsed.Virus,
					Valid: true,
				},
				parseQuality:    parsed.ParseQuality,
				canonicalID:     sql.NullString{},
				canonicalFullID: sql.NullString{},
				canonicalStemID: sql.NullString{},
			}
			chOut <- updated
			continue
		}

		// Generate canonical UUIDs
		canonicalID := gnuuid.New(
			parsed.Canonical.Simple,
		).String()

		// Check if same as before
		if parsedIsSame(r, parsed, canonicalID) {
			continue
		}

		// Handle canonical full
		var canonicalFullID string
		canonicalFull := parsed.Canonical.Full
		if parsed.Canonical.Simple != parsed.Canonical.Full {
			canonicalFullID = gnuuid.New(
				parsed.Canonical.Full,
			).String()
		} else {
			canonicalFull = ""
		}

		// Handle canonical stem
		var canonicalStemID string
		if parsed.Canonical.Stemmed != "" {
			canonicalStemID = gnuuid.New(
				parsed.Canonical.Stemmed,
			).String()
		}

		// Extract year
		var year sql.NullInt16
		if parsed.Authorship != nil &&
			parsed.Authorship.Year != "" {
			yearStr := strings.Trim(
				parsed.Authorship.Year,
				"()",
			)
			var yInt int
			_, err := fmt.Sscanf(yearStr, "%d", &yInt)
			if err == nil {
				year = sql.NullInt16{
					Int16: int16(yInt),
					Valid: true,
				}
			}
		}

		// Extract cardinality
		var cardinality sql.NullInt32
		if parsed.Cardinality > 0 {
			cardinality = sql.NullInt32{
				Int32: int32(parsed.Cardinality),
				Valid: true,
			}
		}

		// Bacteria flag
		bacteriaBool := false
		if parsed.Bacteria != nil {
			bacteriaBool = parsed.Bacteria.Bool()
		}

		updated := reparsed{
			nameStringID:    r.nameStringID,
			name:            r.name,
			canonicalID:     newNullStr(canonicalID),
			canonicalFullID: newNullStr(canonicalFullID),
			canonicalStemID: newNullStr(canonicalStemID),
			canonical:       parsed.Canonical.Simple,
			canonicalFull:   canonicalFull,
			canonicalStem:   parsed.Canonical.Stemmed,
			bacteria:        bacteriaBool,
			virus: sql.NullBool{
				Bool:  parsed.Virus,
				Valid: true,
			},
			surrogate: sql.NullBool{
				Bool:  parsed.Surrogate != nil,
				Valid: true,
			},
			parseQuality: parsed.ParseQuality,
			cardinality:  cardinality,
			year:         year,
		}
		chOut <- updated
	}

	return nil
}

// parsedIsSame checks if parsing result unchanged.
func parsedIsSame(
	r reparsed,
	parsed parsed.Parsed,
	canonicalID string,
) bool {
	if r.canonicalID.String != canonicalID {
		return false
	}

	isNewSurrogate := parsed.Surrogate != nil
	if (isNewSurrogate != r.surrogate.Bool) ||
		(isNewSurrogate && !r.surrogate.Valid) {
		return false
	}

	if r.bacteria != (parsed.Bacteria != nil &&
		parsed.Bacteria.Bool()) {
		return false
	}

	if (parsed.Virus != r.virus.Bool) ||
		(parsed.Virus && !r.virus.Valid) {
		return false
	}

	return true
}

// newNullStr creates a sql.NullString.
func newNullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// saveBatchedNames collects changed names and batches them.
func saveBatchedNames(
	ctx context.Context,
	opt *optimizer,
	chOut <-chan reparsed,
	batchSize int,
) error {
	pool := opt.operator.Pool()
	if pool == nil {
		return &gn.Error{
			Code: errcode.OptimizerReparseError,
			Msg:  "Database connection lost",
			Err:  fmt.Errorf("pool is nil"),
		}
	}

	batch := make([]reparsed, 0, batchSize)

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := bulkInsertToTempTable(ctx, pool, batch)
		if err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r, ok := <-chOut:
			if !ok {
				return flushBatch()
			}

			batch = append(batch, r)
			if len(batch) >= batchSize {
				if err := flushBatch(); err != nil {
					return err
				}
			}
		}
	}
}

// reparseNames orchestrates the name reparsing workflow.
func reparseNames(
	ctx context.Context,
	opt *optimizer,
) (string, error) {
	pool := opt.operator.Pool()
	if pool == nil {
		return "", &gn.Error{
			Code: errcode.OptimizerReparseError,
			Msg:  "Database connection lost",
			Err:  fmt.Errorf("pool is nil"),
		}
	}

	// Create temp table
	slog.Info("Creating temporary table for name processing")
	err := createReparseTempTable(ctx, pool)
	if err != nil {
		return "", err
	}

	defer func() {
		dropCtx := context.Background()
		q := "DROP TABLE IF EXISTS temp_reparse_names"
		_, _ = pool.Exec(dropCtx, q)
	}()

	// Create channels
	chIn := make(chan reparsed)
	chOut := make(chan reparsed)

	// Create errgroup
	g, gCtx := errgroup.WithContext(ctx)

	// Stage 1: Load names
	g.Go(func() error {
		defer close(chIn)
		return loadNamesForReparse(gCtx, opt, chIn)
	})

	// Stage 2: Parse with workers
	workerCount := 50
	batchSize := 50_000 // Hardcoded

	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			return workerReparse(gCtx, chIn, chOut)
		})
	}

	// Stage 3: Batch saves
	g.Go(func() error {
		return saveBatchedNames(gCtx, opt, chOut, batchSize)
	})

	// Close chOut when workers done
	go func() {
		wg.Wait()
		close(chOut)
	}()

	// Wait for pipeline
	if err := g.Wait(); err != nil &&
		!errors.Is(err, context.Canceled) {
		return "", &gn.Error{
			Code: errcode.OptimizerReparseError,
			Msg:  "Failed to reparse name strings",
			Err:  fmt.Errorf("pipeline: %w", err),
		}
	}

	// Stage 4: Batch operations
	slog.Info("Executing batch UPDATE on name_strings")
	rowsUpdated, err := batchUpdateNameStrings(ctx, pool)
	if err != nil {
		return "", err
	}

	msg := "<em>Parsing was identical to the previous one</em>"
	if rowsUpdated > 0 {
		msg = fmt.Sprintf(
			"<em>Updated %s name_strings</em>",
			humanize.Comma(rowsUpdated),
		)
	}
	slog.Info("Inserting unique canonicals")
	err = batchInsertCanonicals(ctx, pool)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
