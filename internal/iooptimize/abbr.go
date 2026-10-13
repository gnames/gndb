package iooptimize

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gnparser"
	"github.com/gnames/gnparser/ent/stemmer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
)

// abbrRow is one epithet of a canonical form with its lookup key.
type abbrRow struct {
	key             string
	pos             int16
	genus           string
	canonicalStemID string
}

// abbrNamesSQL selects canonical forms with their stems. Keys are
// built from the unstemmed canonical forms, the same way gnames
// builds them from text.
//
// Uninomials have no epithets, so names with cardinality below 2 are
// skipped. This also skips surrogates and hybrid formulas, which have
// cardinality 0 (stored as NULL). A canonical form is kept if at least
// one of its name-strings passes the filter.
//
// Hybrids (×), graft chimeras (+) and cultivars (‘) are skipped:
// splitting them by spaces would treat a second genus as an epithet.
// Viruses are skipped: their names are not Latin binomials and their
// canonical forms are not stemmed.
const abbrNamesSQL = `
SELECT c.name, p.canonical_stem_id
FROM (
  SELECT DISTINCT canonical_id, canonical_stem_id
  FROM name_strings
  WHERE canonical_id IS NOT NULL AND canonical_stem_id IS NOT NULL
    AND cardinality > 1
    AND virus IS NOT TRUE
) p
JOIN canonicals c ON c.id = p.canonical_id
WHERE c.name !~ '[×+‘]'`

// abbrStagingSQL creates the table that receives rows from Go.
const abbrStagingSQL = `
CREATE UNLOGGED TABLE abbr_lookups_staging (
  abbr_key varchar(255) NOT NULL,
  pos smallint NOT NULL,
  genus varchar(255) NOT NULL,
  canonical_stem_id uuid NOT NULL
)`

// abbrInsertSQL moves rows from staging to abbr_lookups.
//
// Several canonical forms share a stem (Carex alba, Carex albus), and
// repeated epithets (Bison bison bison) give the same key twice, so
// GROUP BY keeps one row per key and stem with the lowest position.
// ORDER BY stores rows of the same key next to each other on disk.
// The table is refilled on every run, so CLUSTER is not needed.
const abbrInsertSQL = `
INSERT INTO abbr_lookups
  (abbr_key, pos, genus, canonical_stem_id, canonical_stem, ds_num)
WITH ds AS (
  SELECT ns.canonical_stem_id,
         count(DISTINCT nsi.data_source_id) AS ds_num
  FROM name_strings ns
  JOIN name_string_indices nsi ON nsi.name_string_id = ns.id
  WHERE ns.canonical_stem_id IS NOT NULL
  GROUP BY ns.canonical_stem_id
)
SELECT s.abbr_key,
       min(s.pos),
       s.genus,
       s.canonical_stem_id,
       cs.name,
       coalesce(ds.ds_num, 0)::smallint
FROM abbr_lookups_staging s
JOIN canonical_stems cs ON cs.id = s.canonical_stem_id
LEFT JOIN ds ON ds.canonical_stem_id = s.canonical_stem_id
GROUP BY s.abbr_key, s.genus, s.canonical_stem_id, cs.name, ds.ds_num
ORDER BY s.abbr_key`

// abbrIndexSQL creates the lookup index after the bulk load.
// INCLUDE columns allow index-only scans.
const abbrIndexSQL = `
CREATE INDEX abbr_lookups_key_idx ON abbr_lookups (abbr_key)
  INCLUDE (genus, canonical_stem, pos, ds_num)`

// buildAbbrLookup rebuilds the abbr_lookups table used to find
// candidate names for abbreviations like "C. elegans". This is
// Step 6 of the optimization workflow. It must run after reparsing
// (Step 1) and orphan removal (Step 3), and before VACUUM ANALYZE.
//
// Keys are built in Go with stemmer.AbbrKey from gnparser, the same
// function gnames uses to build keys from text.
//
// Workflow:
//  1. Truncate abbr_lookups, drop its index, create a staging table
//  2. Stream canonical forms, build keys, copy them to staging
//  3. Move rows to abbr_lookups without duplicates, in key order
//  4. Create the lookup index, drop the staging table
//  5. Record the gnparser version in a comment on the table
//  6. Log the largest keys to make odd cases easy to spot
//  7. Build abbr_genera with sizes and classifications of genera
//
// Uses: cfg.Database.BatchSize
func buildAbbrLookup(
	ctx context.Context,
	opt *optimizer,
	cfg *config.Config,
) (string, error) {
	pool := opt.operator.Pool()
	if pool == nil {
		return "", &gn.Error{
			Code: errcode.OptimizerAbbrLookupError,
			Msg:  "Database connection lost",
			Err:  fmt.Errorf("pool is nil"),
		}
	}

	slog.Info("Building abbreviation lookup table")

	prepare := []string{
		"TRUNCATE abbr_lookups",
		"DROP INDEX IF EXISTS abbr_lookups_key_idx",
		"DROP TABLE IF EXISTS abbr_lookups_staging",
		abbrStagingSQL,
	}
	for _, q := range prepare {
		if _, err := pool.Exec(ctx, q); err != nil {
			return "", abbrError("prepare", err)
		}
	}

	if err := stageAbbrRows(ctx, pool, cfg); err != nil {
		return "", err
	}

	res, err := pool.Exec(ctx, abbrInsertSQL)
	if err != nil {
		return "", abbrError("insert", err)
	}
	count := res.RowsAffected()

	if _, err := pool.Exec(ctx, abbrIndexSQL); err != nil {
		return "", abbrError("create index", err)
	}

	q := "DROP TABLE abbr_lookups_staging"
	if _, err := pool.Exec(ctx, q); err != nil {
		return "", abbrError("drop staging", err)
	}

	// Keys match only when gndb and gnames use the same stemmer, so
	// clients can compare this version with their own.
	q = fmt.Sprintf(
		"COMMENT ON TABLE abbr_lookups IS 'gnparser %s'",
		gnparser.Version,
	)
	if _, err := pool.Exec(ctx, q); err != nil {
		return "", abbrError("comment", err)
	}

	if err := logLargestAbbrKeys(ctx, pool); err != nil {
		return "", err
	}

	generaNum, err := buildAbbrGenera(ctx, pool)
	if err != nil {
		return "", err
	}

	msg := fmt.Sprintf(
		"<em>Created abbreviation lookup with %s records and %s genera</em>",
		humanize.Comma(count),
		humanize.Comma(generaNum),
	)
	return msg, nil
}

