package iooptimize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cheggaaa/pb/v3"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/parserpool"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/gnames/gnlib/ent/nomcode"
	"github.com/gnames/gnparser/ent/parsed"
	"github.com/gnames/gnuuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

// nameForWords holds name_string data needed for word extraction.
type nameForWords struct {
	ID          string
	Name        string // The actual name string to parse
	CanonicalID string
}

// createWords orchestrates the word extraction and insertion process.
// This is Step 4 of the optimization workflow from gnidump.
//
// Workflow:
//  1. Truncate words and word_name_strings tables
//  2. Load all name_strings with canonical_id
//  3. Parse names and extract words (direct parsing, no cache)
//  4. Deduplicate words and word-name linkages
//  5. Bulk insert words
//  6. Bulk insert word-name-string linkages
//
// Reference: gnidump createWords() in words.go
func createWords(ctx context.Context, o *OptimizerImpl, cfg *config.Config) error {
	slog.Debug("Creating words for words tables")

	// Get database connection
	pool := o.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Step 1: Truncate words tables
	if err := truncateWordsTables(ctx, conn.Conn()); err != nil {
		return err
	}

	// Step 2: Load all name_strings for word extraction
	names, err := getNameStringsForWords(ctx, conn.Conn())
	if err != nil {
		return err
	}

	if len(names) == 0 {
		slog.Info("No names to process for word extraction")
		return nil
	}

	// Step 3: Create parserpool for concurrent parsing
	workerCount := cfg.JobsNumber
	if workerCount == 0 {
		workerCount = 1
	}
	parserPool := parserpool.NewPool(workerCount)
	defer parserPool.Close()

	// Parse names and extract words using parserpool
	slog.Debug("Parsing names and extracting words", "totalNames", len(names))
	words, wordNames, err := parseNamesForWords(ctx, parserPool, names, cfg)
	if err != nil {
		return err
	}

	// Step 4: Deduplicate words and word-name linkages
	uniqueWords := deduplicateWords(words)
	uniqueWordNames := deduplicateWordNameStrings(wordNames)

	// Step 5: Bulk insert words
	slog.Debug("Saving words to database")
	if err := saveWords(ctx, conn.Conn(), uniqueWords, cfg); err != nil {
		return err
	}

	// Step 6: Bulk insert word-name linkages
	slog.Debug("Saving word-name linkages to database")
	if err := saveWordNameStrings(ctx, conn.Conn(), uniqueWordNames, cfg); err != nil {
		return err
	}

	slog.Debug("Completed words creation",
		"totalWords", len(uniqueWords),
		"totalLinks", len(uniqueWordNames))

	return nil
}

// truncateWordsTables clears the words and word_name_strings tables.
// This ensures a clean slate before populating word data.
//
// Reference: gnidump truncateTable() in db.go
func truncateWordsTables(ctx context.Context, conn *pgx.Conn) error {
	tables := []string{"words", "word_name_strings"}

	for _, table := range tables {
		sql := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		_, err := conn.Exec(ctx, sql)
		if err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
		slog.Debug("Truncated table", "table", table)
	}

	return nil
}

