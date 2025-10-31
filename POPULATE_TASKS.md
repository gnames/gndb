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

## Phase 3: Data Import - Metadata & Names ✅ COMPLETE
- [x] Implement `internal/iopopulate/cache.go` (52 lines from spec-kit)
  - [x] Cache directory preparation
  - [x] Clear cache functionality
  - [x] Fixed ioconfig references to use pkg/config
- [x] Implement `internal/iopopulate/metadata.go` (277 lines from spec-kit)
  - [x] Import DataSource records from SFGA
  - [x] Handle metadata overrides from sources.yaml
  - [x] Read SFGA metadata (title, version, etc.)
  - [x] Query record counts
  - [x] Build and insert DataSource records
- [x] Implement `internal/iopopulate/names.go` (252 lines from spec-kit)
  - [x] NameString import from SFGA
  - [x] Canonical/CanonicalFull/CanonicalStem generation
  - [x] Parse names using gnparser
  - [x] Batch insert with pgx CopyFrom
  - [x] Deduplication logic
  - [x] User prompts (marked unused for now)
- [x] Wire up processSource() to call Phase 3 functions
  - [x] Prepare cache directory
  - [x] Fetch and open SFGA file
  - [x] Import metadata
  - [x] Import name-strings
- [x] Fixed PopulatorImpl → populator references
- [x] All tests passing, lint clean
- [x] Note: 19 80-column violations in Phase 3 files (from spec-kit)

## Phase 4: Additional Data Import ✅ COMPLETE
- [x] Implement `internal/iopopulate/vernaculars.go` (357 lines from spec-kit)
  - [x] VernacularString import from SFGA
  - [x] Language handling
  - [x] Batch insert
  - [x] Fixed PopulatorImpl → populator references
- [x] Implement `internal/iopopulate/hierarchy.go` (390 lines from spec-kit)
  - [x] Classification hierarchy building
  - [x] Concurrent parsing with gnparser
  - [x] hNode structure for taxonomy tree
  - [x] Fixed PopulatorImpl → populator references
- [x] Implement `internal/iopopulate/indices.go` (727 lines from spec-kit)
  - [x] NameStringIndex import (links names to sources)
  - [x] Taxonomic status, rank, classification
  - [x] Flat vs full classification support
  - [x] Fixed cfg.Import → cfg.Populate
  - [x] Safe *bool pointer dereferencing
- [x] Wire up processSource() to call Phase 4 functions
  - [x] Import vernaculars (optional, warns on failure)
  - [x] Build hierarchy (optional, warns on failure)
  - [x] Import name indices (required)
- [x] Fixed function signatures in populator.go
  - [x] buildHierarchy(ctx, sfgaDB, cfg.JobsNumber)
  - [x] processNameIndices(ctx, p, sfgaDB, &source, hierarchy, cfg)
- [x] All tests passing, lint clean

## Phase 5: CLI Command & Integration ✅ COMPLETE
- [x] Implement `cmd/populate.go`
  - [x] Flags: --source-ids, --release-version, --release-date, --flat-classification
  - [x] Both long and short forms for flags (-s, -r, -d, -f)
  - [x] User confirmation before population starts
  - [x] Validation for override flags (single source only)
  - [x] Empty database check (prompts to run create first)
  - [x] Progress display with gn.Info
  - [x] Integration with iopopulate.NewPopulator
- [x] Wire populate command in `cmd/root.go`
- [x] Run full test suite: `just test` (all tests passing)
- [x] Run linter: `just lint` (lint clean)
- [x] Command follows patterns from create.go and migrate.go
- [x] Configuration updates via Option functions
- [x] Proper error handling with gn.PrintErrorMessage

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

**Current Status:** Phase 5 Complete - Ready for Phase 6 (Final Verification)
**Started:** 2025-10-30
**Branch:** 4-populate
