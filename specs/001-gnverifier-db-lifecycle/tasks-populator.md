# Tasks: Populator Implementation

**Feature Branch**: `001-gnverifier-db-lifecycle`
**Status**: In Progress (T034-T037 Complete)
**Prerequisites**: T001-T033 complete (interfaces, contracts, CLI wiring)

**Context**:
- Populator interface defined in `pkg/lifecycle/populator.go`
- Stub implementation exists in `internal/io/populate/populator.go`
- Comprehensive implementation plan documented in populator.go comments
- Contract test passing with stub

---

## Phase 4.1: Cache Setup

### T034: [P] Implement Cache Management Functions ✅

**Description**: Create cache directory management helper functions

**Actions**:
1. Create `internal/io/populate/cache.go`
2. Implement `clearCache(cacheDir string) error`:
   - Remove all files in ~/.cache/gndb/sfga/
   - Create directory if doesn't exist
3. Implement `prepareCacheDir() (string, error)`:
   - Call config.GetCacheDir()
   - Append "sfga" subdirectory
   - Clear cache directory
   - Return cache path
4. Write unit tests in `cache_test.go`

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/cache.go` (new)
- `/Users/dimus/code/golang/gndb/internal/io/populate/cache_test.go` (new)

**Success Criteria**:
- [x] clearCache removes all files
- [x] prepareCacheDir returns correct path
- [x] Unit tests pass
- [x] Handles missing directories gracefully

**Dependencies**: None

---

## Phase 4.2: Sources Configuration Loading

### T035: [P] Implement Sources Filtering Logic ✅

**Description**: Add CLI filtering support to LoadSourcesConfig

**Actions**:
1. Add to `pkg/populate/sources.go`:
   - `FilterSources(sources []DataSourceConfig, filter string) ([]DataSourceConfig, error)`
   - Handle "main" (ID < 1000)
   - Handle "exclude main" (ID >= 1000)
   - Handle comma-separated IDs: "1,3,5"
2. Write unit tests in `sources_test.go`

**File Paths**:
- `/Users/dimus/code/golang/gndb/pkg/populate/sources.go`
- `/Users/dimus/code/golang/gndb/pkg/populate/sources_test.go`

**Success Criteria**:
- [x] FilterSources("main") returns ID < 1000
- [x] FilterSources("1,3,5") returns correct sources
- [x] All unit tests pass

**Dependencies**: None

---

### T036: Update populate CLI Command with Filtering Flags ✅

**Description**: Wire sources filtering to CLI command

**Actions**:
1. Update `cmd/gndb/populate.go`:
   - Add --sources flag (string, default: "")
   - Add --release-version flag (string, default: "")
   - Add --release-date flag (string, default: "")
   - Load sources.yaml using populate.LoadSourcesConfig()
   - Apply FilterSources with --sources value
   - Validate override flags (CLI constraint: only single source)
   - Apply version/release-date overrides if single source
2. Add error messages for validation failures
3. Update help text with examples

**File Paths**:
- `/Users/dimus/code/golang/gndb/cmd/gndb/populate.go`

**Success Criteria**:
- [x] Flags available in `gndb populate --help`
- [x] Validation errors displayed clearly
- [x] Correct sources filtered based on flags

**Dependencies**: T035

---

## Phase 4.3: SFGA Fetching & Opening

### T037: [P] Implement SFGA Fetcher Wrapper ✅

**Description**: Create wrapper around sflib Archive.Fetch

**Actions**:
1. Create `internal/io/populate/sfga.go`
2. Implement `fetchSFGA(source DataSourceConfig, cacheDir string) (string, error)`:
   - Use sflib Archive.Fetch to download/extract SFGA to cache
   - Handle local files and URLs
   - Handle .zip extraction automatically
   - Return path to extracted SQLite file
3. Implement `openSFGA(sqlitePath string) (*sql.DB, error)`:
   - Open SQLite database
   - Return database handle
4. Write unit tests with mock files

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/sfga.go` (new)
- `/Users/dimus/code/golang/gndb/internal/io/populate/sfga_test.go` (new)

**Success Criteria**:
- [x] fetchSFGA handles local files
- [x] fetchSFGA handles URLs
- [x] openSFGA returns valid db handle
- [x] Unit tests pass

**Dependencies**: T034

---

## Phase 4.4: Name Strings Processing (Phase 1)

