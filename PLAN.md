# PLAN: Abbreviated names lookup table

## Motivation

Scientific literature contains many abbreviated names, such as
`C. elegans` or `P. t. altaica`. Often a paper has several of them
for the same genus (`C. elegans`, `C. briggsae`, `C. remanei`).
Name-finding tools need a fast way to get candidate full names from
the first letter (or letters) of a genus and a specific or
infraspecific epithet.

The current faceted search in gnames (`g:C. sp:elegans`) can do this,
but it is slow for common epithets. It joins
`words -> word_name_strings -> canonicals -> verification` and filters
by genus only at the end. It also returns full verification records
when only candidate names are needed.

The plan is to add a small denormalized table, `abbr_lookups`, built
by `gndb optimize`, that answers such queries with a single indexed
equality lookup.

## Key idea

Each stemmed canonical form is split into a genus and its epithets.
Every epithet produces one row, keyed by the lowercase first letter
of the genus plus the epithet stem:

```text
Caenorhabditis elegans ->  c-elegans
Panthera tigr altaic   ->  p-tigr, p-altaic
Carex alb              ->  c-alb
```

- **Stems, not full epithets.** When the genus is abbreviated, the
  gender ending in the text often differs from the database
  (*alba*, *albus*, *album*). Stems make the match gender-blind.
  Stems come from gnparser's `stemmer.StemCanonical`, the same
  function that produces `canonical_stems`.
- **Position is stored, not keyed.** Epithets move between ranks:
  *Panthera tigris altaica* and *Panthera altaica*, or varieties
  raised to species. The key ignores position, and `pos` is used only
  as a ranking hint.
- **One row per key and name.** Repeated epithets (*Bison bison
  bison*, *Panthera tigris tigris*) give one row with the lowest
  `pos`, not duplicates.
- **One key, many rows.** `c-alb` maps to many genera. The index is
  non-unique and rows are stored in key order, so a lookup reads one
  contiguous range.
- **Extra genus letters filter, they don't key.** `Cae.`, `Ch.` and
  `Th.` use key `c-...`/`t-...` plus a genus prefix filter on the
  small result set.
- **Synonyms come for free.** Old combinations are in
  `canonical_stems`, so an old paper's abbreviation finds the genus
  the name was in at the time.
- **Hybrids and viruses are skipped.** Canonical forms containing
  `×`, `+` or `‘` are left out, and so are virus names: ICTV binomials
  (*Aichivirus A*) are not Latin, and their stems are not stemmed.

## Key function

The key is built in two places: by gndb when it fills the table, and
by gnames when it reads text. Both must produce exactly the same key,
so there is a single implementation in gnparser, next to the stemmer
it depends on:

```go
// AbbrKey returns a lookup key for a name with an abbreviated genus,
// e.g. ("C.", "elegans") -> "c-elegans", ("Carex", "alba") -> "c-alb".
func AbbrKey(genus, epithet string) string // gnparser ent/stemmer
```

- **Why gnparser.** The key is "how gnparser stems an epithet" plus
  the genus letter. gnames, gndb and gnfinder all depend on gnparser
  already. gnlib cannot hold it: gnparser imports gnlib, so gnlib
  importing the stemmer would create a module cycle.
- **Same function, same input.** gndb calls `AbbrKey` on epithets of
  unstemmed canonical forms, the same kind of input gnames gets from
  text. No key logic is written in SQL.
- **Fixtures live with the function.** gnparser's tests pin known
  keys (`vulgaris` -> `uulgar`, short words, diaereses), so a stemmer
  change that alters keys fails there first.
- **Same gnparser version.** Keys are only comparable when gndb and
  gnames use the same stemmer. gndb records the gnparser version in
  a comment on the table (see below); gnames can check it at start.

`parsed.NormalizeByType` (used for `words.modified`) must **not** be
used here. It differs from `StemCanonical`: it stems words shorter
than 3 characters, and it does not transliterate diaereses.

## Schema

New model in `pkg/schema/models.go`, added to `AllModels()` in
`pkg/schema/gorm.go`. `gndb migrate` creates the table in existing
databases.

```go
// AbbrLookup finds candidate names for abbreviated binomials and
// trinomials, e.g. "C. elegans" -> key "c-elegans".
type AbbrLookup struct {
	// AbbrKey is the lowercase first letter of the genus, a dash,
	// and the epithet stem, e.g. "c-elegans".
	AbbrKey string `gorm:"type:varchar(255);not null"`

	// Pos is the position of the epithet in the canonical form:
	// 2 for a specific epithet, 3+ for infraspecific ones.
	// For repeated epithets, the lowest position is kept.
	Pos int16

	// Genus is the full genus name.
	Genus string `gorm:"type:varchar(255);not null"`

	// CanonicalStemID links to canonical_stems.
	CanonicalStemID string `gorm:"type:uuid;not null"`

	// CanonicalStem is the stemmed canonical form.
	CanonicalStem string `gorm:"type:varchar(255);not null"`

	// DsNum is the number of data sources that have names with
	// this stemmed canonical form. Used for ranking.
	DsNum int16
}
```

