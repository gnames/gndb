# Research: GNverifier Database Lifecycle Implementation

**Date**: 2025-10-02  
**Status**: Complete

## Executive Summary

This document consolidates research findings on PostgreSQL schema design, Atlas migrations, SFGA format handling, and gnidump reference patterns to inform the GNverifier database lifecycle implementation.

---

## 1. PostgreSQL Schema Design for 100M+ Names

### Decision: Hybrid Indexing Strategy

**Chosen Approach**:
- **B-tree indexes** with INCLUDE clause for exact matches and covering indexes
- **GiST trigram indexes** (siglen=256) for fuzzy name matching
- **Partial indexes** for language-specific vernacular names
- **Hash partitioning** (16 partitions) for name_string_occurrences table

**Rationale**:
- Covering indexes eliminate heap lookups, improving read performance by 30-60%
- Trigram indexes with tuned siglen parameter achieve 50%+ speedup vs defaults
- Partial indexes reduce index size for vernacular names organized by language
- Hash partitioning enables partition pruning and parallel queries for 200M occurrence records

**Alternatives Considered**:
- GIN indexes: Rejected for fuzzy matching (slower to build, higher update cost)
- List partitioning by data source: Rejected due to uneven distribution
- BRIN indexes: Reserved only for timestamp/sequential columns

### Decision: Materialized Views for Denormalization

**Chosen Approach**:
Create materialized views for:
- `mv_name_lookup`: Pre-joined names with primary data source
- `mv_vernacular_by_language`: Aggregated vernacular names by language
- `mv_synonym_map`: Denormalized synonym resolution

**Rationale**:
- Eliminates expensive 3-table joins (60-80% query time savings)
- Read-only post-setup database makes refresh overhead acceptable
- Heavily indexed materialized views enable index-only scans

**Trade-offs**:
- Storage cost (duplicated data) vs query speed
- Refresh overhead during restructure phase
- Acceptable for read-heavy workload after setup

---

## 2. Atlas Migration Framework Integration

### Decision: Use Atlas with Versioned Migrations

**Chosen Approach**:
- Versioned migration files: `{timestamp}_{name}.sql`
- `atlas.sum` integrity tracking (Merkle hash tree)
- Transactional migration mode (`--tx-mode all`)
- Linear execution order (`--exec-order linear`)

**Rationale**:
- PostgreSQL supports transactional DDL (automatic rollback on failure)
- atlas.sum prevents concurrent migration conflicts
- Automatic migration generation reduces manual SQL writing
- Native Go SDK integration with cobra CLI

**Integration Pattern**:
```go
import "ariga.io/atlas/atlasexec"

client, _ := atlasexec.NewClient(workdir.Path(), "atlas")
result, _ := client.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
    URL: dbURL,
})
```

**Alternatives Considered**:
- golang-migrate: Rejected (manual SQL, no integrity tracking)
- goose: Rejected (less mature, fewer safety features)
- Custom solution: Rejected (reinventing solved problems)

---

## 3. SFGA Format and sflib Library

### Decision: Stream-Based Import with Batch Inserts

**SFGA Structure** (github.com/sfborg/sfga/schema.sql):
- SQLite-based archive following CoLDP standard
- 30+ tables: metadata, core taxonomic, extended data, controlled vocabularies
- Key tables:
  - `name`: col__id (PK), gn__scientific_name_string, gn__canonical_simple, gn__canonical_full, gn__canonical_stemmed, col__rank_id, authorship fields
  - `taxon`: col__id (PK), col__name_id (FK to name), col__parent_id, taxonomic hierarchy (kingdom→species)
  - `synonym`: col__taxon_id (FK), col__name_id (FK), taxonomic status
  - `vernacular`: col__taxon_id (FK), col__name, col__language

**sflib API Pattern**:
```go
import "github.com/sfborg/sflib/internal/isfga"

sfga := isfga.New()
sfga.SetDb(dbPath)
db, _ := sfga.Connect()

// Stream data via channels
ctx := context.Background()
namesChan := sfga.LoadNames(ctx)
for name := range namesChan {
    // Process name
}
```

**Chosen Import Strategy**:
1. Stream data from SFGA via channels (memory efficient)
2. Batch inserts (1000-5000 records per transaction)
3. Disable indexes during import, rebuild after
4. Use PostgreSQL COPY protocol for maximum speed