### T038: [P] Write Integration Test for Name Strings Import ✅

**Description**: Create failing test for name strings import

**Actions**:
1. Create `internal/io/populate/names_integration_test.go`
2. Test scenario:
   - Create test SFGA with sample names
   - Call processNameStrings()
   - Verify name_strings table populated
   - Verify UUID v5 generation correct
   - Verify ON CONFLICT DO NOTHING works
3. Mark test as integration (skip in short mode)

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/names_integration_test.go` (new)

**Success Criteria**:
- [x] Test fails (function not implemented)
- [x] Test clearly specifies expected behavior
- [x] Uses testdata SFGA file

**Dependencies**: T037

---

### T038.5: [P] Implement GNparser Pool Wrapper ✅

**Description**: Create wrapper around gnparser.NewPool for botanical and zoological parsing

**Actions**:
1. Create `pkg/parserpool/pool.go` (pure - parsing is computation, not I/O)
2. Define `Pool` interface:
   ```go
   type Pool interface {
       Parse(nameString string, code nomcode.Code) (*gnparser.Parsed, error)
       Close()
   }
   ```
3. Implement `PoolImpl` struct in same file:
   - Two channels: botanicalCh, zoologicalCh (chan gnparser.GNparser)
   - poolSize field
4. Implement `NewPool(jobsNum int) Pool`:
   - poolSize = jobsNum (default: runtime.NumCPU() if 0)
   - Create two parser pools using gnparser.NewPool():
     * Botanical: cfg.Code = nomcode.Botanical
     * Zoological: cfg.Code = nomcode.Zoological
   - Example: `botanicalCh := gnparser.NewPool(gnparser.NewConfig(gnparser.OptFormat(gnfmt.CSV), gnparser.OptWithBotanicalCode()), poolSize)`
5. Implement `Parse(nameString string, code nomcode.Code)`:
   - Select channel based on code (botanical vs zoological)
   - Get parser from channel: `parser := <-ch` (blocks if all busy)
   - Parse: `result := parser.ParseName(nameString)`
   - Return parser to channel: `ch <- parser`
   - Return parsed result
6. Implement `Close()`:
   - Close both channels
   - Drain remaining parsers
7. Write unit tests in `pkg/parserpool/pool_test.go`:
   - Test concurrent parsing with multiple goroutines
   - Test nomcode.Botanical and nomcode.Zoological
   - Test blocking when pool exhausted
   - Test Close() cleanup
   - Run with -race flag

**File Paths**:
- `/home/dimus/code/golang/gndb/pkg/parserpool/pool.go` (new, interface + implementation)
- `/home/dimus/code/golang/gndb/pkg/parserpool/pool_test.go` (new)

**Success Criteria**:
- [x] Pool interface defined in pkg/parserpool/
- [x] PoolImpl uses gnparser.NewPool() correctly
- [x] Config.Code set to nomcode.Botanical and nomcode.Zoological
- [x] Concurrent parsing works without race conditions
- [x] Close() properly cleans up resources
- [x] Unit tests pass with -race flag

**Dependencies**: None (can be done in parallel with T038)

**Notes**:
- Uses gnparser.NewPool(cfg Config, size int) chan GNparser
- Config.Code field (nomcode.Code) differentiates botanical vs zoological
- Pool size defaults to runtime.NumCPU() when jobsNum == 0
- Total parsers = 2 * poolSize (one pool per code)
- Used by T041 (hierarchy building) for concurrent canonical parsing
- Channel-based approach from gnparser provides natural backpressure

---

### T039: Implement Name Strings Processing ✅

**Description**: Implement Phase 1 - Name Strings import

**Actions**:
1. Create `internal/io/populate/names.go`
2. Implement `processNameStrings(ctx context.Context, p *PopulatorImpl, sfgaDB *sql.DB, sourceID int) error`:
   - Query SFGA: `SELECT col__id, gn__scientific_name_string, col__scientific_name FROM name`
   - Validate gn__scientific_name_string not empty
   - If empty: prompt user (yes/no/abort) with count
   - Generate UUID v5 using gnuuid.New(nameString).String()
   - Batch insert using parameterized INSERT with ON CONFLICT DO NOTHING
   - Progress logging with humanize.Comma
3. Add helper `promptUser(message string) (string, error)` for interactive prompts
4. Batch size from config (default 5000)

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/names.go` (new)

