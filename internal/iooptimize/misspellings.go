package iooptimize

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gnlib/ent/fuzzy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
)

const (
	// ionDataSourceID is the ID of ION, the source of misspelled names.
	ionDataSourceID = 168

	// misspellingEditDist is the default MaxEditDist of gnmatcher. Edit
	// distance 2 gives too many false positives.
	misspellingEditDist = 1

	// misspellingsFile keeps flagged names for a review.
	misspellingsFile = "misspellings.tsv"
)

// reliableNamesSQL selects canonical forms of reliable sources with their
// stems. Reliable sources are the curated ones, EOL (12), Catalogue of
// Life XR (13) and Ruhoff 1980 (206).
const reliableNamesSQL = `
SELECT DISTINCT cs.name, c.name
FROM name_string_indices nsi
JOIN data_sources ds ON ds.id = nsi.data_source_id
JOIN name_strings ns ON ns.id = nsi.name_string_id
JOIN canonicals c ON c.id = ns.canonical_id
JOIN canonical_stems cs ON cs.id = ns.canonical_stem_id
WHERE ds.is_curated OR ds.id IN (12, 13, 206)`

// ionOnlyNamesSQL selects canonical forms that have 2 or more words and
// are used by ION and by no other source, reliable or not. It guarantees
// that a flag cannot hide a name of another source.
//
// Uninomials are skipped: gnmatcher does not fuzzy-match them by default,
// so they block nothing.
const ionOnlyNamesSQL = `
SELECT p.canonical_id, c.name, cs.name
FROM (
  SELECT ns.canonical_id,
         min(ns.canonical_stem_id::text)::uuid AS canonical_stem_id
  FROM name_string_indices nsi
  JOIN name_strings ns ON ns.id = nsi.name_string_id
  WHERE ns.canonical_id IS NOT NULL
  GROUP BY ns.canonical_id
  HAVING bool_and(nsi.data_source_id = $1)
     AND bool_and(ns.cardinality > 1)
     AND bool_and(ns.virus IS NOT TRUE)
) p
JOIN canonicals c ON c.id = p.canonical_id
JOIN canonical_stems cs ON cs.id = p.canonical_stem_id
ORDER BY c.name`

// ionName is a canonical form that is used only by ION.
type ionName struct {
	canonicalID string
	canonical   string
	stem        string
}

// misspelling is an ION name with a reliable name that it blocks.
type misspelling struct {
	ionName
	reliable string
}

// reliableNames gives access to canonical forms of reliable sources.
type reliableNames struct {
	// canonicals keeps canonical forms by their stem.
	canonicals map[string][]string
	trie       *fuzzy.Trie
}

// flagMisspellings sets canonicals.maybe_misspelling for canonical forms
// that would block fuzzy matching to a reliable name. This is Step 4 of
// the optimization workflow.
//
// Flagged names are saved to misspellings.tsv in the cache directory, so
// they can be reviewed.
//
// Uses: cfg.HomeDir
func flagMisspellings(
	ctx context.Context,
	opt *optimizer,
	cfg *config.Config,
) (string, error) {
	pool := opt.operator.Pool()
	if pool == nil {
		return "", misspellingError(
			"check connection", fmt.Errorf("pool is nil"),
		)
	}

	names, err := findMisspellings(ctx, pool)
	if err != nil {
		return "", err
	}

	// It runs without ION as well, to remove flags of a previous run.
	if err = saveMisspellingFlags(ctx, pool, names); err != nil {
		return "", err
	}

	path := filepath.Join(config.CacheDir(cfg.HomeDir), misspellingsFile)
	if err = writeMisspellings(path, names); err != nil {
		return "", err
	}
	slog.Info("Saved flagged names", "path", path, "count", len(names))

	msg := fmt.Sprintf(
		"<em>Flagged %s possibly misspelled ION names</em>",
		humanize.Comma(int64(len(names))),
	)
	return msg, nil
}

// findMisspellings returns ION names that follow the flagging rule.
// It only reads from the database.
func findMisspellings(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]misspelling, error) {
	ionNames, err := loadIONNames(ctx, pool)
	if err != nil {
		return nil, err
	}
	if len(ionNames) == 0 {
		slog.Info("Database has no ION-only names")
		return nil, nil
	}

	reliable, err := loadReliableNames(ctx, pool)
	if err != nil {
		return nil, err
	}

	var res []misspelling
	for _, name := range ionNames {
		if match, ok := reliable.match(name); ok {
			res = append(res, misspelling{ionName: name, reliable: match})
		}
	}
	return res, nil
}

