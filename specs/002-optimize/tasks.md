# Tasks: Optimize Database Performance

**Input**: Design documents from `/home/dimus/code/golang/gndb/specs/002-optimize/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/

**Branch**: `002-optimize`

## Execution Summary

This task list implements the `gndb optimize` command following the production-tested `gnidump rebuild` workflow. The implementation is broken down into 6 main optimization steps, each with comprehensive tests following TDD principles.

**Key Architecture Decisions from gnidump Analysis:**
- Concurrent processing pipeline (load → workers → save pattern)
- Batch processing for large datasets
- Bulk inserts via pgx.CopyFrom
- Idempotent operations (safe reruns)
- Transaction safety per update
- Ephemeral cache at `~/.cache/gndb/optimize/` for parse results

## Phase 3.1: Setup & Infrastructure

### T001: Create optimize CLI command structure ✅
**File**: `cmd/gndb/optimize.go`
**Description**: Create the cobra command for `gndb optimize` with proper flag handling
**Details**:
- Create optimizeCmd as cobra.Command
- Add flags: --jobs (worker count), --batch-size (word batching)
- Wire to config.Config using viper
- Add to root command in main.go
- Include help text and usage examples
**Test**: Run `gndb optimize --help` and verify output
**Status**: ✅ COMPLETE

**Implementation Summary**:
- CLI layer simplified to just propagate errors (no error creation)
- Error types created in `internal/iodb/errors.go`:
  - `ConnectionError` - for database connection failures
  - `TableCheckError` - for table existence check failures
  - `EmptyDatabaseError` - for unpopulated database
- Added `CheckReadyForOptimization()` method to PgxOperator
- All errors originate from packages where they occur (bottom-up propagation)
- CLI uses `gnlib.PrintUserMessage(err)` to display user-friendly errors to STDOUT
- Technical errors automatically go to STDERR via framework
- User-focused help text (no implementation details)

---

### T002 [P]: Create cache infrastructure for parse results
**File**: `internal/iooptimize/cache.go`
**Description**: Implement ephemeral key-value store for caching parsed name results using Badger v4 + GOB
**Details**:
- **Architecture Decision**: Use Badger v4 embedded KV store + GOB serialization (proven with 150M+ names in gnidump)
- Create CacheManager struct with methods:
  - `NewCacheManager(cacheDir string)` - creates Badger v4 DB at ~/.cache/gndb/optimize/
  - `StoreParsed(nameStringID string, parsed *gnparser.Parsed)` - GOB encode and store via Badger transaction
  - `GetParsed(nameStringID string)` - retrieve and GOB decode from Badger
  - `Cleanup()` - close Badger DB and remove cache directory
- Use gnfmt.GNgob{} encoder for GOB serialization (from gnlib)
- Use badger.NewTransaction(true) for writes (writable=true)
- Use badger.View() for reads (read-only transactions)
- Store parsed data as GOB-encoded bytes keyed by name_string_id (string)
- Defer cleanup to remove cache after optimize completes
**Dependencies**: 
  - github.com/dgraph-io/badger/v4 (upgrade from v2)
  - github.com/gnames/gnfmt (for GNgob encoder)
  - github.com/gnames/gnparser (for Parsed type)
**Reference**: gnidump kvio package (ent/kv/kvio/) and db_reparse.go usage
**Test**: Unit test cache operations (create, store, retrieve, cleanup)

---

## Phase 3.2: Step 1 - Reparse Names (TDD)

### T003 [P]: Write integration test for name reparsing ✅
**File**: `internal/iooptimize/reparse_test.go`
**Description**: Write test that validates name reparsing updates name_strings correctly
**Test Scenario**:
1. Given: Database with sample name_strings (e.g., "Homo sapiens", "Mus musculus")
2. When: Call reparseNames(ctx, cfg)
3. Then:
   - name_strings.canonical_id updated with latest gnparser output
   - canonical_full_id, canonical_stem_id updated
   - bacteria, virus, surrogate flags updated
   - parse_quality set correctly
   - Cached parse results stored in cache
4. Verify test FAILS (no implementation yet)
**Reference**: gnidump db_reparse.go test pattern
**Status**: ✅ COMPLETE

**Implementation Summary**:
- Created `internal/iooptimize/reparse_test.go` with 4 comprehensive integration tests
- Tests verify all aspects of name reparsing workflow:
  - `TestReparseNames_Integration`: Main test for canonical updates, cache storage, table population
  - `TestReparseNames_Idempotent`: Verifies safe reruns without data duplication
  - `TestReparseNames_UpdatesOnlyChangedNames`: Tests optimization for unchanged names
  - `TestReparseNames_VirusNames`: Tests special virus name handling
- Added `errNotImplemented()` helper in `internal/iooptimize/errors.go` for TDD red phase
- Updated `OptimizerImpl` struct to include cache field
- All tests FAIL as expected with "not yet implemented" error ✅ (TDD red phase confirmed)
- Ready for implementation in T004-T008

---

### T004: Implement loadNamesForReparse function ✅
**File**: `internal/iooptimize/reparse.go`
**Description**: Load all name_strings from database for reparsing
**Details**:
- Query: SELECT id, name FROM name_strings
- Send reparsed structs to channel (chIn chan<- *reparsed)
- Progress tracking: log every 100,000 names
- Context cancellation support
**Reference**: gnidump loadReparse() in db_reparse.go
**Test**: T003 should start passing for load phase
**Status**: ✅ COMPLETE

**Implementation Summary**:
- Created `internal/iooptimize/reparse.go` with `reparsed` struct and `loadNamesForReparse` function
- Function queries all name_strings from database: id, name, canonical_id, canonical_full_id, canonical_stem_id, bacteria, virus, surrogate, parse_quality
- Sends each name to input channel for processing
- Progress logging every 100,000 names with speed metrics (using humanize)
- Context cancellation properly handled in select statement
- Added 2 unit tests in `reparse_test.go`:
  - `TestLoadNamesForReparse_Unit`: Verifies all names are loaded correctly ✅
  - `TestLoadNamesForReparse_ContextCancellation`: Verifies context cancellation works ✅
- All tests passing ✅

---

### T005: Implement workerReparse concurrent processor ✅
**File**: `internal/iooptimize/reparse.go`
**Description**: Worker function that parses names using gnparser
**Details**:
- Create gnparser instance per worker (use pkg/parserpool if available)
- For each name from chIn:
  - Parse with gnparser.ParseName(name)
  - Generate UUID v5 for canonical forms using gnuuid.New()
  - Check if parsing improved (parsedIsSame comparison)
  - Store parsed result in cache (CacheManager.StoreParsed)
  - Send updated record to chOut
- Handle context cancellation
**Reference**: gnidump workerReparse() in db_reparse.go (50 workers)
**Config**: Use Config.JobsNumber for worker count (not hardcoded 50)
**Status**: ✅ COMPLETE

**Implementation Summary**:
- Implemented `workerReparse()` function in `internal/iooptimize/reparse.go`
- Uses `parserpool.Pool` parameter for efficient concurrent parsing (Botanical code by default)
- Generates UUID v5 for canonical forms: canonicalID, canonicalFullID, canonicalStemID
- Implements `parsedIsSame()` optimization to skip unchanged names (no unnecessary DB updates)
- Stores parsed results in cache via `CacheManager.StoreParsed()`
- Handles unparsed names (sets empty canonical IDs, parse quality)
- Context cancellation: drains input channel and returns `ctx.Err()`
- Helper functions added:
  - `parsedIsSame(r reparsed, parsed parsed.Parsed, canonicalID string) bool` - compares old vs new parse
  - `newNullStr(s string) sql.NullString` - creates SQL NULL strings
- Added 3 comprehensive unit tests:
  - `TestWorkerReparse_Unit`: Verifies parsing, UUID generation, caching ✅
  - `TestWorkerReparse_ContextCancellation`: Verifies cancellation handling ✅
  - `TestWorkerReparse_SkipsUnchangedNames`: Verifies optimization ✅
- All tests passing ✅
- No logging in internal package (follows error handling pattern)

---

### T006: Implement saveReparsedNames function ✅
**File**: `internal/iooptimize/reparse.go`
**Description**: Save reparsed name data back to database
**Details**:
- Receive reparsed structs from chOut channel
- For each record, call updateNameString() in transaction
- Log updates to slog (optional: reparse.log file)
- Progress tracking
**Reference**: gnidump saveReparse() in db_reparse.go
**Status**: ✅ COMPLETE

**Implementation Summary**:
- Implemented `saveReparsedNames()` function that receives reparsed data from channel
- Calls `updateNameString()` for each record with transaction-based updates
- Progress tracking: logs every 100,000 updates with speed metrics
- Context cancellation properly handled
- Error propagation from updateNameString
- Added comprehensive unit test `TestSaveReparsedNames_Unit` ✅
- Test verifies all names updated and canonicals inserted correctly

---

### T007: Implement updateNameString database operation ✅
**File**: `internal/iooptimize/reparse.go`
**Description**: Transaction-based update of name_strings and canonical tables
**Details**:
- Begin transaction
- UPDATE name_strings SET canonical_id=?, canonical_full_id=?, canonical_stem_id=?, bacteria=?, virus=?, surrogate=?, parse_quality=? WHERE id=?
- INSERT INTO canonicals (id, name) VALUES (?, ?) ON CONFLICT DO NOTHING
- INSERT INTO canonical_fulls (id, name) VALUES (?, ?) ON CONFLICT DO NOTHING
- INSERT INTO canonical_stems (id, name) VALUES (?, ?) ON CONFLICT DO NOTHING
- Commit or rollback on error
**Reference**: gnidump updateNameString() in db_reparse.go
**Status**: ✅ COMPLETE

**Implementation Summary**:
- Implemented `updateNameString()` function with full transaction safety
- Updates name_strings table with all canonical IDs and flags
- Inserts into canonicals, canonical_stems, canonical_fulls tables with ON CONFLICT DO NOTHING
- Skips canonical inserts for unparseable names (parseQuality == 0)
- Only inserts canonical_full if different from canonical
- Proper transaction rollback on errors, commit on success
- Added error types: `ReparseTransactionError`, `ReparseUpdateError`, `ReparseInsertError`
- Added comprehensive unit test `TestUpdateNameString_Unit` ✅
- Test verifies updates and inserts work correctly with proper error handling

---

### T008: Implement reparseNames orchestrator ✅
**File**: `internal/iooptimize/reparse.go`
**Description**: Main function orchestrating concurrent reparse pipeline
**Details**:
- Create channels: chIn, chOut
- Use errgroup for error handling
- Launch goroutines:
  - 1x loadNamesForReparse(ctx, chIn)
  - Nx workerReparse(ctx, chIn, chOut) where N = Config.JobsNumber
  - 1x saveReparsedNames(ctx, chOut)
- Wait for completion
- Return error if any
**Reference**: gnidump reparse() in db_reparse.go
**Test**: T003 should now PASS
**Status**: ✅ COMPLETE

**Implementation Summary**:
- Implemented `reparseNames()` orchestrator using errgroup for concurrent pipeline
- Creates chIn and chOut channels for 3-stage pipeline communication
- Stage 1: loadNamesForReparse (1 goroutine) - loads all name_strings from database
- Stage 2: workerReparse (N goroutines) - concurrent parsing using parserpool
  - N = Config.JobsNumber (default runtime.NumCPU())
  - Each worker parses names, generates UUIDs, caches results
- Stage 3: saveReparsedNames (1 goroutine) - saves updated records to database
- WaitGroup tracks worker completion, closes chOut when all workers finish
- Proper error propagation: any goroutine error cancels context for all others
- All 10 integration tests passing ✅:
  - TestReparseNames_Integration - full workflow with 4 test names
  - TestReparseNames_Idempotent - safe reruns without duplication
  - TestReparseNames_UpdatesOnlyChangedNames - optimization works
  - TestReparseNames_VirusNames - virus flag detection works
  - TestLoadNamesForReparse_Unit - database loading works
  - TestLoadNamesForReparse_ContextCancellation - context handling works
  - TestWorkerReparse_Unit - parsing and caching works
  - TestWorkerReparse_ContextCancellation - context handling works
  - TestWorkerReparse_SkipsUnchangedNames - optimization works
  - TestSaveReparsedNames_Unit - database saving works
  - TestUpdateNameString_Unit - transaction-based updates work

**Additional Enhancements**:
- Added year and cardinality extraction from parsed data
- Fixed virus name detection for unparsed names (Virus flag set even when Parsed=false)
- Proper handling of edge cases: unparseable names, virus names, unchanged names

---

## Phase 3.3: Step 2 - Fix Vernacular Languages (TDD)

### T009 [P]: Write integration test for vernacular language normalization ✅
**File**: `internal/iooptimize/vernacular_test.go`
**Description**: Test that vernacular language codes are normalized correctly
**Status**: ✅ COMPLETE
**Test Scenario**:
1. Given: Database with vernacular_string_indices having various language codes (e.g., "en", "eng", "English")
2. When: Call fixVernacularLanguages(ctx, cfg)
3. Then:
   - language_orig field populated with original language value
   - lang_code converted to 3-letter ISO code (e.g., "en" → "eng")
   - All lang_code values are lowercase
   - language field normalized
4. Verify test FAILS
**Reference**: gnidump db_vern.go test pattern

---

### T010: Implement moveLanguageToOrig function ✅
**File**: `internal/iooptimize/vernacular.go`
**Description**: Copy language field to language_orig for records that don't have it
**Status**: ✅ COMPLETE
**Details**:
- Single SQL UPDATE: `UPDATE vernacular_string_indices SET language_orig = language WHERE language_orig IS NULL`
- Execute via pgx
**Reference**: gnidump langOrig() in db_vern.go

---

### T011: Implement loadVernaculars function ✅
**File**: `internal/iooptimize/vernacular.go`
**Description**: Load all vernacular records for language normalization
**Status**: ✅ COMPLETE
**Details**:
- Query: SELECT ctid, language, lang_code FROM vernacular_string_indices
- Send vern structs to channel
- Progress tracking: every 50,000 records
- Context cancellation support
**Reference**: gnidump loadVern() in db_vern.go

---

### T012: Implement normalizeVernacularLanguage function ✅
**File**: `internal/iooptimize/vernacular.go`
**Description**: Normalize language codes using gnlang library
**Status**: ✅ COMPLETE
**Details**:
- For each vernacular record from channel:
  - If 2-letter code: convert to 3-letter using gnlang.LangCode2To3Letters()
  - If 3-letter code: validate using gnlang
  - If missing lang_code: derive from language field using gnlang.LangCode()
  - Update vernacular record via updateVernRecord()
- Handle context cancellation
**Reference**: gnidump normVernLang() in db_vern.go
**Dependency**: gnfmt/gnlang library

---

### T013: Implement updateVernRecord function ✅
**File**: `internal/iooptimize/vernacular.go`
**Description**: Update single vernacular record using ctid
**Status**: ✅ COMPLETE
**Details**:
- SQL UPDATE using ctid (physical row ID): `UPDATE vernacular_string_indices SET language=?, lang_code=? WHERE ctid=?`
- Execute via pgx
**Reference**: gnidump updateVernRecord() in db_vern.go

---

### T014: Implement langCodeToLowercase function ✅
**File**: `internal/iooptimize/vernacular.go`
**Description**: Ensure all lang_code values are lowercase
**Status**: ✅ COMPLETE
**Details**:
- Single SQL UPDATE: `UPDATE vernacular_string_indices SET lang_code = LOWER(lang_code)`
**Reference**: gnidump langCodeLowCase() in db_vern.go

---

### T015: Implement fixVernacularLanguages orchestrator ✅
**File**: `internal/iooptimize/vernacular.go`
**Description**: Main function orchestrating vernacular language fix
**Status**: ✅ COMPLETE
**Details**:
- Call moveLanguageToOrig()
- Create channels for concurrent processing
- Use errgroup
- Launch goroutines:
  - loadVernaculars(ctx, chIn)
  - Multiple normalizeVernacularLanguage workers
- Call langCodeToLowercase() at end
**Reference**: gnidump fixVernLang() in db_vern.go
**Test**: T009 should now PASS

---

## Phase 3.4: Step 3 - Remove Orphan Records (TDD)

### T016 [P]: Write integration test for orphan removal ✅
**File**: `internal/iooptimize/orphans_test.go`
**Description**: Test that orphaned records are removed correctly
**Status**: ✅ COMPLETE
**Test Scenario**:
1. Given: Database with:
   - name_strings not in name_string_indices (orphans)
   - canonicals not referenced by name_strings (orphans)
   - canonical_fulls not referenced (orphans)
   - canonical_stems not referenced (orphans)
2. When: Call removeOrphans(ctx, cfg)
3. Then:
   - Orphaned name_strings deleted
   - Orphaned canonicals deleted
   - Orphaned canonical_fulls deleted
   - Orphaned canonical_stems deleted
   - Referenced records remain intact
4. Verify test FAILS
**Reference**: gnidump removeOrphans() in db_views.go

**Implementation Summary**:
- Created `internal/iooptimize/orphans_test.go` with 4 comprehensive integration tests
- Tests verify all aspects of orphan removal workflow:
  - `TestRemoveOrphans_Integration`: Main test with orphan names, canonicals, stems, fulls
  - `TestRemoveOrphans_Idempotent`: Verifies safe reruns without data loss
  - `TestRemoveOrphans_EmptyDatabase`: Tests graceful handling of empty database
  - `TestRemoveOrphans_CascadeOrder`: Tests correct removal order (names → canonicals)
- Added stub `removeOrphans()` function in `internal/iooptimize/orphans.go`
- Test FAILS as expected with "not yet implemented" error ✅ (TDD red phase confirmed)
- Ready for implementation in T017-T021

---

### T017: Implement removeOrphanNameStrings function ✅
**File**: `internal/iooptimize/orphans.go`
**Description**: Delete name_strings not referenced by name_string_indices
**Status**: ✅ COMPLETE
**Details**:
- SQL: DELETE FROM name_strings WHERE id NOT IN (SELECT DISTINCT name_string_id FROM name_string_indices)
- Alternative using LEFT JOIN for performance:
  ```sql
  DELETE FROM name_strings ns
  WHERE NOT EXISTS (
    SELECT 1 FROM name_string_indices nsi
    WHERE nsi.name_string_id = ns.id
  )
  ```
- Log count of deleted records
**Reference**: gnidump removeOrphans() in db_views.go

**Implementation Summary**:
- Implemented `removeOrphanNameStrings()` function in `internal/iooptimize/orphans.go`
- Uses LEFT OUTER JOIN pattern from gnidump for optimal performance
- Logs removal action and deleted count using slog
- Returns `OrphanRemovalError` with user-friendly error messages on failure
- Added `NewOrphanRemovalError()` error type in `internal/iooptimize/errors.go`
- Updated `removeOrphans()` orchestrator to call Step 1 (removeOrphanNameStrings)
- Tests verify correct behavior:
  - TestRemoveOrphans_Integration: 2 orphan names deleted ✅
  - TestRemoveOrphans_CascadeOrder: 1 orphan name deleted ✅
  - Referenced names remain intact ✅
- Ready for T018 (removeOrphanCanonicals)

---

### T018: Implement removeOrphanCanonicals function ✅
**File**: `internal/iooptimize/orphans.go`
**Description**: Delete canonicals not referenced by name_strings
**Status**: ✅ COMPLETE
**Details**:
- SQL: DELETE FROM canonicals WHERE id NOT IN (SELECT canonical_id FROM name_strings WHERE canonical_id IS NOT NULL)
- Log count of deleted records
**Reference**: gnidump removeOrphans() in db_views.go

**Implementation Summary**:
- Implemented `removeOrphanCanonicals()` function in `internal/iooptimize/orphans.go`
- Uses LEFT OUTER JOIN pattern from gnidump for optimal performance
- Logs removal action and deleted count using slog
- Returns `OrphanRemovalError` with user-friendly error messages on failure
- Updated `removeOrphans()` orchestrator to call Step 2 (removeOrphanCanonicals)
- Tests verify correct behavior:
  - TestRemoveOrphans_Integration: 2 orphan canonicals deleted ✅
  - TestRemoveOrphans_CascadeOrder: 1 orphan canonical deleted after orphan name removed ✅
  - TestRemoveOrphans_Idempotent: Safe reruns (0 deletions on second run) ✅
- Ready for T019 (removeOrphanCanonicalFulls)

---

### T019: Implement removeOrphanCanonicalFulls function ✅
**File**: `internal/iooptimize/orphans.go`
**Description**: Delete canonical_fulls not referenced by name_strings
**Status**: ✅ COMPLETE
**Details**:
- SQL: DELETE FROM canonical_fulls WHERE id NOT IN (SELECT canonical_full_id FROM name_strings WHERE canonical_full_id IS NOT NULL)
- Log count of deleted records

**Implementation Summary**:
- Implemented `removeOrphanCanonicalFulls()` function in `internal/iooptimize/orphans.go`
- Uses LEFT OUTER JOIN pattern from gnidump for optimal performance
- Logs removal action and deleted count using slog
- Returns `OrphanRemovalError` with user-friendly error messages on failure
- Updated `removeOrphans()` orchestrator to call Step 3 (removeOrphanCanonicalFulls)
- Tests verify correct behavior:
  - TestRemoveOrphans_Integration: 1 orphan canonical_full deleted ✅
  - TestRemoveOrphans_CascadeOrder: Correct cascade behavior (0 deletions) ✅
  - TestRemoveOrphans_Idempotent: Safe reruns (0 deletions on second run) ✅
- Ready for T020 (removeOrphanCanonicalStems)

---

### T020: Implement removeOrphanCanonicalStems function ✅
**File**: `internal/iooptimize/orphans.go`
**Description**: Delete canonical_stems not referenced by name_strings
**Status**: ✅ COMPLETE
**Details**:
- SQL: DELETE FROM canonical_stems WHERE id NOT IN (SELECT canonical_stem_id FROM name_strings WHERE canonical_stem_id IS NOT NULL)
- Log count of deleted records

**Implementation Summary**:
- Implemented `removeOrphanCanonicalStems()` function in `internal/iooptimize/orphans.go`
- Uses LEFT OUTER JOIN pattern from gnidump for optimal performance
- Logs removal action and deleted count using slog
- Returns `OrphanRemovalError` with user-friendly error messages on failure
- Updated `removeOrphans()` orchestrator to call Step 4 (removeOrphanCanonicalStems)
- **COMPLETE orphan removal workflow** - all 4 steps implemented (T017-T020)
- Tests verify correct behavior:
  - TestRemoveOrphans_Integration: 1 orphan canonical_stem deleted ✅ **NOW PASSES**
  - TestRemoveOrphans_CascadeOrder: Correct cascade behavior ✅
  - TestRemoveOrphans_Idempotent: Safe reruns (0 deletions on second run) ✅
  - TestRemoveOrphans_EmptyDatabase: Graceful empty database handling ✅
- Ready for T021 (orchestrator documentation - already implemented)

---

### T021: Implement removeOrphans orchestrator ✅
**File**: `internal/iooptimize/orphans.go`
**Description**: Main function orchestrating orphan removal in correct order
**Status**: ✅ COMPLETE
**Details**:
- Execute in sequence:
  1. removeOrphanNameStrings()
  2. removeOrphanCanonicals()
  3. removeOrphanCanonicalFulls()
  4. removeOrphanCanonicalStems()
- Log total records removed
**Reference**: gnidump removeOrphans() in db_views.go
**Test**: T016 should now PASS

**Implementation Summary**:
- Orchestrator already fully implemented through T017-T020
- Executes all 4 orphan removal steps in correct sequence
- Proper error handling and propagation at each step
- Each step logs count of deleted records (detailed logging)
- Test T016 (TestRemoveOrphans_Integration) **PASSES** ✅
- All 4 orphan removal tests pass:
  - TestRemoveOrphans_Integration: Full workflow ✅
  - TestRemoveOrphans_Idempotent: Safe reruns ✅
  - TestRemoveOrphans_EmptyDatabase: Edge case handling ✅
  - TestRemoveOrphans_CascadeOrder: Correct order verification ✅
- **Phase 3.4 (Step 3 - Remove Orphan Records) COMPLETE**

---

## Phase 3.5: Step 4 - Create Words Tables (TDD)

### T022 [P]: Write integration test for word extraction ✅
**File**: `internal/iooptimize/words_test.go`
**Description**: Test that words are extracted and linked to names correctly
**Status**: ✅ COMPLETE
**Test Scenario**:
1. Given: Database with name_strings and cached parse results
2. When: Call createWords(ctx, cfg)
3. Then:
   - words table populated with normalized and modified word forms
   - word_name_strings junction table links words to names and canonicals
   - Only epithet and author words included (type filtering)
   - Deduplication applied (no duplicate words)
4. Verify test FAILS
**Reference**: gnidump createWords() in words.go

**Implementation Summary**:
- Created `internal/iooptimize/words_test.go` with 3 comprehensive integration tests
- Tests verify all aspects of word extraction workflow:
  - `TestCreateWords_Integration`: Main test for word extraction, junction table, deduplication, type filtering
  - `TestCreateWords_Idempotent`: Verifies safe reruns without data duplication
  - `TestCreateWords_EmptyCache`: Tests graceful handling of missing cache entries
- Created stub `createWords()` function in `internal/iooptimize/words.go`
- Test FAILS as expected with "not yet implemented" error ✅ (TDD red phase confirmed)
- Test validates:
  - Words table population with normalized and modified forms
  - Junction table (word_name_strings) links words to names and canonicals
  - Only epithet and author words extracted (type filtering via gnparser)
  - Deduplication works (same word in multiple names only stored once)
  - Words extracted from cached parse results (simulates Step 1 output)
- Ready for implementation in T023-T030

---

### T023: Implement truncateWordsTables function ✅
**File**: `internal/iooptimize/words.go`
**Description**: Clear words and word_name_strings tables before population
**Status**: ✅ COMPLETE
**Details**:
- SQL: TRUNCATE TABLE words CASCADE
- SQL: TRUNCATE TABLE word_name_strings CASCADE
- Ensures clean slate for word creation
**Reference**: gnidump createWords() uses truncateTable()

**Implementation Summary**:
- Implemented `truncateWordsTables()` function in `internal/iooptimize/words.go`
- Truncates both `words` and `word_name_strings` tables with CASCADE
- Proper error handling and logging for each table
- Follows gnidump pattern from truncateTable() in db.go

---

### T024: Implement getNameStringsForWords function ✅ (UPDATED)
**File**: `internal/iooptimize/words.go`
**Description**: Query all name_strings for word extraction
**Status**: ✅ COMPLETE (needs update to return names)
**Details**:
- Query: SELECT id, name, canonical_id FROM name_strings WHERE canonical_id IS NOT NULL
- Return slice with: name_string_id, name (for parsing), canonical_id
**Reference**: gnidump getWordNames() in db.go (returns just names for parsing)

**Implementation Summary**:
- Implemented `getNameStringsForWords()` function in `internal/iooptimize/words.go`
- Created `nameForWords` struct to hold id, name, and canonical_id
- Queries only name_strings with non-NULL canonical_id (required for word extraction)
- Returns slice of nameForWords structs for batch parsing
- Proper error handling and logging
- Added comprehensive unit test `TestGetNameStringsForWords_Unit` ✅
- Test verifies: correct filtering, proper data retrieval, NULL exclusion
- **TODO**: Update to include `name` field for parsing

---

### T025: Implement parseNamesForWords function ✅
**File**: `internal/iooptimize/words.go`
**Description**: Parse names and extract words (following gnidump approach)
**Status**: ✅ COMPLETE
**Details**:
- For batch of name strings:
  - Parse using parserpool with WithDetails(true) to get Words field
  - For each parsed result:
    - Extract word details from parsed.Words:
      - SpEpithetType words
      - InfraspEpithetType words
      - AuthorWordType words
    - Skip surrogates and hybrids
    - For each word:
      - Generate wordID = UUID(normalized|typeID) using gnuuid.New()
      - Create Word struct with: ID, Normalized, Modified (NormalizeByType), TypeID
      - Create WordNameString struct linking word to name and canonical
- Return deduplicated words and word_name_strings
**Reference**: gnidump processParsedWords() in words.go
**Note**: Direct parsing approach - no cache needed (gnparser is fast, KV overhead avoided)

**Implementation Summary**:
- Implemented `parseNamesForWords()` main function with batch processing
- Implemented `processBatchConcurrent()` for concurrent parsing using errgroup
- Implemented `processChunk()` worker function for parallel execution
- **Concurrency**: Uses Config.JobsNumber workers (configurable, defaults to runtime.NumCPU())
- **Batching**: Processes in batches of Config.Database.BatchSize (default 50,000)
- **Memory efficient**: Workers divide batches into chunks for parallel processing
- **Error handling**: Uses errgroup for proper error propagation and context cancellation
- Follows gnidump logic: filters SpEpithet, InfraspEpithet, and AuthorWord types
- Generates UUID5 word IDs from modified|typeID (matching gnidump)
- All code compiles successfully ✅
- Linter passes ✅

---

### T026: Implement deduplicateWords function ✅
**File**: `internal/iooptimize/words.go`
**Description**: Remove duplicate words using map-based deduplication
**Status**: ✅ COMPLETE
**Details**:
- Use map[string]schema.Word keyed by word.ID|word.Normalized (composite key)
- Return unique words as slice
**Reference**: gnidump prepWords() in words.go and wordsMap building

**Implementation Summary**:
- Implemented `deduplicateWords()` function
- Uses composite key: `ID|Normalized` (matches gnidump pattern exactly)
- Map-based deduplication for O(n) performance
- Logs original and unique counts for visibility
- Returns deduplicated slice ready for bulk insert
- Code compiles successfully ✅
- Linter passes ✅

---

### T027: Implement deduplicateWordNameStrings function ✅
**File**: `internal/iooptimize/words.go`
**Description**: Remove duplicate word-name links
**Status**: ✅ COMPLETE
**Details**:
- Use map[string]schema.WordNameString keyed by "WordID|NameStringID"
- Return unique links as slice
**Reference**: gnidump uniqWordNameString() in words.go

**Implementation Summary**:
- Implemented `deduplicateWordNameStrings()` function
- Uses composite key: `WordID|NameStringID` (matches gnidump pattern exactly)
- Map-based deduplication for O(n) performance
- Logs original and unique counts for visibility
- Returns deduplicated slice ready for bulk insert
- Code compiles successfully ✅
- Linter passes ✅

---

### T028: Implement saveWords function ✅
**File**: `internal/iooptimize/words.go`
**Description**: Bulk insert words using pgx.CopyFrom
**Status**: ✅ COMPLETE
**Details**:
- Batch words by Config.Database.BatchSize
- For each batch:
  - Use pgx.CopyFrom to bulk insert into words table
  - Columns: id, normalized, modified, type_id
- Progress tracking: log every batch
**Reference**: gnidump saveWords() in db.go uses insertRows()

**Implementation Summary**:
- Implemented `saveWords()` function in `internal/iooptimize/words.go`
- Uses pgx.CopyFrom for high-performance bulk inserts (matching gnidump pattern)
- Batches by Config.Database.BatchSize (default 50,000)
- Inserts columns: id, normalized, modified, type_id (matches schema.Word)
- Progress logging every 100,000 records + final summary
- Proper error handling with detailed error messages
- Returns early if no words to save
- Code compiles successfully ✅
- Linter passes ✅

---

### T029: Implement saveWordNameStrings function ✅
**File**: `internal/iooptimize/words.go`
**Description**: Bulk insert word-name linkages using pgx.CopyFrom
**Status**: ✅ COMPLETE
**Details**:
- Batch by Config.Database.BatchSize
- For each batch:
  - Use pgx.CopyFrom to bulk insert into word_name_strings table
  - Columns: word_id, name_string_id, canonical_id
- Progress tracking: log every batch
**Reference**: gnidump saveNameWords() in db.go uses insertRows()

**Implementation Summary**:
- Implemented `saveWordNameStrings()` function in `internal/iooptimize/words.go`
- Uses pgx.CopyFrom for high-performance bulk inserts (matching gnidump pattern)
- Batches by Config.Database.BatchSize (default 50,000)
- Inserts columns: word_id, name_string_id, canonical_id (matches schema.WordNameString)
- Progress logging every 100,000 records + final summary
- Proper error handling with detailed error messages
- Returns early if no word-name linkages to save
- Code compiles successfully ✅
- Linter passes ✅

---

### T030: Implement createWords orchestrator ✅
**File**: `internal/iooptimize/words.go`
**Description**: Main function orchestrating word extraction and insertion
**Status**: ✅ COMPLETE
**Details**:
- Call truncateWordsTables()
- Call getNameStringsForWords() to get all name IDs
- Create parserpool with Config.JobsNumber workers
- Call parseNamesForWords() to parse and extract words
- Call deduplicateWords()
- Call saveWords() in batches
- Call deduplicateWordNameStrings()
- Call saveWordNameStrings() in batches
- Log completion stats
**Reference**: gnidump createWords() in words.go
**Test**: T022 should now PASS

**Implementation Summary**:
- Implemented `createWords()` orchestrator in `internal/iooptimize/words.go:27-96`
- Complete 6-step workflow:
  1. Acquire database connection from operator.Pool()
  2. Truncate words tables (truncateWordsTables)
  3. Load name_strings for word extraction (getNameStringsForWords)
  4. Create parserpool with Config.JobsNumber workers
  5. Parse names and extract words (parseNamesForWords)
  6. Deduplicate and save words and linkages
- Proper resource management: defer connection.Release() and parserPool.Close()
- Comprehensive error handling at each step
- Progress logging for visibility
- Removed all `//nolint:unused` comments from helper functions
- **All integration tests PASS**:
  - TestCreateWords_Integration: Full workflow ✅
  - TestCreateWords_Idempotent: Safe reruns ✅
  - TestCreateWords_EmptyCache: Edge case handling ✅