**Success Criteria**:
- [x] Integration test passes
- [x] Handles empty gn__scientific_name_string
- [x] Progress logged to stderr
- [x] ON CONFLICT works correctly

**Dependencies**: T038

---

## Phase 4.5: Hierarchy Building (Phase 2 Part 1)

### T040: [P] Write Integration Test for Hierarchy Building ✅

**Description**: Create failing test for hierarchy map generation

**Actions**:
1. Create `internal/io/populate/hierarchy_integration_test.go`
2. Test scenario:
   - Create test SFGA with parent-child taxon relationships
   - Call buildHierarchy()
   - Verify hierarchy map built correctly
   - Verify getBreadcrumbs() returns correct classification strings
3. Include edge cases: missing parents, circular refs

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/hierarchy_integration_test.go` (new)

**Success Criteria**:
- [x] Test fails (function not implemented)
- [x] Covers parent-child walking
- [x] Tests flat classification fallback

**Dependencies**: T037

---

### T041: Implement Hierarchy Builder ✅

**Description**: Implement hierarchy map generation from SFGA taxon table

**Actions**:
1. Create `internal/io/populate/hierarchy.go`
2. Define `hNode` struct (id, parentID, name, rank, taxonomicStatus)
3. Implement `buildHierarchy(ctx context.Context, sfgaDB *sql.DB) (map[string]*hNode, error)`:
   - Query: `SELECT t.col__id, t.col__parent_id, t.col__status_id, n.col__scientific_name, n.col__rank_id FROM taxon t JOIN name n ON n.col__id = t.col__name_id`
   - Use errgroup for concurrent parsing with gnparser (botanical code)
   - Build map[id]*hNode with parent relationships
   - Handle self-referencing parent IDs
   - Progress logging
4. Implement `getBreadcrumbs(id string, hierarchy map[string]*hNode, flatClsf map[string]string) (classification, classificationRanks, classificationIDs string)`:
   - Walk up parent chain to root
   - Fallback to flat classification if len(nodes) < 2
   - Return 3 pipe-delimited strings
5. Implement `flatClassification()` helper with predefined ranks

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/hierarchy.go` (new)

**Success Criteria**:
- [x] Integration test passes
- [x] Hierarchy map built correctly
- [x] getBreadcrumbs handles missing parents
- [x] Concurrent parsing works

**Dependencies**: T040

---

## Phase 4.6: Name Indices Processing (Phase 2 Part 2)

### T041.5: [P] Add Outlink Column Configuration with gnoutlink Namespace Support ✅

**Description**: Add outlink_id_column field using "table.column" format with automatic `gnoutlink:` namespace extraction from col__alternative_id

**Actions**:
1. Update `pkg/populate/sources.go`:
   - Add `OutlinkIDColumn string` field to `DataSourceConfig`
   - Format: `"table.column"` (e.g., "taxon.col__id", "name.col__alternative_id")
   - Update Validate() to check OutlinkIDColumn format if IsOutlinkReady=true:
     * Must contain exactly one dot: `table.column`
     * Table must be one of: "taxon", "name", "synonym"
     * Column must be one of: "col__id", "col__name_id", "col__local_id", "col__alternative_id"
     * Exception: "synonym.col__alternative_id" is NOT allowed (synonym table lacks this column)
   - Add helper `ExtractOutlinkID(columnName, value string) string`:
     * If columnName ends with "col__alternative_id": Extract value after "gnoutlink:" from comma-separated list
     * Else: Return value as-is
     * Example: "gbif:123,gnoutlink:Homo_sapiens" → "Homo_sapiens"
2. Update template `pkg/templates/sources.yaml`:
   - Document the gnoutlink namespace convention:
     * Direct columns: "taxon.col__id", "name.col__name_id" (use value as-is)
     * Alternative ID: "taxon.col__alternative_id", "name.col__alternative_id" (auto-extract gnoutlink:)
     * Format in SFGA: "namespace1:id1,gnoutlink:transformed_id"
   - Document table availability by record type:
     * "taxon.*" - Taxa & synonyms only (bare names get empty)
     * "name.*" - All record types (taxa, synonyms, bare names)
     * "synonym.*" - Synonyms only
