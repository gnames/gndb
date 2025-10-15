# Implementation Plan: Optimize Database Performance

**Branch**: `002-optimize` | **Date**: 2025-10-15 | **Spec**: [./spec.md](./spec.md)
**Input**: Feature specification from `/home/dimus/code/golang/gndb/specs/002-optimize/spec.md`

## Summary
This feature introduces a new `optimize` command to `gndb`. This command will perform a series of data processing tasks to optimize the database for use with `gnverifier`. The implementation will be based on the `rebuild` command from the `gnidump` project.

## Technical Context
**Language/Version**: Go 1.21
**Primary Dependencies**: `database/sql`, `pgx`
**Storage**: PostgreSQL
**Testing**: `go test`
**Target Platform**: Linux server, macOS CLI
**Project Type**: single Go project
**Performance Goals**: Faster name verification, vernacular name detection, and synonym resolution.
**Constraints**: Must be restartable from scratch.
**Scale/Scope**: The `gndb` database can be large, so the optimization process needs to be efficient.

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

## Project Structure

### Documentation (this feature)
```
specs/002-optimize/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
│   └── Optimizer.go
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
pkg/
└── lifecycle/             # Pure logic modules
    └── optimizer.go

internal/iooptimize/             # Impure implementations
└── optimizer.go

cmd/
└── gndb/
    └── optimize.go          # CLI entry point
```

**Structure Decision**: The feature will be implemented following the existing project structure, with pure logic in `pkg/lifecycle`, impure implementation in `internal/iooptimize`, and the CLI command in `cmd/gndb`.

## Phase 0: Outline & Research
Completed. See [research.md](./research.md) for details.

## Phase 1: Design & Contracts
Completed. See [data-model.md](./data-model.md), [quickstart.md](./quickstart.md), and [contracts/Optimizer.go](./contracts/Optimizer.go) for details.

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Each contract → contract test task [P]
- Each entity → model creation task [P]
- Each user story → integration test task
- Implementation tasks to make tests pass

**Ordering Strategy**:
- TDD order: Tests before implementation
- Dependency order: Pure modules → io implementations → CLI
- Mark [P] for parallel execution (independent files)

**Estimated Output**: 20-30 numbered, ordered tasks in tasks.md

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)
**Phase 4**: Implementation (execute tasks.md following constitutional principles)
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
No complexity deviations are anticipated.

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
- [X] Complexity deviations documented

---
*Based on Constitution v1.3.2 - See `.specify/memory/constitution.md`*