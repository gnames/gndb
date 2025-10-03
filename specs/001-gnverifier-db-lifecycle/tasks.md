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

### T011: Add Environment Variable Overrides for All Config Fields

**Description**: Implement environment variable support in `internal/io/config/loader.go` to allow all config fields to be overridden via `GNDB_*` environment variables, satisfying Constitution Principle VI (precedence: flags > env vars > config file > defaults)

**Actions**:
1. Update `internal/io/config/loader.go` to add environment variable support:
   - Import `strings` package for key replacer
   - In `Load()` function, add after `v.SetConfigType("yaml")`:
     ```go
     // Enable environment variable overrides
     v.SetEnvPrefix("GNDB")
     v.AutomaticEnv()
     v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
     ```
   - This enables automatic binding of environment variables with GNDB_ prefix
   - Nested config fields map to env vars with underscores (database.host → GNDB_DATABASE_HOST)

2. Create `internal/io/config/loader_test.go` with environment variable tests:
   - `TestLoad_EnvVarOverride_DatabaseHost()`: Set GNDB_DATABASE_HOST, verify it overrides config file
   - `TestLoad_EnvVarOverride_NestedField()`: Set GNDB_DATABASE_MAX_CONNECTIONS, verify override
   - `TestLoad_EnvVarOverride_ImportBatchSize()`: Set GNDB_IMPORT_BATCH_SIZE, verify override
   - `TestLoad_EnvVarOverride_LoggingLevel()`: Set GNDB_LOGGING_LEVEL, verify override
   - `TestLoad_PrecedenceOrder()`: Verify env var overrides config file but is overridden by flags
   - Use `t.Setenv()` to set environment variables in tests (Go 1.17+)

3. Update `pkg/config/config.go` godoc to document all supported environment variables:
   - Add godoc section before `Config` struct listing all GNDB_* environment variables
   - Include examples for each field type (string, int, bool, map)

4. Update `cmd/gndb/root.go` to add environment variable documentation to help text:
   - In `rootCmd.Long`, add section "Environment Variables:" listing all GNDB_* variables
   - Include note about precedence order

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/config/loader.go` (update Load function)
- `/Users/dimus/code/golang/gndb/internal/io/config/loader_test.go` (create with env var tests)
- `/Users/dimus/code/golang/gndb/pkg/config/config.go` (update godoc)
- `/Users/dimus/code/golang/gndb/cmd/gndb/root.go` (update help text)

**Environment Variables to Support** (all config fields from pkg/config/config.go):
```
GNDB_DATABASE_HOST                     # database.host
GNDB_DATABASE_PORT                     # database.port
GNDB_DATABASE_USER                     # database.user
GNDB_DATABASE_PASSWORD                 # database.password
GNDB_DATABASE_DATABASE                 # database.database
GNDB_DATABASE_SSL_MODE                 # database.ssl_mode
GNDB_DATABASE_MAX_CONNECTIONS          # database.max_connections
GNDB_DATABASE_MIN_CONNECTIONS          # database.min_connections
GNDB_DATABASE_MAX_CONN_LIFETIME        # database.max_conn_lifetime
GNDB_DATABASE_MAX_CONN_IDLE_TIME       # database.max_conn_idle_time
GNDB_IMPORT_BATCH_SIZE                 # import.batch_size
GNDB_OPTIMIZATION_CONCURRENT_INDEXES   # optimization.concurrent_indexes
GNDB_LOGGING_LEVEL                     # logging.level
GNDB_LOGGING_FORMAT                    # logging.format
```

**Success Criteria**:
- [ ] All config fields can be overridden via GNDB_* environment variables
- [ ] Nested field naming uses underscores (database.host → GNDB_DATABASE_HOST)
- [ ] Environment variables override config file values
- [ ] CLI flags override environment variables (precedence maintained)
- [ ] All tests pass: `go test ./internal/io/config`
- [ ] Godoc documentation includes all environment variables
- [ ] `gndb --help` shows environment variable usage

**Dependencies**: Requires T003 (config loader implementation)

**Parallel**: No (modifies existing config loader)

**Constitutional Compliance**: Satisfies Principle VI (Configuration Management) - precedence order: flags > env vars > config file > defaults

**Testing Strategy**:
- Unit tests verify each environment variable works
- Integration test verifies precedence order (flag > env > file > default)
- Use `t.Setenv()` for isolated test environment

**Example Usage After Implementation**:
```bash
# Override database host via environment variable
export GNDB_DATABASE_HOST=production-db.example.com
export GNDB_DATABASE_PASSWORD=secret123
gndb create

