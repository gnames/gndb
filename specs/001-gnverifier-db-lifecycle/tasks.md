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

---

### T023: Update migrate Command to Use SchemaManager ✅

---

### T024: Rename restructure → optimize, Use Optimizer Interface ✅

**Description**: Rename command and use lifecycle.Optimizer interface

**Actions**:
1. Rename `cmd/gndb/restructure.go` → `cmd/gndb/optimize.go`
2. Change command name "restructure" → "optimize"
3. Use Optimizer interface
4. Update root.go

**File Paths**:
- `/Users/dimus/code/golang/gndb/cmd/gndb/optimize.go`
- `/Users/dimus/code/golang/gndb/cmd/gndb/root.go`

**Success Criteria**:
- [X] Command renamed
- [X] Uses Optimizer interface
- [X] `gndb optimize` available (returns "not implemented")

**Dependencies**: T021

---

## Phase 3.4: Populator Stub

### T025: [P] Write Populator Contract Test

**Description**: Test for lifecycle.Populator interface compliance

**Actions**:
1. Create `pkg/lifecycle/populator_test.go`
2. Write contract test (will fail until implementation exists)

**File Paths**:
- `/Users/dimus/code/golang/gndb/pkg/lifecycle/populator_test.go`

**Success Criteria**:
- [ ] Test exists and fails

**Parallel**: [P]

---

### T026: Implement Populator Stub

**Description**: Create basic Populator implementation structure

**Actions**:
1. Create `internal/io/populate/populator.go` with stub that returns "not implemented"
2. Run contract test

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/populate/populator.go`

**Success Criteria**:
- [ ] Implements interface
- [ ] Contract test passes
- [ ] Returns "not implemented" error

**Dependencies**: T025

---

### T027: Update populate Command to Use Populator Interface

**Description**: Refactor populate command to use lifecycle.Populator

**Actions**:
1. Update imports
2. Use Populator interface
3. Test: `gndb populate` (returns "not implemented")

**File Paths**:
- `/Users/dimus/code/golang/gndb/cmd/gndb/populate.go`

**Success Criteria**:
- [ ] Uses interface
- [ ] Returns clear message

**Dependencies**: T026

---

## Phase 3.5: Integration Testing

### T028: Integration Test - Create Schema End-to-End

**Description**: Test complete create workflow with new interfaces

**Actions**:
1. Create integration test
2. Test: connect → create → verify tables → verify collation
3. Use testcontainers or test database

**File Paths**:
- `/Users/dimus/code/golang/gndb/cmd/gndb/create_integration_test.go`

**Success Criteria**:
- [ ] Integration test passes
- [ ] Schema created correctly

**Dependencies**: T022

---

### T029: Integration Test - Migrate Schema

**Description**: Test migrate command end-to-end

**Actions**:
1. Create integration test
2. Test: create → migrate → verify

**File Paths**:
- `/Users/dimus/code/golang/gndb/cmd/gndb/migrate_integration_test.go`

**Success Criteria**:
- [ ] Integration test passes

**Dependencies**: T023

---

## Phase 3.6: Cleanup & Documentation

### T030: Remove Obsolete pkg/ Directories

**Description**: Clean up old package structure

**Actions**:
1. Check `pkg/migrate/`, `pkg/restructure/` - if empty/obsolete, remove
2. Verify no broken imports

**File Paths**:
- `/Users/dimus/code/golang/gndb/pkg/migrate/`
- `/Users/dimus/code/golang/gndb/pkg/restructure/`

**Success Criteria**:
- [ ] Obsolete dirs removed
- [ ] Build succeeds

**Dependencies**: T024, T027

---

### T031: Update CLAUDE.md

**Description**: Document refactored architecture

**Actions**:
1. Run `.specify/scripts/bash/update-agent-context.sh claude`
2. Verify architecture documented
3. Keep under 150 lines

**File Paths**:
- `/Users/dimus/code/golang/gndb/CLAUDE.md`

**Success Criteria**:
- [ ] CLAUDE.md updated
- [ ] Architecture accurate
- [ ] Under 150 lines

**Dependencies**: T030

---

### T032: Verify All Tests Pass

**Description**: Run complete test suite

**Actions**:
1. Run `go test ./...`
2. Fix any failures
3. Verify coverage >70% for core packages

**Success Criteria**:
- [ ] All tests pass

**Dependencies**: T031

---

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
