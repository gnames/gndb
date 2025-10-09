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

### T033: Document Populate Implementation Plan

**Description**: Create plan for Populator.Populate() implementation

**Actions**:
1. Review `pkg/populate/sources.go`
2. Document in comments:
   - Read sources.yaml
   - Open SFGA with sflib
   - Transform to models
   - Bulk insert with CopyFrom
   - Progress logging

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/populator.go`

**Success Criteria**:
- [ ] Plan documented in comments
- [ ] Clear next steps

**Dependencies**: T027

---

### T034: Document Optimize Implementation Plan

**Description**: Create plan for Optimizer.Optimize() implementation

**Actions**:
1. Document in comments:
   - Drop existing indexes/views
   - Create indexes
   - Create materialized views
   - Use helper methods (VacuumAnalyze, etc.)
   - Progress logging

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/optimize/optimizer.go`

**Success Criteria**:
- [ ] Plan documented
- [ ] Helper methods referenced

**Dependencies**: T024

---

## Summary

**Total Tasks**: 19 (T016-T034)
**Parallel Tasks**: 6 (T016, T018, T020, T025 - can run contract tests together)
**Critical Path**: T016→T017→T022→T028→T030→T031→T032

**Preserved from T001-T015**:
- Config package (100%)
- DatabaseOperator connection logic (trimmed to 5 methods)
- GORM models (wrapped in SchemaManager)
- SetCollation (moved to SchemaManager)
- Optimization methods (moved to Optimizer)

**Estimated Effort**: 9-12 hours focused work

**Next Major Phase** (after T034): Implement Populator and Optimizer logic

---

*Constitution v1.3.0 | TDD | Minimalist | Preserves working code*