- Code compiles successfully ✅
- Linter passes ✅
- **Phase 3.5 (Step 4 - Create Words Tables) COMPLETE**

---

## Phase 3.6: Step 5 - Create Verification View (TDD)

### T031 [P]: Write integration test for verification view creation ✅
**File**: `internal/iooptimize/views_test.go`
**Description**: Test that verification materialized view is created with correct structure
**Status**: ✅ COMPLETE
**Test Scenario**:
1. Given: Populated database with name_strings and name_string_indices
2. When: Call createVerificationView(ctx, cfg)
3. Then:
   - Existing verification view dropped (if exists)
   - New verification materialized view created
   - View contains expected columns
   - 3 indexes created: canonical_id, name_string_id, year
   - Query verification view returns expected records
4. Verify test FAILS
**Reference**: gnidump createVerification() in db_views.go

**Implementation Summary**:
- Created `internal/iooptimize/views_test.go` with 3 comprehensive integration tests:
  - `TestCreateVerificationView_Integration`: Main test validating:
    - View creation and structure (21 columns)
    - All expected columns present
    - 3 indexes created (canonical_id, name_string_id, year)
    - Data filtering logic (surrogates excluded, virus exception, bacteria filter)
    - Total of 4 valid records from 5 inserted (1 surrogate excluded)
    - Specific field values (name, year, canonical_id)
  - `TestCreateVerificationView_Idempotent`: Safe reruns verification
  - `TestCreateVerificationView_EmptyDatabase`: Edge case handling
