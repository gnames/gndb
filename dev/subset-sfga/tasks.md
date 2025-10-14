# subset-sfga Implementation Tasks

**Goal**: Extract ~30k representative records from large SFGA files while preserving edge cases and hierarchy integrity.

**Approach**: Use sflib Reader/Writer pattern for correctness and simplicity.

---

## Phase 1: Fetch and Open Source SFGA

- [X] **T1.1**: Add sflib dependency import
  - Import `github.com/sfborg/sflib` 
  - Import `github.com/gnames/gndb/internal/ioconfig`
  - Add stub usage to prevent import removal

- [X] **T1.2**: Implement cache directory setup
  - Use `ioconfig.GetCacheDir()` from main project
  - Create `sfga/` subdirectory similar to gndb populate
  - Creates `~/.cache/gndb/sfga/` directory with 0755 permissions

- [X] **T1.3**: Fetch source SFGA
  - Uses `sflib.Fetch()` which transparently handles both URLs and local paths
  - sflib does all the heavy lifting (download, extract, cache)
  - We just pass source to sflib and it figures it out

- [X] **T1.4**: Query leaf nodes directly from SFGA SQLite
  - Open database with `sql.Open("sqlite3", sfgaPath)`
  - Query for leaf taxa: taxa that don't appear as parent_id
  - Gracefully handle missing tables (0-N results is fine)
  - Collect col__id (taxon IDs) into slice for ancestry traversal

---

## Phase 2: Edge Case Detection (Revised Approach)

- [X] **T2.1**: Query orphan names
  - Names in name_string table not in taxon table
  - Keep separate - no ancestry traversal needed
  - SQL: `SELECT DISTINCT ns.id FROM name_string ns WHERE ns.id NOT IN (SELECT DISTINCT name_string_id FROM taxon WHERE name_string_id IS NOT NULL)`
  - Gracefully handle 0-N results

- [X] **T2.2**: Skip special character detection
  - Special characters and punctuation handled by gnparser/sflib
  - Not our responsibility in this tool

- [X] **T2.3**: Query names with synonyms
  - Accepted names that have synonym records
  - SQL: `SELECT DISTINCT t.col__id FROM taxon t WHERE t.col__id IN (SELECT DISTINCT accepted_id FROM synonym WHERE accepted_id IS NOT NULL)`
  - Add to taxonMap for deduplication
  - Need ancestry traversal

- [X] **T2.4**: Query names with vernacular connections
  - Names that appear in vernacular_name_string_indices
  - Connection via taxon table (NOT orphan names)
  - SQL: `SELECT DISTINCT t.col__id FROM taxon t WHERE t.name_string_id IN (SELECT DISTINCT name_string_id FROM vernacular_name_string_indices WHERE name_string_id IS NOT NULL)`
  - Add to taxonMap for deduplication
  - Need ancestry traversal

- [X] **T2.5**: Use map for automatic deduplication
  - Changed from slice to `map[string]bool`
  - Much simpler - no manual deduplication needed
  - Map key = col__id, value = true

---

## Phase 3: Random Sampling (Skipped)

- **Note**: Not implementing random sampling for now. The queries above (leaf nodes, orphans, synonyms, vernaculars) provide sufficient diversity for testing.
- Can be added later if needed with: `SELECT id FROM name_string WHERE id NOT IN (...) ORDER BY RANDOM() LIMIT ?`

---

## Phase 4: Hierarchy Completion

- [X] **T4.1**: Collect all taxon IDs into map
  - Add leaf taxon IDs to `map[string]bool`
  - Add synonym parent IDs to map
  - Add vernacular name IDs to map
  - Map provides automatic deduplication

- [X] **T4.2**: Implement recursive ancestry traversal
  - Function `traverseAncestry()` processes all taxa in map
  - Function `addAncestors()` recursively follows parent_id chain
  - Query: `SELECT parent_id FROM taxon WHERE col__id = ? AND parent_id IS NOT NULL`
  - Stops at root (parent_id IS NULL) or already-seen nodes
  - Adds all ancestors to taxonMap

- [X] **T4.3**: Log hierarchy progress
  - Log initial taxon count (before traversal)
  - Log final taxon count (after traversal)
  - Shows how many ancestors were added

---

## Phase 5: Read Records into COLDP Structs

- [X] **T5.1**: Load metadata from source SFGA
  - Use `sflib.NewSfga()` and `SetDb()` to point to source
  - Call `Connect()` to establish database connection
  - Use `LoadMeta()` to get `*coldp.Meta`
  - Update metadata title and description for subset

- [X] **T5.2**: Read taxon records via channels
  - Use `LoadTaxa()` which streams `coldp.Taxon` to channel
  - Filter taxa where `taxon.ID` is in `taxonMap`
  - Collect into `[]coldp.Taxon` slice

- [X] **T5.3**: Read vernacular records via channels
  - Use `LoadVernaculars()` which streams `coldp.Vernacular` to channel
  - Filter vernaculars where `vern.TaxonID` is in `taxonMap`
  - Collect into `[]coldp.Vernacular` slice

---

## Phase 6: Write Output SFGA

- [X] **T6.1**: Create output SFGA archive
  - Use `sflib.NewSfga()` to create archive
  - Call `Create(outputDir)` with directory path (not file path!)
  - Call `SetDb(outputPath)` to set the SQLite file path
  - Defer `Close()`

- [X] **T6.2**: Write metadata
  - Use `InsertMeta(meta)` to write updated metadata

- [X] **T6.3**: Write taxon records
  - Use `InsertTaxa(selectedTaxa)` to bulk insert
  - Log count of taxa written

- [X] **T6.4**: Write vernacular records
  - Use `InsertVernaculars(selectedVernaculars)` to bulk insert
  - Log count of vernaculars written

---

## Phase 7: Summary and Validation

- [X] **T7.1**: Log extraction summary
  - Taxa count (with ancestry)
  - Vernaculars count
  - Orphan names count
  - Output path

- [ ] **T7.2**: Test with real SFGA file
  - Need valid SFGA source (not empty cache file)
  - Verify all phases execute successfully
  - Check output file is valid

- [ ] **T7.3**: Optional: Add file size validation
  - Log output file size
  - Verify file is readable with `sflib.NewSfga()`

---

## Testing

- [ ] **T8.1**: Test with URL source
  - Use small test source (e.g., Ruhoff 0206)
  - Verify download and caching works

- [ ] **T8.2**: Test with local file source
  - Use local SFGA file
  - Verify direct file access works

- [ ] **T8.3**: Test with gndb populate
  - Create test sources.yaml pointing to output
  - Run `gndb populate` with subset
  - Verify all tables populated correctly

- [ ] **T8.4**: Test edge cases are preserved
  - Query output for empty names
  - Verify special characters present
  - Check hierarchy depth

---

## Notes

- Keep it simple - single file until complexity warrants refactoring
- Reuse sflib as much as possible - don't reinvent the wheel
- Focus on correctness over performance (this is a dev tool)
- Document any SFGA schema assumptions in comments
