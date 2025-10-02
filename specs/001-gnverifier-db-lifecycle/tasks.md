# Tasks: GNverifier Database Lifecycle Management

**Feature Branch**: `001-gnverifier-db-lifecycle`  
**Input**: Design documents from `/Users/dimus/code/golang/gndb/specs/001-gnverifier-db-lifecycle/`  
**Prerequisites**: plan.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

**Strategy**: Start with first 5 foundational tasks to establish project structure and TDD workflow

---

## Phase 3.1: Project Setup

### T001: Initialize Go Module and Project Structure

**Description**: Create the Go module and base directory structure per plan.md architecture

**Actions**:
1. Initialize Go module: `go mod init github.com/gnames/gndb`
2. Create directory structure:
   ```
   mkdir -p pkg/{config,schema,migrate,populate,restructure}
   mkdir -p internal/io/{config,database,sfga}
   mkdir -p cmd/gndb
   mkdir -p migrations
   mkdir -p testdata
   ```
3. Create go.mod with initial dependencies:
   - `github.com/jackc/pgx/v5` (PostgreSQL driver)
   - `github.com/spf13/cobra` (CLI framework)
   - `github.com/spf13/viper` (configuration)
   - `github.com/stretchr/testify` (testing)
4. Run `go mod tidy`

**File Paths**:
- `/Users/dimus/code/golang/gndb/go.mod`
- `/Users/dimus/code/golang/gndb/go.sum`

**Success Criteria**:
- [x] Go module initialized with correct import path
- [x] All directories created per plan.md structure
- [x] Dependencies downloaded without errors
- [x] `go build ./...` succeeds (even with empty packages)

**Parallel**: N/A (foundational task)

---

### T002: Create Configuration Package with Types and Validation

**Description**: Implement pure configuration package in `pkg/config/` with struct definitions, defaults, and validation logic

**Actions**:
1. Create `pkg/config/config.go` with:
   - `Config` struct with nested structs for database, import, optimization, logging
   - `DatabaseConfig` struct (host, port, user, password, database, ssl_mode)
   - `ImportConfig` struct with `BatchSizes` map
   - `OptimizationConfig` struct with `ConcurrentIndexes` bool and `StatisticsTargets` map
   - `LoggingConfig` struct (level, format)
   - `Validate()` method that checks required fields (database connection params)
   - `Defaults()` function returning sensible defaults
2. Create `pkg/config/config_test.go` with:
   - Test for `Defaults()` returning valid config
   - Test for `Validate()` rejecting missing required fields
   - Test for `Validate()` accepting complete config

**File Paths**:
- `/Users/dimus/code/golang/gndb/pkg/config/config.go`
- `/Users/dimus/code/golang/gndb/pkg/config/config_test.go`

**Success Criteria**:
- [x] Config struct has all fields from quickstart.md example YAML
- [x] Validation enforces required database fields
- [x] Defaults provide zero-config operation
- [x] All tests pass: `go test ./pkg/config`
- [x] No imports from internal/io/ (pure package)

**Parallel**: [P] - Independent file, no external dependencies

---

### T003: Implement Configuration Loader (Impure I/O)

**Description**: Implement configuration file and flag loading in `internal/io/config/` using viper

**Actions**:
1. Create `internal/io/config/loader.go` with:
   - `Load(configPath string)` function that:
     - Uses viper to read YAML from configPath (or default locations: ./gndb.yaml, ~/.config/gndb/gndb.yaml)
     - Unmarshals into `pkg/config.Config` struct
     - Returns error if file malformed or validation fails
   - `BindFlags(cmd *cobra.Command, cfg *config.Config)` function that:
     - Binds cobra flags to viper keys
     - Returns updated config with CLI flag overrides