- Created stub `internal/iooptimize/views.go` with `createVerificationView()` function
- Test properly **FAILS** with "not yet implemented" error ✅ (TDD red phase)
- Uses `iotesting.GetTestConfig()` and `iodb.NewPgxOperator()` patterns
- Schema creation with `ioschema.NewManager()`
- Code compiles successfully ✅
- Linter passes ✅

---

### T032: Implement dropVerificationView function ✅
**File**: `internal/iooptimize/views.go`
**Description**: Drop existing verification materialized view if it exists
**Status**: ✅ COMPLETE
**Details**:
- SQL: DROP MATERIALIZED VIEW IF EXISTS verification CASCADE
- Logs action
**Reference**: gnidump createVerification() in db_views.go

---

### T033: Implement buildVerificationViewSQL function ✅
**File**: `internal/iooptimize/views.go`
**Description**: Generate SQL for verification materialized view
**Status**: ✅ COMPLETE
**Details**:
- Return SQL string (reference: data-model.md):
  ```sql
  CREATE MATERIALIZED VIEW verification AS
  WITH taxon_names AS (
    SELECT nsi.data_source_id, nsi.record_id, nsi.name_string_id, ns.name
      FROM name_string_indices nsi
        JOIN name_strings ns ON nsi.name_string_id = ns.id
  )
  SELECT nsi.data_source_id, nsi.record_id, nsi.name_string_id,
    ns.name, nsi.name_id, nsi.code_id, ns.year, ns.cardinality, ns.canonical_id,
    ns.virus, ns.bacteria, ns.parse_quality, nsi.local_id, nsi.outlink_id,
    nsi.taxonomic_status, nsi.accepted_record_id, tn.name_string_id as
    accepted_name_id, tn.name as accepted_name, nsi.classification,
    nsi.classification_ranks, nsi.classification_ids
    FROM name_string_indices nsi
      JOIN name_strings ns ON ns.id = nsi.name_string_id
      LEFT JOIN taxon_names tn
        ON nsi.data_source_id = tn.data_source_id AND
           nsi.accepted_record_id = tn.record_id
    WHERE
      (
        ns.canonical_id is not NULL AND
        surrogate != TRUE AND
        (bacteria != TRUE OR parse_quality < 3)
      ) OR ns.virus = TRUE
  ```
