package iooptimize

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/gnames/gnfmt/gnlang"
	"github.com/jackc/pgx/v5/pgxpool"
)

// vernacular represents a vernacular record for language
// normalization.
type vernacular struct {
	ctID        string
	language    sql.NullString
	langCode    sql.NullString
	newLanguage sql.NullString
	newLangCode sql.NullString
	needsUpdate bool
}

// normalizeVernaculars orchestrates the vernacular language
// normalization process. This implements Step 2 of the optimize
// workflow using batch updates for performance.
//
// Workflow:
//  1. Move language field to language_orig (preserve original)
//  2. Create temporary table for batch updates
//  3. Load all vernacular records and normalize in memory
//  4. Batch insert normalized data to temp table
//  5. Single UPDATE FROM temp table to apply all changes
//  6. Convert all lang_code to lowercase
//
// Performance: Uses batch operations instead of row-by-row
// updates (100x+ faster)
//
// Reference: gnidump fixVernLang() in db_vern.go + batch
// optimization
func normalizeVernaculars(
	ctx context.Context,
	opt *optimizer,
	cfg *config.Config,
) error {
	pool := opt.operator.Pool()
	if pool == nil {
		return &gn.Error{
			Code: errcode.OptimizerVernacularNormalizeError,
			Msg:  "Database connection lost",
			Err:  fmt.Errorf("pool is nil"),
		}
	}

	slog.Info("Moving language data to language_orig")
	err := moveLanguageToOrig(ctx, pool)
	if err != nil {
		return err
	}

	slog.Info("Normalizing vernacular languages")

	// Create temporary table for batch updates
	err = createVernacularTempTable(ctx, pool)
	if err != nil {
		return err
	}
	defer func() {
		dropCtx := context.Background()
		q := "DROP TABLE IF EXISTS temp_vernacular_updates"
		_, _ = pool.Exec(dropCtx, q)
	}()

	// Load and normalize all records
	records, err := loadAndNormalizeVernaculars(ctx, pool)
	if err != nil {
		return err
	}

	// Batch insert to temp table
	err = batchInsertVernacularUpdates(
		ctx,
		pool,
		records,
		cfg,
	)
	if err != nil {
		return err
	}

	// Single UPDATE FROM temp table
	err = applyVernacularBatchUpdate(ctx, pool)
	if err != nil {
		return err
	}

	slog.Info("Making sure all language codes are lowercase")
	err = langCodeToLowercase(ctx, pool)
	if err != nil {
		return err
	}

	return nil
}

// moveLanguageToOrig copies language field to language_orig for
// records that don't have it. This preserves the original
// language value before normalization.
//
// Reference: gnidump langOrig() in db_vern.go
func moveLanguageToOrig(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	q := `
UPDATE vernacular_string_indices
SET language_orig = language
WHERE language_orig IS NULL OR language_orig = ''`

	_, err := pool.Exec(ctx, q)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerVernacularNormalizeError,
			Msg:  "Failed to preserve original language values",
			Err:  fmt.Errorf("move to language_orig: %w", err),
		}
	}
	return nil
}

// createVernacularTempTable creates a temporary table to hold
// batch updates.
func createVernacularTempTable(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	// Drop if exists from previous run
	_, _ = pool.Exec(
		ctx,
		"DROP TABLE IF EXISTS temp_vernacular_updates",
	)

	q := `
CREATE UNLOGGED TABLE temp_vernacular_updates (
	row_ctid tid,
	language TEXT,
	lang_code TEXT
)`

	_, err := pool.Exec(ctx, q)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerVernacularNormalizeError,
			Msg:  "Failed to create temporary table",
			Err:  fmt.Errorf("create temp table: %w", err),
		}
	}
	return nil
}

