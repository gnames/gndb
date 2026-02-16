package iooptimize

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/gnames/gnparser"
	"github.com/gnames/gnparser/ent/parsed"
	"github.com/gnames/gnuuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

// nameForWords holds name_string data needed for word extraction.
type nameForWords struct {
	id          string
	name        string
	canonicalID string
}

// wordResult holds results from word extraction workers.
type wordResult struct {
	words     []schema.Word
	wordNames []schema.WordNameString
}

// extractWords orchestrates the word extraction and insertion
// process. This is Step 4 of the optimization workflow from
// gnidump.
//
// Workflow:
//  1. Truncate words and word_name_strings tables
//  2. Stream name_strings with canonical_id from database
//  3. Parse names and extract words (concurrent processing)
//  4. Deduplicate words globally, save word-name links
//     in batches during streaming
//  5. Bulk insert words after pipeline completes
//
// Reference: gnidump createWords() in words.go
func extractWords(
	ctx context.Context,
	opt *optimizer,
	cfg *config.Config,
) error {
	var msg string
	pool := opt.operator.Pool()
	if pool == nil {
		return &gn.Error{
			Code: errcode.OptimizerWordExtractionError,
			Msg:  "Database connection lost",
			Err:  fmt.Errorf("pool is nil"),
		}
	}

	msg = "Creating words for fuzzy matching"
	slog.Info(msg)

	// Step 1: Truncate words tables.
	if err := truncateWordsTables(ctx, pool); err != nil {
		return err
	}

	// Steps 2-4: Stream, parse, deduplicate words, and save
	// word-name linkages in batches during streaming.
	slog.Info("Streaming names and extracting words")
	wordsMap, totalLinks, err := parseNamesForWords(
		ctx, pool, cfg,
	)
	if err != nil {
		return err
	}

	if len(wordsMap) == 0 {
		slog.Info("No names to process for word extraction")
		gn.Info(
			"<em>No names found for word extraction</em>",
		)
		return nil
	}

	// Convert words map to slice for saving.
	uniqueWords := make(
		[]schema.Word, 0, len(wordsMap),
	)
	for _, w := range wordsMap {
		uniqueWords = append(uniqueWords, w)
	}

	// Step 5: Bulk insert words.
	slog.Info("Saving words to database")
	if err := saveWords(
		ctx, pool, uniqueWords, cfg,
	); err != nil {
		return err
	}

	slog.Info(
		"Completed words creation",
		"totalWords", len(uniqueWords),
		"totalLinks", totalLinks,
	)

	// Report stats.
	msg = fmt.Sprintf(
		"<em>Created %s words and %s word linkages</em>",
		humanize.Comma(int64(len(uniqueWords))),
		humanize.Comma(int64(totalLinks)),
	)
	gn.Info(msg)

	return nil
}

// truncateWordsTables clears the words and word_name_strings
// tables. This ensures a clean slate before populating word data.
//
// Reference: gnidump truncateTable() in db.go
func truncateWordsTables(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	tables := []string{"words", "word_name_strings"}

	for _, table := range tables {
		sql := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		_, err := pool.Exec(ctx, sql)
		if err != nil {
			return &gn.Error{
				Code: errcode.OptimizerWordExtractionError,
				Msg:  fmt.Sprintf("Failed to truncate %s", table),
				Err:  fmt.Errorf("truncate %s: %w", table, err),
			}
		}
		slog.Info("Truncated table", "table", table)
	}

	return nil
}

// loadNamesForWords streams name_strings with canonical_id
// directly to a channel, avoiding loading all records into
// memory.
//
// Reference: gnidump getWordNames() in db.go
func loadNamesForWords(
	ctx context.Context,
	pool *pgxpool.Pool,
	chIn chan<- nameForWords,
) error {
	// Count total names for progress bar.
	var totalCount int
	countQuery := `
SELECT COUNT(*)
FROM name_strings
WHERE canonical_id IS NOT NULL`

	err := pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerWordExtractionError,
			Msg:  "Failed to count names for word extraction",
			Err:  fmt.Errorf("count query: %w", err),
		}
	}

	query := `
SELECT id, name, canonical_id
FROM name_strings
WHERE canonical_id IS NOT NULL`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerWordExtractionError,
			Msg:  "Failed to load names for word extraction",
			Err:  fmt.Errorf("query: %w", err),
		}
	}
	defer rows.Close()

	bar := newProgressBar(totalCount, "Extracting names' words: ")
	defer bar.Finish()

	count := 0
	for rows.Next() {
		var n nameForWords
		err := rows.Scan(&n.id, &n.name, &n.canonicalID)
		if err != nil {
			return &gn.Error{
				Code: errcode.OptimizerWordExtractionError,
				Msg:  "Failed to scan name record",
				Err:  fmt.Errorf("scan: %w", err),
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case chIn <- n:
			count++
			if count%1000 == 0 {
				bar.Add(1000)
			}
		}
	}

	// Add remainder.
	if count%1000 > 0 {
		bar.Add(count % 1000)
	}

	if err := rows.Err(); err != nil {
		return &gn.Error{
			Code: errcode.OptimizerWordExtractionError,
			Msg:  "Failed to iterate name records",
			Err:  fmt.Errorf("rows error: %w", err),
		}
	}

	slog.Info(
		"Loaded names for word extraction",
		"count", count,
	)
	return nil
}

