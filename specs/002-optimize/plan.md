# Implementation Plan: Optimize Database Performance

**Branch**: `002-optimize` | **Date**: 2025-10-18 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/home/dimus/code/golang/gndb/specs/002-optimize/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code or `AGENTS.md` for opencode).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 8. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary

Implement `gndb optimize` command to optimize a populated database for gnverifier compatibility by replicating the production-tested `gnidump rebuild` workflow. The optimization follows a 6-step process: (1) reparse all name_strings with latest gnparser algorithms and cache results, (2) normalize vernacular language codes, (3) remove orphan records, (4) extract and link words for fuzzy matching using cached parse results, (5) create verification materialized view with indexes, and (6) run VACUUM ANALYZE. This ensures the database is optimized exactly as expected by gnverifier's query patterns.

## Technical Context
**Language/Version**: Go 1.21+  
**Primary Dependencies**: pgx/v5 (database operations), gnparser (name parsing), gnlang (language normalization), gnlib/gnuuid (UUID v5 for cache keys)  
**Storage**: PostgreSQL (primary database), temporary key-value store at `~/.cache/gndb/optimize/` (ephemeral parse cache)  
**Testing**: go test with contract tests and integration tests  
**Target Platform**: Linux/macOS CLI (cross-platform)  
**Project Type**: single (Go CLI application)  
**Performance Goals**: Process 100M+ name records with configurable concurrency (Config.JobsNumber workers)  
**Constraints**: Must be idempotent (drop and recreate), reusable cache between steps, fail-fast on errors  
**Scale/Scope**: Optimize databases with 10M-100M+ scientific names from multiple data sources

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Modular Architecture**
- [x] Each feature component is a separate module with single responsibility
  - Pure interface: `pkg/lifecycle/optimizer.go`
  - Impure implementation: `internal/iooptimize/optimizer.go`
- [x] Modules communicate only through interfaces, no direct implementation coupling
  - CLI uses `lifecycle.Optimizer` interface
  - Implementation uses `db.Operator` interface for database access
- [x] Module boundaries clearly defined and documented
  - Optimization logic in `internal/iooptimize/`
  - 6-step workflow separated into distinct functions

**II. Pure/Impure Code Separation**
- [x] Pure logic separated from I/O and state-changing operations
  - Word extraction logic can be pure (normalized/modified forms)
  - Language code normalization uses gnlang library (pure)
- [x] All database, file system, network operations isolated in `io` modules
  - All database queries in `internal/iooptimize/`
  - Cache operations in `internal/iooptimize/cache.go`
- [x] Pure functions do not import or depend on `io` modules
  - No pure modules needed for this feature (optimization is inherently I/O-heavy)

**III. Test-Driven Development** *(NON-NEGOTIABLE)*
- [x] Tests written first and verified to fail before implementation
  - Contract test exists: `pkg/lifecycle/optimizer_test.go`
  - Integration tests will verify each of 6 steps
- [x] Red-Green-Refactor workflow documented in task ordering
  - Tasks will follow: write test → verify failure → implement → verify pass
- [x] All features include passing tests before considered complete
  - Full workflow integration test required

**IV. CLI-First Interface**
- [x] All functionality exposed via CLI commands using subcommands
  - New command: `gndb optimize`
- [x] Database lifecycle phases separated: create, migrate, populate, restructure
  - `optimize` is the final lifecycle phase after `populate`
- [x] Subcommands are independently executable and composable
  - `optimize` operates on existing populated database
- [x] Structured output to stdout, errors to stderr
  - Progress messages to STDOUT, technical errors to STDERR
- [x] No GUI, web, or graphical dependencies introduced
  - Pure CLI implementation

**V. Open Source Readability**
- [x] Public APIs documented with clear godoc comments
  - `pkg/lifecycle/optimizer.go` has godoc
  - Each function will have purpose documentation
- [x] Complex logic includes explanatory comments
  - 6-step workflow will be clearly commented
  - gnidump reference will be documented
- [x] Names follow Go conventions and are self-documenting
  - `Optimize()`, `reparseNames()`, `fixVernaculars()`, etc.

**VI. Configuration Management**
- [x] YAML configuration file support included (gndb.yaml)
  - Uses existing `pkg/config` infrastructure
- [x] CLI flags override file-based configuration settings
  - Follows established precedence: flags > env > config > defaults