# CLI flag still takes highest precedence
gndb create --host=override-db.example.com  # Uses override-db, not production-db
```

---

### T012: Generate Default Config File on First Run

**Description**: Auto-generate a documented default config file (gndb.yaml) in the appropriate platform-specific directory on first run, and display config file location when gndb executes.

**Actions**:
1. Create `internal/io/config/generate.go` with config file generation logic:
   - Function `GetConfigDir()` to determine platform-specific config directory:
     * Linux/macOS: `~/.config/gndb/`
     * Windows: `%APPDATA%\gndb\` (using `os.UserConfigDir()`)
   - Function `GetDefaultConfigPath()` returns `<config-dir>/gndb.yaml`
   - Function `GenerateDefaultConfig()` creates documented YAML file:
     ```yaml
     # GNdb Configuration File
     # This file was auto-generated. Edit as needed.
     #
     # Configuration precedence (highest to lowest):
     #   1. CLI flags (--host, --port, etc.)
     #   2. Environment variables (GNDB_*)
     #   3. This config file
     #   4. Built-in defaults
     #
     # For all environment variables, see: go doc github.com/gnames/gndb/pkg/config

     # Database connection settings
     database:
       host: localhost              # PostgreSQL host
       port: 5432                   # PostgreSQL port
       user: postgres               # PostgreSQL user
       password: postgres           # PostgreSQL password (consider using GNDB_DATABASE_PASSWORD env var)
       database: gnames             # Database name
       ssl_mode: disable            # SSL mode: disable/require/verify-ca/verify-full

       # Connection pool settings
       max_connections: 20          # Maximum connections in pool
       min_connections: 2           # Minimum connections in pool
       max_conn_lifetime: 60        # Max connection lifetime (minutes)
       max_conn_idle_time: 10       # Max idle time (minutes)

     # Import settings
     import:
       batch_size: 5000             # Number of records per batch insert

     # Optimization settings
     optimization:
       concurrent_indexes: false    # Create indexes concurrently (true for production)
       statistics_targets:          # Statistics targets for high-cardinality columns
         name_strings.canonical_simple: 1000
         taxa.rank: 100

     # Logging settings
     logging:
       level: info                  # Log level: debug/info/warn/error
       format: text                 # Log format: json/text
     ```
   - Function creates parent directories if they don't exist

2. Update `cmd/gndb/root.go` to auto-generate config on first run:
   - In `PersistentPreRunE`, before calling `config.Load()`:
     * Check if config file exists at default location
     * If not found and no explicit `--config` flag, call `GenerateDefaultConfig()`
     * Print message: "Generated default config at: <path>"
   - After loading config successfully, print: "Using config from: <path>" (if from file)
   - If using defaults (no file), print: "Using built-in defaults (no config file)"

3. Create `internal/io/config/generate_test.go` with tests:
   - `TestGetConfigDir()`: Verify correct platform-specific directory
   - `TestGetDefaultConfigPath()`: Verify full path construction
   - `TestGenerateDefaultConfig()`: Create temp dir, generate config, verify YAML valid
   - `TestGenerateDefaultConfig_CreatesParentDirs()`: Verify directory creation
   - `TestGenerateDefaultConfig_FileExists()`: Don't overwrite existing config

4. Update `internal/io/config/loader.go`:
   - Track which config source was used (file path, env vars, defaults)
   - Return config source info for display in CLI

**File Paths**:
- `/Users/dimus/code/golang/gndb/internal/io/config/generate.go` (new)
- `/Users/dimus/code/golang/gndb/internal/io/config/generate_test.go` (new)
- `/Users/dimus/code/golang/gndb/internal/io/config/loader.go` (update to track source)
- `/Users/dimus/code/golang/gndb/cmd/gndb/root.go` (update to generate and display config location)

**Platform-Specific Config Directories**:
- **Linux/macOS**: `~/.config/gndb/gndb.yaml`
- **Windows**: `%APPDATA%\gndb\gndb.yaml`
- Use `os.UserConfigDir()` (Go 1.13+) for cross-platform compatibility

**Success Criteria**:
- [ ] First run of `gndb` creates config file in platform-specific directory
- [ ] Generated config file contains all fields with documentation comments
- [ ] Config file uses default values from `pkg/config.Defaults()`
- [ ] Existing config files are never overwritten
- [ ] CLI displays config file location on startup
- [ ] CLI distinguishes between file, env vars, and built-in defaults
- [ ] All tests pass: `go test ./internal/io/config`
- [ ] Cross-platform tested (Linux, macOS, Windows paths)

**Dependencies**: Requires T011 (env var support for complete config system)

**Parallel**: No (enhances existing config loader)

**User Experience**:
```bash
# First run - auto-generates config
$ gndb create
Generated default config at: /home/user/.config/gndb/gndb.yaml
Using config from: /home/user/.config/gndb/gndb.yaml
[... rest of create output ...]