**Batch Size Decision**:
- Scientific names (small records): 5000 per batch
- Taxa (medium records): 2000 per batch
- References (large records): 1000 per batch

**Rationale**:
- Streaming prevents memory exhaustion on 100M+ records
- Batching balances transaction overhead with rollback size
- Disabling indexes during import speeds up bulk load by 10x
- COPY protocol is fastest PostgreSQL import method

---

## 4. SFGA Version Compatibility

### Decision: Enforce Same Version for Initial Ingest

**Versioning Model**:
- Semantic versioning: MAJOR.MINOR.PATCH
- Major: Incompatible schema changes
- Minor: Backward-compatible additions
- Patch: Non-breaking changes

**Version Checking**:
```go
if !sfga.IsCompatible("1.2.0") {
    return fmt.Errorf("incompatible SFGA version")
}
```

**Policy**:
- Initial ingest: ALL data sources MUST use same SFGA version
- Subsequent updates: Different versions supported via version-compatible sflib

**Rationale**:
- Prevents schema conflicts during initial population
- Allows incremental updates with schema evolution
- `IsCompatible()` method enforces version constraints

---

## 5. SFGA to PostgreSQL Mapping

### Decision: Direct Table Mapping with Denormalization

**Core Tables**:

| SFGA Table | PostgreSQL Table | Notes |
|------------|------------------|-------|
| name | name_strings | Stores parsed name components |
| taxon | taxa | Complete hierarchical taxonomy |
| synonym | synonyms | Alternative names |
| vernacular | vernacular_names | Common names by language |
| reference | references | Bibliographic data |
| - | name_string_occurrences | Denormalized occurrence tracking |
| - | data_sources | SFGA source metadata |

**Type Mapping**:
- SQLite TEXT → PostgreSQL TEXT
- SQLite INTEGER → PostgreSQL BIGINT (for large IDs)
- SQLite BOOLEAN (0/1) → PostgreSQL BOOLEAN
- Comma-separated TEXT → PostgreSQL TEXT[] (arrays)

**Special Handling**:
- Reserved keywords: `order` → `order_name`
- IDs: Keep as TEXT (SFGA uses UUIDs/custom IDs)
- Timestamps: Convert TEXT → TIMESTAMP
- Enumerations: Use CHECK constraints or PostgreSQL ENUMs

**Denormalized Columns**:
- `data_source_name` in occurrences (avoid join with data_sources)
- Parsed name components in name_strings (avoid gnparser runtime calls)

---

## 6. Performance Optimizations

### Decision: Three-Phase Restructure Strategy

**Phase Approach**:
1. **Indexes**: Create all secondary indexes (B-tree, GiST trigram, partial)
2. **Materialized Views**: Build denormalized views
3. **Statistics**: Run ANALYZE, tune high-cardinality columns

**Index Creation Order**:
```sql
-- 1. Critical lookup indexes
CREATE INDEX idx_namestrings_canonical ON name_strings (canonical_simple);

-- 2. Fuzzy matching indexes
CREATE INDEX idx_namestrings_name_trgm 
ON name_strings USING GIST (name_string gist_trgm_ops(siglen=256));

-- 3. Language-specific partial indexes
CREATE INDEX idx_vernacular_english 
ON vernacular_names (name_string) WHERE language_code = 'en';

-- 4. Statistics update
ALTER TABLE name_strings ALTER COLUMN canonical_simple SET STATISTICS 1000;
ANALYZE name_strings;
```

**Memory Tuning** (for 64GB server):
```
shared_buffers = 16GB           # 25% of RAM
effective_cache_size = 48GB     # 75% of RAM
work_mem = 256MB                # Per-operation
maintenance_work_mem = 4GB      # For index builds
```

**Connection Pooling**:
- Use PgBouncer transaction pooling
- 20-30 backend connections for 1000 queries/sec
- Reduces memory overhead from 2GB (1000 connections) to 60MB (30 connections)

---

## 7. gnidump Reference Implementation

### Lessons Learned

**Architecture Patterns to Adopt**:
- Separate pkg/ (pure logic) and internal/io/ (database ops)
- Interface-driven design for testability
- CLI subcommands via cobra
- Configuration via viper (YAML + flags)

**Patterns to Avoid**:
- Mixing database logic in CLI handlers
- Direct SQL in business logic (use repository pattern)
- Lack of progress reporting for long operations

