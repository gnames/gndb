
# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

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
[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context
**Language/Version**: [e.g., Go 1.21, Python 3.11, Swift 5.9, Rust 1.75 or NEEDS CLARIFICATION]  
**Primary Dependencies**: [e.g., database/sql, FastAPI, UIKit, LLVM or NEEDS CLARIFICATION]  
**Storage**: [if applicable, e.g., PostgreSQL, CoreData, files or N/A]  
**Testing**: [e.g., go test, pytest, XCTest, cargo test or NEEDS CLARIFICATION]  
**Target Platform**: [e.g., Linux server, macOS CLI, iOS 15+, WASM or NEEDS CLARIFICATION]
**Project Type**: [single/web/mobile - determines source structure]  
**Performance Goals**: [domain-specific, e.g., 1000 req/s, 10k lines/sec, 60 fps or NEEDS CLARIFICATION]  
**Constraints**: [domain-specific, e.g., <200ms p95, <100MB memory, offline-capable or NEEDS CLARIFICATION]  
**Scale/Scope**: [domain-specific, e.g., 10k users, 1M LOC, 50 screens or NEEDS CLARIFICATION]

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Modular Architecture**
- [ ] Each feature component is a separate module with single responsibility
- [ ] Modules communicate only through interfaces, no direct implementation coupling
- [ ] Module boundaries clearly defined and documented

**II. Pure/Impure Code Separation**
- [ ] Pure logic separated from I/O and state-changing operations
- [ ] All database, file system, network operations isolated in `io` modules
- [ ] Pure functions do not import or depend on `io` modules

**III. Test-Driven Development** *(NON-NEGOTIABLE)*
- [ ] Tests written first and verified to fail before implementation
- [ ] Red-Green-Refactor workflow documented in task ordering
- [ ] All features include passing tests before considered complete

**IV. CLI-First Interface**
- [ ] All functionality exposed via CLI commands using subcommands
- [ ] Database lifecycle phases separated: create, migrate, populate, restructure
- [ ] Subcommands are independently executable and composable
- [ ] Structured output to stdout, errors to stderr
- [ ] No GUI, web, or graphical dependencies introduced

**V. Open Source Readability**
- [ ] Public APIs documented with clear godoc comments
- [ ] Complex logic includes explanatory comments
- [ ] Names follow Go conventions and are self-documenting

**VI. Configuration Management**
- [ ] YAML configuration file support included (gndb.yaml)
- [ ] CLI flags override file-based configuration settings
- [ ] Precedence order enforced: flags > env vars > config file > defaults
- [ ] Configuration schema documented and validated at startup
- [ ] Fail-fast with clear errors for invalid configuration

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
pkg/                     # Pure interfaces and types (no I/O)
├── config/              # Configuration types and defaults
│   ├── config.go        # Config struct, validation, defaults
│   └── config_test.go   # Pure config logic tests
├── db/                  # Database operator interface
│   ├── operator.go      # Operator interface (Connect, Pool, TableExists, etc.)
│   └── operator_test.go # Contract test: verifies implementation compliance
├── lifecycle/           # Lifecycle component interfaces
│   ├── schema.go        # SchemaManager interface (Create, Migrate)
│   ├── populator.go     # Populator interface (Populate)
│   ├── optimizer.go     # Optimizer interface (Optimize)
│   ├── *_test.go        # Contract tests for each interface
├── schema/              # GORM models for database schema
│   └── models.go        # NameString, Canonical, DataSource, etc.
├── populate/            # Pure populate logic and types
│   ├── sources.go       # sources.yaml parsing and filtering
│   └── sources_test.go  # Unit tests for source filtering
├── logger/              # Logging configuration
└── templates/           # Embedded templates
    ├── config.yaml      # Default config template
    └── sources.yaml     # Default sources template

internal/                # Impure I/O implementations (io* prefix convention)
├── ioconfig/            # Config file I/O operations
│   ├── loader.go        # Load config from YAML/env/flags (uses viper)
│   ├── generate.go      # Generate default config files
│   └── *_test.go        # Integration tests
├── iodb/                # PostgreSQL database operations
│   ├── operator.go      # PgxOperator implements pkg/db.Operator (pgxpool)
│   └── operator_test.go # Integration tests (requires PostgreSQL)
├── ioschema/            # Schema management via GORM
│   └── manager.go       # Manager implements lifecycle.SchemaManager
├── iopopulate/          # Data population from SFGA sources
│   ├── populator.go     # PopulatorImpl implements lifecycle.Populator
│   ├── sfga.go          # SFGA download and opening (uses sflib)
│   ├── cache.go         # Cache management (~/.cache/gndb/sfga/)
│   ├── names.go         # Phase 1: Name strings processing
│   ├── hierarchy.go     # Phase 1.5: Taxonomy hierarchy
│   ├── indices.go       # Phase 2: Name indices processing
│   ├── vernaculars.go   # Phase 3-4: Vernacular names
│   ├── metadata.go      # Phase 5: Data source metadata
│   └── *_test.go        # Integration tests (require PostgreSQL + SFGA files)
├── iooptimize/          # Database optimization
│   └── optimizer.go     # OptimizerImpl implements lifecycle.Optimizer
└── iotesting/           # Shared test utilities
    └── config.go        # Test config helpers

cmd/gndb/                # CLI application (cobra)
├── main.go              # Entry point, sets up logger and root command
├── root.go              # Root command, loads config
├── create.go            # create command (schema creation)
├── migrate.go           # migrate command (schema migration)
├── populate.go          # populate command (SFGA import)
├── optimize.go          # optimize command (indexes, views)
└── *_test.go            # End-to-end integration tests

testdata/                # Test SFGA files
└── 1002.sqlite          # VASCAN test data (used by integration tests)
```

**Structure Decision**: Single Go project with strict pure/impure separation.

### Architecture Principles

**1. Pure/Impure Separation (Constitution Principle II)**
- **pkg/**: Pure interfaces and types, no I/O dependencies
  - Defines contracts via interfaces (`db.Operator`, `lifecycle.*`)
  - Contains GORM models (schema definition only)
  - Houses pure logic (config parsing, source filtering)
- **internal/**: I/O implementations with `io*` prefix
  - All packages prefixed with `io` for clarity (`ioconfig`, `iodb`, `iopopulate`)
  - Implements pkg/ interfaces with actual I/O operations
  - No circular dependencies back to pkg/

**2. Package Naming Convention**
- Package names match directory names (Go idiom)
- I/O packages use `io` prefix consistently: `ioconfig`, `iodb`, `iopopulate`, etc.
- No import aliases needed since package name = directory name
- Example: `import "github.com/gnames/gndb/internal/iodb"` → use as `iodb.PgxOperator()`

**3. Interface-Based Design (Constitution Principle I)**
```go
// pkg/db/operator.go - Interface definition
type Operator interface {
    Connect(ctx, *config.DatabaseConfig) error
    Pool() *pgxpool.Pool  // Exposes pool for CopyFrom, transactions
    TableExists(ctx, string) (bool, error)
    HasTables(ctx) (bool, error)
    DropAllTables(ctx) error
    Close() error
}

// internal/iodb/operator.go - Implementation
type PgxOperator struct { pool *pgxpool.Pool }
func (p *PgxOperator) Connect(...) { /* pgx implementation */ }
```

**4. Lifecycle Components**
- **SchemaManager**: Creates/migrates schema via GORM AutoMigrate
- **Populator**: Imports SFGA data through 5 phases (names → hierarchy → indices → vernaculars → metadata)
- **Optimizer**: Creates indexes and materialized views (idempotent, always rebuilds)

**5. Populate Workflow (5 Phases)**
```
Phase 0: Fetch SFGA from URL/local, cache at ~/.cache/gndb/sfga/
Phase 1: Process name_strings (scientific names) via gnparser + pgx CopyFrom
Phase 1.5: Build hierarchy (taxonomy tree)
Phase 2: Process name_string_indices (taxa, synonyms, bare names)
Phase 3-4: Process vernaculars (common names by language)
Phase 5: Update data_sources metadata (record counts, timestamps)
```

**6. Configuration Precedence (Constitution Principle VI)**
```
CLI flags > Environment vars > config.yaml > Defaults
```
- Config file: `~/.config/gndb/config.yaml` (or `--config` flag)
- Sources file: `~/.config/gndb/sources.yaml` (or `--sources-yaml` flag)
- Templates in `pkg/templates/` for generation

**7. Cache Strategy**
- SFGA files cached at `~/.cache/gndb/sfga/` (all platforms)
- Format: `{sourceID:04d}.sqlite` (e.g., `0001.sqlite` for Catalogue of Life)
- Avoids redundant downloads across populate runs

**8. Testing Strategy (Constitution Principle III)**
- **Contract tests**: `pkg/lifecycle/*_test.go` verify interface compliance
- **Unit tests**: Pure logic in `pkg/` packages
- **Integration tests**: `internal/io*/*_test.go` require PostgreSQL + SFGA data
- **E2E tests**: `cmd/gndb/*_test.go` test full CLI workflows
- Run with `go test -short` to skip integration tests

**9. CLI Design (Constitution Principle IV)**
```bash
gndb create   # DatabaseOperator → SchemaManager.Create()
gndb migrate  # DatabaseOperator → SchemaManager.Migrate()
gndb populate # DatabaseOperator → Populator.Populate() → 5 phases
gndb optimize # DatabaseOperator → Optimizer.Optimize()
```

**10. Performance Considerations**
- **pgx CopyFrom** for bulk inserts (100M+ records)
- **gnparser pool** for concurrent name parsing
- **Batch processing** with configurable batch sizes
- **Materialized views** for fast lookups (created by Optimizer)

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

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

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., Breaking modular arch] | [current need] | [why simpler approach insufficient] |
| [e.g., Mixing pure/impure] | [specific problem] | [why separation impractical] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [ ] Phase 0: Research complete (/plan command)
- [ ] Phase 1: Design complete (/plan command)
- [ ] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [ ] Initial Constitution Check: PASS
- [ ] Post-Design Constitution Check: PASS
- [ ] All NEEDS CLARIFICATION resolved
- [ ] Complexity deviations documented

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