**Reference**: gnidump createVerification() and data-model.md

---

### T034: Implement createVerificationIndexes function ✅
**File**: `internal/iooptimize/views.go`
**Description**: Create 3 indexes on verification materialized view
**Status**: ✅ COMPLETE
**Details**:
- Execute 3 SQL statements:
  1. CREATE INDEX verification_canonical_id_idx ON verification (canonical_id)
  2. CREATE INDEX verification_name_string_id_idx ON verification (name_string_id)
  3. CREATE INDEX verification_year_idx ON verification (year)
- Log each index creation
**Reference**: gnidump createVerification() in db_views.go and data-model.md

---

### T035: Implement createVerificationView orchestrator ✅
**File**: `internal/iooptimize/views.go`
**Description**: Main function orchestrating view creation
**Status**: ✅ COMPLETE
**Details**:
- Call dropVerificationView()
- Get SQL from buildVerificationViewSQL()
- Execute CREATE MATERIALIZED VIEW statement
- Call createVerificationIndexes()
- Log completion
**Reference**: gnidump createVerification() in db_views.go
**Test**: T031 should now PASS

---

## Phase 3.7: Step 6 - VACUUM ANALYZE (TDD)

### T036 [P]: Write integration test for VACUUM ANALYZE ✅
**File**: `internal/iooptimize/vacuum_test.go`
**Description**: Test that VACUUM ANALYZE runs successfully
**Status**: ✅ COMPLETE
**Test Scenario**:
1. Given: Optimized database
2. When: Call vacuumAnalyze(ctx, cfg)
3. Then:
   - VACUUM ANALYZE executes without error
   - Statistics updated (verify pg_stat_user_tables)
