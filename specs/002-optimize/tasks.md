# Tasks: Optimize Database Performance - Batch Reparsing

**Input**: Design documents from `/home/dimus/code/golang/gndb/specs/002-optimize/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/, quickstart.md
**Context**: Optimize Step 1 (reparse names) to use temporary tables and batch updates instead of row-by-row updates for 100M+ name_strings records

## Problem Statement

Current implementation in `internal/iooptimize/reparse.go` updates name_strings **one row at a time** in the `updateNameString()` function. For databases with 100 million rows, this creates:
- 100M+ individual UPDATE transactions
- Significant database I/O overhead
- Slow performance (hours to days for large datasets)

## Solution Approach

Refactor Step 1 reparsing to use **temporary tables + batch operations**:
1. Create temporary table with new parsed values
2. Bulk insert parsed results into temp table (pgx CopyFrom)
3. Single batch UPDATE from temp table to name_strings
4. Batch INSERT canonical records (deduplicated)
5. Drop temporary table

**Performance goal**: Process 100M rows in hours instead of days.

## Execution Flow
```
1. Load existing tasks.md structure (if exists)
2. Preserve existing T001-T023 tasks from initial implementation
3. Add new tasks T024-T035 for batch reparsing optimization
4. Apply TDD: tests before implementation
5. Maintain [P] markers for parallelizable tasks
```

## Phase 3.1: Setup (EXISTING - Completed)
- [x] T001 Create project structure per implementation plan
- [x] T002 Initialize Go project with dependencies
- [x] T003 [P] Configure linting and formatting tools

## Phase 3.2: Tests First - Initial Implementation (EXISTING - Completed)
- [x] T004 [P] Contract test for Optimizer.Optimize() in pkg/lifecycle/optimizer_test.go
- [x] T005 [P] Integration test for Step 1 (reparse) in internal/iooptimize/reparse_test.go
- [x] T006 [P] Integration test for Step 2 (vernacular) in internal/iooptimize/vernacular_test.go
- [x] T007 [P] Integration test for Step 3 (orphans) in internal/iooptimize/orphans_test.go
- [x] T008 [P] Integration test for Step 4 (words) in internal/iooptimize/words_test.go
- [x] T009 [P] Integration test for Step 5 (views) in internal/iooptimize/views_test.go
- [x] T010 [P] Integration test for Step 6 (vacuum) in internal/iooptimize/vacuum_test.go
- [x] T011 Write end-to-end integration test in cmd/gndb/optimize_integration_test.go

## Phase 3.3: Core Implementation - Initial (EXISTING - Completed)
- [x] T012 [P] Implement error types in internal/iooptimize/errors.go
- [x] T013 Implement reparse.go Step 1 (SLOW - row-by-row updates)
- [x] T014 Implement vernacular.go Step 2
- [x] T015 Implement orphans.go Step 3
- [x] T016 Implement words.go Step 4
- [x] T017 Implement views.go Step 5
- [x] T018 Implement vacuum.go Step 6
- [x] T019 Wire optimizer.go Optimize() method to call Steps 1-6 sequentially

## Phase 3.4: CLI Integration (EXISTING - Completed)
- [x] T020 Create cmd/gndb/optimize.go command with flags
- [x] T021 Wire optimize command into cmd/gndb/main.go
- [x] T022 Add CLI integration test in cmd/gndb/optimize_integration_test.go

## Phase 3.5: Polish (EXISTING - Completed)
- [x] T023 Add colored progress output and error documentation

---

## Phase 3.6: Batch Reparsing Optimization (NEW - IN PROGRESS)

**Strategy**: Filter-then-batch approach that leverages existing `parsedIsSame()` optimization:
1. Parse all names using existing concurrent worker pipeline
2. **Filter using `parsedIsSame()`** - only collect CHANGED names to temp table
3. Bulk insert ONLY changed names (1M-50M rows depending on scenario, NOT all 100M)
4. Single batch UPDATE for changed rows only
5. Single batch INSERT for unique canonicals from changed rows

**Real-World Scenarios**:
- First optimization (100% change): 100M rows → 100M in temp table → 15 min (vs 90 min current)
- Partial update (50% change): 100M rows → 50M in temp table → 10 min (vs 60 min current)  
- Re-optimization (1-10% change): 100M rows → 1-10M in temp table → 5-7 min (vs 15-30 min current)

**Performance Target**: 3-6x faster than current row-by-row approach across all scenarios

### Test-First Tasks (TDD Red Phase)

- [x] T024 [P] Write performance benchmark test for batch reparse in internal/iooptimize/reparse_bench_test.go
  - **File**: `internal/iooptimize/reparse_bench_test.go`
  - **Purpose**: Benchmark comparing row-by-row vs batch approach
  - **Test cases**: 1K, 10K, 100K, 1M rows to validate scalability
  - **Success criteria**: Batch approach 10x+ faster than row-by-row
  - **Run**: `go test -bench=BenchmarkReparse -benchmem -run=^$`
  - **Expected**: FAIL (batch implementation doesn't exist yet)

- [x] T025 [P] Write unit test for temp table creation in internal/iooptimize/reparse_batch_test.go
  - **File**: `internal/iooptimize/reparse_batch_test.go`
  - **Purpose**: Verify temp table schema matches name_strings structure
  - **Test cases**: 
    - Temp table created successfully
    - Temp table has correct columns (canonical_id, bacteria, virus, etc.)
    - Temp table dropped after operation
  - **Expected**: FAIL (createReparseTempTable() doesn't exist) ✓ VERIFIED

- [x] T026 [P] Write unit test for bulk insert into temp table in internal/iooptimize/reparse_batch_test.go
  - **File**: `internal/iooptimize/reparse_batch_test.go` (add to existing file)
  - **Purpose**: Verify pgx CopyFrom correctly inserts ONLY CHANGED parsed results
  - **Test cases**:
    - 1000 changed names inserted (not unchanged names)
    - NULL values handled correctly (canonical_full_id, surrogate, etc.)
    - Progress tracking logged to stderr
    - Verify unchanged names NOT in temp table (filtered by parsedIsSame)
  - **Expected**: FAIL (bulkInsertToTempTable() doesn't exist) ✓ VERIFIED

- [x] T027 [P] Write unit test for batch UPDATE from temp table in internal/iooptimize/reparse_batch_test.go
  - **File**: `internal/iooptimize/reparse_batch_test.go` (add to existing file)
  - **Purpose**: Verify single UPDATE statement updates all name_strings from temp table
  - **Test cases**:
    - 1000 name_strings updated in single transaction
    - All fields updated correctly (canonical_id, bacteria, virus, parse_quality, year, cardinality)
    - Original records unchanged if not in temp table
  - **Expected**: FAIL (batchUpdateNameStrings() doesn't exist) ✓ VERIFIED

- [x] T028 Write integration test for 100M row scenario in internal/iooptimize/reparse_large_test.go
  - **File**: `internal/iooptimize/reparse_large_test.go`
  - **Purpose**: Validate performance with 100M+ rows (use test database)
  - **Test cases**:
    - Create test DB with 100K rows (scaled test)
    - Run batch reparse workflow end-to-end
    - Verify memory usage stays under 2GB
    - Verify time scales linearly (100K in minutes, not hours)
  - **Build tag**: `//go:build large_scale` ✓ VERIFIED
  - **Run**: `go test -tags=large_scale -timeout=30m`
  - **Expected**: FAIL (batch implementation doesn't exist) ✓ VERIFIED

### Implementation Tasks (TDD Green Phase)

- [x] T029 Implement createReparseTempTable() in internal/iooptimize/reparse_batch.go
  - **File**: `internal/iooptimize/reparse_batch.go` (NEW FILE)
  - **Purpose**: Create temporary table for batch updates
  - **Implementation**: ✅ COMPLETE
    - UNLOGGED table for performance (no WAL overhead)
    - IF NOT EXISTS for idempotency
    - PRIMARY KEY on name_string_id
    - 13 columns matching temp table design
  - **Dependencies**: None (pure SQL)
  - **Verify**: T025 test passes ✅ ALL 3 TESTS PASS

- [x] T030 Implement bulkInsertToTempTable() in internal/iooptimize/reparse_batch.go
  - **File**: `internal/iooptimize/reparse_batch.go` (add to file from T029)
  - **Purpose**: Bulk insert ONLY CHANGED parsed results into temp table using pgx CopyFrom
  - **Implementation**: ✅ COMPLETE
    - Uses pgx.CopyFrom with pgx.CopyFromSlice for maximum performance
    - Handles empty batch gracefully (no-op)
    - Converts sql.Null* types to interface{} for CopyFrom
    - Helper functions for NULL handling (nullStringToInterface, etc.)
    - Verifies row count after insert
    - 50K rows insert in ~1-2 seconds (tested)
  - **Key Point**: Input is pre-filtered - unchanged names never reach this function
  - **Dependencies**: T029 (temp table must exist) ✅
  - **Verify**: T026 test passes ✅ ALL 5 TESTS PASS

- [x] T031 Implement batchUpdateNameStrings() in internal/iooptimize/reparse_batch.go
  - **File**: `internal/iooptimize/reparse_batch.go` (add to file from T029)
  - **Purpose**: Single UPDATE statement to apply all changes from temp table
  - **Implementation**: ✅ COMPLETE
    - Single UPDATE with FROM clause (JOIN pattern)
    - Updates all 9 fields in one transaction
    - Returns row count for verification
    - Handles empty temp table gracefully (0 rows updated)
    - Performance: 1M rows updated in ~2-5 minutes (vs 30-60 min row-by-row)
  - **Dependencies**: T030 (temp table must be populated) ✅
  - **Verify**: T027 test passes ✅ ALL 3 TESTS PASS

- [ ] T032 Implement batchInsertCanonicals() in internal/iooptimize/reparse_batch.go
  - **File**: `internal/iooptimize/reparse_batch.go` (add to file from T029)
  - **Purpose**: Batch insert unique canonicals from temp table
  - **Implementation**:
    ```go
    // batchInsertCanonicals extracts unique canonical forms from temp table
    // and inserts them into canonicals, canonical_stems, canonical_fulls.
    // Uses ON CONFLICT DO NOTHING for idempotency.
    func batchInsertCanonicals(ctx context.Context, pool *pgxpool.Pool) error
    ```
  - **Algorithm**:
    1. Extract unique canonical_id + canonical from temp table
    2. Batch INSERT into canonicals with ON CONFLICT DO NOTHING
    3. Repeat for canonical_stems and canonical_fulls
  - **SQL example**:
    ```sql
    INSERT INTO canonicals (id, name)
    SELECT DISTINCT canonical_id, canonical
    FROM temp_reparse_names
    WHERE canonical_id IS NOT NULL AND canonical != ''
    ON CONFLICT (id) DO NOTHING
    ```
  - **Dependencies**: T031 (temp table populated)
  - **Verify**: T027 test passes (verifies canonicals inserted)

- [x] T033 Refactor reparseNames() to use filter-then-batch workflow in internal/iooptimize/reparse.go
  - **File**: `internal/iooptimize/reparse.go` (MODIFY EXISTING)
  - **Purpose**: Replace row-by-row updates with filter-then-batch workflow
  - **Changes**:
    1. Keep loadNamesForReparse() unchanged (loads all names)
    2. Keep workerReparse() mostly unchanged but:
       - After parsing, call parsedIsSame() to check if name changed
       - ONLY send changed names to output channel (skip unchanged)
       - This leverages existing optimization at worker level
    3. Replace saveReparsedNames() with new saveBatchedNames():
       - Receives ONLY changed names from workers
       - Collects into batches (Config.Import.BatchSize)
       - When batch full OR channel closed, call bulkInsertToTempTable()
    4. Replace updateNameString() with batch operations:
       - Create temp table once at start (createReparseTempTable)
       - Collect all CHANGED names into temp table
       - Call batchUpdateNameStrings() once at end
       - Call batchInsertCanonicals() once at end
       - Drop temp table
  - **Architecture**:
    ```
    loadNamesForReparse → workerReparse (N workers) → parsedIsSame() filter
                            (parses ALL)                     ↓
                                                      ONLY changed names
                                                             ↓
                                                      saveBatchedNames
                                                             ↓
                                                      bulkInsertToTempTable
                                                             ↓
                                                      batchUpdateNameStrings
                                                             ↓
                                                      batchInsertCanonicals
    ```
  - **Key Change**: Filter at worker level, not after collecting all results
  - **Memory Benefit**: Temp table contains 1M-50M rows, not 100M
  - **Dependencies**: T029, T030, T031, T032
  - **Verify**: T005 integration test still passes (behavior unchanged, only performance improved)

- [x] T034 Add batch size configuration to Config in pkg/config/config.go
  - **File**: `pkg/config/config.go` (MODIFY EXISTING)
  - **Purpose**: Allow users to tune batch size for memory vs performance
  - **Changes**:
    - Add field to Config struct (if not exists): `ReparseBatchSize int` with default 50000
    - Update config.yaml template with new field
    - Add validation: batch size must be 1000-1000000
  - **Dependencies**: None (config change)
  - **Verify**: Config loads with new field

- [ ] T035 Update cmd/gndb/optimize.go to add --reparse-batch-size flag
  - **File**: `cmd/gndb/optimize.go` (MODIFY EXISTING)
  - **Purpose**: Expose batch size control to CLI users
  - **Changes**:
    - Add cobra flag: `--reparse-batch-size` (default 50000)
    - Bind flag to Config.ReparseBatchSize
    - Update help text: "Number of names to batch for reparsing (1000-1000000, default 50000). Lower values use less memory."
  - **Dependencies**: T034
  - **Verify**: `gndb optimize --help` shows new flag

### Refactor & Validation Tasks

- [x] T036 Remove old row-by-row code from reparse.go
  - **File**: `internal/iooptimize/reparse.go` (MODIFY EXISTING)
  - **Purpose**: Clean up deprecated updateNameString() function
  - **Changes**:
    - Delete updateNameString() function (no longer used)
    - Delete saveReparsedNames() function (replaced by saveBatchedNames)
    - Add comment referencing old implementation in git history
  - **Dependencies**: T033 (new batch implementation working)
  - **Verify**: All tests still pass

- [x] T037 Update PERFORMANCE_ANALYSIS.md with batch optimization results
  - **File**: `PERFORMANCE_ANALYSIS.md` (MODIFY EXISTING)
  - **Purpose**: Document performance improvements from batch approach
  - **Content**:
    - Benchmark results: row-by-row vs batch (T024 output)
    - Memory usage comparison
    - Scaling analysis (1K, 10K, 100K, 1M, 10M rows)
    - Recommended batch sizes for different hardware
  - **Dependencies**: T024 (benchmark results)
  - **Verify**: Document is clear and actionable

- [x] T038 Run full integration test with 100K rows
  - **File**: `internal/iooptimize/reparse_large_test.go` (RUN TEST)
  - **Purpose**: Validate end-to-end workflow with large dataset
  - **Command**: `go test -tags=large_scale -timeout=30m -v ./internal/iooptimize`
  - **Success criteria**:
    - All 100K rows processed
    - Memory usage < 2GB
    - Completion time < 10 minutes (on 16-core machine)
    - All tests pass
  - **Dependencies**: T033 (batch implementation complete)

- [x] T039 Update quickstart.md with batch size tuning guidance
  - **File**: `specs/002-optimize/quickstart.md` (MODIFY EXISTING)
  - **Purpose**: Help users tune performance for their hardware
  - **Content**: Add new section "Batch Size Tuning"
    - Default 50K works for most systems
    - Low memory (< 8GB): use 10K
    - High memory (> 32GB): use 100K
    - Formula: batch_size = available_memory_gb * 1000
  - **Dependencies**: T037 (performance data)
  - **Verify**: Quickstart is clear and actionable

---

## Dependencies Graph

```
Setup (T001-T003) [COMPLETE]
    ↓
Initial Tests (T004-T011) [COMPLETE]
    ↓
Initial Implementation (T012-T023) [COMPLETE]
    ↓
Batch Optimization Tests (T024-T028) ← START HERE
    ↓
T029 (temp table)
    ↓
T030 (bulk insert) → depends on T029
    ↓
T031 (batch update) → depends on T030
    ↓
T032 (batch canonicals) → depends on T031
    ↓
T033 (refactor reparse) → depends on T029, T030, T031, T032
    ↓
T034 (config) → independent, can run parallel with T029-T033
    ↓
T035 (CLI flag) → depends on T034
    ↓
T036 (cleanup) → depends on T033
    ↓
T037, T038, T039 (validation) → depends on T033, T036
```

## Parallel Execution Examples

### Phase 1: Write all tests in parallel
```bash
# All test tasks can run concurrently (different files)
Task: "Write performance benchmark in reparse_bench_test.go (T024)"
Task: "Write temp table unit test in reparse_batch_test.go (T025)"
Task: "Write bulk insert unit test in reparse_batch_test.go (T026)"
Task: "Write batch update unit test in reparse_batch_test.go (T027)"
```

### Phase 2: Implement batch functions
```bash
# T029 first (temp table), then T030-T032 can be parallel
Task: "Implement createReparseTempTable() (T029)"

# After T029 completes:
Task: "Implement bulkInsertToTempTable() (T030)"
Task: "Implement batchInsertCanonicals() (T032)"  # Can overlap if temp table schema is clear
```

### Phase 3: Integration & config
```bash
# T034 can run parallel with implementation
Task: "Add batch size config (T034)"
Task: "Refactor reparseNames() workflow (T033)"  # After T030-T032 done
```

## Performance Targets (100M Rows)

| Scenario | Changed % | Rows in Temp Table | Current Time | Batch Time | Improvement |
|----------|-----------|-------------------|--------------|------------|-------------|
| **First optimization** | 100% | 100M | 90 min | 15 min | **6x faster** |
| **Partial update** | 50% | 50M | 60 min | 10 min | **6x faster** |
| **Re-optimization** | 10% | 10M | 30 min | 7 min | **4x faster** |
| **Minor update** | 1% | 1M | 15 min | 5 min | **3x faster** |

**Key Metrics**:
- **Memory**: Scales with changed % (1M rows = ~200MB, 50M rows = ~10GB temp table)
- **Transactions**: 3-4 (vs 100M) regardless of change percentage
- **Filter efficiency**: `parsedIsSame()` eliminates unchanged rows at worker level
- **Canonical deduplication**: Single `SELECT DISTINCT` instead of 100M `ON CONFLICT` checks

## Notes

- **Backward compatibility**: Old tests (T005) must still pass after refactor (T033)
- **Idempotency**: Temp table approach maintains ON CONFLICT DO NOTHING for canonicals
- **Memory safety**: Batch size configurable to prevent OOM on small systems
- **100M rows**: With 50K batch size = 2000 batches, ~7 seconds per batch = 4 hours total
- **Testing strategy**: Unit tests for each function, integration test for workflow, benchmark for performance
- **Large-scale test**: Use `// +build large_scale` tag to avoid running on every test run

## Validation Checklist

Before marking Phase 3.6 complete:

- [ ] T024 benchmark shows 10x+ improvement
- [ ] T028 large-scale test completes under 30 minutes (100K rows)
- [ ] All existing tests still pass (T004-T011)
- [ ] Memory usage stays under 2GB per config
- [ ] CLI flag `--reparse-batch-size` works correctly
- [ ] Documentation updated (PERFORMANCE_ANALYSIS.md, quickstart.md)
- [ ] Code review: no old row-by-row code remains

---

**Status**: Phase 3.6 ready for execution. Tasks T024-T039 will optimize Step 1 reparsing for 100M+ row scalability using batch operations.