Indexes are not declared in the model. They are created after the
bulk load.

## Optimize step

A new step, "Building abbreviation lookup", goes in
`internal/iooptimize/abbr.go`. The optimizer goes from 6 to 7 steps:

1. Reparse name strings
2. Normalize vernacular languages
3. Remove orphaned records
4. Extract words
5. Create verification view
6. **Build abbreviation lookup** (new)
7. VACUUM ANALYZE

The order matters:

- It must run after step 1, which rewrites `canonical_stem_id` and
  `canonical_stems`, and after step 3, which removes orphan stems.
- It must run before VACUUM ANALYZE. Index-only scans need the
  visibility map that VACUUM sets, and the planner needs statistics
  for the new table. `vacuum.go` vacuums the whole database, so no
  change is needed there.

Keys are built in Go, everything else in PostgreSQL:

1. Truncate `abbr_lookups`, drop its index, create an UNLOGGED
   staging table `abbr_lookups_staging`.
2. Stream distinct pairs of canonical form and stem ID:

   ```sql
   SELECT c.name, p.canonical_stem_id
   FROM (
     SELECT DISTINCT canonical_id, canonical_stem_id
     FROM name_strings
     WHERE canonical_id IS NOT NULL AND canonical_stem_id IS NOT NULL
       AND virus IS NOT TRUE
   ) p
   JOIN canonicals c ON c.id = p.canonical_id
   WHERE c.name !~ '[×+‘]'
   ```

3. In Go, split each canonical form by spaces. Skip uninomials and
   forms whose first word is not capitalized. Skip epithets whose stem
   is empty (`que`), for which `AbbrKey` returns `""`. For every epithet,
   build the key with `stemmer.AbbrKey(genus, epithet)` and keep its
   position. Copy the rows to staging with pgx `CopyFrom` in batches
   of `cfg.Database.BatchSize`.
4. Move rows to the final table:

   ```sql
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
   SELECT s.abbr_key, min(s.pos), s.genus, s.canonical_stem_id,
          cs.name, coalesce(ds.ds_num, 0)::smallint
   FROM abbr_lookups_staging s
   JOIN canonical_stems cs ON cs.id = s.canonical_stem_id
   LEFT JOIN ds ON ds.canonical_stem_id = s.canonical_stem_id
   GROUP BY s.abbr_key, s.genus, s.canonical_stem_id, cs.name,
            ds.ds_num
   ORDER BY s.abbr_key;
   ```

5. Create the index, drop staging, record the gnparser version:

   ```sql
   CREATE INDEX abbr_lookups_key_idx ON abbr_lookups (abbr_key)
     INCLUDE (genus, canonical_stem, pos, ds_num);
   COMMENT ON TABLE abbr_lookups IS 'gnparser v1.x.y';
   ```

Notes:

- The index is dropped before the load, so the `INSERT` does not
  update it row by row on repeated runs.
- `GROUP BY` collapses duplicates into one row with the lowest `pos`.
  Duplicates come from canonical forms that share a stem (*Carex
  alba*, *Carex albus*) and from repeated epithets (*Bison bison
  bison*).
- `ORDER BY abbr_key` stores the rows of a key next to each other on
  disk. Because the table is truncated and refilled on every run, this
  gives the same result as `CLUSTER` without rewriting the table a
  second time or taking an exclusive lock.
- `ds_num` comes from `name_string_indices`, so the step does not
  depend on the verification view.
- The `INCLUDE` columns allow index-only scans for lookups.
- The step reports the row count and logs the 20 largest keys, so odd
  cases are easy to spot.

## Code changes

gnparser:

- `ent/stemmer/abbr.go`: `AbbrKey()` and its tests.

gndb:

- `go.mod`: gnparser with `AbbrKey` (a `replace` to the local copy
  until it is released).
- `pkg/schema/models.go`: `AbbrLookup` model.
- `pkg/schema/gorm.go`: add `AbbrLookup` to `AllModels()`.
- `pkg/errcode/`: new `OptimizerAbbrLookupError`.
- `internal/iooptimize/abbr.go`: the new step.
- `internal/iooptimize/optimizer.go`: call the step, update the doc
  comment and all `Step N/6` messages to `/7`.
- `internal/iooptimize/vacuum.go`: its comment says "Step 6"; change
  it to Step 7.
- `AGENTS.md`: add `AbbrLookup` to the schema highlights and the
  abbreviation table to the optimization description.
- `CHANGELOG.md`: new entry.

## Tests