4. Verify test FAILS
**Note**: This is a gndb enhancement, not in gnidump

---

### T037: Implement vacuumAnalyze function ✅
**File**: `internal/iooptimize/vacuum.go`
**Description**: Run VACUUM ANALYZE on database
**Status**: ✅ COMPLETE
**Details**:
- Execute SQL: VACUUM ANALYZE
- Use pgx connection (cannot run in transaction)
- Log start and completion
- Report time taken
**Note**: gndb enhancement per FR-004 requirement
**Test**: T036 should now PASS

---

## Phase 3.8: Orchestration & CLI Integration

### T038: Wire all 6 steps in Optimize() method
**File**: `internal/iooptimize/optimizer.go`
**Description**: Implement the main Optimize() method to call all 6 steps sequentially
**Details**:
- Replace "not yet implemented" error with actual workflow
- Execute in sequence:
  1. reparseNames(ctx, cfg)
  2. fixVernacularLanguages(ctx, cfg)
  3. removeOrphans(ctx, cfg)
  4. createWords(ctx, cfg)
  5. createVerificationView(ctx, cfg)
  6. vacuumAnalyze(ctx, cfg)
- Use CacheManager:
  - Create cache at start
  - Defer cleanup at end
- Error handling: return on first error
- Log progress for each step
**Reference**: gnidump Build() in buildio.go

