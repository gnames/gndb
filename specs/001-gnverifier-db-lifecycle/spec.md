# Feature Specification: GNverifier Database Lifecycle Management

**Branch**: `001-gnverifier-db-lifecycle` | **Created**: 2025-10-01 | **Status**: Active

## User Scenarios & Testing

### Primary User Story
Researchers/organizations set up local GNverifier instances with custom biodiversity data sources. The CLI manages complete database lifecycle: create schema, migrate, populate from SFGA files, optimize for fast name verification/vernacular lookup/synonym resolution.

Both main service and local instances use identical tooling - only data sources differ.

### Acceptance Scenarios

1. `gndb create` on empty database → creates all tables, indexes, constraints
2. `gndb migrate` on existing database → updates schema to latest version, no data loss
3. `gndb populate <sfga-files>` → imports names/taxon data, ready for queries
4. `gndb optimize` → applies indexes, materialized views for fast name verification
5. Queries (name verification, vernacular lookup, synonyms) → performance matches main service
6. Local instance operates independently, no connection to main service required

### Edge Cases & Recovery
- **Interrupted operations**: Restart phase from beginning
- **Corrupt data**: Re-populate from source SFGA files
- **Optimize re-run**: Always rebuilds from scratch (drops existing, recreates with latest algorithms)
- **Insufficient disk space**: Fails with error; user resolves, restarts phase
- **Non-empty database on create**: Prompts user confirmation to drop existing data
- **Encoding/locale**: Schema includes necessary collation at table/column level

**Recovery Model**: Database is read-only post-setup (convention). Corruption → restart lifecycle from SFGA files.

## Clarifications (2025-10-08)

- **DatabaseOperator**: Provides basic operations (Connect, Close, TableExists, DropAllTables) + exposes pgxpool; high-level components use pool for specialized SQL
- **Schema migrations**: GORM AutoMigrate handles automatically
- **User interface**: CLI commands (create/migrate/populate/optimize); library interfaces minimal
- **Optimize behavior**: Always drop/recreate to apply algorithm improvements
- **Read-only enforcement**: Convention only, not enforced

## Requirements

### Functional Requirements

**Lifecycle Commands**:
- **FR-001**: `gndb create` - Create schema via GORM AutoMigrate
- **FR-002**: `gndb migrate` - Update schema via GORM AutoMigrate
- **FR-003**: `gndb populate <sfga-files>` - Import names/taxon data from SFGA sources
- **FR-004**: `gndb optimize` - Apply indexes/materialized views; always rebuilds from scratch

**Optimization Targets**:
- **FR-005**: Fast scientific name verification (1000 names/sec)
- **FR-006**: Vernacular name detection by language
- **FR-007**: Synonym resolution

**Deployment**:
- **FR-008**: Local instances operate independently (no main service connection)
- **FR-009**: Identical lifecycle for main service and local instances

**Data & Config**:
- **FR-010**: SFGA format required (github.com/sfborg/sfga); sflib for import (github.com/sfborg/sflib)
- **FR-011**: Initial ingest: single SFGA version; updates: version-compatible via sflib
- **FR-012**: YAML config (gndb.yaml) + CLI flags; precedence: flags > env > file > defaults

**Operations**:
- **FR-013**: Phase validation before execution (rules in research.md)
- **FR-014**: Human-friendly progress to stdout; structured logs (JSON) for monitoring/debugging
- **FR-015**: Graceful error handling; recovery via phase restart
- **FR-016**: Read-only convention post-setup (not enforced)

### Performance Requirements

- **PR-001**: 1000 names/sec reconciliation (primary use case)
- **PR-002**: Vernacular/synonym queries optimized (lower priority than reconciliation)
- **PR-003**: Scale: 100M name-strings, 200M occurrences, 10M vernacular, 20M occurrences (main service; local instances may be smaller)

### Key Entities

**Architecture**:
- **DatabaseOperator**: Basic ops (Connect, Close, TableExists, DropAllTables) + exposes pgxpool
- **SchemaManager**: Schema create/migrate via GORM AutoMigrate
- **Populator**: Import names/taxon data from SFGA using sflib, bulk insert via pgx CopyFrom
- **Optimizer**: Create indexes/materialized views; always rebuilds from scratch

**Data**:
- **Scientific Name**: Formal biological name for verification/reconciliation
- **Vernacular Name**: Common names by language
- **Synonym**: Alternative names for same taxon
- **SFGA Data Source**: SQLite files with names/taxon data (github.com/sfborg/sfga)

### Dependencies & Assumptions

**Dependencies**: PostgreSQL, pgxpool, GORM, sflib, SFGA files, Cobra (CLI), Viper (config)

**Assumptions**:
- User has DB permissions (CREATE, ALTER, INSERT)
- Sufficient disk space
- SFGA-normalized data sources
- Initial ingest: single SFGA version; updates: version-compatible
- Name verification logic out of scope (other components)

