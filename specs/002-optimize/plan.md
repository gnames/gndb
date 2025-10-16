
# Implementation Plan: Optimize Database Performance

**Branch**: `002-optimize` | **Date**: 2025-10-16 | **Spec**: [spec.md](./spec.md)
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

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Implement the `gndb optimize` command following the proven gnidump rebuild workflow: reparse scientific names with latest gnparser algorithms, normalize vernacular languages, remove orphan records, populate word decomposition tables, create the verification materialized view with indexes, and run VACUUM ANALYZE. The command uses configurable concurrency (Config.JobsNumber) and provides user-friendly colored progress output (Constitution Principle X) to support diverse hardware capabilities.

**Critical Usage Pattern**: Optimize is a time-consuming final step run ONCE after ALL data sources are imported. Typical workflow: `gndb populate` (multiple times for different sources) → `gndb optimize` (once before production deployment). User should not run optimize between each populate operation.

## Technical Context
**Language/Version**: Go 1.25
**Primary Dependencies**: pgx/v5 (SQL execution), gnparser (name parsing), gnlang (language normalization), fatih/color (terminal colors), slog (logging)
**Storage**: PostgreSQL (primary database) with materialized view, word tables
**Testing**: go test (unit tests for pure logic, integration tests with -short flag support)
**Target Platform**: Linux/macOS/Windows CLI
**Project Type**: Single Go CLI application
**Performance Goals**: Optimize 100M+ name records for fast gnverifier queries (<50ms p95 verification lookups)
**Constraints**: Idempotent (safe to re-run), must verify database is populated, uses Config.JobsNumber for worker concurrency (adapts to hardware), dual-channel output (user to STDOUT, technical to STDERR)
**Scale/Scope**: 6-step optimization workflow (reparse, fix languages, remove orphans, create words, create view, VACUUM ANALYZE) processing all name_strings and vernacular_string_indices records
**Typical Workflow**: Multiple `gndb populate --source-id=N` operations (e.g., 12 different sources) → Single `gndb optimize` (before production) → Database ready for gnverifier

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Modular Architecture**
- [X] Each feature component is a separate module with single responsibility
- [X] Modules communicate only through interfaces, no direct implementation coupling
- [X] Module boundaries clearly defined and documented

**II. Pure/Impure Code Separation**
- [X] Pure logic separated from I/O and state-changing operations
- [X] All database, file system, network operations isolated in `io` modules
- [X] Pure functions do not import or depend on `io` modules

**III. Test-Driven Development** *(NON-NEGOTIABLE)*
- [X] Tests written first and verified to fail before implementation
- [X] Red-Green-Refactor workflow documented in task ordering
- [X] All features include passing tests before considered complete

**IV. CLI-First Interface**
- [X] All functionality exposed via CLI commands using subcommands
- [X] Database lifecycle phases separated: create, migrate, populate, restructure
- [X] Subcommands are independently executable and composable
- [X] Structured output to stdout, errors to stderr
- [X] No GUI, web, or graphical dependencies introduced

**V. Open Source Readability**
- [X] Public APIs documented with clear godoc comments
- [X] Complex logic includes explanatory comments
- [X] Names follow Go conventions and are self-documenting

**VI. Configuration Management**
- [X] YAML configuration file support included (gndb.yaml)
- [X] CLI flags override file-based configuration settings
- [X] Precedence order enforced: flags > env vars > config file > defaults
- [X] Configuration schema documented and validated at startup
- [X] Fail-fast with clear errors for invalid configuration

**VII. Development principles.**
- [X] Follow KISS and Do Not Repeat Yourself principles

**VIII. Contributor-First Minimalism (NON-NEGOTIABLE)**
- [X] Write the simplest code that solves the problem
- [X] Create abstractions only when they improve comprehension or testability
- [X] No "just in case" code

**IX. Dual-Channel Communication**
- [X] User-facing output (STDOUT) is separated from developer-facing output (STDERR)
- [X] Well-formatted error documentation is provided on STDOUT for users
- [X] Technical logs and stack traces are directed to STDERR

**X. User-Friendly Documentation**
- [X] Use terminal colors to enhance readability
- [X] Headers and titles are in a distinct color
- [X] Warnings or dangerous operations are highlighted (time-consuming operation notice)

## Project Structure