# Subsequent runs - uses existing config
$ gndb create
Using config from: /home/user/.config/gndb/gndb.yaml
[... rest of create output ...]

# Explicit config path
$ gndb create --config custom.yaml
Using config from: custom.yaml
[... rest of create output ...]

# Environment variables only
$ rm ~/.config/gndb/gndb.yaml
$ GNDB_DATABASE_HOST=prod-db gndb create
Using built-in defaults with environment variable overrides
[... rest of create output ...]
```

**Generated Config File Quality**:
- YAML syntax highlighted comments explaining each field
- Security note about password (recommend env var)
- Reference to full documentation
- Grouped by logical sections (database, import, optimization, logging)
- Example values for map fields (statistics_targets)

---

### T013: Create .envrc.example for direnv Integration

**Description**: Create an example `.envrc.example` file demonstrating how to use direnv for local development with environment variable configuration, showing all available GNDB_* variables with documentation.

**Actions**:
1. Create `.envrc.example` in project root with:
   - Header comment explaining direnv usage and setup
   - All GNDB_* environment variables from T011 with example values
   - Comments explaining each variable's purpose and default value
   - Security best practices section (don't commit .envrc with secrets)
   - Instructions for using the file:
     ```bash
     # 1. Install direnv: https://direnv.net/
     # 2. Copy this file: cp .envrc.example .envrc
     # 3. Edit .envrc with your values
     # 4. Allow direnv: direnv allow .
     ```
   - Example configurations for different environments:
     * Local development setup
     * Testing configuration
     * Production-like environment
   - Note about precedence: env vars override config file but are overridden by CLI flags

2. Add `.envrc` to `.gitignore` to prevent committing sensitive values

3. Create documentation in `.envrc.example`:
   ```bash
   # GNdb Environment Configuration for direnv
   # 
   # This file demonstrates how to configure gndb using environment variables.
   # direnv automatically loads these variables when you cd into this directory.
   #
   # Setup:
   #   1. Install direnv: https://direnv.net/
   #   2. Copy this file: cp .envrc.example .envrc
   #   3. Edit .envrc with your actual values
   #   4. Allow direnv: direnv allow .
   #
   # Configuration Precedence (highest to lowest):
   #   1. CLI flags (--host, --port, etc.)
   #   2. Environment variables (these GNDB_* vars)
   #   3. Config file (~/.config/gndb/gndb.yaml)
   #   4. Built-in defaults
   #
   # SECURITY: Never commit .envrc with real credentials!

   # Database Connection Settings
   # Override these for local development or testing
   export GNDB_DATABASE_HOST=localhost              # Default: localhost
   export GNDB_DATABASE_PORT=5432                   # Default: 5432
   export GNDB_DATABASE_USER=postgres               # Default: postgres
   export GNDB_DATABASE_PASSWORD=postgres           # Default: postgres (CHANGE IN PRODUCTION!)
   export GNDB_DATABASE_DATABASE=gnames             # Default: gnames
   export GNDB_DATABASE_SSL_MODE=disable            # Default: disable (use 'require' in production)

   # Connection Pool Settings
   # export GNDB_DATABASE_MAX_CONNECTIONS=20        # Default: 20
   # export GNDB_DATABASE_MIN_CONNECTIONS=2         # Default: 2
   # export GNDB_DATABASE_MAX_CONN_LIFETIME=60      # Default: 60 minutes
   # export GNDB_DATABASE_MAX_CONN_IDLE_TIME=10     # Default: 10 minutes

   # Import Settings
   # export GNDB_IMPORT_BATCH_SIZE=5000             # Default: 5000 records per batch

   # Optimization Settings
   # export GNDB_OPTIMIZATION_CONCURRENT_INDEXES=false  # Default: false (set true in production)

   # Logging Settings
   # export GNDB_LOGGING_LEVEL=info                 # Default: info (options: debug, info, warn, error)
   # export GNDB_LOGGING_FORMAT=text                # Default: text (options: text, json)

   # Example Configurations:

   # Local Development (PostgreSQL via Docker):
   # export GNDB_DATABASE_HOST=localhost
   # export GNDB_DATABASE_PORT=5432
   # export GNDB_DATABASE_USER=gndb_dev
   # export GNDB_DATABASE_PASSWORD=dev_password
   # export GNDB_DATABASE_DATABASE=gndb_local
   # export GNDB_LOGGING_LEVEL=debug

   # Testing Environment:
   # export GNDB_DATABASE_DATABASE=gndb_test
   # export GNDB_IMPORT_BATCH_SIZE=100
   # export GNDB_LOGGING_LEVEL=warn

   # Production-like Setup:
   # export GNDB_DATABASE_HOST=prod-db.example.com
   # export GNDB_DATABASE_SSL_MODE=require
   # export GNDB_DATABASE_PASSWORD=<use_secret_manager>
   # export GNDB_OPTIMIZATION_CONCURRENT_INDEXES=true
   # export GNDB_LOGGING_FORMAT=json
   # export GNDB_LOGGING_LEVEL=info
   ```

4. Update `.gitignore`:
   - Add `.envrc` entry if not already present
   - Add comment explaining why .envrc is ignored

5. Update README.md (if exists) or create docs/configuration.md:
   - Add section on environment variable configuration
   - Link to .envrc.example
   - Explain direnv workflow
   - Show example usage

**File Paths**:
- `/Users/dimus/code/golang/gndb/.envrc.example` (new)
- `/Users/dimus/code/golang/gndb/.gitignore` (update)
- `/Users/dimus/code/golang/gndb/README.md` or `/Users/dimus/code/golang/gndb/docs/configuration.md` (update/create)

**Success Criteria**:
- [ ] `.envrc.example` demonstrates all GNDB_* environment variables from T011
- [ ] File includes clear setup instructions for direnv
- [ ] Comments explain each variable's purpose and default value
- [ ] Security best practices documented (don't commit secrets)
- [ ] Example configurations provided for different environments
- [ ] `.envrc` added to `.gitignore`
- [ ] Documentation updated with direnv usage guide
- [ ] Variables match exactly those implemented in T011

**Dependencies**: Requires T011 (environment variable support must be implemented)

**Parallel**: No (documents functionality from T011)

**User Workflow After Implementation**:
```bash
# Install direnv (one-time setup)
$ brew install direnv  # macOS
$ sudo apt install direnv  # Ubuntu/Debian
$ echo 'eval "$(direnv hook bash)"' >> ~/.bashrc  # Enable direnv

