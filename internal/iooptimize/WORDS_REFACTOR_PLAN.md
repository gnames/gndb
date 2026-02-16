# Words Pipeline Refactor Plan

## Problem

`parseNamesForWords()` accumulates all `Word` and `WordNameString`
records in memory maps before writing anything to the database.
For large databases (tens of millions of name_strings), the
`wnsMap` grows to many gigabytes and depletes memory.

## Goal

Follow the gnidump approach: save `word_name_strings` in batches
as we go, only keep `wordsMap` in memory for global dedup.

## Key Insight

- `wordsMap` (unique words): Must stay in memory for global
  dedup. This is bounded — unique words are far fewer than
  word-name links. Acceptable memory usage.
- `wnsMap` (word-name links): Does NOT need global dedup.
  Per-batch dedup is sufficient. This is the memory hog —
  one entry per (word, name_string) pair.

## Refactored Pipeline

### Stage 1: Stream names (unchanged)

`loadNamesForWords` streams rows from DB → `chIn` channel.
No changes needed.

### Stage 2: Workers parse and extract (unchanged)

Workers read from `chIn`, parse with gnparser, extract words
and word-name links, send `wordResult` → `chOut`. No changes
needed.

### Stage 3: Collector — batched writes (NEW)

Replace the current collector (which accumulates into two maps)
with a collector that:

1. Maintains `wordsMap` across the entire run (global dedup
   for words).
2. Accumulates `wordNames` into a slice until it reaches
   batch size (e.g., 50,000).
3. When batch is full:
   - Dedup `wordNames` within the batch (same as gnidump's
     `uniqWordNameString`).
   - Write the batch to DB via `saveWordNameStrings`.
   - Reset the slice.
4. After `chOut` is drained, flush the remaining `wordNames`.

### Stage 4: Save words (mostly unchanged)

After the pipeline completes, convert `wordsMap` to slice and
call `saveWords`. Same as before.

## Changes Required

### File: `words.go`

1. **`parseNamesForWords` return type change:**
   - Currently returns `(map[string]schema.Word,
     map[string]schema.WordNameString, error)`.
   - Change to return `(map[string]schema.Word, error)` since
     word-name links are saved inline.

2. **Collector goroutine rewrite:**
   - Remove `wnsMap`.
   - Add batch accumulation + inline saving for word-name links.
   - The collector needs access to `pool` and `cfg` for saving.

3. **`extractWords` simplification:**
   - Remove `wnsMap` handling, slice conversion, and
     `saveWordNameStrings` call (moved into collector).
   - Only handle `wordsMap` → slice → `saveWords`.

4. **New helper `deduplicateWordNames`:**
   - Takes `[]schema.WordNameString`, returns deduplicated slice.
   - Mirrors gnidump's `uniqWordNameString`.

## Error Handling

The collector goroutine currently runs outside `errgroup`.
To handle save errors from inline writes, either:

- **(A)** Move collector into `errgroup` and propagate errors.
- **(B)** Keep collector as goroutine, capture error in a
  variable, check after `<-collectDone`.

Option (A) is cleaner. The collector becomes another `g.Go`
function. If it fails, the errgroup context cancels the
pipeline.

## Memory Profile After Refactor

- `wordsMap`: Same as before — bounded by unique word count.
- `wordNames` batch: Bounded by batch size (50K entries).
- Peak memory: `wordsMap` + one batch of `wordNames` — orders
  of magnitude less than the current approach.
