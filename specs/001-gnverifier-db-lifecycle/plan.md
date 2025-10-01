# Implementation Plan: GNverifier Database Lifecycle Management

**Branch**: `001-gnverifier-db-lifecycle` | **Date**: 2025-10-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/Users/dimus/code/golang/gndb/specs/001-gnverifier-db-lifecycle/spec.md`

## Summary
Build a complete database lifecycle management tool for GNverifier that enables users to create, migrate, populate, and optimize PostgreSQL databases for scientific name verification. The system must handle 100M+ scientific names with 1000 names/sec reconciliation throughput, support SFGA format data sources via sflib, and provide independent subcommands for each lifecycle phase (create, migrate, populate, restructure). Both the main GNverifier service and local user instances use identical tooling.

## Technical Context
**Language/Version**: Go 1.21+  
**Primary Dependencies**:  
- **CLI Framework**: cobra (command structure), viper (configuration)  
- **Database**: pgx (PostgreSQL driver), atlas (migrations)  
- **Data Import**: github.com/sfborg/sflib (SFGA format)  
- **Name Parsing**: github.com/gnames/gnparser  
- **Reference Implementation**: github.com/gnames/gnidump (prototype)  
- **Testing**: testify/assert (unit tests), custom integration test framework
- **Task Runner**: just (command runner for development tasks)

**Storage**: PostgreSQL (main database), SQLite (SFGA format data sources)  
**Testing**: Go standard testing with testify/assert  
**Target Platform**: Linux/macOS/Windows CLI  
**Project Type**: single (Go CLI tool)  
**Performance Goals**:  
- 1000 names/sec reconciliation throughput (baseline)  
- Support 100M scientific names + 200M occurrences  
- Support 10M vernacular names + 20M occurrences  

**Constraints**:  
- Read-only database after setup (optimized for queries)  
- Recovery via restart-from-scratch model  
- All data sources must use same SFGA version for initial ingest  

**Scale/Scope**:  
- Main service: 100M+ names  
- Local instances: flexible scale based on hardware  

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Modular Architecture**
- [x] Each feature component is a separate module with single responsibility
  - `create` module: schema creation logic
  - `migrate` module: schema migration logic
  - `populate` module: data import logic
  - `restructure` module: optimization logic
- [x] Modules communicate only through interfaces, no direct implementation coupling
  - Database interface defines operations
  - Data source interface for SFGA reading
  - Configuration interface for settings
- [x] Module boundaries clearly defined and documented
  - pkg/ for pure logic, internal/io/ for impure operations

**II. Pure/Impure Code Separation**
- [x] Pure logic separated from I/O and state-changing operations
  - Schema definitions in pkg/schema
  - Migration logic in pkg/migrate
  - Data transformation in pkg/transform
- [x] All database, file system, network operations isolated in `io` modules
  - internal/io/pg for PostgreSQL operations
  - internal/io/sfga for SFGA file reading
  - internal/io/config for configuration loading
- [x] Pure functions do not import or depend on `io` modules
  - pkg/ modules define interfaces that io/ implements

**III. Test-Driven Development** *(NON-NEGOTIABLE)*
- [x] Tests written first and verified to fail before implementation
  - Contract tests for each module interface
  - Integration tests for subcommands
- [x] Red-Green-Refactor workflow documented in task ordering
  - Tasks explicitly separate test creation from implementation
- [x] All features include passing tests before considered complete
  - CI gates on test passage

**IV. CLI-First Interface**
- [x] All functionality exposed via CLI commands using subcommands
  - `gndb create` - schema creation
  - `gndb migrate` - schema migration
  - `gndb populate` - data population
  - `gndb restructure` - optimization
- [x] Database lifecycle phases separated: create, migrate, populate, restructure
  - Each subcommand is independent
- [x] Subcommands are independently executable and composable
  - Can run individually or in sequence
- [x] Structured output to stdout, errors to stderr
  - JSON and human-readable formats supported
- [x] No GUI, web, or graphical dependencies introduced
  - Pure CLI tool

**V. Open Source Readability**
- [x] Public APIs documented with clear godoc comments
  - All exported functions/types documented
- [x] Complex logic includes explanatory comments
  - Schema definitions, optimization strategies commented
- [x] Names follow Go conventions and are self-documenting
  - Clear, descriptive names throughout

**VI. Configuration Management**
- [x] YAML configuration file support included (gndb.yaml)
  - Database connection, SFGA paths, etc.
- [x] CLI flags override file-based configuration settings
  - viper handles precedence
- [x] Precedence order enforced: flags > env vars > config file > defaults
  - Built into viper configuration
- [x] Configuration schema documented and validated at startup
  - Validation in pkg/config
- [x] Fail-fast with clear errors for invalid configuration
  - Early validation before operations

## Project Structure

### Documentation (this feature)
```
specs/001-gnverifier-db-lifecycle/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (interface definitions)
└── tasks.md             # Phase 2 output (NOT created by /plan)
```

### Source Code (repository root)
```
gndb/
├── cmd/
│   └── gndb/
│       ├── main.go           # Root command, cobra setup
│       ├── create.go         # Create subcommand
│       ├── migrate.go        # Migrate subcommand
│       ├── populate.go       # Populate subcommand
│       └── restructure.go    # Restructure subcommand
├── pkg/                      # Pure logic modules (public APIs)
│   ├── config/
│   │   ├── config.go         # Configuration types
│   │   ├── config_test.go    # Config validation tests
│   │   └── interfaces.go     # Config loader interface
│   ├── schema/
│   │   ├── schema.go         # Schema definitions
│   │   ├── schema_test.go
│   │   └── interfaces.go     # Schema operations interface
│   ├── migrate/
│   │   ├── migrate.go        # Migration logic
│   │   ├── migrate_test.go
│   │   └── interfaces.go
│   ├── populate/
│   │   ├── populate.go       # Data population logic
│   │   ├── populate_test.go
│   │   └── interfaces.go
│   ├── restructure/
│   │   ├── optimize.go       # Optimization logic
│   │   ├── optimize_test.go
│   │   └── interfaces.go
│   └── model/
│       ├── namestring.go     # Core data models
│       ├── datasource.go
│       ├── vernacular.go
│       └── synonym.go
├── internal/io/              # Impure implementations
│   ├── config/
│   │   ├── loader.go         # YAML/flag loading (implements pkg/config)
│   │   └── loader_test.go
│   ├── pg/
│   │   ├── client.go         # PostgreSQL client (implements pkg/schema, etc.)
│   │   ├── client_test.go
│   │   ├── schema.go         # Schema operations
│   │   ├── migrate.go        # Migration operations
│   │   ├── populate.go       # Data insertion
│   │   └── optimize.go       # Index/view creation
│   ├── sfga/
│   │   ├── reader.go         # SFGA file reading via sflib
│   │   └── reader_test.go
│   └── gnparser/
│       ├── client.go         # Name parsing integration
│       └── client_test.go
├── migrations/               # Atlas migration files
│   ├── 001_initial_schema.sql
│   ├── 002_add_vernacular.sql
│   └── README.md
├── testdata/                 # Test fixtures
│   ├── sample.sfga          # Sample SFGA file
│   └── config_sample.yaml
├── gndb.yaml.example        # Example configuration
├── go.mod
├── go.sum
└── README.md
```

**Structure Decision**: Single Go project structure with clear pkg/internal separation per constitutional requirements. CLI subcommands in cmd/gndb/, pure logic in pkg/, I/O implementations in internal/io/.

## Phase 0: Outline & Research

**Status**: Starting research phase

### Research Tasks

1. **PostgreSQL Schema Design for 100M+ Names**
   - Research indexing strategies for scientific name matching
   - Investigate trigram/fuzzy matching indexes for name verification
   - Study partitioning strategies for large-scale name tables
   - Review PostgreSQL performance tuning for read-heavy workloads

2. **SFGA Format and sflib Integration**
   - Study sflib API and data structures
   - Understand SFGA versioning and compatibility requirements
   - Identify how to handle version differences during updates
   - Map SFGA entities to PostgreSQL schema

3. **Atlas Migration Framework**
   - Review atlas migration workflow and best practices
   - Understand versioning and rollback capabilities (if any)
   - Determine how to integrate with gndb CLI
   - Reference github.com/gnames/gnames/migrations/README.md

4. **Name Verification Optimization Strategies**
   - Research materialized views for name lookups
   - Study denormalization strategies for performance
   - Investigate caching patterns for frequent queries
   - Review vernacular name language indexing approaches

5. **gnidump as Reference Implementation**
   - Analyze gnidump architecture and patterns
   - Extract reusable components and approaches
   - Identify areas where gndb should differ
   - Document lessons learned

**Output**: research.md consolidating all findings with decisions, rationales, and alternatives considered

## Phase 1: Design & Contracts

**Status**: Pending Phase 0 completion

### Planned Artifacts

1. **data-model.md**: PostgreSQL schema design
   - name_strings table (scientific names)
   - name_string_occurrences table (links to data sources)
   - vernacular_names table (common names by language)
   - synonyms table (alternative names)
   - data_sources table (SFGA source metadata)
   - schema_version table (migration tracking)

2. **contracts/** directory: Go interface definitions
   - `DatabaseManager` interface: CRUD operations
   - `SchemaCreator` interface: schema creation
   - `Migrator` interface: migration operations  
   - `DataPopulator` interface: bulk data insertion
   - `Optimizer` interface: index/view creation
   - `ConfigLoader` interface: configuration reading
   - `SFGAReader` interface: SFGA file parsing

3. **quickstart.md**: End-to-end workflow test
   - Setup PostgreSQL database
   - Run `gndb create` with sample config
   - Run `gndb populate` with testdata/sample.sfga
   - Run `gndb restructure` to optimize
   - Query verification: measure throughput
   - Cleanup steps

4. **Contract Tests**: Failing tests for each interface
   - Test files in pkg/*/interfaces_test.go
   - Assert interface contracts without implementation
   - Verify failure before implementation phase

**Output**: Detailed design documents, interface definitions, failing contract tests

## Phase 2: Task Planning Approach
*Description only - tasks.md created by /tasks command*

### Task Generation Strategy

**From Contracts**:
- Each interface definition → contract test task [P]
- Each interface → implementation task in internal/io/

**From Data Model**:
- Each table → schema definition task [P]
- Indexes and constraints → optimization tasks
- Migration files → versioned SQL creation tasks

**From Subcommands**:
- create → cobra command setup + schema creation integration
- migrate → cobra command + atlas integration
- populate → cobra command + SFGA reading + bulk insert
- restructure → cobra command + optimization execution

**Ordering Strategy**:
1. Setup & Configuration (config module, viper/cobra setup)
2. Tests First (all contract tests, integration test scaffolding) [P]
3. Core Pure Modules (pkg/config, pkg/schema, pkg/model) [P]
4. I/O Implementations (internal/io/pg, internal/io/sfga, internal/io/config)
5. CLI Integration (cmd/gndb subcommands)
6. Migration Setup (atlas configuration, initial migrations)
7. Integration Tests (quickstart validation)
8. Documentation & Examples

**Estimated Output**: 40-50 tasks in dependency order with parallel execution markers

## Complexity Tracking
*No constitutional violations - all principles satisfied*

No complexity deviations. The design aligns with all constitutional principles:
- Modular architecture with clear boundaries
- Pure/impure separation enforced
- TDD workflow planned
- CLI-first with subcommands
- Configuration management built-in

## Progress Tracking

**Phase Status**:
- [ ] Phase 0: Research complete (/plan command)
- [ ] Phase 1: Design complete (/plan command)
- [ ] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [ ] Post-Design Constitution Check: PASS (pending Phase 1)
- [ ] All NEEDS CLARIFICATION resolved (will be resolved during Phase 0/1)
- [x] Complexity deviations documented (none)

---
*Based on Constitution v1.2.0 - See `.specify/memory/constitution.md`*
