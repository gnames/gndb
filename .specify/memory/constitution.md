<!--
SYNC IMPACT REPORT
Version: 1.1.0 → 1.2.0
Change Type: MINOR (principle expanded with material guidance)
Modified Principles: IV. CLI-First Interface (expanded with subcommand architecture)
Added Sections: None
Removed Sections: None
Templates Status:
  ⚠ .specify/templates/plan-template.md (requires update - add subcommand check)
  ✅ .specify/templates/spec-template.md (validated - no conflicts)
  ✅ .specify/templates/tasks-template.md (validated - no conflicts)
Follow-up TODOs: None
-->

# GNdb Constitution

## Core Principles

### I. Modular Architecture
- Every feature MUST be implemented as a separate, self-contained module
- Modules MUST communicate through well-defined interfaces only
- Each module MUST have a clear, single responsibility
- No direct dependencies between implementation modules; use interface contracts
- Module boundaries MUST be respected; no reaching into internal implementation details

**Rationale**: Interface-based design enables independent testing, clear boundaries, and maintainable growth as the database tooling evolves.

### II. Pure/Impure Code Separation
- Pure logic (data transformations, business rules) MUST be separated from impure code
- All state changes, I/O operations, and side effects MUST be isolated in dedicated `io` modules
- Pure functions MUST be the default; impure code implements interfaces defined by pure modules
- Database operations, file system access, and network calls MUST live in `io` boundaries
- Pure code MUST NOT import or depend on `io` modules

**Rationale**: Separating pure from impure code makes testing straightforward, reasoning about behavior clear, and refactoring safe.

### III. Test-Driven Development (NON-NEGOTIABLE)
- TDD is mandatory for all new functionality
- Workflow: Write test → Test fails → Implement → Test passes → Refactor
- Tests MUST be written and verified to fail before any implementation code
- Red-Green-Refactor cycle is strictly enforced
- No feature is complete without passing tests

**Rationale**: TDD ensures code correctness, prevents regressions, and serves as living documentation for expected behavior.

### IV. CLI-First Interface
- All functionality MUST be exposed via command-line interface
- CLI MUST use subcommands to break down complexity into independent operations
- Each database lifecycle phase MUST be a separate subcommand:
  * `create` - Database creation
  * `migrate` - Schema migrations
  * `populate` - Data population
  * `restructure` - Internal reorganization
- Subcommands MUST be independently executable and composable
- Input: command-line arguments and flags
- Output: structured data to stdout (JSON, TSV, or human-readable formats)
- Errors: descriptive messages to stderr with appropriate exit codes
- No GUI, web server, or graphical dependencies

**Rationale**: Subcommand architecture enables focused, testable operations that align with database lifecycle phases, supports Unix composition patterns, and keeps cognitive load manageable.

### V. Open Source Readability
- Code MUST be written for human understanding, not just machine execution
- Public APIs and exported functions MUST have clear documentation
- Complex algorithms MUST include explanatory comments
- Variable and function names MUST be descriptive and follow Go conventions
- Code reviews MUST verify readability before merge

**Rationale**: Open source projects thrive on contributor understanding; readable code lowers barriers to contribution and maintenance.

### VI. Configuration Management
- External configuration via YAML files MUST be supported (gndb.yaml)
- CLI flags and arguments MUST override file-based configuration settings
- Configuration precedence order MUST be enforced: flags > environment variables > config file > defaults
- Configuration schema MUST be documented and validated at startup
- Configuration parsing MUST fail fast with clear error messages for invalid settings
- Default values MUST be sensible and allow zero-config operation where practical

**Rationale**: Consistent configuration management across features prevents user confusion, enables automation, and supports both interactive and scripted usage patterns.

### VII. Development principles.

Follow KISS and Do Not Repeat Yourself principles. Keep code without unnecessary repetitions including documentation. Write documentation concise and clear, oriented for human reading as well as LLM.

## Development Workflow

### Testing Requirements
- Unit tests for pure logic modules
- Integration tests for `io` module contracts
- Contract tests verify interface compliance
- All tests MUST pass before merge
- Test coverage for critical paths is mandatory
- Each subcommand MUST have integration tests verifying end-to-end behavior

### Code Organization
```
project/
├── pkg/                 # Pure logic modules (public APIs)
│   ├── config/          # Configuration types and validation
│   │   ├── config.go
│   │   └── config_test.go
│   └── module/
│       ├── interfaces.go    # Interface definitions
│       ├── logic.go         # Pure implementations
│       └── logic_test.go    # Unit tests
├── internal/io/        # Impure implementations
│   ├── config/          # Config file reading, flag parsing
│   │   ├── loader.go
│   │   └── loader_test.go
│   └── module/
│       ├── implementation.go    # Implements pkg/module interfaces
│       └── integration_test.go  # I/O integration tests
└── cmd/                # CLI entry points
    └── gndb/
        ├── main.go      # Root command setup
        ├── create.go    # Create subcommand
        ├── migrate.go   # Migrate subcommand
        ├── populate.go  # Populate subcommand
        └── restructure.go  # Restructure subcommand
```

### Quality Gates
- All code MUST pass `go vet` and `golint` checks
- Tests MUST pass: `go test ./...`
- Code reviews verify principle compliance
- Documentation MUST be updated alongside code changes
- Each subcommand MUST have help text and usage examples

## Governance

### Amendment Process
This constitution supersedes all other development practices and patterns. Amendments require:
1. Documented proposal with rationale
2. Impact analysis on existing codebase
3. Team approval (for multi-person teams) or maintainer approval
4. Migration plan for affected code
5. Version increment following semantic versioning

### Versioning Policy
- **MAJOR**: Backward-incompatible principle changes or removals
- **MINOR**: New principles added or significant expansions
- **PATCH**: Clarifications, wording improvements, non-semantic fixes

### Compliance Review
- All pull requests MUST verify constitutional compliance
- Violations MUST be justified or design must be revised
- Complexity introduced MUST solve real problems, not imagined ones
- When in doubt, choose simplicity over sophistication

**Version**: 1.2.0 | **Ratified**: 2025-10-01 | **Last Amended**: 2025-10-01
