# Implementation Plan: GNverifier Database Lifecycle Management

**Branch**: `001-gnverifier-db-lifecycle` | **Date**: 2025-10-03 | **Spec**: [/Users/dimus/code/golang/gndb/specs/001-gnverifier-db-lifecycle/spec.md]

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
This project will enable a user to go through the GNverifier database lifecycle. It would start with empty database, create schema, migrate schema, create performance critical optimization of database as well as modification of data that will speed up name verification (reconciliation and reconciliation) as well as optimizing database data for vernacular names detection by languages and figuring out synonyms of input scientific names. After this project is complete it will allow users to setup GNverifier functionality locally, independent from main gnverifier service. It will be also used at the main service the same way. As a result users who have their own data-sources that are not included on the main site to create local GNverifier that is able to query these data-sources.

## Technical Context
**Language/Version**: Go 1.25
**Primary Dependencies**: pgx/v5, cobra, viper, gorm
**Storage**: PostgreSQL
**Testing**: go test
**Target Platform**: Linux server, macOS CLI
**Project Type**: single Go project
**Performance Goals**: 1000 names/sec reconciliation
**Constraints**: Offline-capable
**Scale/Scope**: 100 million name-strings

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
│   ├── Importer.go
│   ├── MigrationRunner.go
│   ├── Optimizer.go
│   └── SFGAReader.go
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
pkg/
├── config/
│   ├── config.go
│   └── config_test.go
├── migrate/
├── populate/
├── restructure/
└── schema/
    ├── gorm.go
    ├── models.go
    └── schema_test.go
internal/io/
├── config/
│   ├── generate.go
│   ├── generate_test.go
│   ├── loader.go
│   └── loader_test.go
├── database/
│   ├── operator.go
│   └── operator_test.go
└── sfga/
cmd/
└── gndb/
    ├── main.go
    ├── create.go
    ├── migrate.go
    ├── populate.go
    └── restructure.go
```

**Structure Decision**: The project is a single Go project, and the structure is already in place. The new feature will be implemented within the existing structure.

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
| | | |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [X] Phase 0: Research complete (/plan command)
- [X] Phase 1: Design complete (/plan command)
- [ ] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [X] Initial Constitution Check: PASS
- [X] Post-Design Constitution Check: PASS
- [X] All NEEDS CLARIFICATION resolved
- [ ] Complexity deviations documented

---
*Based on Constitution v1.2.0 - See `.specify/memory/constitution.md`*