2. Create `internal/io/config/loader_test.go` with:
   - Integration test: create temp YAML file, load it, verify struct populated
   - Test: YAML file missing required field → validation error
   - Test: CLI flag override works (flag value takes precedence)

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/config/loader.go`
- `/Users/dimus/code/golang/gndb/internal/io/config/loader_test.go`

**Success Criteria**:
- [x] Loads YAML from file path or default locations
- [x] Unmarshals into pkg/config.Config correctly
- [x] Validation errors propagate to caller
- [x] Integration tests pass with temp YAML files
- [x] Precedence order working: flags > config file > defaults

**Dependencies**: Requires T002 (pkg/config types)

**Parallel**: No (depends on T002)

---

## Phase 3.2: Schema Models (TDD - Tests First)

### T004: [P] Write Schema Model Tests (MUST FAIL)

**Description**: Write tests for Go schema models and DDL generation BEFORE implementation exists

**Actions**:
1. Create `pkg/schema/schema_test.go` with tests that WILL FAIL:
   - `TestDataSourceTableDDL()`: Assert DataSource{}.TableDDL() contains "CREATE TABLE data_sources"
   - `TestNameStringTableDDL()`: Assert NameString{}.TableDDL() contains "CREATE TABLE name_strings" and "canonical_simple TEXT"
   - `TestNameStringIndexDDL()`: Assert NameString{}.IndexDDL() returns slice containing trigram index DDL
   - `TestTaxonTableDDL()`: Assert Taxon{}.TableDDL() includes foreign key to name_strings
   - `TestSchemaVersionTableDDL()`: Assert SchemaVersion{}.TableDDL() creates schema_versions table
2. Run tests and verify they FAIL: `go test ./pkg/schema`
3. Document failure output (expected - no implementation yet)

**File Paths**:
- `/Users/dimus/code/golang/gndb/pkg/schema/schema_test.go`

**Success Criteria**:
- [x] Tests written covering DDL generation for all core models
- [x] Tests FAIL when run (cannot compile or assertions fail)
- [x] Test expectations match data-model.md specifications
- [x] No implementation code written (tests only)

**TDD GATE**: This task MUST be completed and MUST show failing tests before T005

**Parallel**: [P] - Test file only, no implementation

---

### T005: Implement Schema Models with DDL Generation

**Description**: Implement Go schema models in `pkg/schema/models.go` to make T004 tests pass

**Actions**:
1. Create `pkg/schema/models.go` with struct definitions per data-model.md:
   - `DataSource` struct with db and ddl struct tags
   - `NameString` struct with parse_quality CHECK constraint
   - `Taxon` struct with foreign key tags
   - `Synonym` struct
   - `VernacularName` struct
   - `Reference` struct
   - `SchemaVersion` struct
2. Create `pkg/schema/ddl.go` with:
   - `DDLGenerator` interface (TableDDL, IndexDDL, TableName methods)
   - Helper function `generateDDL(model interface{})` that reflects on struct tags to build CREATE TABLE SQL
   - Implement `TableDDL()` method for each model
   - Implement `IndexDDL()` method for models that need secondary indexes
3. Run tests from T004 and verify they now PASS: `go test ./pkg/schema`

**File Paths**:
- `/Users/dimus/code/golang/gndb/pkg/schema/models.go`
- `/Users/dimus/code/golang/gndb/pkg/schema/ddl.go`

**Success Criteria**:
- [x] All model structs match data-model.md specifications
- [x] DDL generation uses reflection on struct tags
- [x] Generated DDL matches expected PostgreSQL syntax
- [x] All tests from T004 now PASS
- [x] No database I/O (pure models only)

**Dependencies**: Requires T004 (tests must exist and be failing)

**Parallel**: No (must wait for T004 tests)

---

---

## Phase 3.3: Database Operations (TDD - Tests First)

### T006: [P] Write DatabaseOperator Contract Tests (MUST FAIL)

**Description**: Write contract tests for DatabaseOperator interface BEFORE implementation exists

**Actions**:
1. Create `pkg/schema/database_test.go` with contract tests that WILL FAIL:
   - `TestDatabaseOperator_Connect()`: Mock operator must implement Connect() method
   - `TestDatabaseOperator_CreateSchema()`: Verify CreateSchema() accepts DDL slice
   - `TestDatabaseOperator_TableExists()`: Assert TableExists() returns bool, error
   - `TestDatabaseOperator_EnableExtension()`: Verify EnableExtension() is callable
   - `TestDatabaseOperator_ExecuteDDLBatch()`: Test batch execution contract
2. Create mock implementation that compiles but panics:
   ```go
   type MockDatabaseOperator struct{}
   func (m *MockDatabaseOperator) Connect(ctx context.Context, dsn string) error {
       panic("not implemented")
   }
   // ... other methods
   ```
3. Run tests and verify they FAIL: `go test ./pkg/schema`

**File Paths**:
- `/Users/dimus/code/golang/gndb/pkg/schema/database_test.go`

**Success Criteria**:
- [ ] Contract tests verify all DatabaseOperator interface methods exist
- [ ] Tests FAIL (mock panics or assertions fail)
- [ ] Test expectations match contracts/DatabaseOperator.go
- [ ] No real implementation code written

**TDD GATE**: This task MUST be completed and MUST show failing tests before T007

**Parallel**: [P] - Test file only, independent of other work

---

### T007: Implement Database Operator with pgxpool

**Description**: Implement DatabaseOperator interface in `internal/io/database/` using pgxpool for connection pooling

**Actions**:
1. Create `internal/io/database/operator.go` with:
   - `PgxOperator` struct holding `*pgxpool.Pool`
   - Implement `Connect()` using pgxpool.New() with config from DatabaseConfig
   - Implement `CreateSchema()` executing DDL in transaction
   - Implement `TableExists()` querying information_schema
   - Implement `EnableExtension()` with CREATE EXTENSION IF NOT EXISTS
   - Implement `ExecuteDDLBatch()` with transaction support
   - Implement other DatabaseOperator methods per contract
2. Create `internal/io/database/operator_test.go` with:
   - Integration test using testcontainers-go for PostgreSQL
   - Test Connect() → CreateSchema() → TableExists() flow
   - Test EnableExtension() for pg_trgm
   - Test error handling (invalid DSN, malformed DDL)
3. Run tests from T006 and verify they now PASS

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/database/operator.go`
- `/Users/dimus/code/golang/gndb/internal/io/database/operator_test.go`