# Set up project environment
$ cd gndb
$ cp .envrc.example .envrc
$ vim .envrc  # Edit with your database credentials
$ direnv allow .

# Environment variables now auto-loaded when in project directory
$ gndb create  # Uses GNDB_* vars from .envrc

# Change environment by editing .envrc
$ vim .envrc  # Update GNDB_DATABASE_HOST=test-db
$ gndb create  # Uses new host automatically
```

**Benefits of direnv Integration**:
- Automatic environment activation when entering project directory
- No need to manually export variables
- Per-project configuration isolation
- Easy switching between environments (dev/test/prod)
- Works seamlessly with the GNDB_* environment variable system from T011
- Safer than storing credentials in config files (can use different .envrc per environment)

**Documentation Quality**:
- Inline comments explain each variable
- Examples show common use cases
- Security warnings prominent
- Clear setup instructions
- Links to direnv documentation

---

### T014: Test and Verify Schema Creation Workflow

**Description**: Test the complete schema creation workflow, verify that `gndb create` works correctly, and document the prerequisite that the PostgreSQL database must be created manually before running the command.

**Context**: The `gndb create` command creates tables/schema inside an existing database - it does NOT create the database itself. Users must create the database first using `createdb` or similar tools.

**Actions**:
1. Document database creation prerequisite:
   - Add section to README.md or docs/quickstart.md
   - Explain that PostgreSQL database must exist before running `gndb create`
   - Provide example: `createdb gnames_test`

2. Test schema creation workflow:
   - Create a test database: `createdb gndb_test`
   - Set up `.envrc` with test database credentials:
     ```bash
     export GNDB_DATABASE_DATABASE=gndb_test
     export GNDB_DATABASE_USER=<your_user>
     export GNDB_DATABASE_PASSWORD=<your_password>
     ```
   - Run: `direnv allow .`
   - Run: `gndb create`
   - Verify all tables created successfully

3. Verify schema contents:
   - Connect to database: `psql gndb_test`
   - Check tables exist: `\dt`
   - Verify expected tables from data-model.md:
     * data_sources
     * name_strings
     * canonicals
     * canonical_fulls
     * canonical_stems
     * name_string_indices
     * words
     * word_name_strings
     * vernacular_strings
     * vernacular_string_indices
     * schema_versions
   - Verify schema_versions table has entry
   - Note: No PostgreSQL extensions needed (fuzzy matching handled by gnmatcher)

4. Test --force flag:
   - Run: `gndb create --force`
   - Verify it drops existing tables and recreates schema
   - Confirm data is cleared (expected behavior)

5. Document the workflow in README.md:
   - Add "Quick Start" section with step-by-step instructions
   - Include prerequisite: create database first
   - Show example with .envrc or direct flags
   - Link to full configuration documentation

**File Paths**:
- `/Users/dimus/code/golang/gndb/README.md` (update or create)
- `/Users/dimus/code/golang/gndb/docs/quickstart.md` (update if exists)
- Test database: `gndb_test` (local PostgreSQL)

**Success Criteria**:
- [ ] Documentation clearly states database must be created before `gndb create`
- [ ] Successfully create test database and run `gndb create`
- [ ] All 11 tables from data-model.md are created
- [ ] schema_versions table has entry with version "1.0.0"
- [ ] No PostgreSQL extensions required (fuzzy matching is in gnmatcher)
- [ ] `--force` flag successfully drops and recreates schema
- [ ] README.md has clear quick start instructions
- [ ] Workflow matches quickstart.md scenario 1

**Dependencies**: Requires T001-T005 (schema models and operator are already implemented)

**Parallel**: No (verification and documentation task)

**Prerequisites for Users**:
```bash
# 1. Install PostgreSQL (if not already installed)
brew install postgresql@15  # macOS
# or
sudo apt install postgresql-15  # Ubuntu