3. Write unit tests in `sources_test.go`:
   - Validate(): Test valid formats "taxon.col__id", "name.col__alternative_id"
   - Validate(): Test invalid formats (no dot, wrong table, wrong column)
   - ExtractOutlinkID(): Test direct column (returns value as-is)
   - ExtractOutlinkID(): Test alternative_id with gnoutlink namespace
   - ExtractOutlinkID(): Test alternative_id with multiple namespaces
   - ExtractOutlinkID(): Test alternative_id without gnoutlink (returns empty)

**File Paths**:
- `/home/dimus/code/golang/gndb/pkg/populate/sources.go`
- `/home/dimus/code/golang/gndb/pkg/populate/sources_test.go`
- `/home/dimus/code/golang/gndb/pkg/templates/sources.yaml`

**Success Criteria**:
- [x] OutlinkIDColumn field uses "table.column" format
- [x] Validation enforces allowed table and column names
- [x] ExtractOutlinkID() auto-detects col__alternative_id and extracts gnoutlink:
- [x] Template documents gnoutlink namespace convention
- [x] Unit tests cover validation and extraction logic

**Dependencies**: T035 (sources filtering)

**Note**: Complex transformations (URL encoding, string replacement) should be done during SFGA creation and stored in col__alternative_id with "gnoutlink:" prefix. This preserves original SFGA data while supporting custom outlink IDs.

**Synonym Table Limitation**: The synonym table does not have col__alternative_id. When using "taxon.col__alternative_id", synonyms will get the accepted taxon's outlink ID (pointing to the accepted taxon's page). If synonym-specific outlinks are needed, transformations must be stored in "name.col__alternative_id" instead.

---

### T042: [P] Write Integration Test for Name Indices Import ✅

**Description**: Create failing test for name indices with classification

**Actions**:
1. Create `internal/io/populate/indices_integration_test.go`
2. Test scenarios:
   - Names with taxon records → full classification
   - Names without taxon records → bare names
   - Empty taxon table → all bare names
   - Verify classification strings correct
   - Verify DELETE old records works
