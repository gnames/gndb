# Feature Specification: GNverifier Database Lifecycle Management

**Feature Branch**: `001-this-project-will`
**Created**: 2025-10-01
**Status**: Draft
**Input**: User description: "this project will enable a user to go through the GNverifier database lifecycle. It would start with empty database, create schema, migrate schema, create performance critical optimization of database as well as modification of data that will speed up name verification (reconciliation and reconciliation) as well as optimizing database data for vernacular names detection by languages and figuring out synonyms of input scientific names. After this project is complete it will allow users to setup GNverifier functionality locally, independent from main gnverifier service. It will be also used at the main service the same way. As a result users who have their own data-sources that are not included on the main site to create local GNverifier that is able to query these data-sources."

## User Scenarios & Testing

### Primary User Story
A researcher or organization has biodiversity nomenclature data sources that are not included in the main GNverifier service. They want to set up a local GNverifier instance to verify and reconcile scientific names against their custom data sources.

The user starts with an empty database and progresses through the complete lifecycle: creating the initial schema, applying migrations as the schema evolves, populating the database with their data sources, and running optimizations that enable fast name verification, vernacular name detection by language, and synonym resolution.

**Important**: The main GNverifier service uses this exact same tooling and process. There is no distinction between "main service" and "local setup" - both use identical database lifecycle management. The main service simply operates on the canonical public data sources, while local instances can use custom data sources.

### Acceptance Scenarios

1. **Given** an empty database, **When** the user runs the create schema command, **Then** the database contains all necessary tables, indexes, and constraints for GNverifier functionality

2. **Given** an existing database with an older schema version, **When** the user runs the migrate command, **Then** the database schema is updated to the latest version without data loss

3. **Given** a database with schema in place, **When** the user runs the populate command with their data sources, **Then** the database is populated with nomenclature data ready for verification queries

4. **Given** a populated database, **When** the user runs the restructure/optimize command, **Then** performance-critical optimizations are applied (indexes, materialized views, denormalization) that enable fast name verification

5. **Given** a fully set up local GNverifier database, **When** the user queries for scientific name verification, vernacular names by language, or synonyms, **Then** results are returned with performance comparable to the main GNverifier service

6. **Given** a local GNverifier instance, **When** the user wants to use it independently, **Then** no connection to the main GNverifier service is required

### Edge Cases
- What happens when the schema migration is interrupted mid-process?
  * System restarts the entire process from the beginning
- How does the system handle corrupt or invalid data during population?
  * System restarts from the beginning; database can be completely restored by repopulating from the same data files
- What happens if the database is corrupted by any unfortunate event?
  * System restarts the entire lifecycle from the beginning
- What happens if optimization commands are run multiple times?
  * Idempotent operations - can be safely re-run
- How does the system handle insufficient disk space during population?
  * Process fails; user must resolve space issues and restart from beginning
- What happens when attempting to create schema on a non-empty database?
  * System asks user if current data should be destroyed; if yes, nukes old data and rebuilds tables and indexes; if no, aborts operation
- How does the system handle different database encoding or locale settings?
  * Database schema includes necessary locale and collation data at table or column level as required for proper nomenclature handling

**Recovery Model**: After database setup is complete, the database is read-only for query operations. Any corruption or failed lifecycle phase is resolved by restarting the entire process from scratch using the original data files.

## Requirements

### Functional Requirements

- **FR-001**: System MUST support creating a complete GNverifier database schema from an empty database
- **FR-002**: System MUST support schema migrations to update existing databases to newer schema versions.
- **FR-003**: System MUST support populating the database with user-provided nomenclature data sources
- **FR-004**: System MUST apply performance optimizations that enable fast scientific name verification
- **FR-005**: System MUST optimize data for vernacular name detection organized by language
- **FR-006**: System MUST optimize data for synonym resolution of scientific names
- **FR-007**: System MUST enable local GNverifier instances to operate independently without connecting to the main service
- **FR-008**: System MUST support the same database lifecycle operations used by the main GNverifier service
- **FR-009**: System MUST allow users with custom data sources not included in the main service to create functional local instances
- **FR-010**: System MUST organize database lifecycle phases as separate, independent operations (create, migrate, populate, restructure)
- **FR-011**: System MUST validate database state before executing each lifecycle phase [NEEDS CLARIFICATION: specific validation rules for each phase to be determined during planning]
- **FR-012**: System MUST provide feedback on progress for long-running operations [NEEDS CLARIFICATION: progress indicators, logging level, output format to be determined during planning]
- **FR-013**: System MUST handle errors gracefully using transactions where possible; recovery is achieved by restarting the whole process or individual phases (e.g., reimporting a specific data source removes corrupted data and starts fresh)
- **FR-014**: System MUST support configuration via YAML file and CLI flags per constitutional requirements
- **FR-015**: System MUST validate input data sources before population; all data source files MUST be normalized to SFGA format (https://github.com/sfborg/sfga)
- **FR-016**: System MUST ensure all data sources in the initial ingest use the same version of SFGA format
- **FR-017**: System MUST support subsequent data updates that may use different SFGA format versions by using the compatible version of sflib library (https://github.com/sfborg/sflib) for import

### Performance Requirements

- **PR-001**: System MUST support reconciliation throughput of at least 1000 names per second (most performant use case, without synonym detection or vernacular name search)
- **PR-002**: Database MUST be optimized to support vernacular name queries (performance target lower priority than pure name reconciliation)
- **PR-003**: Database MUST be optimized to support synonym resolution queries (performance target lower priority than pure name reconciliation)
- **PR-004**: Database population MUST handle at least 100 million scientific name-strings with 200 million name-string occurrences across data sources, and 10 million vernacular name-strings with 20 million occurrences (main service target; local users with slower hardware, less memory, and smaller datasets may use significantly smaller volumes)

### Key Entities

- **Database Schema**: Represents the structure of tables, indexes, constraints, and relationships required for GNverifier functionality
- **Schema Version**: Tracks the current version of the database schema to support migrations
- **Nomenclature Data Source**: External data containing scientific names, vernacular names, synonyms, and taxonomic information
- **Scientific Name**: A formal biological name to be verified, reconciled, or resolved
- **Vernacular Name**: Common names in various languages associated with scientific names
- **Synonym**: Alternative, currently not accepted scientific names that refer to the same taxonomic entity
- **Optimization Artifact**: Performance-enhancing database structures (indexes, materialized views, denormalized tables) created during restructure phase
- **Migration**: A versioned change to the database schema

### Dependencies and Assumptions

**Dependencies**:
- PostgreSQL database system for GNverifier data storage
- SQLite for SFGA format data source files
- Access to user-provided nomenclature data sources in SFGA format
- sflib library (https://github.com/sfborg/sflib) for data import
- Configuration file (gndb.yaml) or equivalent configuration source

**Assumptions**:
- Users have appropriate database permissions (CREATE, ALTER, INSERT, etc.)
- Sufficient disk space for database and optimization artifacts
- Data sources are normalized to SFGA format before ingestion
- Users understand basic database lifecycle concepts
- For initial ingest, all data sources use the same SFGA format version
- Name detection, resolution, and reconciliation logic are out of scope (handled by other components); this project focuses on database lifecycle and optimization
- Validation rules for lifecycle phases and progress feedback formats will be defined during planning and implementation based on discovered requirements

## Review & Acceptance Checklist

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [ ] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous (except marked items)
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Execution Status

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [ ] Review checklist passed (pending clarifications)
