# Optimizer Implementation Plan

## Overview
Implement the optimizer component for GNdb - the final step in creating a GNverifier database. The optimizer prepares the database for fast name verification by reparsing names with the latest gnparser, creating canonical forms, building word indexes for fuzzy matching, and creating materialized views.

**Based on**: ~/tmp/gndb spec-kit branch implementation (~6,649 lines of code)
**Complexity**: Large - 6 sequential phases with concurrent processing
**Estimated files**: ~15 new files in internal/iooptimize/

## Current State
✅ Completed:
- Schema management (create, migrate)
- Data population (SFGA import)
- Database operator (pgx connection pooling)
- Configuration system
- Logging infrastructure
- End-to-end populate tests

❌ Missing:
- Optimizer implementation (all phases)
- Optimize command in CLI
- Integration tests for optimizer

## Optimizer Workflow (6 Phases)

### Phase 1: Reparse Name Strings
**Purpose**: Update all name_strings with latest gnparser algorithms

**Implementation**:
- Load all name_strings from database
- Parse in parallel using gnparser (concurrent workers)
- Extract canonical forms (simple, full, stem)
- Update name_strings table with:
  - canonical_id, canonical_full_id, canonical_stem_id
  - cardinality, year, parse_quality
  - bacteria, virus, surrogate flags
- Insert new canonical records (canonicals, canonical_fulls, canonical_stems)

**Key files** (from spec-kit):
- `reparse.go` - Main orchestration (~400 lines)
- `reparse_batch.go` - Batch processing logic (~300 lines)
- `progress.go` - Progress bar helper (~50 lines)

**Complexity**: High
- Concurrent parsing with worker pools
- Batch insertions with CopyFrom
- Progress tracking with pb library
- ~1M+ name strings to process

**Dependencies**:
- `github.com/gnames/gnparser` - Scientific name parsing
- `github.com/gnames/gnuuid` - UUID generation for canonicals
- `github.com/cheggaaa/pb/v3` - Progress bars
- `golang.org/x/sync/errgroup` - Concurrent error handling

---

### Phase 2: Normalize Vernacular Languages
**Purpose**: Standardize language codes and names in vernacular_string_indices

**Implementation**:
- Load all vernacular_string_indices
- Normalize language codes:
  - Convert 2-letter codes to 3-letter (ISO 639-2/3)
  - Validate 3-letter codes
  - Derive codes from language names
  - Set full language names
- Batch update using temp table (PostgreSQL optimization)
- Convert all lang_code to lowercase

**Key files**:
- `vernacular.go` - Language normalization (~350 lines)

**Complexity**: Medium
- In-memory normalization
- Batch updates via temp table
- Language code conversions

**Dependencies**:
- `github.com/gnames/gnfmt/gnlang` - Language code utilities

---

### Phase 3: Remove Orphaned Records
**Purpose**: Clean up unreferenced canonical forms and name strings

**Implementation**:
- Remove name_strings not in name_string_indices (LEFT JOIN)
- Remove canonicals not in name_strings
- Remove canonical_fulls not in name_strings
- Remove canonical_stems not in name_strings
- Report deletion counts

**Key files**:
- `orphans.go` - Orphan removal (~200 lines)

**Complexity**: Low
- Simple DELETE queries with LEFT JOINs
- Sequential execution

---