### Documentation (this feature)
```
specs/002-optimize/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (COMPLETE - gnidump analysis)
├── data-model.md        # Phase 1 output (COMPLETE - verification view SQL)
├── quickstart.md        # Phase 1 output (pending)
├── contracts/           # Phase 1 output (pending)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
pkg/
├── lifecycle/
│   ├── optimizer.go         # Optimizer interface (EXISTING)
│   └── optimizer_test.go    # Contract test (EXISTING)
└── config/
    └── config.go            # JobsNumber field (EXISTING)

internal/iooptimize/         # Single package, organized by files
├── optimizer.go             # OptimizerImpl + Optimize() orchestration (EXISTING stub)
├── reparse.go               # Step 1: Reparse names with gnparser (NEW)
├── vernacular.go            # Step 2: Fix vernacular languages (NEW)
├── orphans.go               # Step 3: Remove orphan records (NEW)
├── words.go                 # Step 4: Create word tables (NEW)
├── verification.go          # Step 5: Create materialized view (NEW)
├── vacuum.go                # Step 6: VACUUM ANALYZE (NEW)
├── progress.go              # Colored progress output helpers (NEW)
└── optimizer_test.go        # Integration tests (NEW)

cmd/gndb/
├── optimize.go              # CLI command (NEW)
└── root.go                  # Add optimize subcommand (MODIFY)
```

**Structure Decision**: Single Go project following existing gndb architecture. All implementation in `internal/iooptimize/` package, organized across multiple files by workflow step for readability. No separate packages per step - follows Contributor-First Minimalism principle (Constitution VIII). CLI command follows existing cobra pattern.

## Phase 0: Outline & Research

**Status**: COMPLETE ✓

Research was conducted on gnidump rebuild implementation at `${HOME}/code/golang/gnidump/`. All findings documented in `research.md`:

- **Decision**: Follow gnidump rebuild 6-step workflow exactly
- **Rationale**: Production-tested, ensures gnverifier compatibility
- **Key Adaptations**: Use Config.JobsNumber instead of hardcoded 50 workers, add user-friendly colored output, add VACUUM ANALYZE step
- **Reference Implementation**: `internal/io/buildio/buildio.go` and related files

No NEEDS CLARIFICATION items remained - all technical details resolved through gnidump source code analysis.

**Output**: research.md (COMPLETE)

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

**Status**: Executing now

1. **Extract entities from feature spec** → `data-model.md`: ✓ COMPLETE
   - Verification materialized view with SQL
   - Word tables (words, word_name_strings)
   - 6-step optimization workflow

2. **Define interface contracts**: Optimizer interface already exists
   - pkg/lifecycle/optimizer.go defines Optimize(ctx, cfg) contract
   - No additional interfaces needed - reuses existing db.Operator

3. **Contract tests**: ✓ Exist
   - pkg/lifecycle/optimizer_test.go verifies interface compliance

4. **Extract test scenarios** from user stories → quickstart.md: Executing next

5. **Update agent file**: Executing next via script

**Output**: data-model.md ✓, contracts/ (N/A - interface exists), quickstart.md (executing), CLAUDE.md (executing)

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from research.md (6-step workflow) and data-model.md (verification view SQL)
- No new schema models needed - all tables exist
- Focus on implementing internal/iooptimize/ functions and CLI command

**Ordering Strategy** (TDD):
1. Create cache directory helpers (test + impl)
2. Step 1: Reparse names - test → impl [uses gnparser, kvSci cache]
3. Step 2: Fix vernacular - test → impl [uses gnlang]
4. Step 3: Remove orphans - test → impl [SQL DELETE]
5. Step 4: Create words - test → impl [uses kvSci cache]
6. Step 5: Create verification view - test → impl [SQL CREATE MATERIALIZED VIEW]
7. Step 6: VACUUM ANALYZE - test → impl [SQL VACUUM]
8. Progress output helpers - test → impl [fatih/color]
9. Main Optimize() orchestration - test → impl
10. CLI command - test → impl [cobra]
11. Integration test - full workflow validation

**Estimated Output**: ~15-20 numbered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

No constitutional violations. All principles followed:
- Modular architecture via Optimizer interface
- Pure/impure separation (pkg/ vs internal/iooptimize/)
- TDD workflow documented
- CLI-first with `gndb optimize` subcommand
- Configurable via existing Config struct
- Reuses proven gnidump logic - no overengineering

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [X] Phase 0: Research complete (/plan command)
- [X] Phase 1: Design complete (/plan command)
- [X] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [X] Initial Constitution Check: PASS
- [X] Post-Design Constitution Check: PASS
- [X] All NEEDS CLARIFICATION resolved
- [X] Complexity deviations documented (N/A - no violations)

---
*Based on Constitution v1.5.0 - See `.specify/memory/constitution.md`*