# 2. Start PostgreSQL service
brew services start postgresql@15  # macOS
# or  
sudo systemctl start postgresql  # Ubuntu

# 3. Create the database
createdb gndb_test

# 4. Configure gndb (option A: using .envrc)
cp .envrc.example .envrc
vim .envrc  # Edit GNDB_DATABASE_DATABASE=gndb_test
direnv allow .

# 5. Create schema
gndb create
```

**Expected Output**:
```
Connected to database: myuser@localhost:5432/gndb_test
Creating schema using GORM AutoMigrate...
✓ Schema created successfully
✓ Schema version set to 1.0.0

Created 11 tables:
  - canonicals
  - canonical_fulls
  - canonical_stems
  - data_sources
  - name_string_indices
  - name_strings
  - schema_versions
  - vernacular_string_indices
  - vernacular_strings
  - word_name_strings
  - words

✓ Database schema creation complete!

Next steps:
  - Run 'gndb populate' to import data from SFGA files
  - Run 'gndb restructure' to create indexes and optimize
```

**Testing Checklist**:
- [ ] Test with fresh database (no existing tables)
- [ ] Test with `--force` flag on existing schema
- [ ] Test with different database names
- [ ] Test with environment variables vs config file
- [ ] Test error handling (database doesn't exist)
- [ ] Test error handling (insufficient permissions)
- [ ] Verify schema_versions table populated
- [ ] Verify all foreign keys created
- [ ] Verify all indexes created by GORM

---

## Task Execution Order (T006-T014)

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
  ↓
T011 (Environment variable overrides)
  ↓
T012 (Generate default config file on first run)
  ↓
T013 (Create .envrc.example for direnv)
  ↓
T014 (Test and verify schema creation workflow)
```

