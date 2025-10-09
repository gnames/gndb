# Tasks: GNverifier Database Lifecycle Management

**Feature Branch**: `001-gnverifier-db-lifecycle`
**Status**: Architecture Refactored (T001-T015 complete)
**Next Phase**: Interface Integration & CLI Wiring

**Context**:
- T001-T015: Initial implementation complete (config, basic DatabaseOperator, schema models)
- Architecture refactored per Constitution v1.3.0 Principle VIII
- Interfaces created in pkg/, implementations preserved in internal/io/
- DatabaseOperator trimmed from 14 methods → 5 core methods
- SchemaManager and Optimizer extracted as separate components
- See PRESERVE.md for details on preserved code

---

## Phase 3.2: Interface Integration & Wiring

### T016: [P] Write Contract Test for database.Operator Interface ✅

### T017: Update PgxOperator to Implement pkg/database.Operator ✅

### T018: [P] Write Contract Test for lifecycle.SchemaManager ✅

### T019: Verify SchemaManager Implementation ✅

### T020: [P] Write Contract Test for lifecycle.Optimizer ✅

### T021: Verify Optimizer Implementation ✅


## Phase 3.3: CLI Command Updates

### T022: Update create Command to Use New Interfaces ✅

### T023: Update migrate Command to Use SchemaManager ✅

### T024: Rename restructure → optimize, Use Optimizer Interface ✅

## Phase 3.4: Populator Stub

### T025: [P] Write Populator Contract Test ✅

### T026: Implement Populator Stub ✅

### T027: Update populate Command to Use Populator Interface ✅

## Phase 3.5: Integration Testing

### T028: Integration Test - Create Schema End-to-End ✅

### T029: Integration Test - Migrate Schema ✅

## Phase 3.6: Cleanup & Documentation

### T030: Remove Obsolete pkg/ Directories ✅

### T031: Update CLAUDE.md ✅

### T032: Verify All Tests Pass ✅

## Phase 3.7: Future Implementation Plans

### T033: Document Populate Implementation Plan ✅


## Summary

**Total Tasks**: 18 (T016-T033)
**Status**: Phase 3 Complete ✅
**Parallel Tasks**: 6 (T016, T018, T020, T025 - contract tests)
**Critical Path**: T016→T017→T022→T028→T030→T031→T032→T033

**Completed Work**:
- Interface architecture refactored (DatabaseOperator, SchemaManager, Optimizer, Populator)
- Contract tests for all interfaces
- CLI commands wired to new interfaces
- Integration tests for create and migrate workflows
- Comprehensive populate implementation plan documented
- Config/cache infrastructure unified across platforms

**Preserved from T001-T015**:
- Config package (100%)
- DatabaseOperator connection logic (trimmed to 5 methods)
- GORM models (wrapped in SchemaManager)
- SetCollation (moved to SchemaManager)
- Optimization methods (moved to Optimizer)

**Next Major Phase**: Implement Populator logic following documented plan
- Optimizer implementation plan postponed until after Populator (requirements may evolve)

---

*Constitution v1.3.0 | TDD | Minimalist | Preserves working code*
