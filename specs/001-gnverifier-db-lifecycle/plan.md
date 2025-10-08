
# Implementation Plan: GNverifier Database Lifecycle Management

**Branch**: `001-gnverifier-db-lifecycle` | **Date**: 2025-10-08 | **Spec**: [spec.md](./spec.md)
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
Enable users to manage the complete GNverifier database lifecycle locally: create schema, migrate, populate with custom data sources, and optimize for fast name verification. The DatabaseOperator interface provides basic database managerial commands and exposes pgxpool connections; high-level components (SchemaManager, Populator, Optimizer) receive these connections and implement specialized SQL operations internally.

## Technical Context
**Language/Version**: Go 1.25
**Primary Dependencies**: pgx/v5 (pgxpool for connection pooling), GORM (AutoMigrate for schema management), cobra (CLI), viper (config), sflib (SFGA data import)
**Storage**: PostgreSQL (primary), SQLite (SFGA format data sources)
**Testing**: go test (unit tests for pure logic, integration tests for io modules)
**Target Platform**: Linux server, macOS CLI
**Project Type**: single (Go CLI application)
**Performance Goals**: 1000 names/sec reconciliation throughput
**Constraints**: Offline-capable, idempotent optimization (rebuild from scratch)
**Scale/Scope**: 100M scientific name-strings, 200M occurrences, 10M vernacular names, 20M occurrences
**User Input**: DatabaseOperator interface must provide pgxpool database connections to high-level lifecycle components

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
specs/001-gnverifier-db-lifecycle/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
│   ├── DatabaseOperator.go
│   ├── SchemaManager.go
│   ├── Populator.go
│   └── Optimizer.go
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
pkg/
├── config/              # Configuration types and validation (pure)
│   ├── config.go
│   └── config_test.go
├── database/            # Database operation interfaces (pure)
│   ├── operator.go      # DatabaseOperator interface
│   └── operator_test.go
├── lifecycle/           # Lifecycle phase interfaces (pure)
│   ├── schema.go        # SchemaManager interface
│   ├── populator.go     # Populator interface
│   ├── optimizer.go     # Optimizer interface
│   └── *_test.go
└── model/               # Data model definitions (pure)
    ├── datasource.go
    ├── namestring.go
    └── *_test.go

internal/io/             # Impure implementations
├── database/
│   ├── operator.go      # DatabaseOperator implementation using pgxpool
│   └── operator_test.go # Integration tests
├── schema/
│   ├── manager.go       # SchemaManager implementation using GORM
│   └── manager_test.go
├── populate/
│   ├── importer.go      # Populator implementation using sflib
│   └── importer_test.go
├── optimize/
│   ├── optimizer.go     # Optimizer implementation
│   └── optimizer_test.go
└── config/
    ├── loader.go        # Config file/flag loading
    └── loader_test.go

cmd/gndb/
├── main.go              # Root command setup
├── create.go            # Create subcommand
├── migrate.go           # Migrate subcommand
├── populate.go          # Populate subcommand
└── optimize.go          # Optimize subcommand
```

**Structure Decision**: Single Go project structure following constitutional pure/impure separation. DatabaseOperator (pkg/database) defines the interface for basic database operations and connection management. Implementations in internal/io/database provide pgxpool-based connections. High-level lifecycle components (SchemaManager, Populator, Optimizer) receive database connections and execute specialized SQL internally.

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
- Follow TDD workflow strictly (RED-GREEN-REFACTOR):
  1. Contract test tasks (verify interface compliance, must fail initially) [P]
  2. Implementation tasks to make contract tests pass
  3. Integration test tasks (end-to-end lifecycle scenarios)
  4. CLI subcommand tasks (create, migrate, populate, optimize)
  5. Configuration and validation tasks
  6. Documentation updates (CLAUDE.md)

**Key Task Categories**:
1. **DatabaseOperator**:
   - Contract test for Pool() method
   - pgxpool implementation with connection management
   - Integration tests for TableExists, DropAllTables
2. **SchemaManager**:
   - Contract test for Create/Migrate methods
   - GORM AutoMigrate implementation
   - Integration tests with test database
3. **Populator**:
   - Contract test for Populate method
   - sflib integration for SFGA reading
   - pgx CopyFrom for bulk inserts
   - Progress logging
4. **Optimizer**:
   - Contract test for Optimize method
   - Idempotent drop/recreate logic
   - Index and materialized view creation
5. **CLI Subcommands**:
   - create, migrate, populate, optimize commands
   - Configuration loading (viper)
   - Error handling and user prompts

**Ordering Strategy**:
- TDD order: Contract tests → implementations → integration tests → CLI
- Dependency order: pkg/config → pkg/database → pkg/lifecycle → internal/io/* → cmd/gndb
- Mark [P] for parallel execution (independent contracts/tests)

**Estimated Output**: 25-35 numbered, ordered tasks in tasks.md

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
- [X] Phase 0: Research complete (/plan command)
- [X] Phase 1: Design complete (/plan command)
- [X] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [X] Initial Constitution Check: PASS
- [X] Post-Design Constitution Check: PASS
- [X] All NEEDS CLARIFICATION resolved (FR-011, FR-012 documented in research.md)
- [X] Complexity deviations documented (None - design follows all constitutional principles)

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
