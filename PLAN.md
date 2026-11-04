# Plan: ION misspellings and shared fuzzy matching

<!-- vim-markdown-toc GFM -->

* [Goals](#goals)
* [Background](#background)
  * [Why ION misspellings hurt
    verification](#why-ion-misspellings-hurt-verification)
  * [Why they must not be deleted](#why-they-must-not-be-deleted)
* [Measurement](#measurement)
  * [How it was measured](#how-it-was-measured)
  * [Results](#results)
  * [Samples](#samples)
  * [Known false positives](#known-false-positives)
* [Design](#design)
  * [Where the exact match comes from](#where-the-exact-match-comes-from)
  * [Flag on canonicals](#flag-on-canonicals)
  * [Rule for flagging](#rule-for-flagging)
  * [Optimizer step](#optimizer-step)
  * [Lifecycle](#lifecycle)
  * [Changes outside gndb](#changes-outside-gndb)
* [Moving fuzzy matching to gnlib](#moving-fuzzy-matching-to-gnlib)
* [Steps](#steps)
* [Open questions](#open-questions)

<!-- vim-markdown-toc -->

## Goals

1. Stop misspelled ION names from blocking fuzzy matching in GNverifier.
2. Keep these names available for gndict, which uses them for name-finding
   in GNfinder.
3. Move the trie and edit distance rules of gnmatcher to gnlib, so gndb
   and gnmatcher use exactly the same fuzzy matching.

## Background

ION (data source 168) has names collected from closed literature by many
people. It has many unique names, but also OCR and human errors.

### Why ION misspellings hurt verification

gnmatcher tries an exact match first and runs fuzzy matching only if
the exact match fails. If ION has `Eriopvgodes imbecilla` (an OCR error),
this string matches ION exactly, and the user never gets
`Eriopygodes imbecilla` from a reliable source.

### Why they must not be deleted

OCR errors repeat in texts. gndict (dictionaries for GNfinder) reads all
ION names from the database since commit `f804d9b`, so GNfinder can find
such misspellings. GNverifier then can fuzzy-match them to correct names.
So misspellings are useful for name-finding, but harmful for exact
matching.

## Measurement

### How it was measured

A throwaway program (gndict session, 2026-11-04) mirrored gnmatcher:

* Reliable sources: `is_curated = true`, plus GBIF (11), EOL (12) and
  Ruhoff 1980 (206). It is the same set gndict uses.
* A trie of stemmed canonical forms (`canonical_stems`) from reliable
  sources, built with `github.com/dvirsky/levenshtein` (`NewMinTree`),
  as gnmatcher does in `internal/io/trie/trie.go`.
* ION-only canonicals: ION canonicals that are not in any reliable source.
* For every ION-only canonical:
  * `stem-shared`: its stem is also a stem of a reliable name. Exact stem
    matching already returns reliable names, so it does not block
    anything.
  * `fuzzy`: `trie.FuzzyMatches(stem, 1)` finds a reliable stem, and both
    stem and canonical pass gnmatcher's `fuzzy.EditDistance`
    (`internal/ent/fuzzy/levenshtein.go`, includes `checkED` for words
    shorter than 5 letters). Such a name blocks fuzzy matching.
  * `unique`: nothing reliable nearby. These are the valuable unique ION
    names.

Loading reliable canonicals took 54s, building the trie 42s, matching
710k names 55s.

### Results

| Data                       | Count     |
|----------------------------|-----------|
| reliable canonicals        | 7,981,530 |
| reliable stems in the trie | 7,764,797 |
| ION canonicals             | 3,951,558 |
| ION-only canonicals        | 710,314   |

| ION-only   | fuzzy (blocks) | stem-shared | unique  |
|------------|----------------|-------------|---------|
| 1 word     | 27,126         | 0           | 19,447  |
| 2 words    | **134,930**    | 31,949      | 407,525 |
| 3+ words   | **14,811**     | 3,188       | 71,338  |

About 150k binomials and trinomials block fuzzy matching.

Uninomials should be skipped:

* gnmatcher does not fuzzy-match uninomials by default
  (`WithUninomialFuzzyMatch` is false), so they block nothing.
* Their samples are noisy: many are real names of different ranks
  (`Echinasterinae` / `Echinasteridae`, `Monarditina` / `Monarditini`),
  or real different genera (`Attax` / `Atrax`).

### Samples

ION-only name → reliable name.

Binomials, mostly real errors:

* OCR: `Hipposideros ator → ater`, `Tapes aurcus → aureus`,
  `Sarotherodon mossamblcus → mossambicus`,
  `Lopholaimus antarctlcus → antarcticus`
* letters lost or added: `Typhlodromus occiddentalis → occidentalis`,
  `Larus glauscescens → glaucescens`, `Polyrhacis → Polyrhachis`
* transpositions: `Tribolium castanuem → castaneum`,
  `Deudorix staudingeir → staudingeri`
* trinomials: `Coturnlx coturnix japonica`, `Canis lupus nubllus`,
  `Scorpaena occi pitalis → occipitalis`

### Known false positives

By eye roughly 5% of binomial samples are doubtful:

* possibly different real names: `Botys communis` (`Botys` is a real
  genus), `Ostrea caillaudi → Astrea`, `Oncidium coriacea → Onchidium`,
  `Monograptus corona → colonus`, `Cardiophorus wahlbergii → sahlbergi`,
  `Helix meadii → media`
* the reliable source has the misspelling and ION is correct:
  `Brevipalpus lewsi → lewis`, `Pseudione nephropis → nephropsi`. Flagging
  them is acceptable: fuzzy matching still returns the reliable record.

The `kind` column of the measurement (ocr, substitution, indel,
transposition, space) is approximate: `space` mostly means both words
differ (genus letter plus gender ending), not a split word.

## Design

### Where the exact match comes from

gnmatcher does not use the `verification` view for matching:

* bloom filters are made from all ids of `canonicals` and
  `canonical_stems` (`internal/io/bloom/db_filters.go`);
* the stems key-value store is made from `canonical_stems` joined with
  `name_strings`, `canonicals` and `name_string_indices`
  (`internal/io/trie/stemskv.go`);
* the trie is made from all names of `canonical_stems`
  (`internal/io/trie/trie.go`).

So hiding ION rows only from the `verification` view is not enough: the
canonical would still match exactly.

### Flag on canonicals

Add a boolean column to `canonicals`, for example `maybe_misspelling`
(default false).

A flagged canonical is in ION only, by definition (see the rule below).
So every `name_strings` row with this canonical belongs to ION, and the
flag cannot hide a name of another source. Nothing is moved or deleted:

* ION rows stay in `name_string_indices`, gndict keeps reading them as
  it does now;
* orphan removal is not involved;
* the flag is derived data, optimize recomputes it on every run.

Why `canonicals` and not `canonical_stems`: `canonical_stems` is a
separate table of stem strings, linked through
`name_strings.canonical_stem_id`. Several canonicals can share a stem
(gender variants: `Aus alba`, `Aus albus`), and only some of them may be
flagged. A flag on a stem would be ambiguous, a flag on a canonical is
exact, because the rule decides per canonical.

### Rule for flagging

Flag a canonical if:

1. it is used only by ION, no reliable source has it;
2. it has 2 or more words;
3. its stem is not a stem of any reliable name;
4. the trie of reliable stems gives a match with edit distance 1
   (gnmatcher default `MaxEditDist`), and both the stem and the canonical
   pass gnmatcher's `EditDistance` checks.

Use the same code as gnmatcher (see
[Moving fuzzy matching to gnlib](#moving-fuzzy-matching-to-gnlib)), so
the flag matches exactly the cases where gnmatcher would otherwise run
fuzzy matching.

### Optimizer step

Add a step to `internal/iooptimize` after reparsing (step 1), because it
needs current canonical forms and stems. The step:

1. resets the flag on all canonicals;
2. builds the trie of reliable stems;
3. sets the flag on canonicals that match the rule.

Current steps are:

1. Reparse all name_strings
2. Normalize vernacular language codes
3. Remove orphaned records
4. Extract and link words for fuzzy matching
5. Create verification materialized view
6. Build abbreviation lookup table
7. VACUUM ANALYZE

The new step can go after step 3, so it does not look at orphans. Run it
only if ION is in the database, report the number of flagged canonicals.
A dry-run mode that writes flagged pairs (ION name, reliable name) to a
file would help to review results.

### Lifecycle

An ION update:

1. `gndb populate -s 168` imports ION as any other source.
2. `gndb optimize` reparses names, removes orphans, recomputes the flag,
   builds words, the `verification` view, abbreviations, vacuums.
3. gnmatcher rebuilds its bloom filters, stems store and trie, and skips
   flagged canonicals.
4. gndict reads ION names as before, flagged ones included, so GNfinder
   still finds OCR spellings.

An update of a reliable source works the same way: the flag is
recomputed, so a misspelling that became a real name in a reliable
source is unflagged by the next optimize.

### Changes outside gndb

gnmatcher has to skip flagged canonicals where it builds its data:

1. bloom filter of canonicals: `SELECT id FROM canonicals WHERE NOT
   maybe_misspelling`;
2. bloom filter of stems and the trie: only stems that have at least one
   unflagged canonical, through `name_strings.canonical_stem_id` and
   `name_strings.canonical_id`. A stem shared by a flagged and an
   unflagged canonical stays;
3. stems key-value store (`stemskv.go`): it already joins `canonicals`,
   add `WHERE NOT c.maybe_misspelling`.

gnmatcher has to work with databases without the column too, or require
a gndb version that has it.

The `verification` view can stay as it is: GNverifier looks up records
only for canonicals that gnmatcher returns.

gndict needs no changes.

## Moving fuzzy matching to gnlib

gnmatcher has the code in `internal`, so gndb cannot import it:

* `internal/ent/fuzzy/levenshtein.go`: `EditDistance(s1, s2, relax)` and
  `checkED`, constants `charsPerED = 4`, `maxEditDistance = 6`. Checks
  are applied only to the second string.
* `internal/io/trie/trie.go`: building `levenshtein.MinTree` from sorted
  stems, `FuzzyMatches(stem, maxEditDist)`, saving and loading the trie.
* the order of checks in `internal/ent/matcher/fuzzy.go` (`matchFuzzy`):
  stem edit distance first (`EditDistance(stemMatch, stem)`), then
  canonical edit distance for every match item
  (`EditDistance(inputCanonical, matchCanonical)`).

Proposal:

1. Move `EditDistance` and `checkED` to gnlib (for example
   `gnlib/ent/fuzzy`) with their tests.
2. Move the trie wrapper (build from sorted strings, fuzzy matches,
   save/load) to gnlib, so gndb and gnmatcher build the trie the same way.
3. Switch gnmatcher to gnlib; its tests must not change.
4. Use the gnlib code in gndb.

Dependencies: `github.com/dvirsky/levenshtein` and
`github.com/gnames/levenshtein` (`ent/editdist`).

## Steps

1. Move fuzzy edit distance and the trie to gnlib, switch gnmatcher,
   release gnlib.
2. In gndb, write the flagging code with a dry-run report; compare its
   numbers with the measurement above (about 150k flagged canonicals).
3. Review a sample of the dry-run report.
4. Add `maybe_misspelling` to `canonicals` (schema, migration), add the
   optimizer step.
5. Make gnmatcher skip flagged canonicals in bloom filters, the stems
   store and the trie.
6. Rebuild a test database, check with GNverifier that
   `Eriopvgodes imbecilla` fuzzy-matches `Eriopygodes imbecilla`, and that
   unique ION names still match exactly.

## Open questions

* Name of the column, and whether other unreliable sources need the same
  treatment later.
* The set of reliable sources is defined by gndict. Should gndb have it
  in configuration instead of hard-coded ids (2 is Wikispecies, curated
  but has junk; 11 GBIF, 12 EOL, 206 Ruhoff are not curated but used)?
* Edit distance 2 (`MaxEditDist` 2 is allowed in gnmatcher) was not
  measured.
* Should a dry-run report be kept as a regular command option?