- [x] Precedence order enforced: flags > env vars > config file > defaults
  - Handled by existing `internal/ioconfig` loader
- [x] Configuration schema documented and validated at startup
  - Config.JobsNumber for concurrency control
  - Config.Import.BatchSize for word batching
- [x] Fail-fast with clear errors for invalid configuration
  - Validation in config loader

**VII. Development principles.**
- [x] Follow KISS and Do Not Repeat Yourself principles.
  - Reuse gnidump proven algorithms instead of reinventing
  - Reuse existing config, logger, db operator infrastructure

**VIII. Contributor-First Minimalism (NON-NEGOTIABLE)**
- [x] Write the simplest code that solves the problem
  - Follow gnidump's straightforward approach
  - 6 sequential steps, each with clear purpose
- [x] Create abstractions only when they improve comprehension or testability
  - Each step as separate function for testing
  - Cache abstraction for parse result reuse
- [x] No "just in case" code
  - Only implement documented 6-step workflow

**IX. Dual-Channel Communication**
- [x] User-facing output (STDOUT) is separated from developer-facing output (STDERR).
  - Progress: "Reparsing 10M names..." to STDOUT
  - Errors: Stack traces to STDERR
- [x] Well-formatted error documentation is provided on STDOUT for users.
  - Clear error messages with actionable steps
- [x] Technical logs and stack traces are directed to STDERR.
  - Database errors, connection failures to STDERR

**X. User-Friendly Documentation**
- [x] Use terminal colors to enhance readability.
  - Step headers in green
  - Progress percentages in cyan
  - Warnings in yellow, errors in red
- [x] Headers and titles are in a distinct color.
  - "Step 1/6: Reparsing names..." in green
- [x] Warnings or dangerous operations are highlighted.
  - "Dropping verification view..." in yellow

## Project Structure

### Documentation (this feature)
```
specs/002-optimize/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (COMPLETE)
├── data-model.md        # Phase 1 output (COMPLETE)
├── quickstart.md        # Phase 1 output (COMPLETE)
├── contracts/           # Phase 1 output (COMPLETE)
│   └── Optimizer.go     # Interface contract
└── tasks.md             # Phase 2 output (/tasks command - TO BE UPDATED)
```

### Source Code (repository root)
```
pkg/
├── lifecycle/
│   ├── optimizer.go         # Optimizer interface (EXISTING)
│   └── optimizer_test.go    # Contract test (EXISTING)
├── config/                  # Configuration types (EXISTING)
├── db/                      # Database operator interface (EXISTING)
└── schema/                  # GORM models (EXISTING - words tables defined)

internal/iooptimize/
├── optimizer.go             # OptimizerImpl stub (EXISTING - needs implementation)
├── optimizer_test.go        # Integration tests (TO BE CREATED)
├── reparse.go              # Step 1: Reparse names (TO BE CREATED)
├── vernacular.go           # Step 2: Fix vernacular languages (TO BE CREATED)
├── orphans.go              # Step 3: Remove orphan records (TO BE CREATED)
├── words.go                # Step 4: Create words tables (TO BE CREATED)
├── views.go                # Step 5: Create verification view (TO BE CREATED)
├── vacuum.go               # Step 6: VACUUM ANALYZE (TO BE CREATED)
└── cache.go                # Parse result cache (kvSci) (TO BE CREATED)

cmd/gndb/
├── optimize.go             # CLI command (TO BE CREATED)
└── main.go                 # Root command (EXISTING - wire optimize)
```

**Structure Decision**: Single Go project following existing gndb architecture. Pure interface in `pkg/lifecycle/optimizer.go`, impure implementation in `internal/iooptimize/` with 6-step workflow separated into distinct files for clarity. CLI command in `cmd/gndb/optimize.go`. Reuses existing infrastructure: `pkg/config`, `pkg/db`, `pkg/logger`, `pkg/parserpool`.

## Phase 0: Outline & Research

**Status**: ✅ COMPLETE

Research findings documented in [research.md](research.md):

1. **Decision**: Follow `gnidump rebuild` logic from `${HOME}/code/golang/gnidump/`
   - **Rationale**: Production-tested workflow, proven gnverifier compatibility
   - **Alternatives considered**: New implementation rejected due to risk

