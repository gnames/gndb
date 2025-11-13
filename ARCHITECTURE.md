# Architecture

GNdb follows Clean Architecture principles with clear separation between business
logic, use cases, and implementation details.

## Core Principles

1. **Dependency Rule**: Dependencies point inward - outer layers depend on inner
   layers, never the reverse
2. **Interface Segregation**: Small, focused interfaces for each business
   operation
3. **Separation of Concerns**: Business logic is independent of frameworks,
   databases, and I/O

## Layer Structure

```
cmd/                    Frameworks & Drivers (CLI, entry point)
  ↓ depends on
internal/               Interface Adapters (implementations)
  ├── iopopulate/      - Populator implementation
  ├── ioschema/        - SchemaManager implementation
  ├── iooptimize/      - Optimizer implementation
  ├── iodb/            - Database adapter (pgx)
  └── iofs/            - File system adapter
  ↓ depends on
pkg/gndb/              Use Cases (core operations)
  ├── SchemaManager    - Database schema operations
  ├── Populator        - SFGA data import
  └── Optimizer        - Performance optimizations
  ↓ depends on
pkg/                   Entities (domain models)
  ├── schema/          - Database entities (DataSource, NameString, etc.)
  ├── sources/         - Source configuration entities
  ├── config/          - Application configuration
  └── db/              - Database gateway interface
```

## Package Responsibilities

### `pkg/` - Business Logic (Portable, Framework-Independent)

- **`config/`** - Application configuration with Option pattern
- **`gndb/`** - Core use case interfaces and version info
- **`schema/`** - Database models (GORM entities for scientific names)
- **`sources/`** - SFGA source configuration and validation
- **`db/`** - Database operator interface (connection, basic operations)
- **`errcode/`** - Error codes for programmatic handling

### `internal/` - Implementation Details (I/O, Frameworks)

- **`iopopulate/`** - Implements Populator (reads SFGA, bulk inserts)
- **`ioschema/`** - Implements SchemaManager (wraps GORM AutoMigrate)
- **`iooptimize/`** - Implements Optimizer (creates indexes, views)
- **`iodb/`** - Implements database Operator (pgx connection pooling)
- **`iofs/`** - File system operations (config directories)
- **`iologger/`** - Logging setup

### `cmd/` - CLI Interface

- Cobra/Viper framework integration
- Command definitions (schema, populate, optimize)
- Flag parsing and configuration initialization

## Key Design Patterns

### Interface-Driven Design

All core operations defined as interfaces in `pkg/gndb/`:

```go
type SchemaManager interface {
    Create(ctx, cfg) error
    Migrate(ctx, cfg) error
}

type Populator interface {
    Populate(ctx) error
}

type Optimizer interface {
    Optimize(ctx, cfg) error
}
```

Implementations live in `internal/` with private structs, public constructors:

```go
// internal/ioschema/manager.go
type manager struct { /* private */ }

func NewManager(op db.Operator) gndb.SchemaManager {
    return &manager{operator: op}
}
```

### Configuration Immutability

Config is immutable after creation - all mutations via Option functions:

```go
cfg := config.New()                           // Always valid
cfg.Update(config.OptDatabaseHost("localhost")) // Safe mutation
```

### Separation of I/O from Validation

- `pkg/sources/` - Pure validation (data structure checks)
- `internal/iopopulate/sources.go` - I/O validation (file existence)

## Benefits

1. **Testability** - Business logic tested without I/O or frameworks
2. **Maintainability** - Clear boundaries, one-way dependencies
3. **Flexibility** - Swap implementations (GORM to sqlc, pgx to database/sql)
4. **Portability** - Business logic independent of CLI, can add HTTP API
5. **Screaming Architecture** - Package names reveal domain, not technical
   patterns

## Data Flow Example

```text
CLI Command (cmd/populate.go)
  ↓
Creates Populator (internal/iopopulate/populator.go)
  ↓
Calls Populate() via gndb.Populator interface
  ↓
Uses entities (pkg/sources/, pkg/schema/)
  ↓
Writes to database via db.Operator interface
  ↓
pgx implementation (internal/iodb/pgx_operator.go)
```

Dependencies always point inward: `cmd` → `internal` → `pkg/gndb` → entities.

## Directory Layout

```text
gndb/
├── cmd/                 # CLI commands (Cobra)
├── pkg/                 # Public API (importable)
│   ├── gndb/           # Core interfaces + version
│   ├── schema/         # Database models
│   ├── sources/        # Source configuration
│   ├── config/         # Configuration
│   ├── db/             # Database interface
│   └── errcode/        # Error codes
├── internal/            # Private implementation
│   ├── iopopulate/     # Population implementation
│   ├── ioschema/       # Schema implementation
│   ├── iooptimize/     # Optimization implementation
│   ├── iodb/           # Database implementation
│   ├── iofs/           # File system operations
│   └── iologger/       # Logging setup
└── main.go             # Entry point
```

## References

- Clean Architecture by Robert C. Martin
- [Project layout standards](https://github.com/golang-standards/project-layout)