3. Use testdata SFGA

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/indices_integration_test.go` (new)

**Success Criteria**:
- [x] Test fails (function not implemented)
- [x] Covers all name index scenarios
- [x] Tests classification integration

**Dependencies**: T037

---

### T043: Implement Name Indices Processing ✅

**Description**: Implement Phase 2 - Name Indices with classification

**Actions**:
1. Create `internal/io/populate/indices.go`
2. Implement `processNameIndices(ctx context.Context, p *PopulatorImpl, sfgaDB *sql.DB, sourceID int, hierarchy map[string]*hNode, cfg *config.Config) error`:
   - Clean old data: `DELETE FROM name_string_indices WHERE data_source_id = $1`
   - Three-phase processing:
     * Phase 1: processTaxa() - accepted names with classification from hierarchy
     * Phase 2: processSynonyms() - synonyms linked to accepted taxa
     * Phase 3: processBareNames() - orphan names with "bare-name-" prefix
   - Use getBreadcrumbs() for classification (respects WithFlatClassification flag)
   - Bulk insert using pgx.CopyFrom
   - Progress logging
3. Handle empty taxon table (all names are bare)
4. Include rank, taxonomic_status, accepted_record_id

**File Paths**:
- `/home/dimus/code/golang/gndb/internal/io/populate/indices.go` (new)

**Success Criteria**:
- [x] Integration test passes (33,297 records from vascan)
- [x] Classifications generated correctly (pipe-delimited strings)
- [x] Bare names handled (with "bare-name-{id}" prefix)
- [x] Old data cleaned (idempotency test passes)
- [x] Synonyms linked to accepted taxa via AcceptedRecordID

**Dependencies**: T041, T042

---

### T043.5: Read Outlink ID from SFGA Columns During Import ✅

**Description**: Update name indices processing to read outlink_id from SFGA tables and populate name_string_indices.outlink_id column

**Actions**:
1. Add helper function in `internal/iopopulate/indices.go`:
   ```go
   // buildOutlinkColumn maps table.column format to query alias
   // Returns column expression for SELECT or empty string if not available
   func buildOutlinkColumn(outlinkColumn string, queryType string) string
   ```
   - Parse `"table.column"` format (e.g., "taxon.col__id" → table="taxon", column="col__id")
   - Map table name to query alias based on queryType:
     * "taxa": taxon→t, name→n
     * "synonyms": taxon→t (accepted), synonym→s, name→n
     * "bare_names": name→name (no alias)
   - Return formatted column (e.g., "t.col__id", "n.col__name_id") or empty if table not available
   - Write unit tests for buildOutlinkColumn with all combinations

2. Update `internal/iopopulate/indices.go`:
   - Add `source DataSourceConfig` parameter to `processNameIndices()`, `processTaxa()`, `processSynonyms()`, `processBareNames()`
   - In each function:
     * Call `buildOutlinkColumn(source.OutlinkIDColumn, queryType)`
     * If result not empty, add to SELECT: `, {result} AS outlink_id`
     * Scan outlink_id from query results
     * Insert into name_string_indices.outlink_id via bulk insert
   - Example for processTaxa():
     ```go
     outlinkCol := buildOutlinkColumn(source.OutlinkIDColumn, "taxa")
     query := `SELECT t.col__id, n.col__id, n.gn__scientific_name_string, ...`
     if outlinkCol != "" {
         query += `, ` + outlinkCol + ` AS outlink_id`
     }
     query += ` FROM taxon t JOIN name n ON n.col__id = t.col__name_id`
     
     // Scan including outlink_id
     var outlinkID string
     err := rows.Scan(&taxonID, &nameID, &nameString, ..., &outlinkID)
     
     // Use in bulk insert to name_string_indices
     record := []interface{}{sourceID, recordID, nameStringID, outlinkID, ...}
     ```

3. Update integration test in `indices_integration_test.go`:
   - Test "taxon.col__id": Taxa & synonyms get value, bare names get empty
   - Test "name.col__id": All record types get value (including bare names)
   - Test with is_outlink_ready=false (all get empty)
   - Verify name_string_indices.outlink_id populated correctly

**File Paths**:
- `/home/dimus/code/golang/gndb/internal/iopopulate/indices.go`
- `/home/dimus/code/golang/gndb/internal/iopopulate/indices_test.go`
- `/home/dimus/code/golang/gndb/internal/iopopulate/indices_integration_test.go`

**Success Criteria**:
- [x] buildOutlinkColumn() correctly maps table.column to query aliases
- [x] Taxa records populate outlink_id in name_string_indices
- [x] Synonyms records populate outlink_id in name_string_indices
- [x] Bare names populate outlink_id if using name table (e.g., "name.col__id")
- [x] Bare names get empty outlink_id if using taxon table (e.g., "taxon.col__id")
- [x] Integration test verifies all scenarios
- [x] Unit tests for buildOutlinkColumn pass

**Dependencies**: T041.5, T043

**Note**: This change requires updating the `processNameIndices()` call site in `populator.go` (T048) to pass the source config.

---

## Phase 4.7: Vernacular Processing (Phases 3 & 4)

### T044: [P] Write Integration Test for Vernaculars ✅

**Description**: Create failing test for vernacular names import

**Actions**:
1. Create `internal/io/populate/vernaculars_integration_test.go`
2. Test scenarios:
   - Vernacular strings inserted uniquely
   - Vernacular indices link to name strings
   - ON CONFLICT DO NOTHING works
   - Verify counts correct
3. Use testdata SFGA with vernaculars

**File Paths**:
- `/home/dimus/code/golang/gndb/internal/io/populate/vernaculars_integration_test.go` (new)

**Success Criteria**:
- [x] Test fails (function not implemented: "undefined: processVernaculars")
- [x] Covers vernacular strings and indices (3 test scenarios)
- [x] Tests uniqueness (deduplication of "Common plantain")
- [x] Tests idempotency (data cleaning)
- [x] Tests empty table handling

**Dependencies**: T037

---

### T045: Implement Vernacular Processing ✅

**Description**: Implement Phases 3 & 4 - Vernacular strings and indices

**Actions**:
1. Create `internal/io/populate/vernaculars.go`
2. Implement `processVernaculars(ctx context.Context, p *PopulatorImpl, sfgaDB *sql.DB, sourceID int) error`:
   - Phase 1: Read SFGA vernacular table
     * Generate UUID v5 for vernacular strings
     * Batch insert with ON CONFLICT DO NOTHING (30k batch size)
     * UTF-8 fixing and truncation (500 char limit)
   - Phase 2: Create VernacularStringIndex records
     * Clean old data (DELETE WHERE data_source_id)
     * Link to vernacular strings via UUID v5
     * Include language, locality, country_code metadata
     * Bulk insert using pgx.CopyFrom
   - Progress logging for both phases

**File Paths**:
- `/home/dimus/code/golang/gndb/internal/io/populate/vernaculars.go` (new, 350 lines)

**Success Criteria**:
- [x] All 3 integration tests pass
- [x] Vernacular strings unique (ON CONFLICT DO NOTHING)
- [x] Indices link correctly via UUID v5
- [x] Progress logged (Phase 1, Phase 2, counts)
- [x] Idempotency verified (data cleaning works)
- [x] Empty table handling works

**Dependencies**: T044

---

## Phase 4.8: Data Source Metadata (Phase 5)

### T046: [P] Write Integration Test for Data Source Metadata ✅

**Description**: Create failing test for data_sources record creation

**Actions**:
1. Create `internal/io/populate/metadata_integration_test.go`
2. Test scenarios:
   - New data source created
   - Existing data source updated
   - Metadata from SFGA + sources.yaml merged
   - Counts (names, vernaculars) queried correctly
   - updated_at timestamp set
3. Verify all metadata fields populated

**File Paths**:
- `/home/dimus/code/golang/gndb/internal/io/populate/metadata_integration_test.go` (new, 400 lines)

**Success Criteria**:
- [x] Test fails (function not implemented: "undefined: updateDataSourceMetadata")
- [x] Tests create and update scenarios (3 test cases)
- [x] Verifies counts (name_string_indices and vernacular_string_indices)
- [x] Tests metadata merging (SFGA metadata + sources.yaml config)
- [x] Tests idempotency (existing data source updated)
- [x] Tests empty/null metadata handling

**Dependencies**: T037

---

### T047: Implement Data Source Metadata ✅

**Description**: Implement Phase 5 - Data source record creation

**Actions**:
1. Create `internal/io/populate/metadata.go`
2. Implement `updateDataSourceMetadata(ctx context.Context, p *PopulatorImpl, source DataSourceConfig, sfgaDB *sql.DB) error`:
   - Read SFGA metadata (title, description, doi)
   - Merge with sources.yaml overrides (title, description can be overridden)
   - Query counts:
     * `SELECT COUNT(*) FROM name_string_indices WHERE data_source_id = $1`
     * `SELECT COUNT(*) FROM vernacular_string_indices WHERE data_source_id = $1`
   - DELETE existing data_sources record (idempotency)
   - INSERT new data_sources record
   - Set updated_at = NOW()
3. Handle empty/NULL SFGA metadata gracefully

**File Paths**:
- `/home/dimus/code/golang/gndb/internal/io/populate/metadata.go` (new, ~240 lines)

**Success Criteria**:
- [x] All 3 integration tests pass
- [x] Metadata merged correctly (SFGA + sources.yaml)
- [x] Counts accurate (queried from indices tables)
- [x] Idempotent (DELETE + INSERT pattern, tested with 2 updates)
- [x] Empty/NULL metadata handled gracefully
- [x] UpdatedAt timestamp set correctly

**Dependencies**: T046

---

## Phase 4.9: Main Orchestration

### T048: Wire All Phases in Populator.Populate() ✅

**Description**: Implement main Populate method orchestrating all phases

**Actions**:
1. Update `internal/io/populate/populator.go` Populate() method:
   - Load sources.yaml with filtering (use T036 flags)
   - Validate override flags (T035)
   - For each source:
     * Prepare cache directory (T034)
     * Fetch SFGA (T037)
     * Open SFGA database (T037)
     * Run Phase 1: processNameStrings (T039)
     * Build hierarchy (T041)
     * Run Phase 2: processNameIndices (T043)
     * Run Phases 3-4: processVernaculars (T045)
     * Run Phase 5: updateDataSourceMetadata (T047)
     * Close SFGA database
   - Error handling with context cancellation
   - Log overall progress and timing
2. Remove "not implemented" error

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/populator.go`