// getNameStringsForWords queries all name_strings with canonical_id for word extraction.
// Only names with canonical forms are used for word extraction.
//
// Reference: gnidump getWordNames() in db.go
func getNameStringsForWords(ctx context.Context, conn *pgx.Conn) ([]nameForWords, error) {
	// First, count total names for progress tracking
	var totalCount int
	countQuery := `
		SELECT COUNT(*)
		FROM name_strings
		WHERE canonical_id IS NOT NULL
	`
	err := conn.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count name_strings: %w", err)
	}

	query := `
		SELECT id, name, canonical_id
		FROM name_strings
		WHERE canonical_id IS NOT NULL
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query name_strings: %w", err)
	}
	defer rows.Close()

	var names []nameForWords
	for rows.Next() {
		var n nameForWords
		if err := rows.Scan(&n.ID, &n.Name, &n.CanonicalID); err != nil {
			return nil, fmt.Errorf("failed to scan name_string: %w", err)
		}
		names = append(names, n)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	slog.Debug("Loaded name_strings for word extraction", "count", len(names))
	return names, nil
}

// parseNamesForWords parses a batch of names and extracts words for fuzzy matching.
// This follows the gnidump processParsedWords pattern but uses parserpool for concurrency.
//
// The parserpool already handles concurrency (JobsNumber parsers in pool), so this function
// processes names sequentially, relying on the pool's internal concurrency.
//
// For each name:
//   - Parse using parserpool to get Words field (WithDetails must be true)
//   - Skip unparsed names, surrogates, and hybrids
//   - Extract words of types: SpEpithetType, InfraspEpithetType, AuthorWordType
//   - Generate word ID from modified form and type: UUID5(modified|typeID)
//   - Create Word and WordNameString records
//
// Reference: gnidump processParsedWords() in words.go
func parseNamesForWords(
	ctx context.Context,
	pool parserpool.Pool,
	names []nameForWords,
	cfg *config.Config,
) ([]schema.Word, []schema.WordNameString, error) {
	wordNames := make([]schema.WordNameString, 0, len(names)*5)
	words := make([]schema.Word, 0, len(names)*5)

	// Process in batches for better memory management
	batchSize := cfg.Database.BatchSize
	if batchSize == 0 {
		batchSize = 10000 // Default batch size
	}

	// Create progress bar with known total
	bar := pb.Full.Start(len(names))
	bar.Set("prefix", "Processing names for words: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	for i := 0; i < len(names); i += batchSize {
		end := min(i+batchSize, len(names))
		batch := names[i:end]

		// Process batch concurrently using errgroup
		batchWords, batchWordNames, err := processBatchConcurrent(ctx, pool, batch, cfg.JobsNumber)
		if err != nil {
			return nil, nil, err
		}

		words = append(words, batchWords...)
		wordNames = append(wordNames, batchWordNames...)

		// Update progress bar
		bar.Add(len(batch))
	}

	slog.Debug(
		"Completed parsing names for words",
		"totalWords",
		len(words),
		"totalLinks",
		len(wordNames),
	)
	return words, wordNames, nil
}

// processBatchConcurrent processes a batch of names concurrently using multiple workers.
// Each worker parses names and extracts words.
func processBatchConcurrent(
	ctx context.Context,
	pool parserpool.Pool,
	batch []nameForWords,
	jobsNum int,
) ([]schema.Word, []schema.WordNameString, error) {
	if jobsNum == 0 {
		jobsNum = 1 // At least 1 worker
	}

	// Divide batch into chunks for workers
	chunkSize := len(batch) / jobsNum
	if chunkSize == 0 {
		chunkSize = 1
	}

	// Channels to collect results
	type result struct {
		words     []schema.Word
		wordNames []schema.WordNameString
	}
	resultsCh := make(chan result, jobsNum)

	g, ctx := errgroup.WithContext(ctx)

	// Launch workers
	for i := 0; i < jobsNum; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == jobsNum-1 {
			end = len(batch) // Last worker takes remainder
		}
		if start >= len(batch) {
			break
		}

		chunk := batch[start:end]
		g.Go(func() error {
			words, wordNames := processChunk(pool, chunk)
			select {
			case resultsCh <- result{words: words, wordNames: wordNames}:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}

	// Wait for all workers
	if err := g.Wait(); err != nil {
		close(resultsCh)
		return nil, nil, err
	}
	close(resultsCh)

	// Collect results
	var allWords []schema.Word
	var allWordNames []schema.WordNameString
	for r := range resultsCh {
		allWords = append(allWords, r.words...)
		allWordNames = append(allWordNames, r.wordNames...)
	}

	return allWords, allWordNames, nil
}

// processChunk processes a chunk of names and extracts words (worker function).
func processChunk(
	pool parserpool.Pool,
	chunk []nameForWords,
) ([]schema.Word, []schema.WordNameString) {
	words := make([]schema.Word, 0, len(chunk)*5)
	wordNames := make([]schema.WordNameString, 0, len(chunk)*5)

	for _, n := range chunk {
		// Parse the name using parserpool (Botanical code by default)
		p, err := pool.Parse(n.Name, nomcode.Botanical)
		if err != nil {
			slog.Warn("Failed to parse name", "name", n.Name, "error", err)
			continue
		}

		// Skip unparsed names, surrogates, and hybrids (following gnidump logic)
		if !p.Parsed || p.Surrogate != nil || p.Hybrid != nil {
			continue
		}

		// Extract words from parsed result
		for _, w := range p.Words {
			wt := w.Type
			// Generate modified form using gnparser's NormalizeByType
			mod := parsed.NormalizeByType(w.Normalized, wt)
			// Generate word ID from modified|typeID (matching gnidump)
			idstr := fmt.Sprintf("%s|%d", mod, int(wt))
			wordID := gnuuid.New(idstr).String()

			// Create WordNameString junction record
			wns := schema.WordNameString{
				WordID:       wordID,
				NameStringID: n.ID,
				CanonicalID:  n.CanonicalID,
			}

			// Only include specific word types (following gnidump filter)
			switch wt {
			case
				parsed.SpEpithetType,
				parsed.InfraspEpithetType,
				parsed.AuthorWordType:
				// Create Word record
				word := schema.Word{
					ID:         wordID,
					Normalized: w.Normalized,
					Modified:   mod,
					TypeID:     int(wt),
				}
				words = append(words, word)
				wordNames = append(wordNames, wns)
			}
		}
	}

	return words, wordNames
}

// deduplicateWords removes duplicate word entries using map-based deduplication.
// The key is composed of ID|Normalized to ensure uniqueness.
// This matches the gnidump pattern where words are deduplicated in the wordsMap.
//
// Reference: gnidump createWords() wordsMap building
func deduplicateWords(words []schema.Word) []schema.Word {
	// Use map with composite key: ID|Normalized (matching gnidump)
	wordsMap := make(map[string]schema.Word)

	for _, w := range words {
		key := w.ID + "|" + w.Normalized
		wordsMap[key] = w
	}

	// Convert map back to slice
	uniqueWords := make([]schema.Word, 0, len(wordsMap))
	for _, w := range wordsMap {
		uniqueWords = append(uniqueWords, w)
	}

	slog.Debug("Deduplicated words", "original", len(words), "unique", len(uniqueWords))
	return uniqueWords
}

// deduplicateWordNameStrings removes duplicate word-name-string linkages.
// The key is composed of WordID|NameStringID to ensure uniqueness.
// This matches the gnidump uniqWordNameString pattern.
//
// Reference: gnidump uniqWordNameString() in words.go
func deduplicateWordNameStrings(wordNames []schema.WordNameString) []schema.WordNameString {
	// Use map with composite key: WordID|NameStringID (matching gnidump)
	wnsMap := make(map[string]schema.WordNameString)

	for _, wn := range wordNames {
		key := wn.WordID + "|" + wn.NameStringID
		wnsMap[key] = wn
	}

	// Convert map back to slice
	uniqueWordNames := make([]schema.WordNameString, 0, len(wnsMap))
	for _, wn := range wnsMap {
		uniqueWordNames = append(uniqueWordNames, wn)
	}

	slog.Debug(
		"Deduplicated word-name links",
		"original",
		len(wordNames),
		"unique",
		len(uniqueWordNames),
	)
	return uniqueWordNames
}

// saveWords performs bulk insert of words into the database using pgx.CopyFrom.
// Words are processed in batches according to Config.Database.BatchSize.
//
// Reference: gnidump saveWords() in db.go using insertRows()
func saveWords(ctx context.Context, conn *pgx.Conn, words []schema.Word, cfg *config.Config) error {
	if len(words) == 0 {
		slog.Info("No words to save")
		return nil
	}

	batchSize := cfg.Database.BatchSize
	if batchSize == 0 {
		batchSize = 50000 // Default batch size
	}

	columns := []string{"id", "normalized", "modified", "type_id"}
	totalSaved := 0

	// Create progress bar for saving words
	bar := pb.Full.Start(len(words))
	bar.Set("prefix", "Saving words: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	for i := 0; i < len(words); i += batchSize {

		end := min(i+batchSize, len(words))
		batch := words[i:end]

		// Prepare rows for CopyFrom
		rows := make([][]any, len(batch))
		for j, w := range batch {
			rows[j] = []any{w.ID, w.Normalized, w.Modified, w.TypeID}
		}

		// Bulk insert using CopyFrom
		copyCount, err := conn.CopyFrom(
			ctx,
			pgx.Identifier{"words"},
			columns,
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("failed to save words batch: %w", err)
		}

		totalSaved += int(copyCount)

		// Update progress bar
		bar.Add(len(batch))
	}

	slog.Debug("Completed saving words", "total", totalSaved)
	return nil
}

// saveWordNameStrings performs bulk insert of word-name-string linkages using pgx.CopyFrom.
// Linkages are processed in batches according to Config.Database.BatchSize.
//
// Reference: gnidump saveNameWords() in db.go using insertRows()
func saveWordNameStrings(
	ctx context.Context,
	conn *pgx.Conn,
	wordNames []schema.WordNameString,
	cfg *config.Config,
) error {
	if len(wordNames) == 0 {
		slog.Info("No word-name linkages to save")
		return nil
	}

	batchSize := cfg.Database.BatchSize
	if batchSize == 0 {
		batchSize = 50000 // Default batch size
	}

	columns := []string{"word_id", "name_string_id", "canonical_id"}
	totalSaved := 0

	// Create progress bar for saving word-name linkages
	bar := pb.Full.Start(len(wordNames))
	bar.Set("prefix", "Saving word-name linkages: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	for i := 0; i < len(wordNames); i += batchSize {
		end := min(i+batchSize, len(wordNames))
		batch := wordNames[i:end]

		// Prepare rows for CopyFrom
		rows := make([][]any, len(batch))
		for j, wn := range batch {
			rows[j] = []any{wn.WordID, wn.NameStringID, wn.CanonicalID}
		}

		// Bulk insert using CopyFrom
		copyCount, err := conn.CopyFrom(
			ctx,
			pgx.Identifier{"word_name_strings"},
			columns,
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("failed to save word-name linkages batch: %w", err)
		}

		totalSaved += int(copyCount)

		// Update progress bar
		bar.Add(len(batch))
	}

	slog.Debug("Completed saving word-name linkages", "total", totalSaved)
	return nil
}
