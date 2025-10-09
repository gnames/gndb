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

### T038: [P] Write Integration Test for Name Strings Import

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
- [ ] Test fails (function not implemented)
- [ ] Test clearly specifies expected behavior
- [ ] Uses testdata SFGA file

**Dependencies**: T037

---

### T039: Implement Name Strings Processing

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
- [ ] Integration test passes
- [ ] Handles empty gn__scientific_name_string
- [ ] Progress logged to stderr
- [ ] ON CONFLICT works correctly

**Dependencies**: T038

---

## Phase 4.5: Hierarchy Building (Phase 2 Part 1)

### T040: [P] Write Integration Test for Hierarchy Building

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
- [ ] Test fails (function not implemented)
- [ ] Covers parent-child walking
- [ ] Tests flat classification fallback

**Dependencies**: T037

---

### T041: Implement Hierarchy Builder

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
- [ ] Integration test passes
- [ ] Hierarchy map built correctly
- [ ] getBreadcrumbs handles missing parents
- [ ] Concurrent parsing works

**Dependencies**: T040

---

## Phase 4.6: Name Indices Processing (Phase 2 Part 2)

### T042: [P] Write Integration Test for Name Indices Import

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
- [ ] Test fails (function not implemented)
- [ ] Covers all name index scenarios
- [ ] Tests classification integration

**Dependencies**: T037

---

### T043: Implement Name Indices Processing

**Description**: Implement Phase 2 - Name Indices with classification

**Actions**:
1. Create `internal/io/populate/indices.go`
2. Implement `processNameIndices(ctx context.Context, p *PopulatorImpl, sfgaDB *sql.DB, sourceID int, hierarchy map[string]*hNode) error`:
   - Clean old data: `DELETE FROM name_string_indices WHERE data_source_id = $1`
   - Query SFGA name + taxon tables
   - For each name:
     * If in taxon: call getBreadcrumbs() for classification
     * If not in taxon: record_id = "bare-name-{col__id}", taxonomic_status = "bare name"
   - Create NameStringIndex records
   - Bulk insert using pgx.CopyFrom
   - Progress logging
3. Handle empty taxon table (all names are bare)
4. Include rank, taxonomic_status, accepted_record_id

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/indices.go` (new)

**Success Criteria**:
- [ ] Integration test passes
- [ ] Classifications generated correctly
- [ ] Bare names handled
- [ ] Old data cleaned

**Dependencies**: T041, T042

---

## Phase 4.7: Vernacular Processing (Phases 3 & 4)

### T044: [P] Write Integration Test for Vernaculars

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
- `/Users/dimus/code/golang/gndb/internal/io/populate/vernaculars_integration_test.go` (new)

**Success Criteria**:
- [ ] Test fails (function not implemented)
- [ ] Covers vernacular strings and indices
- [ ] Tests uniqueness

**Dependencies**: T037

---

### T045: Implement Vernacular Processing

**Description**: Implement Phases 3 & 4 - Vernacular strings and indices

**Actions**:
1. Create `internal/io/populate/vernaculars.go`
2. Implement `processVernaculars(ctx context.Context, p *PopulatorImpl, sfgaDB *sql.DB, sourceID int) error`:
   - Phase 3: Read SFGA VernacularName table
     * Generate UUID v5 for vernacular strings
     * Batch insert with ON CONFLICT DO NOTHING
   - Phase 4: Create VernacularStringIndex records
     * Link to name_string_id
     * Include language, locality
     * Bulk insert using CopyFrom
   - Progress logging for both phases

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/vernaculars.go` (new)

**Success Criteria**:
- [ ] Integration test passes
- [ ] Vernacular strings unique
- [ ] Indices link correctly
- [ ] Progress logged

**Dependencies**: T044

---

## Phase 4.8: Data Source Metadata (Phase 5)

### T046: [P] Write Integration Test for Data Source Metadata

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
- `/Users/dimus/code/golang/gndb/internal/io/populate/metadata_integration_test.go` (new)