// match returns a reliable canonical form that gnmatcher would find for
// the ION name, if its exact match did not exist. It repeats the order of
// checks of matchFuzzy in gnmatcher, including the order of arguments of
// EditDistance, because the checks apply only to the second argument.
func (r *reliableNames) match(name ionName) (string, bool) {
	// A shared stem gives reliable names already, nothing is blocked.
	if _, ok := r.canonicals[name.stem]; ok {
		return "", false
	}

	for _, stem := range r.trie.Matches(name.stem, misspellingEditDist) {
		if fuzzy.EditDistance(stem, name.stem, false) == -1 {
			continue
		}
		for _, canonical := range r.canonicals[stem] {
			if fuzzy.EditDistance(name.canonical, canonical, false) != -1 {
				return canonical, true
			}
		}
	}
	return "", false
}

func loadIONNames(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]ionName, error) {
	slog.Info("Loading ION-only names")
	rows, err := pool.Query(ctx, ionOnlyNamesSQL, ionDataSourceID)
	if err != nil {
		return nil, misspellingError("query ION names", err)
	}
	defer rows.Close()

	var res []ionName
	for rows.Next() {
		var name ionName
		err = rows.Scan(&name.canonicalID, &name.canonical, &name.stem)
		if err != nil {
			return nil, misspellingError("scan ION name", err)
		}
		res = append(res, name)
	}
	if err = rows.Err(); err != nil {
		return nil, misspellingError("read ION names", err)
	}
	return res, nil
}

func loadReliableNames(
	ctx context.Context,
	pool *pgxpool.Pool,
) (*reliableNames, error) {
	slog.Info("Loading reliable names")
	rows, err := pool.Query(ctx, reliableNamesSQL)
	if err != nil {
		return nil, misspellingError("query reliable names", err)
	}
	defer rows.Close()

	canonicals := make(map[string][]string)
	for rows.Next() {
		var stem, canonical string
		if err = rows.Scan(&stem, &canonical); err != nil {
			return nil, misspellingError("scan reliable name", err)
		}
		canonicals[stem] = append(canonicals[stem], canonical)
	}
	if err = rows.Err(); err != nil {
		return nil, misspellingError("read reliable names", err)
	}

	slog.Info("Building trie of reliable stems", "stems", len(canonicals))
	return newReliableNames(canonicals)
}

func newReliableNames(
	canonicals map[string][]string,
) (*reliableNames, error) {
	stems := make([]string, 0, len(canonicals))
	for stem := range canonicals {
		stems = append(stems, stem)
	}
	slices.Sort(stems)

	trie, err := fuzzy.NewTrie(stems)
	if err != nil {
		return nil, misspellingError("build trie", err)
	}
	return &reliableNames{canonicals: canonicals, trie: trie}, nil
}

// saveMisspellingFlags replaces flags in one transaction, so a failure
// leaves flags of the previous run intact.
func saveMisspellingFlags(
	ctx context.Context,
	pool *pgxpool.Pool,
	names []misspelling,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return misspellingError("begin transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
UPDATE canonicals SET maybe_misspelling = false WHERE maybe_misspelling`)
	if err != nil {
		return misspellingError("reset flags", err)
	}

	_, err = tx.Exec(ctx, `
CREATE TEMP TABLE misspelled_canonicals (id uuid NOT NULL) ON COMMIT DROP`)
	if err != nil {
		return misspellingError("create temporary table", err)
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"misspelled_canonicals"},
		[]string{"id"},
		pgx.CopyFromSlice(len(names), func(i int) ([]any, error) {
			return []any{names[i].canonicalID}, nil
		}),
	)
	if err != nil {
		return misspellingError("copy canonical ids", err)
	}

	_, err = tx.Exec(ctx, `
UPDATE canonicals c SET maybe_misspelling = true
FROM misspelled_canonicals m
WHERE c.id = m.id`)
	if err != nil {
		return misspellingError("set flags", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return misspellingError("commit flags", err)
	}
	return nil
}

// writeMisspellings saves pairs of ION and reliable names to a file.
func writeMisspellings(path string, names []misspelling) error {
	f, err := os.Create(path)
	if err != nil {
		return misspellingError("create report", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, name := range names {
		_, err = fmt.Fprintf(w, "%s\t%s\n", name.canonical, name.reliable)
		if err != nil {
			return misspellingError("write report", err)
		}
	}
	if err = w.Flush(); err != nil {
		return misspellingError("write report", err)
	}
	return f.Close()
}

func misspellingError(action string, err error) error {
	return &gn.Error{
		Code: errcode.OptimizerMisspellingError,
		Msg:  "Failed to flag ION misspellings",
		Err:  fmt.Errorf("%s: %w", action, err),
	}
}