// parseNamesForWords streams names from the database, parses
// them with concurrent workers, deduplicates words globally,
// and saves word-name linkages in batches during streaming.
// Returns deduplicated words map and total linkages saved.
//
// Pipeline:
//
//	Stage 1: loadNamesForWords streams rows → chIn
//	Stage 2: Workers parse and extract words → chOut
//	Stage 3: Collector deduplicates words into map, saves
//	         word-name linkages in batches to database
func parseNamesForWords(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *config.Config,
) (map[string]schema.Word, int, error) {
	jobsNum := cfg.JobsNumber
	if jobsNum == 0 {
		jobsNum = 20
	}

	batchSize := cfg.Database.BatchSize
	if batchSize == 0 {
		batchSize = 50000
	}

	// Create channels.
	chIn := make(chan nameForWords)
	chOut := make(chan wordResult)

	// Create errgroup.
	g, gCtx := errgroup.WithContext(ctx)

	// Stage 1: Stream names from database.
	g.Go(func() error {
		defer close(chIn)
		return loadNamesForWords(gCtx, pool, chIn)
	})

	// Stage 2: Parse with workers.
	var wg sync.WaitGroup
	for i := 0; i < jobsNum; i++ {
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			return workerExtractWords(gCtx, chIn, chOut)
		})
	}

	// Close chOut when workers done.
	go func() {
		wg.Wait()
		close(chOut)
	}()

	// Stage 3: Collect words into map, save word-name
	// linkages in batches.
	wordsMap := make(map[string]schema.Word)
	totalLinks := 0
	var wnsBatch []schema.WordNameString

	g.Go(func() error {
		for r := range chOut {
			for _, w := range r.words {
				key := w.ID + "|" + w.Normalized
				wordsMap[key] = w
			}

			wnsBatch = append(wnsBatch, r.wordNames...)

			if len(wnsBatch) >= batchSize {
				deduped := deduplicateWordNames(wnsBatch)
				err := saveWordNameStrings(
					gCtx, pool, deduped, cfg,
				)
				if err != nil {
					return err
				}
				totalLinks += len(deduped)
				wnsBatch = wnsBatch[:0]
			}
		}

		// Flush remaining word-name linkages.
		if len(wnsBatch) > 0 {
			deduped := deduplicateWordNames(wnsBatch)
			err := saveWordNameStrings(
				gCtx, pool, deduped, cfg,
			)
			if err != nil {
				return err
			}
			totalLinks += len(deduped)
		}

		return nil
	})

	// Wait for pipeline.
	err := g.Wait()

	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, 0, &gn.Error{
			Code: errcode.OptimizerWordExtractionError,
			Msg:  "Failed to parse names for words",
			Err:  fmt.Errorf("pipeline: %w", err),
		}
	}

	slog.Info(
		"Completed parsing names for words",
		"uniqueWords", len(wordsMap),
		"totalLinks", totalLinks,
	)
	return wordsMap, totalLinks, nil
}

// deduplicateWordNames removes duplicate word-name linkages
// within a batch. Deduplication key is WordID|NameStringID.
func deduplicateWordNames(
	wns []schema.WordNameString,
) []schema.WordNameString {
	seen := make(map[string]struct{}, len(wns))
	res := make([]schema.WordNameString, 0, len(wns))
	for _, wn := range wns {
		key := wn.WordID + "|" + wn.NameStringID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		res = append(res, wn)
	}
	return res
}

