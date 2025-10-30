# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GNdb is a CLI tool for managing the lifecycle of a PostgreSQL database used by GNverifier (a scientific name verification service). It enables users to set up and maintain local GNverifier instances with custom biodiversity data sources.

The tool handles:
- Database schema creation and migration using GORM
- Data population from SFGA (Species File Group Archive) files
- Database optimization for fast name verification and synonym resolution

## Development Commands

### Building and Installing
```bash
# Build binary to bin/gndb
just build

# Install to ~/go/bin/gndb
just install

# Build multi-platform releases
just release
```

### Testing
```bash
# Run unit tests (skip integration tests)
just test

# Run all tests including integration tests (requires PostgreSQL)
just test-all

# Run tests with coverage report
just test-coverage

# Run tests for specific package
just test-pkg pkg/config

# Run tests with race detector
just test-race
```

### Code Quality
```bash
# Format code
just fmt

# Run linter (requires golangci-lint)
just lint

# Tidy dependencies
just tidy

# Full verification (format, tidy, test, build)
just verify
```

### Development Setup

The project uses direnv for environment configuration:
1. Copy `.envrc.example` to `.envrc`
2. Edit database credentials and settings
3. Run `direnv allow .`

## Architecture

### Application Directories

GNdb uses standard XDG-compliant directories for different types of data:

- **Config Directory** (`config.ConfigDir(homeDir)`): `~/.config/gndb/`
  - Contains `config.yaml` - main configuration file
  - Created automatically on first run

- **Cache Directory** (`config.CacheDir(homeDir)`): `~/.cache/gndb/`
  - Used for downloaded SFGA files and temporary data
  - Safe to delete; will be recreated as needed

- **Log Directory** (`config.LogDir(homeDir)`): `~/.local/share/gndb/logs/`
  - Contains `gndb.log` - structured JSON logs (or text, depending on config)
  - Fresh log file created on each application run
  - Bootstrap logs preserved during logger reconfiguration

All directories are created automatically during bootstrap (cmd/root.go).

### Package Structure