- gnparser, unit tests for `stemmer.AbbrKey()`, table-driven:
  - full and abbreviated genus (`Caenorhabditis`, `C.`, `Cae.`);
  - gender endings giving the same key (`alba`, `albus`, `album`);
  - `v` -> `u` (`vulgaris` -> `s-uulgar`);
  - an epithet shorter than 3 characters (left unstemmed);
  - an epithet with a diaeresis;
  - keys from text match keys from `StemCanonical` output.
- gndb, unit tests for splitting canonical forms into rows.
- gndb, integration tests for the optimize step on testdata:
  - a known name is found by its key;
  - no duplicate rows per key and stem;
  - hybrids produce no rows;
  - binomials are found by the key `AbbrKey` builds from text;
  - running the step twice gives the same row count.

## Results on real data

Measured on the local `gnames` database (8.7M canonical stems,
9M canonicals, 32M name-strings, PostgreSQL 18).

First, SQL-only version of the step:

- 9.1M rows, 2.6M distinct keys. Table 825 MB, index 602 MB.
- Build: 90 s for the insert, 2 s for the index.
- Bucket sizes: median 1, 90% 5, 99% 41, 99.9% 229 rows. The largest
  keys are `p-minor` (2,400), `p-gracil`, `c-minor`, `c-gracil` and
  `c-elegans` (1,834).
- Lookup of `c-alb` (1,156 rows): index-only scan, 0.3 ms.
- `{c-elegans,c-briggs,c-remane}` ranks *Caenorhabditis* first with
  all 3 hits.
- 3,811 stems with `×`, `+` or `‘` are skipped. No stem starts with a
  non-capital letter.

Go version (keys from `stemmer.AbbrKey`), on a copy of the needed
columns in `gndb_test`:

- 9,058,538 rows, built in 80–100 s. `ds_num` was not measured: the
  copy of `name_string_indices` failed, so its join cost is missing.
- Compared with keys taken from `canonical_stems` by SQL, all rows
  match except:
  - virus names (~33K rows). Their stored stems are not stemmed
    (`Alphaendornavirus fuphaseoli`) and epithets keep upper case,
    so viruses are now excluded;
  - the epithet `que`, which the stemmer reduces to an empty string.
    `AbbrKey` now returns `""` for it instead of leaking its
    placeholder, and such epithets are skipped;
  - one name, *Ataenius pereirai*, whose `canonical_stem_id` points
    to the stem `A. uesp` — a data problem in `gnames`. The Go build
    gives the correct key, `a-pereira`.
- 219 rows come from junk epithets ending with `-` (`sp-`,
  `undefined-`). They are harmless and left in.
- The stemmer replaces `v` with `u` and `j` with `i`
  (`vulgaris` -> `uulgar`), one more reason to always build keys with
  `stemmer.AbbrKey()`.

## Usage on the gnames side (later)

Keys are always built with gnparser's `stemmer.AbbrKey()`, and gnames
should check that the gnparser version in the table comment matches
its own.

Single abbreviation, with optional extra genus letters:

```sql
SELECT canonical_stem, genus, pos, ds_num
FROM abbr_lookups
WHERE abbr_key = $1                         -- 'c-elegans'
  AND ($2 = '' OR starts_with(genus, $2))   -- '' or 'Cae'
ORDER BY ds_num DESC;
```

The genus prefix is case-sensitive, so gnames normalizes it first
(`cae` -> `Cae`). `starts_with` avoids having to escape `%` and `_`,
as `LIKE` would need.

Several abbreviations from one paper, ranking genera by how many of
them they explain:

```sql
SELECT genus,
       count(DISTINCT abbr_key) AS hits,
       array_agg(DISTINCT canonical_stem) AS names
FROM abbr_lookups
WHERE abbr_key = ANY($1)   -- {c-elegans,c-briggs,c-remane}
GROUP BY genus
ORDER BY hits DESC, sum(ds_num) DESC
LIMIT 10;
```

Winning stems expand to real canonicals with the existing index on
`name_strings.canonical_stem_id`. The exact epithet ending in the
text (*alba* vs *albus*) then gives a ranking bonus.

## Future: taxonomic context

Publications are often mostly about one group (plants, insects, ...).
Candidates should be re-ranked, not filtered, by overlap with that
context. A hard filter would drop host plants in entomology papers,
and context is what resolves cross-code homonyms such as *Morus*
(a plant and a bird).

A second pass would add:

- `clade text[]`: higher taxa from a preferred classification
  (Catalogue of Life first, then GBIF, then the most common one).
- `code smallint`: botanical, zoological, bacterial or unknown.

Context could come from three levels, strongest first: abbreviations
grouped within the paper, full names detected in the same paper, and
the journal or book subject profile. Choosing the preferred
classification needs its own decision, so this is out of scope for
the first step.
