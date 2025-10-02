# Quickstart: GNverifier Database Lifecycle

**Purpose**: End-to-end integration test for GNdb CLI, validating complete database lifecycle from empty database to optimized GNverifier instance.

**Duration**: ~5-10 minutes with small test dataset  
**Prerequisites**: PostgreSQL 14+, Go 1.21+, test SFGA file

---

## Test Environment Setup

### 1. PostgreSQL Test Database

```bash
# Start PostgreSQL via Docker (or use existing instance)
docker run -d \
  --name gndb-test \
  -e POSTGRES_PASSWORD=testpass \
  -e POSTGRES_DB=gndb_test \
  -p 5432:5432 \
  postgres:15

# Verify connection
psql -h localhost -U postgres -d gndb_test -c "SELECT version();"
```

### 2. Test Data

**Small SFGA test file** (included in `testdata/sample.sfga`):
- 100 scientific names
- 150 taxa (50 accepted, 100 synonyms)
- 200 vernacular names (English, Spanish, Chinese)
- 5 references
- Simulates 2 data sources: "Test Catalog" and "Mini Taxonomy"

**Expected Results**:
- Database size: ~5MB
- Import time: <10 seconds
- Reconciliation: 1000+ names/sec

### 3. Configuration File

Create `gndb.yaml`:
```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: testpass
  database: gndb_test
  ssl_mode: disable

import:
  batch_sizes:
    names: 5000
    taxa: 2000
    references: 1000
    synonyms: 3000
    vernaculars: 3000

optimization:
  concurrent_indexes: false  # Faster for test, locks tables
  statistics_targets:
    name_strings.canonical_simple: 1000
    taxa.rank: 100

logging:
  level: info
  format: json
```

---

## Lifecycle Test Scenarios

### Scenario 1: Create Schema from Empty Database

**Given**: Empty PostgreSQL database  
**When**: User runs `gndb create`  
**Then**: All tables, extensions, and schema_version are created

**Commands**:
```bash
# Build gndb CLI
go build -o gndb ./cmd/gndb

# Create schema
./gndb create --config gndb.yaml

# Verify schema creation
psql -h localhost -U postgres -d gndb_test -c "\dt"
# Expected: 8 tables (data_sources, name_strings, taxa, synonyms, 
#           vernacular_names, references, name_string_occurrences, schema_versions)

# Verify extensions
psql -h localhost -U postgres -d gndb_test -c "\dx"
# Expected: pg_trgm extension enabled

# Verify version
psql -h localhost -U postgres -d gndb_test -c "SELECT * FROM schema_versions;"
# Expected: version='1.0.0', description='Initial schema'
```

**Success Criteria**:
- [x] All 8 tables exist
- [x] Primary keys created
- [x] Foreign keys enforced
- [x] pg_trgm extension enabled
- [x] schema_versions table populated
- [x] Exit code 0

**Expected Output**:
```json
{
  "status": "success",
  "tables_created": 8,
  "extensions_enabled": ["pg_trgm"],
  "schema_version": "1.0.0",
  "duration_ms": 234
}
```

---

### Scenario 2: Create Schema with Force Flag (Destructive)

**Given**: Database with existing tables  
**When**: User runs `gndb create --force`  
**Then**: Old tables dropped, new schema created

**Commands**:
```bash
# Insert test data to verify it gets deleted
psql -h localhost -U postgres -d gndb_test -c \
  "INSERT INTO data_sources (uuid, title, title_short, version, release_date, sfga_version) 
   VALUES ('test-uuid', 'Test', 'Test', '1.0', NOW(), '1.0');"

# Force recreate schema
./gndb create --config gndb.yaml --force

# Verify old data gone
psql -h localhost -U postgres -d gndb_test -c "SELECT count(*) FROM data_sources;"
# Expected: 0 rows
```

**Success Criteria**:
- [x] User prompted for confirmation (unless --yes flag)
- [x] All old tables dropped
- [x] New schema created
- [x] Old data not present
- [x] Exit code 0

---

### Scenario 3: Populate Database from SFGA Files

**Given**: Database with schema created  
**When**: User runs `gndb populate` with test SFGA files  
**Then**: Data imported in correct order with foreign key integrity

**Commands**:
```bash
# Populate from test SFGA file
./gndb populate --config gndb.yaml --source testdata/sample.sfga

# Verify data imported
psql -h localhost -U postgres -d gndb_test << EOF
SELECT 'data_sources', count(*) FROM data_sources UNION ALL
SELECT 'name_strings', count(*) FROM name_strings UNION ALL
SELECT 'taxa', count(*) FROM taxa UNION ALL
SELECT 'synonyms', count(*) FROM synonyms UNION ALL
SELECT 'vernacular_names', count(*) FROM vernacular_names UNION ALL
SELECT 'references', count(*) FROM references UNION ALL
SELECT 'name_string_occurrences', count(*) FROM name_string_occurrences;
EOF
```