// stageAbbrRows reads canonical forms, builds a row for every
// epithet, and copies the rows to the staging table in batches.
func stageAbbrRows(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *config.Config,
) error {
	batchSize := cfg.Database.BatchSize
	if batchSize == 0 {
		batchSize = 50000
	}

	rows, err := pool.Query(ctx, abbrNamesSQL)
	if err != nil {
		return abbrError("query names", err)
	}
	defer rows.Close()

	var total int
	batch := make([]abbrRow, 0, batchSize)
	for rows.Next() {
		var canonical, stemID string
		if err := rows.Scan(&canonical, &stemID); err != nil {
			return abbrError("scan names", err)
		}

		batch = append(batch, abbrRows(canonical, stemID)...)
		if len(batch) >= batchSize {
			if err := copyAbbrRows(ctx, pool, batch); err != nil {
				return err
			}
			total += len(batch)
			batch = batch[:0]
		}
	}
	if err := rows.Err(); err != nil {
		return abbrError("read names", err)
	}

	if err := copyAbbrRows(ctx, pool, batch); err != nil {
		return err
	}
	total += len(batch)

	slog.Info("Staged abbreviation rows", "rows", total)
	return nil
}

// abbrRows splits a canonical form into a genus and epithets and
// returns one row per epithet. Uninomials and names that do not
// start with a capital letter give no rows. Epithets with an empty
// stem (such as "que") are skipped.
//
// Names with an abbreviated genus ("C. elegans") or with a broken
// character in the genus (U+FFFD from a wrong encoding) give no rows:
// such genera cannot be candidates for an abbreviation found in text.
func abbrRows(canonical, stemID string) []abbrRow {
	words := strings.Fields(canonical)
	if len(words) < 2 {
		return nil
	}

	genus := words[0]
	first, _ := utf8.DecodeRuneInString(genus)
	if !unicode.IsUpper(first) {
		return nil
	}

	if strings.HasSuffix(genus, ".") ||
		strings.ContainsRune(genus, utf8.RuneError) {
		return nil
	}

	res := make([]abbrRow, 0, len(words)-1)
	for i, epithet := range words[1:] {
		key := stemmer.AbbrKey(genus, epithet)
		if key == "" {
			continue
		}
		res = append(res, abbrRow{
			key:             key,
			pos:             int16(i + 2),
			genus:           genus,
			canonicalStemID: stemID,
		})
	}
	return res
}

// copyAbbrRows bulk inserts rows to the staging table.
func copyAbbrRows(
	ctx context.Context,
	pool *pgxpool.Pool,
	batch []abbrRow,
) error {
	if len(batch) == 0 {
		return nil
	}

	columns := []string{"abbr_key", "pos", "genus", "canonical_stem_id"}
	_, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"abbr_lookups_staging"},
		columns,
		pgx.CopyFromSlice(len(batch), func(i int) ([]any, error) {
			r := batch[i]
			return []any{r.key, r.pos, r.genus, r.canonicalStemID}, nil
		}),
	)
	if err != nil {
		return abbrError("copy rows", err)
	}
	return nil
}

// logLargestAbbrKeys logs the 20 keys with the most rows.
func logLargestAbbrKeys(ctx context.Context, pool *pgxpool.Pool) error {
	q := `
SELECT abbr_key, count(*) AS num
FROM abbr_lookups
GROUP BY abbr_key
ORDER BY num DESC, abbr_key
LIMIT 20`

	rows, err := pool.Query(ctx, q)
	if err != nil {
		return abbrError("largest keys", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var num int64
		if err := rows.Scan(&key, &num); err != nil {
			return abbrError("scan largest keys", err)
		}
		slog.Info("Large abbreviation key", "key", key, "rows", num)
	}
	if err := rows.Err(); err != nil {
		return abbrError("largest keys", err)
	}
	return nil
}

func abbrError(stage string, err error) error {
	return &gn.Error{
		Code: errcode.OptimizerAbbrLookupError,
		Msg:  "Failed to build abbreviation lookup",
		Err:  fmt.Errorf("abbr lookup %s: %w", stage, err),
	}
}