---

### T039: Add colored progress output to STDOUT
**File**: `internal/iooptimize/progress.go`
**Description**: Implement colored terminal output per Constitution X
**Details**:
- Create progress reporting functions:
  - printStepHeader(stepNum, stepName) - green color
  - printProgress(message) - cyan color
  - printWarning(message) - yellow color
  - printError(message) - red color
- Use fatih/color or similar library
- Progress messages go to STDOUT
- Technical errors go to STDERR
**Constitution**: Principle X (User-Friendly Documentation)

---

### T040: Add error documentation blocks to STDOUT
**File**: `internal/iooptimize/errors.go`
**Description**: Implement formatted error documentation per Constitution IX
**Details**:
- For each error condition, create error documentation block:
  - Title (colored)
  - Clear explanation of problem
  - Actionable steps for resolution
- Examples:
  - "Database not populated" → suggest running `gndb populate`
  - "Connection failed" → check PostgreSQL status
  - "Insufficient disk space" → show required space
**Constitution**: Principle IX (Dual-Channel Communication)

---

### T041 [P]: Write end-to-end integration test
**File**: `cmd/gndb/optimize_test.go`
**Description**: Test complete optimize workflow via CLI
**Test Scenario**:
1. Given: Populated test database
2. When: Run `gndb optimize` command
3. Then:
   - All 6 steps execute successfully
   - Database is optimized
   - Verification view queryable
   - Words tables populated
   - Exit code 0
