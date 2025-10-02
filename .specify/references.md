# GNdb References

This document contains important reference links and locations for the GNdb project.

## Local Code References

**Note**: Paths use `${HOME}` for cross-platform compatibility (works on Linux/macOS/WSL)

### Schema Models (Source of Truth)
- **gnidump models**: `${HOME}/code/golang/gnidump/pkg/ent/model/model.go`
  - Our schema MUST match this for gnverifier compatibility
  - Uses UUID v5 for deterministic IDs
  - Defines all database tables and their structure

### Related GNames Projects
- **gnidump**: `${HOME}/code/golang/gnidump/`
  - Database population and schema management
  - Reference implementation for data models

- **gnparser**: `${HOME}/code/golang/gnparser/`
  - Name parsing library
  - Generates canonical forms and metadata

- **gnverifier**: `${HOME}/code/golang/gnverifier/`
  - Name verification cli and Web UI
  - Consumer of the database we're building

- **gnames**: `${HOME}/code/golang/gnames/`
  - Name verification service that gnverifier has to use remotly (currently)

- **gmatcher**: `${HOME}/code/golang/gnmatcher/`
  - Name matching service that gnames has to use remotly (currently)

## External References

### SFGA Format
- **SFGA Schema**: `${HOME}/code/sql/sfga/schema.sql`
  - URL: https://github.com/sfborg/sfga
  - Standard Format for Global names Archive
  - SQLite-based archive format for nomenclature data
  - Our import source format

- **sflib Library**:  `${HOME}/code/golang/sflib/`
  - URL: https://github.com/sfborg/sflib
  - Official SFGA library for Go
  - Used for reading SFGA files
  - Version compatibility checking

- **sf**:  `${HOME}/code/golang/sflib/`
  - CLI consumer of sflib library.

- **to-gn**: `${HOME}/code/golang/to-gn/`
  - CLI importer of SFGA files to gnames database
  - It is a good reference to see importing of SFGA to gnames database.
  - gnames database is the default name for gnverifier database.

### Database & Tools

- **PostgreSQL Documentation**: https://www.postgresql.org/docs/
  - pg_trgm extension for fuzzy text matching
  - UUID support and generation
  - Performance tuning guides

- **Atlas Migration Tool**: https://atlasgo.io/
  - Go-native database migration tool
  - Versioned migrations with integrity checking
  - Used for schema evolution

### Go Libraries

- **pgx**: https://github.com/jackc/pgx
  - PostgreSQL driver for Go
  - Native protocol, better performance than database/sql
  - Connection pooling

- **cobra**: https://github.com/spf13/cobra
  - CLI framework
  - Subcommand architecture for create/migrate/populate/restructure

- **viper**: https://github.com/spf13/viper
  - Configuration management
  - YAML files, flags, environment variables

### Project Documentation

- **GNdb Specification**: `specs/001-gnverifier-db-lifecycle/spec.md`
  - Feature requirements and user scenarios
  - Acceptance criteria

- **Implementation Plan**: `specs/001-gnverifier-db-lifecycle/plan.md`
  - Technical approach and architecture
  - Constitution compliance

- **Research Notes**: `specs/001-gnverifier-db-lifecycle/research.md`
  - Technical decisions and rationale
  - Performance benchmarks and targets

- **Data Model**: `specs/001-gnverifier-db-lifecycle/data-model.md`
  - Database schema documentation
  - Entity relationships

- **Quickstart Guide**: `specs/001-gnverifier-db-lifecycle/quickstart.md`
  - End-to-end integration tests
  - Setup instructions

- **Tasks**: `specs/001-gnverifier-db-lifecycle/tasks.md`
  - Implementation task breakdown
  - Current: First 5 foundational tasks complete

## GNames Ecosystem

- **GNames Website**: https://globalnames.org/
  - Global Names Architecture project
  - Biodiversity nomenclature tools

## Key Concepts

### UUID v5 Generation
- Deterministic UUIDs based on content
- Namespace: DNS "globalnames.org"
- Same name-string always generates same UUID
- Enables distributed data consistency

### Data Source IDs
- Hard-coded SMALLINT values
- Historical compatibility with older resolver versions
- Not auto-incrementing

### Canonical Forms
- **Simple**: Uninomial/binomial without authorship (e.g., "Homo sapiens")
- **Full**: Complete with infraspecific ranks (e.g., "Homo sapiens sapiens")
- **Stemmed**: Normalized for fuzzy matching (e.g., "hom sapien")

### Nomenclatural Codes
- 0: No info
- 1: ICZN (Zoological)
- 2: ICN (Botanical)
- 3: ICNP (Prokaryotes)
- 4: ICTV (Viruses)

## Notes

- Always check gnidump models before making schema changes
- UUID v5 generation must use DNS:"globalnames.org" namespace
- DataSource IDs are hard-coded, not auto-generated
- Canonical forms stored in separate tables, not embedded
- Word-level indexing enables fuzzy name matching