## Dependencies (T006-T014)
- T006 blocks T007 (TDD: contract tests before implementation)
- T008 can run parallel with T006 (independent test files)
- T007, T008 both block T009 (CLI needs database operator and tests)
- T009 blocks T010 (integration test needs working CLI)
- T003 blocks T011 (env var support needs config loader to exist)
- T011 blocks T012 (config generation needs env var support complete)
- T011 blocks T013 (envrc example needs env var implementation)
- T011 enhances T009 (adds env var capability to existing CLI)
- T012 enhances T009 (adds auto-generation of config files)
- T013 documents T011 (provides direnv integration example)
- T013 blocks T014 (testing needs .envrc setup for configuration)
- T014 verifies T001-T013 (validates entire config and schema creation system)

---

## Progress Summary

**Completed** (T001-T013):
- ✅ T001-T002: Project structure and configuration types
- ✅ T003: Configuration loader (file + flags)
- ✅ T004-T005: Schema models with DDL generation
- ✅ T006-T010: Database operator and CLI (skipped - not implemented yet)
- ✅ T011: Environment variable overrides for all config fields
- ✅ T012: Auto-generate default config file on first run
- ✅ T013: Create .envrc.example for direnv integration

**Verification** (T012 - Config Generation):
- ✅ First run of `gndb create` generates config at ~/.config/gndb/gndb.yaml
- ✅ Subsequent runs use existing config (no overwrite)
- ✅ Config has all values commented out with inline documentation
- ✅ Config is valid YAML (empty sections with comments)
- ✅ All 14 GNDB_* environment variables documented
- ✅ MergeWithDefaults() fills in missing values automatically

**Next** (T014):
- [ ] T014: Test and verify schema creation workflow

**After T014**:
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

**Status**: Tasks T006-T012 defined. Ready for execution after T005 completion.