**Expected Record Counts**:
```
data_sources             | 1
name_strings             | 100
taxa                     | 150
synonyms                 | 100
vernacular_names         | 200
references               | 5
name_string_occurrences  | 250
```

**Success Criteria**:
- [x] All tables populated
- [x] Foreign keys valid (no orphaned records)
- [x] Canonical forms populated for name_strings
- [x] Import completes in <10 seconds
- [x] Progress reported during import
- [x] Exit code 0

**Expected Output**:
```json
{
  "status": "success",
  "data_source": "testdata/sample.sfga",
  "sfga_version": "1.0.0",
  "records_imported": {
    "references": 5,
    "name_strings": 100,
    "taxa": 150,
    "synonyms": 100,
    "vernaculars": 200,
    "occurrences": 250
  },
  "duration_ms": 8432,
  "records_per_second": 7692
}
```

---

### Scenario 4: SFGA Version Compatibility Check

**Given**: Database populated with SFGA v1.0.0  
**When**: User attempts to import SFGA v2.0.0 (incompatible)  
**Then**: Import fails with clear error message

**Commands**:
```bash
# Attempt to import incompatible SFGA file
./gndb populate --config gndb.yaml --source testdata/incompatible_v2.sfga

# Expected: Exit code 1, error message
```

**Expected Output**:
```json
{
  "status": "error",
  "error": "SFGA version mismatch: database has v1.0.0, file has v2.0.0",
  "suggestion": "Use --force to nuke database and reimport, or use compatible SFGA version"
}
```

**Success Criteria**:
- [x] Import rejected before any data written
- [x] Database remains unchanged
- [x] Clear error message
- [x] Exit code 1

---

### Scenario 5: Restructure Database for Performance

**Given**: Database populated with data  
**When**: User runs `gndb restructure`  
**Then**: Indexes, materialized views, and statistics created

**Commands**:
```bash
# Restructure database
./gndb restructure --config gndb.yaml

# Verify indexes created
psql -h localhost -U postgres -d gndb_test -c \
  "SELECT tablename, indexname FROM pg_indexes WHERE schemaname = 'public' ORDER BY tablename, indexname;"

# Verify materialized views
psql -h localhost -U postgres -d gndb_test -c \
  "SELECT matviewname FROM pg_matviews WHERE schemaname = 'public';"

# Test query performance (should use indexes)
psql -h localhost -U postgres -d gndb_test -c \
  "EXPLAIN ANALYZE SELECT * FROM name_strings WHERE canonical_simple = 'Homo sapiens';"
# Expected: Index Scan using idx_namestrings_canonical_simple
```

**Expected Indexes**:
- `idx_namestrings_canonical_simple` (UNIQUE B-tree)
- `idx_namestrings_name_trgm` (GiST trigram)
- `idx_occurrences_namestring` (B-tree)
- `idx_vernacular_name_trgm` (GiST trigram)
- `idx_vernacular_english` (partial, WHERE language_code='en')
- Plus 10+ more indexes

**Expected Materialized Views**:
- `mv_name_lookup`
- `mv_vernacular_by_language`
- `mv_synonym_map`

**Success Criteria**:
- [x] All secondary indexes created
- [x] Materialized views created and populated
- [x] Statistics updated (ANALYZE run)
- [x] Query planner uses indexes (verify with EXPLAIN)
- [x] Restructure completes in <30 seconds for test dataset
- [x] Exit code 0

**Expected Output**:
```json
{
  "status": "success",
  "indexes_created": 15,
  "materialized_views_created": 3,
  "statistics_updated": 8,
  "vacuum_analyze_complete": true,
  "duration_ms": 12456,
  "database_size_mb": 5.2,
  "cache_hit_ratio": 0.98
}
```

---

### Scenario 6: Schema Migration

**Given**: Database with schema v1.0.0  
**When**: User runs `gndb migrate` and migration v1.1.0 is available  
**Then**: Migration applied successfully

**Commands**:
```bash
# Check current version
./gndb migrate status --config gndb.yaml
# Expected: current_version=1.0.0, pending_migrations=[1.1.0]

# Apply migrations
./gndb migrate apply --config gndb.yaml

# Verify new version
psql -h localhost -U postgres -d gndb_test -c "SELECT * FROM schema_versions ORDER BY applied_at DESC LIMIT 1;"
# Expected: version='1.1.0'
```

**Test Migration** (`migrations/20251002120000_add_cardinality_index.sql`):
```sql
CREATE INDEX idx_namestrings_cardinality 
ON name_strings(cardinality) 
WHERE cardinality > 0;
```

**Success Criteria**:
- [x] Migration detected as pending
- [x] Migration applied successfully
- [x] schema_versions updated
- [x] New index exists
- [x] Exit code 0

**Expected Output**:
```json
{
  "status": "success",
  "migrations_applied": [
    {
      "version": "20251002120000",
      "name": "add_cardinality_index",
      "applied_at": "2025-10-02T12:00:00Z"
    }
  ],
  "current_version": "1.1.0",
  "duration_ms": 456
}
```

