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
- 001-gnverifier-db-lifecycle: Added Go 1.25 + pgx/v5 (pgxpool for connection pooling), GORM (AutoMigrate for schema management), cobra (CLI), viper (config), sflib (SFGA data import)
- 001-gnverifier-db-lifecycle: Added Go 1.21+

<!-- MANUAL ADDITIONS START -->

## Architecture (Refactored v1.3.0)

### Package Structure
- **pkg/**: Pure interfaces and types (no I/O)
  - `config/`: Configuration types
  - `database/`: DatabaseOperator interface (connection management)
  - `lifecycle/`: SchemaManager, Populator, Optimizer interfaces
  - `schema/`: GORM models for database schema
  - `logger/`: Logging configuration

- **internal/io/**: Impure I/O implementations
  - `database/`: PgxOperator (PostgreSQL connections via pgxpool)
  - `schema/`: SchemaManager (GORM AutoMigrate)
  - `populate/`: Populator (SFGA import via sflib)
  - `optimize/`: Optimizer (indexes, materialized views)
  - `config/`: Config loading (viper)

- **cmd/gndb/**: CLI commands (cobra)
  - `create`: Create database schema
  - `migrate`: Apply schema migrations
  - `populate`: Import SFGA data
  - `optimize`: Create indexes and optimize

### Design Principles
1. **Pure/Impure Separation**: Interfaces in pkg/, implementations in internal/io/
2. **DatabaseOperator Pattern**: Exposes `*pgxpool.Pool` to lifecycle components
3. **Lifecycle Interfaces**: SchemaManager, Populator, Optimizer for each phase
4. **GORM AutoMigrate**: Schema versioning handled by GORM
5. **pgx CopyFrom**: Bulk inserts for 100M+ records performance

### Key Interfaces
```go
// pkg/database/operator.go
type Operator interface {
    Connect(ctx, *config.DatabaseConfig) error
    Pool() *pgxpool.Pool  // For CopyFrom, transactions
    TableExists(ctx, string) (bool, error)
    HasTables(ctx) (bool, error)
    DropAllTables(ctx) error
}

// pkg/lifecycle/schema.go
type SchemaManager interface {
    Create(ctx, *config.Config) error
    Migrate(ctx, *config.Config) error
}

// pkg/lifecycle/populator.go
type Populator interface {
    Populate(ctx, *config.Config) error
}

// pkg/lifecycle/optimizer.go
type Optimizer interface {
    Optimize(ctx, *config.Config) error
}
```

### Workflow
1. `gndb create`: DatabaseOperator → SchemaManager.Create() → GORM AutoMigrate
2. `gndb migrate`: DatabaseOperator → SchemaManager.Migrate() → GORM AutoMigrate
3. `gndb populate`: DatabaseOperator → Populator.Populate() → sflib + pgx CopyFrom
4. `gndb optimize`: DatabaseOperator → Optimizer.Optimize() → CREATE INDEX/MATERIALIZED VIEW

### Testing
- Unit tests: pkg/ packages (pure logic)
- Contract tests: pkg/lifecycle/*_test.go (interface compliance)
- Integration tests: cmd/gndb/*_integration_test.go (requires PostgreSQL)
- Run with `go test -short` to skip integration tests

### Configuration
- YAML: `~/.config/gndb/config.yaml` or `./gndb.yaml`
- Environment: `GNDB_DATABASE_HOST`, `GNDB_DATABASE_USER`, etc.
- CLI flags: `--config`, `--force`
- Precedence: flags > env vars > config file > defaults

### Database Schema
- 10 core tables: data_sources, name_strings, canonicals, etc.
- GORM models in pkg/schema/models.go
- Collation "C" for scientific name sorting (internal/io/schema/manager.go:82)

<!-- MANUAL ADDITIONS END -->
