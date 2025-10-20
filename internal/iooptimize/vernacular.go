package iooptimize

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gnfmt/gnlang"
	"golang.org/x/sync/errgroup"
)

// vern represents a vernacular record for language normalization.
// Uses ctid (PostgreSQL physical row ID) for updates.
type vern struct {
	ctID     string
	language sql.NullString
	langCode sql.NullString
}

// fixVernacularLanguages orchestrates the vernacular language normalization process.
// This implements Step 2 of the optimize workflow.
//
// Workflow:
//  1. Move language field to language_orig (preserve original)
//  2. Load all vernacular records
//  3. Normalize language codes using gnlang (2-letter → 3-letter, validate)
//  4. Convert all lang_code to lowercase
//
// Reference: gnidump fixVernLang() in db_vern.go
func fixVernacularLanguages(ctx context.Context, opt *OptimizerImpl, cfg *config.Config) error {
	slog.Info("Moving new language data to language_orig")
	err := moveLanguageToOrig(ctx, opt)
	if err != nil {
		return err
	}

	slog.Info("Normalizing vernacular language")
	chIn := make(chan vern)
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		defer close(chIn)
		return loadVernaculars(gCtx, opt, chIn)
	})

	g.Go(func() error {
		return normalizeVernacularLanguage(gCtx, opt, chIn)
	})

	if err := g.Wait(); err != nil {
		return err
	}

	slog.Info("Making sure all language codes are low case")
	err = langCodeToLowercase(ctx, opt)
	if err != nil {
		slog.Error("Could not set all language codes to low case", "error", err)
		return err
	}

	return nil
}

// moveLanguageToOrig copies language field to language_orig for records that don't have it.
// This preserves the original language value before normalization.
//
// Reference: gnidump langOrig() in db_vern.go
func moveLanguageToOrig(ctx context.Context, opt *OptimizerImpl) error {
	q := `
UPDATE vernacular_string_indices
	SET language_orig = language
	WHERE language_orig IS NULL
`
	_, err := opt.operator.Pool().Exec(ctx, q)
	return err
}

// loadVernaculars loads all vernacular records from database for language normalization.
// Sends records to channel for concurrent processing.
//
// Reference: gnidump loadVern() in db_vern.go
func loadVernaculars(ctx context.Context, opt *OptimizerImpl, ch chan<- vern) error {
	timeStart := time.Now().UnixNano()
	q := `
SELECT ctid, language, lang_code
	FROM vernacular_string_indices
`
	rows, err := opt.operator.Pool().Query(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		count++
		var v vern
		if err := rows.Scan(&v.ctID, &v.language, &v.langCode); err != nil {
			return err
		}

		select {
		case ch <- v:
		case <-ctx.Done():
			return ctx.Err()
		}

		if count%50_000 == 0 {
			timeSpent := float64(time.Now().UnixNano()-timeStart) / 1_000_000_000
			speed := int64(float64(count) / timeSpent)
			fmt.Fprintf(os.Stderr, "\r%s", strings.Repeat(" ", 60))
			fmt.Fprintf(os.Stderr, "\rParsed %s names, %s names/sec",
				humanize.Comma(int64(count)), humanize.Comma(speed))
		}
	}

	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 60))
	slog.Info("Finished normalization of vernacular languages")

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

// normalizeVernacularLanguage processes vernacular records from channel,
// normalizing language codes and language names using gnlang library.
//
// Logic (matching gnidump exactly):
//   - 2-letter codes: Convert to 3-letter, set language to full name
//   - 3-letter codes: Validate, set language to full name
//   - Missing lang_code: Derive from language field
//
// Reference: gnidump normVernLang() in db_vern.go
func normalizeVernacularLanguage(ctx context.Context, opt *OptimizerImpl, ch <-chan vern) error {
	for v := range ch {
		switch {
		case len(v.language.String) == 2:
			// 2-letter code: convert to 3-letter
			lang3, err := gnlang.LangCode2To3Letters(v.language.String)
			if err != nil {
				continue
			}
			if len(v.langCode.String) != 3 {
				v.langCode = sql.NullString{String: lang3, Valid: true}
			}
			lang := gnlang.Lang(lang3)
			if lang != "" {
				v.language = sql.NullString{String: lang, Valid: true}
			}

		case len(v.language.String) == 3:
			// 3-letter code: validate and normalize
			_, err := gnlang.LangCode3To2Letters(v.language.String)
			if err != nil {
				continue
			}
			if len(v.langCode.String) != 3 {
				v.langCode = v.language
			}
			lang := gnlang.Lang(v.language.String)
			if lang != "" {
				v.language = sql.NullString{String: lang, Valid: true}
			}

		case len(v.langCode.String) != 3:
			// Missing lang_code: derive from language field
			lang3 := gnlang.LangCode(v.language.String)
			if lang3 != "" {
				v.langCode = sql.NullString{String: lang3, Valid: true}
				// Also normalize language to full name
				lang := gnlang.Lang(lang3)
				if lang != "" {
					v.language = sql.NullString{String: lang, Valid: true}
				}
			}

		default:
			continue
		}

		if err := updateVernRecord(ctx, opt, v); err != nil {
			slog.Error("Failed to update vernacular record",
				"ctid", v.ctID, "error", err)
			return err
		}
	}

	return nil
}

// updateVernRecord updates a single vernacular record using ctid (physical row ID).
//
// Reference: gnidump updateVernRecord() in db_vern.go
func updateVernRecord(ctx context.Context, opt *OptimizerImpl, v vern) error {
	q := `
UPDATE vernacular_string_indices
  SET language = $1, lang_code = $2
  WHERE ctid = $3
`
	_, err := opt.operator.Pool().Exec(ctx, q, v.language, v.langCode, v.ctID)
	return err
}

// langCodeToLowercase ensures all lang_code values are lowercase.
//
// Reference: gnidump langCodeLowCase() in db_vern.go
func langCodeToLowercase(ctx context.Context, opt *OptimizerImpl) error {
	q := `
UPDATE vernacular_string_indices
	SET lang_code = LOWER(lang_code)
`
	_, err := opt.operator.Pool().Exec(ctx, q)
	return err
}
