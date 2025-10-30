# Populate Implementation Tasks

This is a temporary tracking file for implementing the populate command.
Delete this file when populate implementation is complete.

## Phase 1: Foundation (Data structures & Configuration) ✅ COMPLETE
- [x] Review `pkg/lifecycle/` interfaces for Populator
- [x] Implement `pkg/populate/sources.go` - SFGA source configuration
  - [x] Source struct with ID, title, URL, metadata (already existed)
  - [x] Sources validation (IDs < 1000 official, >= 1000 custom)
  - [x] YAML parsing for sources.yaml
  - [x] Tests for sources.go (1009 lines, 13 test functions, all passing)
- [x] Review existing schema models in `pkg/schema/`
- [x] Add populate error codes to `pkg/errcode/` (10 new codes)

## Phase 2: Core Populator & SFGA Handling ✅ COMPLETE
- [x] Implement `internal/iopopulate/populator.go`
  - [x] Private populator struct implementing lifecycle.Populator
  - [x] Constructor NewPopulator(db.Operator)
  - [x] Main Populate(ctx, *config.Config) method
  - [x] Source filtering logic (SourceIDs)
  - [x] Progress reporting
  - [x] Stub processSource() for Phase 3/4 implementation
- [x] Implement `internal/iopopulate/sfga.go` (437 lines from spec-kit)
  - [x] SFGA file discovery and validation (local and remote)
  - [x] File pattern matching (multiple ID formats)
  - [x] SQLite connection handling (stubbed for Phase 3)
  - [x] Version checking (stubbed for Phase 3)
  - [x] Added nolint comments for unused functions
- [x] Implement `internal/iopopulate/errors.go`
  - [x] 9 error functions following gn.Error pattern
  - [x] Tests for errors.go (9 test functions, all passing)
- [x] Added dependencies: gnlib, sflib, gnparser, sqlite
- [x] All tests passing, lint clean

## Phase 3: Data Import - Metadata & Names
- [ ] Implement `internal/iopopulate/metadata.go`
  - [ ] Import DataSource records from SFGA
  - [ ] Handle metadata overrides from sources.yaml
  - [ ] Batch insert optimization
  - [ ] Tests for metadata.go
- [ ] Implement `internal/iopopulate/cache.go`
  - [ ] Name parsing cache (gnparser integration)
  - [ ] Cache hit/miss tracking
  - [ ] Tests for cache.go
- [ ] Implement `internal/iopopulate/names.go` (LARGEST COMPONENT)
  - [ ] NameString import from SFGA
  - [ ] Canonical/CanonicalFull/CanonicalStem generation
  - [ ] Parse names using gnparser
  - [ ] Batch insert with pgx CopyFrom
  - [ ] Deduplication logic
  - [ ] Tests for names.go

## Phase 4: Additional Data Import
- [ ] Implement `internal/iopopulate/vernaculars.go`
  - [ ] VernacularString import from SFGA
  - [ ] Language handling
  - [ ] Batch insert
  - [ ] Tests for vernaculars.go
- [ ] Implement `internal/iopopulate/hierarchy.go`
  - [ ] Classification hierarchy import
  - [ ] Flat vs full classification support
  - [ ] Tests for hierarchy.go
- [ ] Implement `internal/iopopulate/indices.go`
  - [ ] NameStringIndex import (links names to sources)
  - [ ] Taxonomic status, rank, classification
  - [ ] Tests for indices.go

## Phase 5: CLI Command & Integration
- [ ] Implement `cmd/populate.go`
  - [ ] Flags: --source-ids, --release-version, --release-date, --flat-classification
  - [ ] Both long and short forms for flags
  - [ ] User confirmation if schema is empty
  - [ ] Progress display with gn.Info
  - [ ] Integration with iopopulate.NewPopulator
  - [ ] Tests for populate.go
- [ ] Wire populate command in `cmd/root.go`
- [ ] Run integration tests
- [ ] Run full test suite: `just test`
- [ ] Run linter: `just lint`
- [ ] Update documentation if needed

## Phase 6: Final Verification
- [ ] Test with real SFGA file (if available in testdata)
- [ ] Verify database population works end-to-end
- [ ] Check performance with batch operations
- [ ] Review code for 80-column compliance
- [ ] Final commit and close issue

## Notes & Dependencies

**Key Dependencies:**
- gnparser for name parsing (github.com/gnames/gnparser)
- sflib for SFGA file handling (github.com/gnames/sflib)
- pgx CopyFrom for bulk inserts

**Architecture Patterns:**
- Private implementation structs (lowercase)
- gn.Error pattern for all errors
- Pass full *config.Config to functions
- Use pgxpool for connection pooling
- Batch operations with configurable batch size

**Current Status:** Phase 1 - Foundation
**Started:** 2025-10-30
**Branch:** 4-populate