**Reusable Components**:
- Database connection management
- SFGA file validation
- Progress tracking utilities
- Error recovery strategies

---

## 8. Technology Stack Decisions

### Final Stack

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Language** | Go 1.21+ | Type safety, concurrency, stdlib richness |
| **CLI Framework** | cobra | Industry standard, subcommand support |
| **Configuration** | viper | YAML/flag/env precedence |
| **PostgreSQL Driver** | pgx | Best performance, native protocol support |
| **Migrations** | atlas | Modern, safe, Go-native |
| **SFGA Import** | sflib | Official SFGA library |
| **Name Parsing** | gnparser | GNames ecosystem integration |
| **Testing** | testify/assert | Familiar, assertion-rich |

**Not Chosen**:
- database/sql: Rejected (pgx outperforms by 2-3x)
- goose migrations: Rejected (less feature-rich than atlas)
- GORM: Rejected (overhead not needed for bulk operations)

---

## 9. Implementation Roadmap

### Phase Sequence

**Phase 0: Setup** (gndb create)
1. Connect to PostgreSQL
2. Create base tables (no indexes except primary keys)
3. Create partitions for occurrences table
4. Insert schema_version tracking

**Phase 1: Populate** (gndb populate)
1. Validate SFGA version compatibility
2. Stream data from SFGA files
3. Batch insert into PostgreSQL (COPY protocol)
4. Order: references → name_strings → taxa → synonyms → vernaculars → occurrences

**Phase 2: Restructure** (gndb restructure)
1. Create all secondary indexes
2. Build materialized views
3. Update statistics (ANALYZE)
4. Create denormalized columns
5. VACUUM ANALYZE

**Phase 3: Migrate** (gndb migrate)
1. Check current schema version
2. Apply pending Atlas migrations
3. Update schema_version table
4. Option to nuke and rebuild from scratch

---

## 10. Performance Targets and Validation

### Success Criteria

| Metric | Target | Validation Method |
|--------|--------|-------------------|
| **Reconciliation Throughput** | 1000 names/sec | pgbench with canonical_form lookups |
| **Fuzzy Match Latency** | <100ms per query | EXPLAIN ANALYZE on trigram searches |
| **Import Speed** | 10K records/sec | Monitor during populate phase |
| **Index Build Time** | <2 hours (100M rows) | Time restructure phase |
| **Database Size** | <500GB (100M names) | pg_total_relation_size() |

### Monitoring Queries

```sql
-- Index usage
SELECT indexrelname, idx_scan, idx_tup_read 
FROM pg_stat_user_indexes 
WHERE schemaname = 'public' 
ORDER BY idx_scan DESC;

-- Cache hit ratio (should be >95%)
SELECT sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) AS cache_hit_ratio
FROM pg_statio_user_tables;

-- Table sizes
SELECT tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename))
FROM pg_tables WHERE schemaname = 'public';
```

---

## 11. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **SFGA version conflicts** | Data corruption | Enforce version checking before import |
| **Out of memory during import** | Process crash | Stream data via channels, batch inserts |
| **Index build timeout** | Deployment failure | Incremental index creation, monitor progress |
| **Slow fuzzy queries** | User dissatisfaction | Tune siglen parameter, use partial indexes |
| **Data source incompatibility** | Import failure | Validate schema before bulk operations |

---

## 12. Next Steps (Phase 1)

1. Generate **data-model.md**: Concrete PostgreSQL schema DDL
2. Create **contracts/** directory: Go interface definitions
3. Write **quickstart.md**: End-to-end integration test
4. Generate **contract tests**: Failing tests for all interfaces
5. Re-evaluate Constitution Check with design decisions

---

## References

- **PostgreSQL Performance**: TigerData optimization studies, pg_trgm documentation
- **Atlas Framework**: atlasgo.io official docs, Go SDK reference
- **SFGA/sflib**: github.com/sfborg/sfga, github.com/sfborg/sflib
- **GNames Ecosystem**: github.com/gnames/gnidump, github.com/gnames/gnparser
- **GNverifier Spec**: /Users/dimus/code/golang/gndb/specs/001-gnverifier-db-lifecycle/spec.md

---

**Research Complete**: All technical unknowns resolved. Ready for Phase 1 design.