**cmd/** - CLI command definitions using Cobra
- Root command and subcommands
- Flag parsing and configuration initialization
- Bootstrap logic for config directories

**pkg/** - Public API packages
- `config/` - Configuration management with Viper (precedence: flags > env vars > config file > defaults)
- `db/` - Database operator interface (connection pooling, basic operations)
- `lifecycle/` - Core interfaces: SchemaManager, Populator, Optimizer
- `schema/` - GORM models aligned with gnverifier database schema
- `populate/` - SFGA data source configuration and validation
- `templates/` - Embedded YAML templates for config files

**internal/** - Internal implementation (not importable by external code)
- `iofs/` - File system operations for config and data directories

### Key Data Flow

1. **Configuration Loading** (config package):
   - Loads from `~/.config/gndb/gndb.yaml`
   - Overrides with environment variables (GNDB_* prefix)
   - Final overrides from CLI flags
   - Configuration precedence: flags > env vars > config file > defaults

2. **Database Schema** (lifecycle.SchemaManager):
   - Uses GORM AutoMigrate for idempotent schema creation/migration
   - Models defined in pkg/schema/models.go
   - Key tables: DataSource, NameString, Canonical, NameStringIndex, VernacularString, Word

3. **Data Population** (lifecycle.Populator):
   - Reads sources.yaml configuration (in testdata/ for examples)
   - Connects to SFGA SQLite files using sflib
   - Transforms data to PostgreSQL schema
   - Performs bulk inserts using pgx CopyFrom for performance

4. **Optimization** (lifecycle.Optimizer):
   - Creates indexes for fast lookups
   - Builds materialized views
   - Generates denormalized tables for performance
   - Always rebuilds from scratch to apply algorithm improvements

### Database Schema Highlights

The schema is aligned with gnidump for gnverifier compatibility:

- **DataSource**: Metadata about biodiversity datasets (ID, title, version, URLs, curation flags)
- **NameString**: Scientific name-strings with parsing metadata (UUID-based IDs, canonical forms, cardinality)
- **Canonical/CanonicalFull/CanonicalStem**: Different canonical name representations for matching
- **NameStringIndex**: Links name-strings to data sources with taxonomic metadata (rank, status, classification)
- **Word/WordNameString**: Word-level indexes for fuzzy matching
- **VernacularString/VernacularStringIndex**: Common names in multiple languages

### Configuration System

**Environment Variable Design Principle:**
Environment variables are available for **persistent configuration fields only** - i.e., fields included in `ToOptions()`.
These are settings appropriate for `config.yaml` that users may want to override per-environment.

Environment variables use GNDB_ prefix with underscores for nested fields:
```bash
# Persistent fields (in ToOptions) - available as env vars
GNDB_DATABASE_HOST=localhost
GNDB_DATABASE_PORT=5432
GNDB_DATABASE_USER=postgres
GNDB_DATABASE_PASSWORD=secret
GNDB_DATABASE_DATABASE=gnames
GNDB_DATABASE_SSL_MODE=disable
GNDB_DATABASE_BATCH_SIZE=50000
GNDB_LOG_LEVEL=info
GNDB_LOG_FORMAT=json
GNDB_LOG_DESTINATION=file
GNDB_JOBS_NUMBER=8

# Runtime-only fields - NOT available as env vars (set via CLI only)
# HomeDir, Populate.SourceIDs, Populate.ReleaseVersion, Populate.ReleaseDate, Populate.WithFlatClassification
```

**Key Alignment:** `ToOptions()` fields = config.yaml fields = env var fields  
This ensures consistency across all configuration sources.

### SFGA Data Sources

Sources are configured in `sources.yaml` (see testdata/sources.yaml for examples):
- Each source has an ID (< 1000 for official, >= 1000 for custom)
- Parent directory or URL contains SFGA SQLite files
- Metadata can override SFGA col__* fields
- Outlink configuration for generating links to original records

### Interface-Driven Design

Core functionality is defined through interfaces in pkg/lifecycle/:
- **SchemaManager**: Database schema creation and migration
- **Populator**: SFGA data import
- **Optimizer**: Performance optimizations

This allows for testing and alternative implementations.

### CLI Interface

Cobra and Viper are used as a framework for CLI and they are located under cmd directory. There is only one reference in the project to cmd in main.go. Except that cmd module is self-contained

## Testing Conventions

- Unit tests use `testing.Short()` to skip integration tests
- Integration tests require PostgreSQL database (gndb_test)
- Tests use testdata/ directory for fixtures (SFGA files, sources.yaml)
- Table-driven tests with subtests for readability

## Common Patterns

### Error Handling
- Use `gn.Error` struct for user-facing errors with dual output:
  - `Msg` field: User-friendly message displayed to STDOUT (supports `<em>` tags for emphasis)
  - `Err` field: Detailed error with context written to log file (hidden by default)
  - `Code` field: Error codes from pkg/errcode/ for programmatic handling
- Pattern: `return &gn.Error{Code: errcode.SomeError, Msg: "User message", Vars: []any{args}, Err: fmt.Errorf("detailed context: %w", err)}`
- Use `gn.Info()` for user-facing informational messages
- Use `gn.Warn()` for non-fatal warnings
- By default, detailed logs are written to file, not displayed to user

### Database Operations
- Use pgxpool for connection pooling (not individual connections)
- Batch operations use configurable batch size (default 50,000)
- Use pgx CopyFrom for bulk inserts (much faster than individual inserts)
- GORM for schema migrations, pgx for performance-critical operations

### Configuration Updates

**Core Principles:**
- Configuration is immutable after creation via `config.New()`
- **All mutations go through Option functions** - this is the only way to modify Config
- Default config is always valid (no validation needed on creation)
- Option functions validate inputs and reject invalid values with `gn.Warn()`
- Invalid options are silently skipped - config remains in valid state

**Configuration Flow:**
1. Create default config: `cfg := config.New()` (always valid)
2. Load from config.yaml: `cfgViper.ToOptions()` → `cfg.Update(opts)`
3. Apply environment variables: Viper unmarshals directly into cfgViper
4. Apply CLI flags: Create Option functions directly → `cfg.Update(opts)`

**Key Design Decisions:**
- **CLI flags → Option functions** (not flags → Config → ToOptions)
  - Flags call Option functions directly (e.g., `OptDatabaseHost(flagValue)`)
  - Check `cmd.Flags().Changed()` to only apply explicitly set flags
  - Maintains single validation path through Option functions
  
- **ToOptions() scope:**
  - Only converts fields appropriate for config.yaml (persistent settings)
  - Includes: Database settings, Log settings, JobsNumber
  - Excludes: Runtime-only fields (HomeDir, SourceIDs, ReleaseVersion/Date, WithFlatClassification)
  - Used for config.yaml round-trip, not for flag processing

**Validation Strategy:**
- Validation happens in Option functions via `isValidString()`, `isValidInt()`, `isValidEnum()`
- Invalid values trigger `gn.Warn()` with user-friendly message
- Config silently keeps previous valid value (option not applied)
- System continues with valid configuration - no abort on invalid inputs

**Passing Config to Functions:**
- **Always pass the whole `*config.Config`**, not substructs
- Rationale: Uniform, simple, flexible, matches "easy to understand" philosophy
- Most functions need multiple config sections (Database + Log + JobsNumber + HomeDir)
- Document what each function actually uses in comments
- Internal implementations can extract substructs for local readability
- Example:
  ```go
  // Populate imports data from SFGA sources.
  // Uses: cfg.Database, cfg.Populate, cfg.JobsNumber, cfg.HomeDir
  func (p *populator) Populate(ctx context.Context, cfg *config.Config) error {
      dbCfg := &cfg.Database  // Extract locally if helpful
      // ...
  }
  ```

## Code Style

- Follow standard Go formatting (enforced by `go fmt`)
- Use golangci-lint for linting
- Run `go mod tidy` before finalizing tasks to avoid lint warnings
- Comments use full sentences with periods
- Exported types/functions have doc comments

## Human Development Oriented Style

- Functions implemented with human reader in mind 
- Code must be easy to understand
- Documentation aims to be concise and clear