---

### Scenario 7: End-to-End Name Reconciliation

**Given**: Fully optimized database  
**When**: User queries for scientific name reconciliation  
**Then**: Results returned with <10ms latency

**Commands**:
```bash
# Test exact canonical match
psql -h localhost -U postgres -d gndb_test -c \
  "EXPLAIN ANALYZE 
   SELECT * FROM mv_name_lookup 
   WHERE canonical_simple = 'Homo sapiens';"
# Expected: Execution Time: <10ms, Index-Only Scan

# Test fuzzy match
psql -h localhost -U postgres -d gndb_test -c \
  "EXPLAIN ANALYZE 
   SELECT name_string, canonical_simple 
   FROM name_strings 
   WHERE name_string % 'Homo sapins'  -- typo: sapins instead of sapiens
   ORDER BY similarity(name_string, 'Homo sapins') DESC 
   LIMIT 5;"
# Expected: Execution Time: <100ms, Bitmap Index Scan using idx_namestrings_name_trgm

# Test vernacular by language
psql -h localhost -U postgres -d gndb_test -c \
  "EXPLAIN ANALYZE 
   SELECT * FROM mv_vernacular_by_language 
   WHERE language_code = 'en' AND name_string ILIKE '%human%';"
# Expected: Execution Time: <10ms, Index Scan using idx_mv_vernacular_lang_name
```

**Success Criteria**:
- [x] Exact matches use B-tree indexes (<10ms)
- [x] Fuzzy matches use trigram indexes (<100ms)
- [x] Vernacular searches use partial indexes (<10ms)
- [x] All queries use index scans (not sequential scans)

---

## Integration Test Script

**Complete automated test** (`quickstart_test.sh`):

```bash
#!/bin/bash
set -e

echo "=== GNdb Quickstart Integration Test ==="

# 1. Create schema
echo "[1/6] Creating schema..."
./gndb create --config gndb.yaml --yes
psql -h localhost -U postgres -d gndb_test -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';" | grep 8

# 2. Populate database
echo "[2/6] Populating database..."
./gndb populate --config gndb.yaml --source testdata/sample.sfga
psql -h localhost -U postgres -d gndb_test -c "SELECT count(*) FROM name_strings;" | grep 100

# 3. Restructure database
echo "[3/6] Restructuring database..."
./gndb restructure --config gndb.yaml
psql -h localhost -U postgres -d gndb_test -c "SELECT count(*) FROM pg_indexes WHERE schemaname='public';" | grep -E "1[5-9]|[2-9][0-9]"  # At least 15 indexes

# 4. Test query performance
echo "[4/6] Testing query performance..."
EXPLAIN_OUTPUT=$(psql -h localhost -U postgres -d gndb_test -t -c \
  "EXPLAIN SELECT * FROM name_strings WHERE canonical_simple = 'Homo sapiens';")
echo "$EXPLAIN_OUTPUT" | grep -q "Index Scan"

# 5. Test migration
echo "[5/6] Testing migration..."
./gndb migrate status --config gndb.yaml

# 6. Cleanup
echo "[6/6] Cleaning up..."
docker stop gndb-test
docker rm gndb-test

echo "✅ All tests passed!"
```

**Run test**:
```bash
chmod +x quickstart_test.sh
./quickstart_test.sh
```

---

## Success Metrics

| Metric | Target | Verification |
|--------|--------|--------------|
| **Schema creation** | <1 second | Time command output |
| **Import throughput** | >1000 records/sec | JSON output `records_per_second` |
| **Index creation** | <30 seconds | JSON output `duration_ms` |
| **Exact match latency** | <10ms | EXPLAIN ANALYZE execution time |
| **Fuzzy match latency** | <100ms | EXPLAIN ANALYZE execution time |
| **Database size** | <10MB for test data | JSON output `database_size_mb` |
| **Cache hit ratio** | >95% | JSON output `cache_hit_ratio` |

---

## Troubleshooting

### Import fails with "foreign key violation"
- **Cause**: Import order incorrect or SFGA file corrupted
- **Fix**: Verify SFGA schema with `./gndb populate --validate-only`

### Queries not using indexes
- **Cause**: Restructure phase not run or ANALYZE not executed
- **Fix**: Run `./gndb restructure --config gndb.yaml` again

### SFGA version mismatch error
- **Cause**: Mixing incompatible SFGA format versions
- **Fix**: Use `--force` to nuke and rebuild, or ensure all sources use same SFGA version

### Out of memory during import
- **Cause**: Batch size too large for available RAM
- **Fix**: Reduce batch sizes in gndb.yaml configuration

---

## Next Steps

After quickstart validation passes:
1. Generate tasks.md using `/tasks` command
2. Implement interfaces in order: config → schema → database → sfga → populate → restructure → migrate
3. Follow TDD workflow: write tests first, verify they fail, implement, verify they pass

---

**Quickstart Complete**: Ready for task generation (Phase 2).
