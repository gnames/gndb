
# Implementation Plan: GNverifier Database Lifecycle Management

**Branch**: `001-gnverifier-db-lifecycle` | **Date**: 2025-10-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/Users/dimus/code/golang/gndb/specs/001-gnverifier-db-lifecycle/spec.md`

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
**Primary Requirement**: Enable local setup of GNverifier database lifecycle, from empty database through schema creation, migration, data population, and performance optimization for scientific name verification at 1000+ names/sec.

**Technical Approach** (from research.md):
- **Go models approach**: Use Go structs with GORM tags (matching gnidump's model.go) to define schema as single source of truth
- **Schema creation**: GORM AutoMigrate for DDL generation (proven from gnidump)
- **Connection pooling**: pgxpool.Pool for concurrent goroutine access during data population
- **Queries**: Raw SQL with pgx v5 (GORM is slow/inflexible for queries)
- **Migration**: Atlas framework with versioned migrations and integrity tracking
- **Import**: Stream-based SFGA ingestion via sflib with batch inserts using PostgreSQL COPY protocol, concurrent workers using pgxpool
- **Optimization**: Hybrid indexing (B-tree + GiST trigram), materialized views for denormalization, statistics tuning

## Technical Context
**Language/Version**: Go 1.21+  
**Primary Dependencies**: 
- pgx v5 with pgxpool (PostgreSQL driver - native protocol, 2-3x faster than database/sql, connection pooling for concurrent goroutines)
- GORM v2 (ORM - ONLY for schema creation/migration, NOT for queries)
- cobra (CLI framework with subcommand support)
- viper (configuration management: YAML + flags + env)
- atlasexec (Atlas Go SDK for migrations)
- sflib (SFGA format import library)
- gnparser (name parsing for GNames ecosystem)
- testify/assert (testing framework)

**Storage**: PostgreSQL 14+ (requires transactional DDL, COPY protocol, pg_trgm extension)

**Testing**: go test with testify assertions, integration tests for I/O modules, contract tests for interfaces

**Target Platform**: Linux/macOS CLI (cross-platform, no GUI dependencies)

**Project Type**: Single Go project (pure logic in pkg/, impure I/O in internal/io/, CLI in cmd/)

**Performance Goals**: 
- 1000+ names/sec reconciliation throughput (primary use case)
- <100ms p95 fuzzy match latency
- 10K records/sec import speed
- <2 hours index build time for 100M records

**Constraints**: 
- Read-only database post-setup (simplifies consistency model)
- Transactional operations where possible; phase-level restart for recovery
- 64GB RAM target for main service (local instances may use less)
- <500GB database size for 100M name-strings

**Scale/Scope**: 
- 100M scientific name-strings with 200M occurrences (main service target)
- 10M vernacular name-strings with 20M occurrences
- Support for smaller local datasets with reduced hardware requirements

**User-Provided Context**: 
- Use Go models with GORM tags (matching gnidump's model.go) for schema creation
- GORM used ONLY for database creation via AutoMigrate (single source of truth)
- GORM NOT used for queries (too slow/inflexible) - use raw SQL with pgx instead
- Schema changes only require updating Go models, not maintaining separate SQL DDL
- Proven approach from gnidump (will be archived when gndb is functional)

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Modular Architecture**
- [x] Each feature component is a separate module with single responsibility
  - pkg/schema: Go model definitions and DDL generation
  - pkg/migrate: Migration orchestration logic
  - pkg/populate: SFGA import coordination
  - pkg/restructure: Optimization logic (indexes, materialized views)
  - internal/io/database: PostgreSQL operations
  - internal/io/sfga: SFGA file I/O via sflib
  - cmd/gndb: CLI subcommands
- [x] Modules communicate only through interfaces, no direct implementation coupling
  - DatabaseOperator interface defines DB ops, implemented by internal/io/database
  - SFGAReader interface defines data source access, implemented by internal/io/sfga
- [x] Module boundaries clearly defined and documented
  - Architecture documented in constitution and enforced by import rules

**II. Pure/Impure Code Separation**
- [x] Pure logic separated from I/O and state-changing operations
  - pkg/ contains pure business logic (model definitions, validation, coordination)
  - internal/io/ contains all database and file system operations
- [x] All database, file system, network operations isolated in `io` modules
  - PostgreSQL queries/transactions in internal/io/database
  - SFGA file reading in internal/io/sfga
- [x] Pure functions do not import or depend on `io` modules
  - pkg/ modules define interfaces; internal/io/ implements them

**III. Test-Driven Development** *(NON-NEGOTIABLE)*
- [x] Tests written first and verified to fail before implementation
  - Contract tests for all interfaces created in Phase 1 before implementation
  - Integration tests for each CLI subcommand written before handlers
- [x] Red-Green-Refactor workflow documented in task ordering
  - tasks.md will follow TDD ordering: tests → implementation → refactor
- [x] All features include passing tests before considered complete
  - No task complete until tests pass; enforced in task definitions

**IV. CLI-First Interface**
- [x] All functionality exposed via CLI commands using subcommands
  - Four lifecycle subcommands: create, migrate, populate, restructure
- [x] Database lifecycle phases separated: create, migrate, populate, restructure
  - Each phase is independent subcommand with dedicated handler
- [x] Subcommands are independently executable and composable
  - Users can run phases in order or restart individual phases
- [x] Structured output to stdout, errors to stderr
  - JSON output for machine consumption, human-readable for interactive use
- [x] No GUI, web, or graphical dependencies introduced
  - Pure CLI tool using cobra framework

**V. Open Source Readability**
- [x] Public APIs documented with clear godoc comments
  - All exported types, functions, and interfaces will have godoc
- [x] Complex logic includes explanatory comments
  - Schema generation, batch sizing, optimization strategies documented inline
- [x] Names follow Go conventions and are self-documenting
  - Idiomatic Go naming enforced (DatabaseOperator, not IDatabase)

**VI. Configuration Management**
- [x] YAML configuration file support included (gndb.yaml)
  - viper loads gndb.yaml from current directory or ~/.config/gndb/
- [x] CLI flags override file-based configuration settings
  - cobra flags take precedence over viper config file values
- [x] Precedence order enforced: flags > env vars > config file > defaults
  - viper handles precedence automatically
- [x] Configuration schema documented and validated at startup
  - Config struct with validation tags; fail on missing required fields
- [x] Fail-fast with clear errors for invalid configuration
  - Startup validation before any operations begin

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
pkg/
├── config/                      # Configuration types and validation (pure)
│   ├── config.go                # Config struct, defaults, validation
│   └── config_test.go           # Unit tests for validation logic
├── schema/                      # Database schema models (pure)
│   ├── models.go                # Go structs with GORM tags (matches gnidump)
│   ├── gorm.go                  # GORM AutoMigrate wrapper
│   ├── interfaces.go            # DatabaseOperator interface
│   └── schema_test.go           # Tests for GORM migration
├── migrate/                     # Migration orchestration (pure)
│   ├── interfaces.go            # MigrationRunner interface
│   ├── orchestrator.go          # Migration coordination logic
│   └── orchestrator_test.go     # Unit tests
├── populate/                    # Data population coordination (pure)
│   ├── interfaces.go            # SFGAReader, Importer interfaces
│   ├── coordinator.go           # Import orchestration, batching logic
│   ├── validator.go             # SFGA version compatibility checks
│   └── populate_test.go         # Unit tests
└── restructure/                 # Optimization coordination (pure)
    ├── interfaces.go            # Optimizer interface
    ├── indexing.go              # Index creation logic
    ├── materialized_views.go    # Materialized view definitions
    └── restructure_test.go      # Unit tests

internal/io/                     # Impure implementations
├── config/                      # Configuration file/flag loading
│   ├── loader.go                # viper integration, file reading
│   └── loader_test.go           # Integration tests
├── database/                    # PostgreSQL operations
│   ├── operator.go              # Implements schema.DatabaseOperator
│   ├── connection.go            # pgx connection pooling
│   ├── migrations.go            # Atlas SDK integration
│   └── database_test.go         # Integration tests (requires testcontainers)
└── sfga/                        # SFGA file I/O
    ├── reader.go                # Implements populate.SFGAReader
    ├── streamer.go              # Channel-based streaming via sflib
    └── sfga_test.go             # Integration tests

cmd/
└── gndb/
    ├── main.go                  # Root command, flag setup
    ├── create.go                # `gndb create` subcommand
    ├── migrate.go               # `gndb migrate` subcommand
    ├── populate.go              # `gndb populate` subcommand
    └── restructure.go           # `gndb restructure` subcommand

migrations/                      # Atlas migration files
├── atlas.sum                    # Merkle tree integrity tracking
└── {timestamp}_{name}.sql       # Versioned SQL migrations

testdata/                        # Test fixtures
├── sample.sfga                  # Small SFGA file for testing
└── expected_schema.sql          # Reference DDL for validation
```

