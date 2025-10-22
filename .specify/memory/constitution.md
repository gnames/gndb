<!--
SYNC IMPACT REPORT
Version: 1.6.0 → 1.6.1
Change Type: PATCH (code style clarification)
Modified Principles:
  - V. Open Source Readability (added 80-column line width requirement)
Added Sections:
  - None
Removed Sections:
  - None
Templates Status:
  ✅ .specify/templates/plan-template.md (no changes needed)
  ✅ .specify/templates/spec-template.md (no changes needed)
  ✅ .specify/templates/tasks-template.md (no changes needed)
  ✅ .specify/templates/commands/*.md (no changes needed)
Rationale:
  - Added 80-column line width guideline to Principle V for consistency
  - PATCH increment as this is a code style clarification, not new principle
  - Includes practical exceptions for impossible/impractical cases
Follow-up TODOs:
  - None (guideline is immediately applicable)
-->

# GNdb Constitution

## Core Principles

### I. Modular Architecture
- Every feature MUST be implemented as a separate, self-contained module
- Modules MUST communicate through well-defined interfaces only
- Each module MUST have a clear, single responsibility
- No direct dependencies between implementation modules; use interface
  contracts
- Module boundaries MUST be respected; no reaching into internal
  implementation details

**Rationale**: Interface-based design enables independent testing, clear
boundaries, and maintainable growth as the database tooling evolves.

### II. Pure/Impure Code Separation
- Pure logic (data transformations, business rules) MUST be separated
  from impure code
- All state changes, I/O operations, and side effects MUST be isolated in
  dedicated `io` modules
- Pure functions MUST be the default; impure code implements interfaces
  defined by pure modules
- Database operations, file system access, and network calls MUST live in
  `io` boundaries

**Import Direction Rules** (enforced by design):
- `pkg/` MUST NOT import `cmd/` or `internal/io/`
- `internal/io/` MAY import `pkg/` (to implement interfaces)
- `internal/io/` MUST NOT import `cmd/`
- `cmd/` MAY import both `pkg/` and `internal/io/` (wiring layer)

**Dependency Flow**:
```
cmd/       ──→  pkg/         (uses pure interfaces)
  └──────→  internal/io/  (creates implementations)

internal/io/  ──→  pkg/      (implements interfaces)

pkg/       ──X──  cmd/       (FORBIDDEN)
pkg/       ──X──  internal/io/ (FORBIDDEN)
internal/io/ ──X── cmd/       (FORBIDDEN)
```

**Rationale**: Separating pure from impure code makes testing
straightforward, reasoning about behavior clear, and refactoring safe.
Strict import rules prevent architectural drift and ensure pure logic
remains reusable.

### III. Test-Driven Development (NON-NEGOTIABLE)
- TDD is mandatory for all new functionality
- Workflow: Write test → Test fails → Implement → Test passes → Refactor
- Tests MUST be written and verified to fail before any implementation
  code
- Red-Green-Refactor cycle is strictly enforced
- No feature is complete without passing tests

**Rationale**: TDD ensures code correctness, prevents regressions, and
serves as living documentation for a given behavior.

### IV. CLI-First Interface
- All functionality MUST be exposed via command-line interface
- CLI MUST use subcommands to break down complexity into independent
  operations
- Each database lifecycle phase MUST be a separate subcommand:
  * `create` - Database creation
  * `migrate` - Schema migrations
  * `populate` - Data population
  * `restructure` - Internal reorganization
- Subcommands MUST be independently executable and composable
- Input: command-line arguments and flags
- Output: structured data to stdout (JSON, TSV, or human-readable
  formats)
- Errors: descriptive messages to stderr with appropriate exit codes
- No GUI, web server, or graphical dependencies

**Rationale**: Subcommand architecture enables focused, testable
operations that align with database lifecycle phases, supports Unix
composition patterns, and keeps cognitive load manageable.

### V. Open Source Readability
- Code MUST be written for human understanding, not just machine
  execution
- Public APIs and exported functions MUST have clear documentation
- Complex algorithms MUST include explanatory comments
- Variable and function names MUST be descriptive and follow Go
  conventions
- Code reviews MUST verify readability before merge
- **Line Width**: Code, documentation, and comments SHOULD target
  80-column line width where practical
  * Exception: Long string literals, URLs, or technical constraints where
    breaking would harm clarity or functionality
  * Exception: Generated code or third-party dependencies
  * Rationale: 80-column width enables side-by-side diffs, improves
    readability in terminal editors, and forces concise expression

**Rationale**: Open source projects thrive on contributor understanding;
readable code lowers barriers to contribution and maintenance.

### VI. Configuration Management
- External configuration via YAML files MUST be supported (gndb.yaml)
- CLI flags and arguments MUST override file-based configuration settings
- Configuration precedence order MUST be enforced: flags > environment
  variables > config file > defaults
- Configuration schema MUST be documented and validated at startup
- Configuration parsing MUST fail fast with clear error messages for
  invalid settings
- Default values MUST be sensible and allow zero-config operation where
  practical

**Rationale**: Consistent configuration management across features
prevents user confusion, enables automation, and supports both interactive
and scripted usage patterns.

### VII. Development principles.

Follow KISS and Do Not Repeat Yourself principles. Keep code without
unnecessary repetitions including documentation. Write documentation
concise and clear, oriented for human reading as well as LLM.

### VIII. Contributor-First Minimalism (NON-NEGOTIABLE)

This project is designed for hybrid human-LLM collaboration and rapid
contributor onboarding.

**Code**:
- Write the simplest code that solves the problem
- Create abstractions when they improve comprehension or testability
  * Good abstraction: Clear name reveals intent in 2 seconds vs.
    30 seconds reading code
  * Good abstraction: Isolates testable logic from I/O complexity
  * Bad abstraction: Generic frameworks without concrete use cases
  * Bad abstraction: Premature optimization for imagined future needs
- No "just in case" code - implement what's needed now
- Prefer explicit over implicit, clear over clever
- Each function should fit on one screen when practical

**Documentation**:
- Godoc comments: state purpose in 1-2 sentences, skip the obvious
- No redundant prose ("This function does X" when function is named DoX)
- Code should be self-documenting through naming
- Comments explain "why", not "what"

**Tests**:
- Test behavior, not implementation details
- Minimal setup code - inline when possible
- Test names serve as specification (e.g.,
  `TestPopulate_EmptyDatabase_CreatesRecords`)
- One clear assertion per test when practical

**Encouraged**:
- Named functions that make code read like prose
- Pure logic extracted for independent testing
- Interfaces that clarify contracts and enable mocking
- Helper functions that eliminate cognitive load

**Discouraged**:
- Generic abstractions without concrete use cases
- "Framework" patterns (e.g., custom middleware stacks, plugin systems)
- Verbose doc comments that restate the code
- Complex test fixtures for simple scenarios
- Single-use helpers that obscure rather than clarify

**Rationale**: Contributors (human or LLM) should understand a module in
<5 minutes. Every abstraction is a tax on comprehension. Optimize for
change velocity, not architectural purity. This enables rapid onboarding
and sustained contributor engagement.

### IX. Dual-Channel Communication
- The CLI MUST distinguish between user-facing and developer-facing
  output.
- **User-Facing Output (`STDOUT`)**:
    - Progress messages MUST be clear, concise, and keep the user
      informed of the application's state.
    - For every distinct error condition, there MUST be a corresponding
      well-formatted documentation block. This documentation should
      include a title, a clear explanation of the problem, and actionable
      steps for resolution. It is intended for the user and MUST be
      printed to `STDOUT`.
- **Developer-Facing Output (`STDERR`)**:
    - Technical error details, stack traces, and verbose logging MUST be
      directed to `STDERR`. This channel is for developers and system
      administrators.
- The tone of all user-facing communication MUST be helpful and
  encouraging.

**Rationale**: Separating user-friendly guidance from technical logs
allows `gndb` to serve two audiences simultaneously. Users receive clear,
actionable help on `STDOUT` without being overwhelmed by technical jargon,
while developers can diagnose issues using the detailed logs on `STDERR`.
This dual-channel approach improves usability for end-users and maintains
a high level of debuggability for developers.

### X. User-Friendly Documentation
- Documentation, especially for the CLI, MUST be user-friendly.
- Use terminal colors to enhance readability and draw attention to
  important information.
- Headers and titles SHOULD be in a distinct color (e.g., green).
- Important warnings or dangerous operations SHOULD be highlighted in a
  cautionary color (e.g., red).
- Key points or highlights SHOULD be in another distinct color (e.g.,
  yellow).
- The color scheme SHOULD be consistent across the application.

**Rationale**: Using colors in terminal output makes the documentation
more engaging and easier to parse visually. It helps users quickly
identify the most critical pieces of information, improving the overall
user experience.

### XI. Human-in-the-Loop Task Execution (NON-NEGOTIABLE)

AI agents MUST NOT automatically proceed with multiple tasks in sequence
without explicit human approval between tasks.

**Rules**:
- After completing a task, agents MUST stop and wait for human developer
  input
- Agents MUST present completed work and await explicit instruction to
  continue
- Agents MUST NOT chain multiple task executions autonomously
- Between tasks, human developers MUST have opportunity to:
  * Review code changes
  * Provide feedback or corrections
  * Adjust direction or priorities
  * Verify work meets requirements

**Exceptions**:
- Sub-steps within a single task MAY proceed automatically (e.g., write
  test → implement → verify)
- Cleanup operations that are part of the current task (e.g., removing
  temporary files)
- Automated test runs that verify the current task

**Rationale**: 
- **Resource Management**: When AI agent resources (tokens, compute)
  deplete mid-task, work stops incomplete, wasting effort and leaving the
  codebase in an inconsistent state
- **Human Oversight**: Developers need checkpoints to review code, catch
  issues early, and provide course corrections before significant work
  compounds on incorrect assumptions
- **Quality Control**: Incremental review enables catching bugs,
  architectural issues, or misunderstandings immediately rather than after
  multiple tasks have built upon flawed foundations
- **Learning Opportunity**: Humans learn from reviewing intermediate
  results; fully automated chains bypass this critical feedback loop

**Implementation**:
- After task completion, agent outputs summary and stops
- Human reviews changes, runs tests, examines code
- Human explicitly invokes next task when ready (e.g., `/implement T002`)
- Task tools/commands default to single-task execution

## Development Workflow

### Testing Requirements
- Unit tests for pure logic modules
- Integration tests for `io` module contracts
- Contract tests verify interface compliance
- All tests MUST pass before merge
- Test coverage for critical paths is mandatory
- Each subcommand MUST have integration tests verifying end-to-end
  behavior

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
This constitution supersedes all other development practices and patterns.
Amendments require:
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

**Version**: 1.6.1 | **Ratified**: 2025-10-01 | **Last Amended**: 2025-10-22