**Success Criteria**:
- [ ] PgxOperator uses pgxpool with MaxConnections, MinConnections from config
- [ ] All DatabaseOperator interface methods implemented
- [ ] Connection pool configured with lifetime and idle timeout settings
- [ ] Integration tests pass with real PostgreSQL via testcontainers
- [ ] Contract tests from T006 now PASS

**Dependencies**: Requires T006 (contract tests must exist and be failing)

**Parallel**: No (must wait for T006 tests)

---

### T008: [P] Write CLI Root Command Tests (MUST FAIL)

**Description**: Write tests for cobra root command structure BEFORE implementation

**Actions**:
1. Create `cmd/gndb/root_test.go` with tests that WILL FAIL:
   - `TestRootCommand_HasSubcommands()`: Assert root command has create, migrate, populate, restructure subcommands
   - `TestRootCommand_ConfigFlag()`: Verify --config flag exists and binds correctly
   - `TestRootCommand_Help()`: Assert help text includes usage examples
   - `TestRootCommand_VersionFlag()`: Verify --version flag outputs version info
2. Run tests and verify they FAIL: `go test ./cmd/gndb`

**File Paths**:
- `/Users/dimus/code/golang/gndb/cmd/gndb/root_test.go`

**Success Criteria**:
- [ ] Tests verify root command structure per plan.md
- [ ] Tests FAIL (no implementation yet)
- [ ] Test expectations match CLI design

**TDD GATE**: This task MUST be completed and MUST show failing tests before T009

**Parallel**: [P] - Test file only, independent

---

### T009: Implement CLI Root Command and Create Subcommand

**Description**: Implement cobra CLI with root command and `gndb create` subcommand

**Actions**:
1. Create `cmd/gndb/main.go`:
   - Initialize cobra root command
   - Add --config flag for config file path
   - Add --version flag
   - Set up subcommands
2. Create `cmd/gndb/root.go`:
   - Define `rootCmd` with cobra
   - Add persistent flags (--config, --log-level)
   - Load configuration using internal/io/config.Load()
