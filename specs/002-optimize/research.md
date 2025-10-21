# Research for Optimize Database Performance

## Decision: Follow `gnidump rebuild` Logic

**Rationale**: The `gndb optimize` command replicates the production-tested `gnidump rebuild` workflow. This approach is proven in production use for gnverifier compatibility and ensures the database is optimized exactly as expected by gnverifier's query patterns.

**Alternatives considered**: A new optimization implementation from scratch was considered but rejected in favor of the proven gnidump approach to ensure compatibility and leverage production battle-testing.

**Reference**: gnidump implementation at `${HOME}/code/golang/gnidump/`
- Main orchestration: `internal/io/buildio/buildio.go`
- CLI command: `cmd/rebuild.go`

## Complete 5-Step Workflow from `gnidump rebuild`

Analysis of gnidump source code reveals the following 5-step process (order preserved from production):

### Step 1: Reparse Names (`db_reparse.go`)
- **Purpose**: Update all name_strings with latest gnparser algorithms and cache results
- **Operation**:
  - Load all name_strings from database
  - Parse using gnparser with concurrent workers (gnidump uses 50)
  - Update database: canonical_id, canonical_full_id, canonical_stem_id, bacteria, virus, surrogate, parse_quality
  - Insert missing canonical records into canonicals, canonical_stems, canonical_fulls tables
  - **Cache parsed results in kvSci (key-value store) keyed by UUID v5** for use in Step 4
- **Why needed**: Parser algorithms improve over time; reparsing applies latest classification logic
- **gndb adaptation**: Use Config.JobsNumber for worker count (defaults to runtime.NumCPU()) instead of hardcoded 50

### Step 2: Fix Vernacular Language (`db_vern.go`)
- **Purpose**: Normalize language codes using gnlang library
- **Operation**:
  - Convert 2-letter codes to 3-letter ISO codes
  - Store original in language_orig before normalization
  - Make all codes lowercase
- **Why needed**: Consistent language codes enable reliable vernacular lookups
- **gndb adaptation**: Use Config.JobsNumber for parallel workers

### Step 3: Remove Orphans (`db_views.go`)
- **Purpose**: Clean up unreferenced records
- **Operations** (in order):
  1. Delete orphan name_strings not in name_string_indices
  2. Delete orphan canonicals not referenced by name_strings
  3. Delete orphan canonical_fulls not referenced by name_strings
  4. Delete orphan canonical_stems not referenced by name_strings
- **Why needed**: Reduces database size and improves query performance
- **Note**: Done after reparsing (not before) to maintain exact gnidump workflow order for safety

### Step 4: Create Words (`words.go`)
- **Purpose**: Extract individual words for fuzzy matching
- **Operation**:
  - Load name_strings from database (names with canonical_id)
  - Parse names using parserpool with WithDetails(true) to get Words field
  - Extract words from parsed.Words for each name
  - Store in words table with normalized and modified forms
  - Create word_name_strings junction table linking words to names and canonicals
  - Process in batches (use Config.Import.BatchSize)
- **Why needed**: Enables word-level fuzzy matching in gnverifier
- **gndb decision**: Direct parsing (like gnidump) - gnparser is fast, avoids KV store overhead

### Step 5: Create Verification Materialized View (`db_views.go`)
- **Purpose**: Denormalize data for fast verification queries
- **Operation**:
  - Drop existing `verification` materialized view if exists
  - Create single `verification` materialized view (see data-model.md for SQL)
  - Create 3 indexes: canonical_id, name_string_id, year
- **Why needed**: Pre-joins tables for O(1) verification lookups instead of expensive runtime joins

## Additional gndb Enhancements

**VACUUM ANALYZE**: gnidump rebuild does NOT run VACUUM ANALYZE. gndb optimize will add this per FR-004 requirement as Step 6.

**Configurable Concurrency**: gnidump hardcodes 50 workers for reparsing. gndb will use Config.JobsNumber to adapt to diverse hardware (from laptops to servers). This supports the use case where gndb is used by many people with varying hardware capabilities, unlike gnidump which was designed for single-user scenarios.

**User-Friendly Output**: Per Constitution Principle X, gndb optimize must add colored terminal output for progress updates to STDOUT. gnidump does not have this - it's a gndb enhancement following the dual-channel communication principle (user messages to STDOUT, technical logs to STDERR).

**Simplified Cache Strategy**: Unlike gnidump which uses kvSci cache, gndb optimize follows a simpler approach - Step 1 (reparse) uses a minimal cache for canonical lookups, and Step 4 (words) re-parses names directly using parserpool. This avoids KV store overhead since gnparser is already highly optimized. The cache at `~/.cache/gndb/optimize/` is used only for Step 1 canonical form storage, not for word extraction. Note: Vernacular names are not parsed, only language-normalized, so no kvVern needed.