2. **6-Step Workflow Analyzed**:
   - Step 1: Reparse names with gnparser, cache results in kvSci
   - Step 2: Fix vernacular languages with gnlang
   - Step 3: Remove orphan records (names, canonicals, stems, fulls)
   - Step 4: Create words tables using cached parse results
   - Step 5: Create verification materialized view with 3 indexes
   - Step 6: VACUUM ANALYZE (gndb enhancement, not in gnidump)

3. **Key Technical Decisions**:
   - Cache location: `~/.cache/gndb/optimize/` (ephemeral)
   - Concurrency: Config.JobsNumber (vs gnidump's hardcoded 50)
   - Batch size: Config.Import.BatchSize for word processing
   - UUID v5 as cache key (matching gnidump)

4. **Dependencies Validated**:
   - gnparser: Name parsing with latest algorithms
   - gnlang: Language code normalization
   - gnlib/gnuuid: UUID v5 for consistent cache keys
   - pgx/v5: Bulk operations via CopyFrom

All NEEDS CLARIFICATION items resolved.

## Phase 1: Design & Contracts

**Status**: ✅ COMPLETE

### Artifacts Generated

1. **Data Model** ([data-model.md](data-model.md)):
   - No new tables (uses existing schema)
   - Materialized view: `verification` with 3 indexes
   - Existing tables populated: `words`, `word_name_strings`
   - Operations: reparse, normalize, delete orphans, insert words

2. **Interface Contracts** ([contracts/Optimizer.go](contracts/Optimizer.go)):
   - `lifecycle.Optimizer` interface with `Optimize(ctx, cfg) error` method
   - Contract test in `pkg/lifecycle/optimizer_test.go` (EXISTING)

3. **Quickstart** ([quickstart.md](quickstart.md)):
   - Prerequisites: Populated database
   - Command: `gndb optimize`
   - User scenarios from spec validated

4. **Implementation Stub** (`internal/iooptimize/optimizer.go`):
   - OptimizerImpl struct with TODO for 6-step logic
   - Returns "not yet implemented" error (test will fail)

### Constitution Re-Check

All principles satisfied (see Constitution Check section above). No violations to track.

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:

The /tasks command will generate tasks following this structure:

1. **Test-First Tasks** (TDD Red Phase):
   - T001: Write integration test for Step 1 (reparse names) - verify failure
   - T002: Write integration test for Step 2 (vernacular fix) - verify failure
   - T003: Write integration test for Step 3 (remove orphans) - verify failure
   - T004: Write integration test for Step 4 (create words) - verify failure
   - T005: Write integration test for Step 5 (create view) - verify failure
   - T006: Write integration test for Step 6 (vacuum) - verify failure
   - T007: Write end-to-end integration test for full workflow - verify failure

2. **Implementation Tasks** (TDD Green Phase):
   - T008: Implement cache.go (kvSci store for parse results) [P]
   - T009: Implement reparse.go (Step 1: load, parse via parserpool, update, cache)
   - T010: Implement vernacular.go (Step 2: gnlang normalization)
   - T011: Implement orphans.go (Step 3: delete cascade in correct order)
   - T012: Implement words.go (Step 4: extract from kvSci cache, bulk insert)
   - T013: Implement views.go (Step 5: DROP + CREATE view, 3 indexes)
   - T014: Implement vacuum.go (Step 6: VACUUM ANALYZE via pgx)
   - T015: Wire optimizer.go Optimize() method to call Steps 1-6 sequentially

3. **CLI Tasks**:
   - T016: Create cmd/gndb/optimize.go command with flags
   - T017: Wire optimize command into cmd/gndb/main.go
   - T018: Add CLI integration test (populate → optimize → verify)

4. **Refactor Tasks** (TDD Refactor Phase):
   - T019: Add colored progress output to STDOUT (Constitution X)
   - T020: Add error documentation blocks to STDOUT (Constitution IX)
   - T021: Verify all tests pass with final implementation

**Ordering Strategy**:
- Tests before implementation (strict TDD)
- Cache first (T008) - needed by Steps 1 and 4
- Steps 1-6 in sequence (dependency order from gnidump)
- CLI after core logic (T016-T018)
- Polish last (T019-T021)

**Parallelizable Tasks**: Only T008 (cache) can be parallel with test writing.

**Estimated Output**: ~21 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking

No constitutional violations. All principles satisfied by design.

## Progress Tracking

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning approach documented (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented (none)

---
*Based on Constitution v1.5.0 - See `.specify/memory/constitution.md`*