**Success Criteria**:
- [X] All phases execute in sequence
- [X] Error handling works
- [X] Context cancellation respected
- [X] Progress logged clearly

**Dependencies**: T036, T039, T041, T043, T045, T047

---

## Phase 4.10: End-to-End Testing

### T049: Create End-to-End Populate Integration Test ✅

**Description**: Test complete populate workflow from CLI to database

**Actions**:
1. Create `cmd/gndb/populate_e2e_test.go`
2. Test workflow:
   - Create test database
   - Generate test sources.yaml
   - Create test SFGA with all tables (name, taxon, vernacular)
   - Run populate command
   - Verify all tables populated:
     * name_strings
     * name_string_indices with classifications
     * vernacular_strings
     * vernacular_string_indices
     * data_sources
   - Verify counts match
   - Test idempotency (run twice, same result)
3. Clean up test database

**File Paths**:
- `/Users/dimus/code/golang/gndb/cmd/gndb/populate_e2e_test.go` (new)

**Success Criteria**:
- [X] E2E test passes
- [X] All phases execute
- [X] Data verified in database
- [X] Idempotent

**Dependencies**: T048

---

### T050: [P] Update Quickstart Documentation

**Description**: Update quickstart.md with actual populate usage

**Actions**:
1. Update `specs/001-gnverifier-db-lifecycle/quickstart.md`:
   - Update section 3.3 with real sources.yaml path
   - Add examples of --sources filtering
   - Add example of --release-version/--release-date override
   - Document cache location for debugging
   - Add troubleshooting section
