package iooptimize

import (
	"context"
	"fmt"
	"log/slog"

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
//  3. Extract words from cached parse results (no re-parsing)
//  4. Deduplicate words
//  5. Bulk insert words
//  6. Bulk insert word-name-string linkages
//
// Reference: gnidump createWords() in words.go
func createWords(ctx context.Context, o *OptimizerImpl, cfg *config.Config) error {
	return fmt.Errorf("createWords is not yet implemented")
}

// truncateWordsTables clears the words and word_name_strings tables.
// This ensures a clean slate before populating word data.
//
// Reference: gnidump truncateTable() in db.go
//
//nolint:unused // Will be used in createWords orchestrator (T030)
func truncateWordsTables(ctx context.Context, conn *pgx.Conn) error {
	tables := []string{"words", "word_name_strings"}

	for _, table := range tables {
		sql := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		_, err := conn.Exec(ctx, sql)
		if err != nil {
			slog.Error("Cannot truncate table", "table", table, "error", err)
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
		slog.Info("Truncated table", "table", table)
	}

	return nil
}

// getNameStringsForWords queries all name_strings with canonical_id for word extraction.
// Only names with canonical forms are used for word extraction.
//
// Reference: gnidump getWordNames() in db.go
//
//nolint:unused // Will be used in createWords orchestrator (T030)
func getNameStringsForWords(ctx context.Context, conn *pgx.Conn) ([]nameForWords, error) {
	query := `
		SELECT id, name, canonical_id
		FROM name_strings
		WHERE canonical_id IS NOT NULL
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		slog.Error("Cannot query name_strings for word extraction", "error", err)
		return nil, fmt.Errorf("failed to query name_strings: %w", err)
	}
	defer rows.Close()

	var names []nameForWords
	for rows.Next() {
		var n nameForWords
		if err := rows.Scan(&n.ID, &n.Name, &n.CanonicalID); err != nil {
			slog.Error("Cannot scan name_string row", "error", err)
			return nil, fmt.Errorf("failed to scan name_string: %w", err)
		}
		names = append(names, n)
	}

	if err := rows.Err(); err != nil {
		slog.Error("Error iterating name_string rows", "error", err)
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	slog.Info("Loaded name_strings for word extraction", "count", len(names))
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
//
//nolint:unused // Will be used in createWords orchestrator (T030)
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

	for i := 0; i < len(names); i += batchSize {
		end := i + batchSize
		if end > len(names) {
			end = len(names)
		}
		batch := names[i:end]

		// Process batch concurrently using errgroup
		batchWords, batchWordNames, err := processBatchConcurrent(ctx, pool, batch, cfg.JobsNumber)
		if err != nil {
			return nil, nil, err
		}

		words = append(words, batchWords...)
		wordNames = append(wordNames, batchWordNames...)

		if (i+batchSize)%100000 == 0 {
			slog.Info("Parsed names for words", "count", i+batchSize)
		}
	}

	slog.Info("Completed parsing names for words", "totalWords", len(words), "totalLinks", len(wordNames))
	return words, wordNames, nil
}

// processBatchConcurrent processes a batch of names concurrently using multiple workers.
// Each worker parses names and extracts words.
//
//nolint:unused // Will be used in createWords orchestrator (T030)
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
//
//nolint:unused // Will be used by processBatchConcurrent (T030)
func processChunk(pool parserpool.Pool, chunk []nameForWords) ([]schema.Word, []schema.WordNameString) {
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