**Success Criteria**:
- [ ] Test fails (function not implemented)
- [ ] Tests create and update scenarios
- [ ] Verifies counts

**Dependencies**: T037

---

### T047: Implement Data Source Metadata

**Description**: Implement Phase 5 - Data source record creation

**Actions**:
1. Create `internal/io/populate/metadata.go`
2. Implement `updateDataSourceMetadata(ctx context.Context, p *PopulatorImpl, source DataSourceConfig, sfgaDB *sql.DB) error`:
   - Read SFGA metadata (title, description, home_url, etc.)
   - Merge with sources.yaml overrides
   - Query counts:
     * `SELECT COUNT(*) FROM name_string_indices WHERE data_source_id = $1`
     * `SELECT COUNT(*) FROM vernacular_string_indices WHERE data_source_id = $1`
   - INSERT or UPDATE data_sources table
   - Set updated_at = NOW()
3. Handle NULL version/release_date gracefully

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/metadata.go` (new)

**Success Criteria**:
- [ ] Integration test passes
- [ ] Metadata merged correctly
- [ ] Counts accurate
- [ ] Idempotent (can run multiple times)

**Dependencies**: T046

---

## Phase 4.9: Main Orchestration

### T048: Wire All Phases in Populator.Populate()

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
- [ ] All phases execute in sequence
- [ ] Error handling works
- [ ] Context cancellation respected
- [ ] Progress logged clearly

**Dependencies**: T036, T039, T041, T043, T045, T047

---

## Phase 4.10: End-to-End Testing

### T049: Create End-to-End Populate Integration Test

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
- [ ] E2E test passes
- [ ] All phases execute
- [ ] Data verified in database
- [ ] Idempotent

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
- [ ] Quickstart updated with real examples
- [ ] All commands verified
- [ ] Cache debugging documented

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
- [ ] Architecture documented clearly
- [ ] Under 150 lines total
- [ ] Key patterns highlighted

**Dependencies**: T049

---

## Summary

**Total Tasks**: 18 (T034-T051)
**Completed**: 4 (T034 ✅, T035 ✅, T036 ✅, T037 ✅)
**Parallel Tasks**: 10 (T034, T035, T037, T038, T040, T042, T044, T046, T050, T051)
**Critical Path**: T034→T036→T037→T039→T041→T043→T045→T047→T048→T049→T050

**Phase Breakdown**:
- Phase 4.1: Cache Setup (T034) - 1 task [P] ✅
- Phase 4.2: Sources Config (T035-T036) - 2 tasks, 1 [P] ✅
- Phase 4.3: SFGA Fetching (T037) - 1 task [P] ✅
- Phase 4.4: Name Strings (T038-T039) - 2 tasks, 1 [P]
- Phase 4.5: Hierarchy (T040-T041) - 2 tasks, 1 [P]
- Phase 4.6: Name Indices (T042-T043) - 2 tasks, 1 [P]
- Phase 4.7: Vernaculars (T044-T045) - 2 tasks, 1 [P]
- Phase 4.8: Metadata (T046-T047) - 2 tasks, 1 [P]
- Phase 4.9: Orchestration (T048) - 1 task
- Phase 4.10: Testing & Docs (T049-T051) - 3 tasks, 2 [P]

**Estimated Effort**: 13-17 hours focused work (4 tasks complete)

**Key Patterns**:
- TDD: Integration tests before implementation
- Sequential phases: Each phase builds on previous
- Parallel where possible: Different files, no dependencies
- Reference implementation: github.com/sfborg/to-gn patterns

**Next Steps**:
1. T038: Write Integration Test for Name Strings Import
2. T039: Implement Name Strings Processing
3. Continue through phases 4.5-4.9 (hierarchy, indices, vernaculars, metadata)
4. T048: Wire all phases in main Populate() orchestration
5. T049-T051: End-to-end testing and documentation

---

*Constitution v1.3.0 | TDD | Pure/Impure Separation | CLI-First*