2. Verify all commands work as documented

**File Paths**:
- `/Users/dimus/code/golang/gndb/specs/001-gnverifier-db-lifecycle/quickstart.md`

**Success Criteria**:
- [X] Quickstart updated with real examples
- [X] All commands verified
- [X] Cache debugging documented

**Dependencies**: T049

---

### T051: [P] Update CLAUDE.md with Populate Architecture

**Description**: Document populate implementation in agent context

**Actions**:
1. Update `CLAUDE.md` MANUAL ADDITIONS section:
   - Add populate workflow overview
   - Document 5-phase processing approach
   - Note cache strategy (~/.cache/gndb/sfga/)
   - List key files and their responsibilities
   - Keep under 150 lines total
2. Run `.specify/scripts/bash/update-agent-context.sh claude`

**File Paths**:
- `/Users/dimus/code/golang/gndb/CLAUDE.md`

**Success Criteria**:
- [X] Architecture documented clearly
- [X] Under 150 lines total (110 lines)
- [X] Key patterns highlighted

**Dependencies**: T049

---

## Summary

**Total Tasks**: 20 (T034-T051, including T041.5 and T043.5)
**Completed**: 4 (T034 ✅, T035 ✅, T036 ✅, T037 ✅)
**Parallel Tasks**: 11 (T034, T035, T037, T038, T040, T041.5, T042, T044, T046, T050, T051)
**Critical Path**: T034→T036→T037→T039→T041→T041.5→T043→T043.5→T045→T047→T048→T049→T050

**Phase Breakdown**:
- Phase 4.1: Cache Setup (T034) - 1 task [P] ✅
- Phase 4.2: Sources Config (T035-T036) - 2 tasks, 1 [P] ✅
- Phase 4.3: SFGA Fetching (T037) - 1 task [P] ✅
- Phase 4.4: Name Strings (T038-T039) - 2 tasks, 1 [P]
- Phase 4.5: Hierarchy (T040-T041) - 2 tasks, 1 [P]
- Phase 4.6: Name Indices (T041.5, T042-T043, T043.5) - 4 tasks, 2 [P]
- Phase 4.7: Vernaculars (T044-T045) - 2 tasks, 1 [P]
- Phase 4.8: Metadata (T046-T047) - 2 tasks, 1 [P]
- Phase 4.9: Orchestration (T048) - 1 task
- Phase 4.10: Testing & Docs (T049-T051) - 3 tasks, 2 [P]

**Estimated Effort**: 15-19 hours focused work (4 tasks complete, 2 new tasks added)

**Key Patterns**:
- TDD: Integration tests before implementation
- Sequential phases: Each phase builds on previous
- Parallel where possible: Different files, no dependencies
- Reference implementation: github.com/sfborg/to-gn patterns

**Next Steps**:
1. T038: Write Integration Test for Name Strings Import
2. T039: Implement Name Strings Processing
3. T041.5: Implement Outlink ID Extraction Logic (NEW - for outlink URL templates)
4. Continue through phases 4.5-4.9 (hierarchy, indices with outlink IDs, vernaculars, metadata)
5. T043.5: Update Name Indices Processing to Use Outlink IDs (NEW)
6. T048: Wire all phases in main Populate() orchestration
7. T049-T051: End-to-end testing and documentation

---

*Constitution v1.3.0 | TDD | Pure/Impure Separation | CLI-First*