3. Create `cmd/gndb/create.go`:
   - Define `createCmd` subcommand
   - Add --force flag (drop existing tables)
   - Handler logic:
     1. Load config
     2. Connect to database using DatabaseOperator
     3. Generate DDL from schema models
     4. Call CreateSchema() with force flag
     5. Enable pg_trgm extension
     6. Set schema version
     7. Output success message with table count
4. Run tests from T008 and verify they now PASS

**File Paths**:
- `/Users/dimus/code/golang/gndb/cmd/gndb/main.go`
- `/Users/dimus/code/golang/gndb/cmd/gndb/root.go`
- `/Users/dimus/code/golang/gndb/cmd/gndb/create.go`
- `/Users/dimus/code/golang/gndb/cmd/gndb/root_test.go` (update tests)

**Success Criteria**:
- [ ] `gndb --help` shows all subcommands
- [ ] `gndb create --help` shows usage and flags
- [ ] `gndb create` successfully creates schema in PostgreSQL
- [ ] --force flag drops and recreates tables
- [ ] All tests from T008 now PASS
- [ ] Binary builds: `go build -o gndb ./cmd/gndb`

**Dependencies**: Requires T007 (DatabaseOperator), T008 (CLI tests)

**Parallel**: No (depends on T007 and T008)

---

### T010: Integration Test - Quickstart Scenario 1 (Create Schema)

**Description**: Implement integration test for Quickstart Scenario 1 (create schema from empty database)

**Actions**:
1. Create `tests/integration/create_test.go`:
   - Use testcontainers-go to spin up PostgreSQL
   - Create gndb.yaml config pointing to test container
   - Execute `gndb create` CLI command
   - Verify all 11 tables exist (data_sources, name_strings, canonicals, etc.)
   - Verify pg_trgm extension enabled
   - Verify schema_versions table has entry
   - Test with --force flag (drop and recreate)
2. Create `testdata/sample.sfga` (empty for now, will be used in T011+)
3. Add integration test to justfile: `just test-integration`

**File Paths**:
- `/Users/dimus/code/golang/gndb/tests/integration/create_test.go`
- `/Users/dimus/code/golang/gndb/testdata/sample.sfga` (stub)

**Success Criteria**:
- [ ] Integration test spins up PostgreSQL container
- [ ] Test executes `gndb create` and verifies schema
- [ ] Test passes with fresh database
- [ ] Test passes with --force flag on existing schema
- [ ] Matches Quickstart Scenario 1 exactly
- [ ] `just test-integration` runs successfully

**Dependencies**: Requires T009 (gndb create command)

**Parallel**: No (depends on T009)

---

## Task Execution Order (T006-T010)

```
T006 [P] (DatabaseOperator contract tests - MUST FAIL)
  ↓
T007 (DatabaseOperator implementation - make tests pass)
  ↓
T008 [P] (CLI tests - MUST FAIL) ←┐
  ↓                                │
T009 (CLI root + create) ──────────┘
  ↓
T010 (Integration test - Scenario 1)
```

## Dependencies (T006-T010)
- T006 blocks T007 (TDD: contract tests before implementation)
- T008 can run parallel with T006 (independent test files)
- T007, T008 both block T009 (CLI needs database operator and tests)
- T009 blocks T010 (integration test needs working CLI)

---

## Progress Summary

**Completed** (T001-T005):
- ✅ Project structure initialized
- ✅ Configuration loading (pure + impure)
- ✅ Schema models with GORM AutoMigrate
- ✅ Connection pool configuration

**Next** (T006-T010):
- [ ] Database operator with pgxpool
- [ ] CLI root command and create subcommand
- [ ] Integration test for schema creation

**After T010**:
- Migration operations (Atlas integration)
- SFGA import (populate phase)
- Optimization (restructure phase)

---

## Validation Checklist

- [x] All tasks specify exact file paths
- [x] TDD workflow enforced (tests before implementation)
- [x] Pure/impure separation maintained
- [x] Parallel tasks [P] are truly independent
- [x] Tasks match plan.md and contracts/
- [x] Integration test matches quickstart.md

---

**Status**: Tasks T006-T010 defined. Ready for execution after T005 completion.