// loadAndNormalizeVernaculars loads all vernacular records and
// normalizes them in memory. This is much faster than row-by-row
// updates.
func loadAndNormalizeVernaculars(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]vernacular, error) {
	// Count total vernacular records for progress
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM vernacular_string_indices`
	err := pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, &gn.Error{
			Code: errcode.OptimizerVernacularNormalizeError,
			Msg:  "Failed to count vernacular records",
			Err:  fmt.Errorf("count query: %w", err),
		}
	}

	q := `
SELECT ctid, language, lang_code
FROM vernacular_string_indices`

	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, &gn.Error{
			Code: errcode.OptimizerVernacularNormalizeError,
			Msg:  "Failed to query vernacular records",
			Err:  fmt.Errorf("query: %w", err),
		}
	}
	defer rows.Close()

	// Create progress bar with known total
	bar := newProgressBar(
		totalCount,
		"Loading and normalizing: ",
	)
	defer bar.Finish()

	var records []vernacular
	count := 0
	for rows.Next() {
		var v vernacular
		err := rows.Scan(&v.ctID, &v.language, &v.langCode)
		if err != nil {
			return nil, &gn.Error{
				Code: errcode.OptimizerVernacularNormalizeError,
				Msg:  "Failed to scan vernacular record",
				Err:  fmt.Errorf("scan: %w", err),
			}
		}

		// Normalize in memory
		normalizeVernacularRecord(&v)

		// Only keep records that need updating
		if v.needsUpdate {
			records = append(records, v)
		}

		count++
		if count%1000 == 0 {
			bar.Add(1000)
		}
	}

	// Add remainder
	if count%1000 > 0 {
		bar.Add(count % 1000)
	}

	if err := rows.Err(); err != nil {
		return nil, &gn.Error{
			Code: errcode.OptimizerVernacularNormalizeError,
			Msg:  "Failed to process vernacular records",
			Err:  fmt.Errorf("rows error: %w", err),
		}
	}

	slog.Info(
		"Finished normalization",
		"recordsToUpdate", len(records),
		"total", totalCount,
	)
	return records, nil
}

// normalizeVernacularRecord normalizes a single vernacular
// record in memory. Sets needsUpdate flag if the record was
// modified.
//
// Logic (matching gnidump exactly):
//   - 2-letter codes: Convert to 3-letter, set language to
//     full name
//   - 3-letter codes: Validate, set language to full name
//   - Missing lang_code: Derive from language field
//
// Reference: gnidump normVernLang() in db_vern.go
func normalizeVernacularRecord(v *vernacular) {
	v.newLanguage = v.language
	v.newLangCode = v.langCode
	v.needsUpdate = false

	switch {
	case len(v.language.String) == 2:
		// 2-letter code: convert to 3-letter
		lang3, err := gnlang.LangCode2To3Letters(
			v.language.String,
		)
		if err != nil {
			return
		}
		if len(v.langCode.String) != 3 {
			v.newLangCode = sql.NullString{
				String: lang3,
				Valid:  true,
			}
			v.needsUpdate = true
		}
		lang := gnlang.Lang(lang3)
		if lang != "" && lang != v.language.String {
			v.newLanguage = sql.NullString{
				String: lang,
				Valid:  true,
			}
			v.needsUpdate = true
		}

	case len(v.language.String) == 3:
		// 3-letter code: validate and normalize
		_, err := gnlang.LangCode3To2Letters(
			v.language.String,
		)
		if err != nil {
			return
		}
		if len(v.langCode.String) != 3 {
			v.newLangCode = v.language
			v.needsUpdate = true
		}
		lang := gnlang.Lang(v.language.String)
		if lang != "" && lang != v.language.String {
			v.newLanguage = sql.NullString{
				String: lang,
				Valid:  true,
			}
			v.needsUpdate = true
		}

	case len(v.langCode.String) != 3:
		// Missing lang_code: derive from language field
		lang3 := gnlang.LangCode(v.language.String)
		if lang3 != "" {
			v.newLangCode = sql.NullString{
				String: lang3,
				Valid:  true,
			}
			v.needsUpdate = true
			// Also normalize language to full name
			lang := gnlang.Lang(lang3)
			if lang != "" && lang != v.language.String {
				v.newLanguage = sql.NullString{
					String: lang,
					Valid:  true,
				}
			}
		}
	}
}

// batchInsertVernacularUpdates inserts normalized records into
// temp table using parameterized inserts.
func batchInsertVernacularUpdates(
	ctx context.Context,
	pool *pgxpool.Pool,
	records []vernacular,
	cfg *config.Config,
) error {
	if len(records) == 0 {
		slog.Info("No vernacular records need updating")
		return nil
	}

	// PostgreSQL parameter limit is 65535
	// Each record uses 3 parameters (ctid, language, lang_code)
	// Max safe batch size: 65535 / 3 = 21845
	const maxBatchSize = 21845

	batchSize := cfg.Database.BatchSize
	if batchSize == 0 || batchSize > maxBatchSize {
		batchSize = maxBatchSize
	}

	bar := newProgressBar(len(records), "Saving updates: ")
	defer bar.Finish()

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		// Build batch insert
		var values []any
		var valuePlaceholders []string
		argIdx := 1
		for _, v := range batch {
			ph := fmt.Sprintf(
				"($%d::tid, $%d, $%d)",
				argIdx,
				argIdx+1,
				argIdx+2,
			)
			valuePlaceholders = append(valuePlaceholders, ph)
			values = append(
				values,
				v.ctID,
				v.newLanguage,
				v.newLangCode,
			)
			argIdx += 3
		}

		q := fmt.Sprintf(`
INSERT INTO temp_vernacular_updates (row_ctid, language, lang_code)
VALUES %s
`, strings.Join(valuePlaceholders, ", "))

		_, err := pool.Exec(ctx, q, values...)
		if err != nil {
			return &gn.Error{
				Code: errcode.OptimizerVernacularNormalizeError,
				Msg:  "Failed to insert batch updates",
				Err:  fmt.Errorf("batch insert: %w", err),
			}
		}

		bar.Add(len(batch))
	}

	return nil
}

// applyVernacularBatchUpdate applies all updates from temp table
// in a single UPDATE.
func applyVernacularBatchUpdate(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	q := `
UPDATE vernacular_string_indices v
SET
	language = t.language,
	lang_code = t.lang_code
FROM temp_vernacular_updates t
WHERE v.ctid = t.row_ctid`

	result, err := pool.Exec(ctx, q)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerVernacularNormalizeError,
			Msg:  "Failed to apply batch updates",
			Err:  fmt.Errorf("batch update: %w", err),
		}
	}

	rowsUpdated := result.RowsAffected()
	msg := "<em>All vernacular records are normalized</em>"
	if rowsUpdated > 0 {
		msg = fmt.Sprintf(
			"<em>Normalized %s vernacular records</em>",
			humanize.Comma(rowsUpdated),
		)
	}
	gn.Info(msg)
	return nil
}

// langCodeToLowercase ensures all lang_code values are lowercase.
//
// Reference: gnidump langCodeLowCase() in db_vern.go
func langCodeToLowercase(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	q := `
UPDATE vernacular_string_indices
SET lang_code = LOWER(lang_code)`

	_, err := pool.Exec(ctx, q)
	if err != nil {
		return &gn.Error{
			Code: errcode.OptimizerVernacularNormalizeError,
			Msg:  "Failed to lowercase language codes",
			Err:  fmt.Errorf("lowercase: %w", err),
		}
	}
	return nil
}