**Structure Decision**: Single Go project following GNdb constitution:
- **pkg/**: Pure business logic modules defining interfaces and orchestration
- **internal/io/**: Impure implementations of pkg/ interfaces for database and file I/O
- **cmd/gndb/**: CLI subcommands (create, migrate, populate, restructure) using cobra
- **Go models approach**: pkg/schema/models.go defines database tables as Go structs with GORM tags (matching gnidump's model.go), enabling schema creation via AutoMigrate while queries use raw SQL with pgx for performance

## Phase 0: Outline & Research
✅ **Status**: COMPLETE (research.md already exists)

**Completed Research Topics**:
1. PostgreSQL schema design for 100M+ names → Hybrid indexing strategy (B-tree + GiST trigram)
2. Atlas migration framework → Versioned migrations with integrity tracking
3. SFGA format and sflib library → Stream-based import with batch inserts
4. SFGA version compatibility → Enforce same version for initial ingest
5. SFGA to PostgreSQL mapping → Direct table mapping with denormalization
6. Performance optimizations → Three-phase restructure (indexes, materialized views, statistics)
7. gnidump reference patterns → Pure/impure separation, interface-driven design
8. Technology stack decisions → Go 1.21+, pgx, cobra, viper, Atlas, sflib
9. Go models approach → Inspired by gnidump's model.go for schema creation and SQL mapping

**Output**: [research.md](./research.md) with all technical unknowns resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Define interface contracts** from functional requirements:
   - For each module → interface definition
   - For each user action → CLI command signature
   - Output interface definitions to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per interface
   - Assert interface compliance
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh claude`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

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

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

**No violations detected**. All constitutional principles are satisfied by the proposed architecture.


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented (none - all principles satisfied)

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