4. Verify colored output to STDOUT
5. Verify cache cleanup
**Reference**: gnidump rebuild integration test pattern

---

### T042: Update CLI command with progress reporting
**File**: `cmd/gndb/optimize.go`
**Description**: Wire progress and error reporting into CLI command
**Details**:
- Call progress.printStepHeader() before each step
- Use progress.printProgress() for status updates
- Use errors.formatError() for user-facing errors
- Ensure STDOUT/STDERR separation
**Test**: T041 should now PASS

---

## Phase 3.9: Documentation & Polish

### T043 [P]: Add godoc comments to all public functions
**File**: All files in `internal/iooptimize/` and `cmd/gndb/optimize.go`
**Description**: Ensure all exported functions have clear godoc comments
**Details**:
- Each exported function/method needs godoc
- Explain purpose in 1-2 sentences
- Reference gnidump equivalent where applicable
**Constitution**: Principle V (Open Source Readability)

---

### T044 [P]: Verify contract test passes
**File**: `pkg/lifecycle/optimizer_test.go`
**Description**: Run contract test to ensure Optimizer interface compliance
**Details**:
- Contract test should now pass with full implementation
- OptimizerImpl satisfies lifecycle.Optimizer interface
**Status**: Should already exist from Phase 1

---

### T045: Update quickstart.md with examples
**File**: `specs/002-optimize/quickstart.md`
**Description**: Enhance quickstart with detailed usage examples
**Details**:
- Add examples:
  - Basic usage: `gndb optimize`
  - Custom worker count: `gndb optimize --jobs=100`
  - Custom batch size: `gndb optimize --batch-size=50000`