// workerExtractWords processes names and extracts words
// (worker function).
func workerExtractWords(
	ctx context.Context,
	chIn <-chan nameForWords,
	chOut chan<- wordResult,
) error {
	prsCfg := gnparser.NewConfig(
		gnparser.OptWithDetails(true),
	)
	prs := gnparser.New(prsCfg)

	for n := range chIn {
		select {
		case <-ctx.Done():
			for range chIn {
			}
			return ctx.Err()
		default:
		}

		// Parse name.
		p := prs.ParseName(n.name)

		// Skip unparsed names, surrogates, and hybrids.
		if !p.Parsed || p.Surrogate != nil || p.Hybrid != nil {
			continue
		}

		// Extract words from parsed result.
		var words []schema.Word
		var wordNames []schema.WordNameString

		for _, w := range p.Words {
			wt := w.Type
			// Generate modified form using gnparser.
			mod := parsed.NormalizeByType(w.Normalized, wt)
			// Generate word ID from modified|typeID.
			idstr := fmt.Sprintf("%s|%d", mod, int(wt))
			wordID := gnuuid.New(idstr).String()

			// Create WordNameString junction record.
			wns := schema.WordNameString{
				WordID:       wordID,
				NameStringID: n.id,
				CanonicalID:  n.canonicalID,
			}

			// Only include specific word types.
			switch wt {
			case
				parsed.SpEpithetType,
				parsed.InfraspEpithetType,
				parsed.AuthorWordType:
				// Create Word record.
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

		if len(words) > 0 {
			chOut <- wordResult{
				words:     words,
				wordNames: wordNames,
			}
		}
	}

	return nil
}

// saveWords performs bulk insert of words using pgx.CopyFrom.
//
// Reference: gnidump saveWords() in db.go
func saveWords(
	ctx context.Context,
	pool *pgxpool.Pool,
	words []schema.Word,
	cfg *config.Config,
) error {
	if len(words) == 0 {
		slog.Info("No words to save")
		return nil
	}

	batchSize := cfg.Database.BatchSize
	if batchSize == 0 {
		batchSize = 50000
	}

	columns := []string{
		"id", "normalized", "modified", "type_id",
	}
	totalSaved := 0

	bar := newProgressBar(len(words), "Saving words: ")
	defer bar.Finish()

	for i := 0; i < len(words); i += batchSize {
		end := i + batchSize
		if end > len(words) {
			end = len(words)
		}
		batch := words[i:end]

		// Prepare rows for CopyFrom.
		rows := make([][]any, len(batch))
		for j, w := range batch {
			rows[j] = []any{
				w.ID,
				w.Normalized,
				w.Modified,
				w.TypeID,
			}
		}

		// Bulk insert using CopyFrom.
		copyCount, err := pool.CopyFrom(
			ctx,
			pgx.Identifier{"words"},
			columns,
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return &gn.Error{
				Code: errcode.OptimizerWordExtractionError,
				Msg:  "Failed to save words",
				Err:  fmt.Errorf("copy from: %w", err),
			}
		}

		totalSaved += int(copyCount)
		bar.Add(len(batch))
	}

	slog.Info("Completed saving words", "total", totalSaved)
	return nil
}

// saveWordNameStrings performs bulk insert of word-name-string
// linkages using pgx.CopyFrom.
//
// Reference: gnidump saveNameWords() in db.go
func saveWordNameStrings(
	ctx context.Context,
	pool *pgxpool.Pool,
	wordNames []schema.WordNameString,
	cfg *config.Config,
) error {
	if len(wordNames) == 0 {
		slog.Info("No word-name linkages to save")
		return nil
	}

	batchSize := cfg.Database.BatchSize
	if batchSize == 0 {
		batchSize = 50000
	}

	columns := []string{
		"word_id",
		"name_string_id",
		"canonical_id",
	}
	totalSaved := 0

	bar := newProgressBar(
		len(wordNames),
		"Saving word linkages: ",
	)
	defer bar.Finish()

	for i := 0; i < len(wordNames); i += batchSize {
		end := min(i+batchSize, len(wordNames))
		batch := wordNames[i:end]

		// Prepare rows for CopyFrom.
		rows := make([][]any, len(batch))
		for j, wn := range batch {
			rows[j] = []any{
				wn.WordID,
				wn.NameStringID,
				wn.CanonicalID,
			}
		}

		// Bulk insert using CopyFrom.
		copyCount, err := pool.CopyFrom(
			ctx,
			pgx.Identifier{"word_name_strings"},
			columns,
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return &gn.Error{
				Code: errcode.OptimizerWordExtractionError,
				Msg:  "Failed to save word linkages",
				Err:  fmt.Errorf("copy from: %w", err),
			}
		}

		totalSaved += int(copyCount)
		bar.Add(len(batch))
	}

	slog.Info(
		"Completed saving word-name linkages",
		"total", totalSaved,
	)
	return nil
}