### Phase 4: Extract and Link Words
**Purpose**: Build word index for fuzzy matching (gnverifier's fuzzy search)

**Implementation**:
- Truncate words and word_name_strings tables
- Load all name_strings with canonical_id
- Parse names to extract words (concurrent)
- Deduplicate words and word-name linkages
- Bulk insert words table
- Bulk insert word_name_strings table (many-to-many)

**Key files**:
- `words.go` - Word extraction and insertion (~500 lines)

**Complexity**: High
- Concurrent parsing
- Word deduplication
- Bulk insertions with CopyFrom
- Many-to-many relationship management

**Dependencies**:
- `github.com/gnames/gnparser` - Word extraction from parsed names

---

### Phase 5: Create Verification View
**Purpose**: Build materialized view for fast verification queries

**Implementation**:
- Drop existing verification view (if exists)
- Create materialized view with SQL:
  ```sql
  CREATE MATERIALIZED VIEW verification AS
  WITH taxon_names AS (...)
  SELECT ... FROM name_string_indices nsi
  JOIN name_strings ns ON ...
  LEFT JOIN canonicals c ON ...
  -- Complex join with classification, ranks, etc.
  ```
- Create indexes on:
  - canonical_id
  - name_string_id  
  - year
- Report record count

**Key files**:
- `views.go` - Materialized view creation (~150 lines, but complex SQL)

**Complexity**: Medium
- Large complex SQL statement
- Index creation
- Materialized view management

**SQL Complexity**: High - denormalizes data from multiple tables for performance

---

### Phase 6: Vacuum Analyze
**Purpose**: Reclaim space and update query planner statistics

**Implementation**:
- Execute `VACUUM ANALYZE` on entire database
- Must run outside transaction
- Updates PostgreSQL statistics for optimal query plans

**Key files**:
- `vacuum.go` - VACUUM execution (~40 lines)

**Complexity**: Low
- Single SQL command
- No transaction needed

---

## Implementation Structure

```
internal/iooptimize/
├── optimizer.go          # Main Optimize() orchestration
├── errors.go            # Error definitions (Step1Error, etc.)
├── reparse.go           # Phase 1: Name reparsing
├── reparse_batch.go     # Phase 1: Batch processing
├── progress.go          # Progress bar helpers
├── vernacular.go        # Phase 2: Language normalization
├── orphans.go           # Phase 3: Orphan removal
├── words.go             # Phase 4: Word extraction
├── views.go             # Phase 5: Materialized view
└── vacuum.go            # Phase 6: VACUUM ANALYZE

cmd/
└── optimize.go          # CLI command (call Optimizer.Optimize)

Tests:
internal/iooptimize/
├── reparse_test.go
├── reparse_batch_test.go
├── reparse_bench_test.go
├── reparse_large_test.go
├── vernacular_test.go
├── orphans_test.go
├── words_test.go
├── views_test.go
└── vacuum_test.go
```

## Dependencies to Add

Check and add if missing in go.mod:
```
github.com/cheggaaa/pb/v3       # Progress bars
github.com/dustin/go-humanize   # Number formatting (1,234,567)
github.com/gnames/gnparser      # Name parsing
github.com/gnames/gnuuid        # UUID generation
github.com/gnames/gnfmt/gnlang  # Language codes
golang.org/x/sync/errgroup      # Concurrent error handling
```

## CLI Integration

### Command: `gndb optimize`

**Flags**:
- `--jobs` - Number of concurrent workers (default: CPU count)
- `--batch-size` - Batch size for bulk operations (default: 50000)

**Prerequisites Check**:
- Database must exist (has tables)
- Database must be populated (has data in name_strings)
- Implemented in `db.Operator.CheckReadyForOptimization()`

**Example**:
```bash
gndb optimize
gndb optimize --jobs 16
gndb optimize --batch-size 100000
```

## Database Schema Updates

**Tables Used** (already exist from schema):
- name_strings - Updated with canonical IDs
- canonicals - Inserted during reparse
- canonical_fulls - Inserted during reparse
- canonical_stems - Inserted during reparse
- vernacular_string_indices - Updated language codes
- words - Truncated and repopulated
- word_name_strings - Truncated and repopulated

**Views Created**:
- verification (materialized view) - Main query optimization

**Indexes Created** (on verification view):
- idx_verification_canonical_id
- idx_verification_name_string_id
- idx_verification_year

## Performance Characteristics

**Timing** (approximate, depends on data size):
- Phase 1 (Reparse): 5-30 minutes (1M names, depends on CPU cores)
- Phase 2 (Vernacular): 1-5 minutes
- Phase 3 (Orphans): < 1 minute
- Phase 4 (Words): 10-40 minutes (depends on name count and CPU)
- Phase 5 (Views): 2-10 minutes (depends on data size)
- Phase 6 (Vacuum): 1-5 minutes

**Total**: ~20-90 minutes for full optimization (varies widely)

**Memory Usage**:
- Moderate - batch processing limits memory
- Peak during word deduplication (in-memory maps)
- Configurable via batch_size

**Concurrency**:
- Phase 1: Parallel parsing (jobs * workers)
- Phase 4: Parallel word extraction (jobs * workers)
- Other phases: Sequential

## Implementation Strategy

### Step 1: Foundation Files
1. Create `internal/iooptimize/optimizer.go` - Main orchestration
2. Create `internal/iooptimize/errors.go` - Error definitions
3. Create `internal/iooptimize/progress.go` - Progress helpers
4. Create `cmd/optimize.go` - CLI command

### Step 2: Phase-by-Phase Implementation
Implement phases in order (each depends on previous):

1. **Phase 6 (Vacuum)** - Simplest, test infrastructure
2. **Phase 3 (Orphans)** - Simple queries, no parsing
3. **Phase 2 (Vernacular)** - Medium complexity
4. **Phase 5 (Views)** - SQL-heavy, no parsing
5. **Phase 1 (Reparse)** - Complex, concurrent parsing
6. **Phase 4 (Words)** - Complex, concurrent parsing

### Step 3: Testing
1. Unit tests for each phase
2. Integration test with small dataset
3. End-to-end test with VASCAN (from populate tests)
4. Performance benchmarks (reparse_bench_test.go)

### Step 4: Polish
1. Add progress bars
2. Add user-friendly messages
3. Add error handling
4. Documentation

## Testing Strategy

### Unit Tests
- Test each phase independently
- Mock database responses
- Test error cases

### Integration Tests
- Require PostgreSQL (skip with -short)
- Use VASCAN test data (source 1002)
- Test full optimize workflow:
  1. Create schema
  2. Populate with test data
  3. Run optimize
  4. Verify results (canonical counts, word counts, view exists)

### Benchmark Tests
- `reparse_bench_test.go` - Measure parsing performance
- Compare concurrent vs sequential
- Test different batch sizes

### Large Tests
- `reparse_large_test.go` - Test with large datasets
- Skip by default (requires special test data)

## Key Challenges

1. **Concurrency Management**
   - Use errgroup for coordinated cancellation
   - Manage worker pools (don't spawn unlimited goroutines)
   - Progress tracking across workers

2. **Memory Management**
   - Batch processing (don't load all names at once)
   - Word deduplication (maps can grow large)
   - Configurable batch sizes

3. **Progress Reporting**
   - pb library for progress bars
   - Need total counts before processing
   - Update frequency (not every record)

4. **SQL Complexity**
   - Verification view SQL is complex
   - Must match gnidump exactly for compatibility
   - Test thoroughly

5. **Error Handling**
   - Wrap errors with context
   - User-friendly messages
   - Preserve detailed logs

## Success Criteria

✅ All 6 phases implemented and tested
✅ CLI command works end-to-end
✅ Progress bars display correctly
✅ Integration test passes with VASCAN data
✅ Verification view created with correct structure
✅ Word tables populated correctly
✅ Canonicals created during reparse
✅ Performance acceptable (< 2 hours for large datasets)
✅ Code passes linter
✅ Documentation complete

## Next Steps

1. Review this plan - discuss any concerns
2. Create feature branch: `git checkout -b 5-optimizer`
3. Start with Step 1 (foundation files)
4. Implement phases one at a time
5. Test each phase before moving to next
6. Create PR when complete

## References

- Original implementation: ~/tmp/gndb (spec-kit branch)
- gnidump source: https://github.com/gnames/gnidump
- PostgreSQL VACUUM docs: https://www.postgresql.org/docs/current/sql-vacuum.html
- gnparser: https://github.com/gnames/gnparser
