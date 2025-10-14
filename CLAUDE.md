# gndb Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-10-02

## Active Technologies
- Go 1.21+ (001-gnverifier-db-lifecycle)
- Go 1.25 + pgx/v5 (pgxpool for connection pooling), GORM (AutoMigrate for schema management), cobra (CLI), viper (config), sflib (SFGA data import) (001-gnverifier-db-lifecycle)
- PostgreSQL (primary), SQLite (SFGA format data sources) (001-gnverifier-db-lifecycle)

## Project Structure
```
src/
tests/
```

## Commands
# Add commands for Go 1.21+

## Code Style
Go 1.21+: Follow standard conventions

## Recent Changes
- 001-gnverifier-db-lifecycle: Added [if applicable, e.g., PostgreSQL, CoreData, files or N/A]
- 001-gnverifier-db-lifecycle: Added Go 1.25 + pgx/v5 (pgxpool for connection pooling), GORM (AutoMigrate for schema management), cobra (CLI), viper (config), sflib (SFGA data import)
- 001-gnverifier-db-lifecycle: Added Go 1.21+

<!-- MANUAL ADDITIONS START -->

## Architecture (v1.4.0 - Flattened Structure)

### Package Structure
- **pkg/**: Pure interfaces and types (no I/O)
  - `config/`: Configuration types, validation, defaults
  - `db/`: DatabaseOperator interface (connection management)
  - `lifecycle/`: SchemaManager, Populator, Optimizer interfaces
  - `schema/`: GORM models for database schema
  - `populate/`: Pure populate logic (sources.yaml parsing, filtering)
  - `templates/`: Embedded config/sources.yaml templates

- **internal/**: Impure I/O implementations (io* prefix convention)
  - `iodb/`: PgxOperator (PostgreSQL via pgxpool)
  - `ioschema/`: SchemaManager (GORM AutoMigrate)
  - `iopopulate/`: Populator (SFGA import via sflib, 5-phase workflow)
  - `iooptimize/`: Optimizer (indexes, materialized views)
  - `ioconfig/`: Config loading (viper, YAML/env/flags)
  - `iotesting/`: Shared test utilities

- **cmd/gndb/**: CLI commands (cobra)
  - `create`: Create schema
  - `migrate`: Apply migrations
  - `populate`: Import SFGA data (5 phases)
  - `optimize`: Create indexes

### Design Principles
1. **Pure/Impure Separation**: Interfaces in pkg/, implementations in internal/
2. **Package Naming**: Names match directories; io* prefix for all I/O packages
3. **No Import Aliases**: Package name = directory name (idiomatic Go)
4. **Interface-Based**: db.Operator, lifecycle.{SchemaManager,Populator,Optimizer}
5. **Performance**: pgx CopyFrom for bulk inserts (100M+ records)

### Key Interfaces
```go
// pkg/db/operator.go
type Operator interface {
    Connect(ctx, *config.DatabaseConfig) error
    Pool() *pgxpool.Pool  // For CopyFrom, transactions
    TableExists(ctx, string) (bool, error)
    HasTables(ctx) (bool, error)
    DropAllTables(ctx) error
    Close() error
}

// pkg/lifecycle/populator.go
type Populator interface {
    Populate(ctx, *config.Config) error
}
```

### Populate Workflow (5 Phases)
**Phase 0: Fetch & Cache SFGA**
- Download from URL or use local parent directory
- Cache at `~/.cache/gndb/sfga/{sourceID:04d}.sqlite`
- Open SQLite database via sflib

**Phase 1: Name Strings**
- Read from SFGA `name_string` table
- Parse via gnparser (botanical code, concurrent pool)
- Insert into `name_strings` table via pgx CopyFrom
- Files: `internal/iopopulate/names.go`

**Phase 1.5: Hierarchy**
- Build taxonomy tree from SFGA `taxon` table
- Store in memory map for breadcrumb generation
- Files: `internal/iopopulate/hierarchy.go`

**Phase 2: Name Indices**
- Process taxa (accepted names), synonyms, bare names
- Link to `name_strings` via name_string_id
- Insert into `name_string_indices` via CopyFrom
- Files: `internal/iopopulate/indices.go`

**Phase 3-4: Vernaculars**
- Phase 3: Vernacular strings → `vernacular_string_indices` table
- Phase 4: Vernacular indices → `vernacular_name_string_indices` table
- Files: `internal/iopopulate/vernaculars.go`

**Phase 5: Metadata**
- Update `data_sources` table with record counts, timestamps
- Files: `internal/iopopulate/metadata.go`

### Key Files by Responsibility
- **Orchestration**: `internal/iopopulate/populator.go` (Populate method)
- **SFGA I/O**: `internal/iopopulate/sfga.go` (download, open)
- **Cache**: `internal/iopopulate/cache.go` (directory management)
- **Sources Config**: `pkg/populate/sources.go` (parsing, filtering)
- **CLI Entry**: `cmd/gndb/populate.go` (flags, validation)

### Configuration
- Config file: `~/.config/gndb/config.yaml`
- Sources file: `~/.config/gndb/sources.yaml`
- Precedence: CLI flags > env vars > YAML > defaults
- Cache: `~/.cache/gndb/sfga/` (all platforms)

### Testing
- Contract tests: `pkg/lifecycle/*_test.go` (interface compliance)
- Unit tests: `pkg/` packages (pure logic)
- Integration tests: `internal/io*/*_test.go` (require PostgreSQL + SFGA)
- E2E tests: `cmd/gndb/*_test.go` (full workflows)
- Run `go test -short` to skip integration tests

### Database Schema
- 10 core tables: data_sources, name_strings, canonicals, name_string_indices, etc.
- GORM models: `pkg/schema/models.go`
- Collation "C": Scientific name sorting (internal/ioschema/manager.go)

<!-- MANUAL ADDITIONS END -->
