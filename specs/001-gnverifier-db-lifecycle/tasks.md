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

## Next Steps

After completing T001-T005, we'll have:
- ✅ Project structure initialized
- ✅ Configuration loading working (pure and impure separation)
- ✅ Schema models with DDL generation (TDD workflow established)

**Then** we can add the next batch of tasks:
- Database operator interface and implementation
- Schema creation CLI command (`gndb create`)
- Integration tests for schema creation
- SFGA reader interface and implementation
- Population logic

---

## Task Execution Order

```
T001 (setup)
  ↓
T002 (config types) ←┐
  ↓                  │ [P] can run in parallel with T004
T003 (config loader) │
                     │
T004 [P] (schema tests - MUST FAIL)
  ↓
T005 (schema models - make tests pass)
```

## Dependencies
- T002 blocks T003 (loader needs config types)
- T004 blocks T005 (TDD: tests before implementation)
- T001 must complete first (project foundation)

---

## Validation Checklist

- [x] All tasks specify exact file paths
- [x] TDD workflow enforced (T004 tests before T005 implementation)
- [x] Pure/impure separation maintained (pkg/ vs internal/io/)
- [x] Parallel tasks [P] are truly independent
- [x] Tasks match plan.md architecture
- [x] Tasks reference data-model.md specifications

---

**Status**: First 5 tasks defined. Ready for execution in order: T001 → T002 → T003, T004 [parallel] → T005