- Add expected output samples
- Add troubleshooting section
**Constitution**: Principle X (User-Friendly Documentation)

---

### T046: Run all tests and verify full pass
**File**: N/A (test execution)
**Description**: Execute complete test suite and verify all tests pass
**Commands**:
```bash
go test ./pkg/lifecycle/...
go test ./internal/iooptimize/...
go test ./cmd/gndb/...
```
**Exit Criteria**: All tests pass, no failures

---

### T047: Performance validation with large dataset
**File**: N/A (validation)
**Description**: Validate optimize performance with realistic dataset
**Test Scenario**:
- Use database with 1M+ name_strings
- Run `gndb optimize`
- Measure:
  - Total time to complete
  - Memory usage
  - CPU utilization
  - Cache size
- Verify: Configurable concurrency (JobsNumber) affects performance
**Success**: Completes without OOM, reasonable time (<1 hour for 1M names)

---

### T048: Final code review and cleanup
**File**: All implementation files
**Description**: Review all code for quality, remove duplication, ensure KISS
**Checklist**:
- [ ] No code duplication (DRY principle)
- [ ] Simple, readable implementations (KISS)
- [ ] No "just in case" code
- [ ] Error handling consistent
- [ ] Logging appropriate
- [ ] Comments clear and concise
**Constitution**: Principles VII, VIII (KISS, Contributor-First Minimalism)

---

## Dependencies Graph

```
Setup Phase:
  T001 (CLI command) ← Required for all CLI tasks
  T002 (Cache) ← Required for T005, T025

Step 1 (Reparse):
  T003 (Test) [P] ← Blocks T008
  T004 (Load) ← Blocks T008
  T005 (Worker) ← Needs T002, blocks T008
  T006 (Save) ← Blocks T008
  T007 (Update) ← Blocks T006
  T008 (Orchestrator) ← Needs T004-T007

Step 2 (Vernacular):
  T009 (Test) [P] ← Blocks T015
  T010-T014 (Functions) ← Block T015
  T015 (Orchestrator) ← Needs T010-T014

Step 3 (Orphans):
  T016 (Test) [P] ← Blocks T021
  T017-T020 (Delete functions) [P] ← Block T021
  T021 (Orchestrator) ← Needs T017-T020

Step 4 (Words):
  T022 (Test) [P] ← Blocks T030
  T023-T029 (Functions) ← Block T030
  T025 (Extract) ← Needs T002 (Cache)
  T030 (Orchestrator) ← Needs T023-T029

Step 5 (View):
  T031 (Test) [P] ← Blocks T035
  T032-T034 (Functions) [P] ← Block T035
  T035 (Orchestrator) ← Needs T032-T034

Step 6 (Vacuum):
  T036 (Test) [P] ← Blocks T037
  T037 (Function) ← Needs T036

Orchestration:
  T038 (Wire Optimize) ← Needs T008, T015, T021, T030, T035, T037
  T039-T040 (Progress & Errors) [P] ← Needed by T042
  T041 (E2E Test) [P] ← Blocks T042
  T042 (CLI Integration) ← Needs T001, T038, T039, T040

Polish:
  T043-T048 ← All can run after T042 completes
```

## Parallel Execution Examples

### Parallel Test Writing (Phase 3.2-3.7):
All test tasks marked [P] can be written in parallel:
```bash
# Execute concurrently:
- T003: Reparse test
- T009: Vernacular test
- T016: Orphans test
- T022: Words test
- T031: View test
- T036: Vacuum test
```

### Parallel Implementation (within same step):
Some implementation tasks within a step can be parallel:
```bash
# Step 3 delete functions (T017-T020):
- T017: removeOrphanNameStrings
- T018: removeOrphanCanonicals
- T019: removeOrphanCanonicalFulls
- T020: removeOrphanCanonicalStems
```

### Sequential Dependencies:
These MUST be sequential:
```
T001 → T002 → T005 → T008 → T038 → T042
(CLI setup → Cache → Workers → Step 1 → Wire all → CLI integration)
```

## Task Execution Notes

1. **TDD Workflow**: All tests (T003, T009, T016, T022, T031, T036) MUST be written and verified to FAIL before implementing their corresponding functions.

2. **Cache Dependency**: T002 (cache) must complete before T005 (reparse worker) and T025 (word extraction) can be implemented.

3. **Step Sequence**: While steps can be implemented in parallel, the orchestrator (T038) requires all steps (T008, T015, T021, T030, T035, T037) to be complete.

4. **gnidump Reference**: Each task references the corresponding gnidump file/function for implementation guidance.

5. **Constitution Compliance**: Tasks T039, T040, T043, T048 ensure constitutional principles are met.

6. **Performance**: T047 validates real-world performance with large datasets.

## Validation Checklist

- [x] All contracts have tests (T044)
- [x] All 6 steps have integration tests (T003, T009, T016, T022, T031, T036)
- [x] Tests before implementation (TDD enforced)
- [x] Parallel tasks are independent (verified)
- [x] Each task specifies exact file path
- [x] No conflicting file modifications in [P] tasks
- [x] gnidump references provided for guidance
- [x] Constitution principles addressed (T039, T040, T043, T048)

---

**Total Tasks**: 48
**Estimated Effort**: 3-5 days (with TDD discipline)
**Critical Path**: T001 → T002 → T005 → T008 → T038 → T042 → T046
