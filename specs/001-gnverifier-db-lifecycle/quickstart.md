# Quickstart: GNverifier Database Lifecycle

**Purpose**: End-to-end integration test validating complete database lifecycle  
**Duration**: ~30-60 minutes (depending on data size)  
**Prerequisites**: PostgreSQL 15+, Go 1.21+, sample SFGA file

---

## Setup

### 1. Install Prerequisites

```bash
# Install PostgreSQL (if not installed)
brew install postgresql@15  # macOS
# or apt-get install postgresql-15  # Linux

# Start PostgreSQL
brew services start postgresql@15

# Install Atlas CLI
curl -sSf https://atlasgo.io/install.sh | sh

# Clone and build gndb
git clone https://github.com/gnames/gndb.git
cd gndb
go build -o gndb cmd/gndb/main.go
```

### 2. Create Test Database

```bash
# Create empty test database
createdb gndb_test

# Verify connection
psql gndb_test -c "SELECT version();"
```

### 3. Prepare Test Data

```bash
# Download sample SFGA file (example: Catalog of Life subset)
wget https://example.com/sample_col.sfga -O testdata/sample.sfga

# Verify SFGA file
sqlite3 testdata/sample.sfga "SELECT version FROM version LIMIT 1;"
```

---

## Test Workflow

### Step 1: Create Schema

```bash
# Run create command
./gndb create \
  --url "postgres://localhost:5432/gndb_test?sslmode=disable" \
  --config gndb.yaml

# Expected output:
# ✓ Extensions created (uuid-ossp, pg_trgm)
# ✓ Tables created (12 tables)
# ✓ Partitions created (16 partitions for name_string_indices)
# ✓ Schema validation passed
# Database schema created successfully
```

**Verification**:
```bash
psql gndb_test -c "\dt"
# Should show: canonicals, canonical_fulls, canonical_stems, data_sources,
#              name_strings, name_string_indices_p00-p15, vernacular_strings, etc.

psql gndb_test -c "\dx"
# Should show: uuid-ossp, pg_trgm extensions
```

### Step 2: Populate Data

```bash
# Import sample SFGA data
./gndb populate \
  --url "postgres://localhost:5432/gndb_test?sslmode=disable" \
  --sfga testdata/sample.sfga \
  --progress

# Expected output:
# Validating SFGA file...
# ✓ SFGA version: 1.0.0 (compatible)
# ✓ Metadata loaded: Sample Dataset (10,000 names, 2,000 vernaculars)
# 
# Importing data...
# [====================] 100% | 10,000/10,000 names | 5,000 names/sec
# [====================] 100% | 2,000/2,000 vernaculars | 8,000 vern/sec
# 
# Import complete:
#   - Data source ID: 999
#   - Name strings: 10,000
#   - Canonicals: 9,500
#   - Indices: 10,000
#   - Vernaculars: 2,000
#   - Duration: 3.2s
```

**Verification**:
```bash
# Check data counts
psql gndb_test -c "SELECT COUNT(*) FROM name_strings;"
# Should show: 10000

psql gndb_test -c "SELECT COUNT(*) FROM canonicals;"
# Should show: ~9500 (some names share canonicals)

psql gndb_test -c "SELECT COUNT(*) FROM vernacular_strings;"
# Should show: 2000

psql gndb_test -c "SELECT id, title, record_count FROM data_sources WHERE id = 999;"
# Should show: Sample Dataset with 10,000 records
```

### Step 3: Restructure/Optimize

```bash
# Run optimization
./gndb restructure \
  --url "postgres://localhost:5432/gndb_test?sslmode=disable" \
  --progress

# Expected output:
# Creating indexes...
# [1/6] ✓ idx_name_strings_name (45s)
# [2/6] ✓ idx_name_strings_name_trgm (120s)
# [3/6] ✓ idx_canonicals_name (30s)
# [4/6] ✓ idx_vernacular_english (2s)
# [5/6] ✓ idx_nsi_name_string_id (25s)
# [6/6] ✓ idx_nsi_canonical_id (22s)
# 
# Creating materialized views...
# ✓ mv_name_lookup (15s)
# 
# Updating statistics...
# ✓ ANALYZE complete (8s)
# 
# Optimization complete. Total time: 4m 27s
```

**Verification**:
```bash
# Check indexes
psql gndb_test -c "\di"
# Should show 30+ indexes

# Verify materialized view
psql gndb_test -c "SELECT COUNT(*) FROM mv_name_lookup;"
# Should show: 10000

# Check statistics
psql gndb_test -c "SELECT tablename, n_live_tup FROM pg_stat_user_tables ORDER BY n_live_tup DESC;"
```

### Step 4: Performance Validation

```bash
# Test exact canonical match (should use index)
psql gndb_test -c "EXPLAIN ANALYZE SELECT * FROM name_strings WHERE name = 'Homo sapiens';"
# Should show: Index Scan using idx_name_strings_name
# Execution Time: < 1ms

# Test fuzzy match (should use GiST trigram index)
psql gndb_test -c "EXPLAIN ANALYZE SELECT name FROM canonicals WHERE name % 'Homo sapienz' LIMIT 10;"
# Should show: Bitmap Index Scan using idx_canonicals_name_trgm
# Execution Time: < 50ms

# Test vernacular lookup
psql gndb_test -c "EXPLAIN ANALYZE SELECT * FROM vernacular_strings WHERE language = 'en' AND name = 'human' LIMIT 10;"
# Should show: Index Scan using idx_vernacular_english
# Execution Time: < 5ms

# Test materialized view query
psql gndb_test -c "EXPLAIN ANALYZE SELECT * FROM mv_name_lookup WHERE canonical = 'Homo sapiens' LIMIT 10;"
# Should show: Index Scan (index-only scan if possible)
# Execution Time: < 2ms
```

### Step 5: Test Throughput (1000 names/sec target)

```bash
# Generate 1000 random canonical lookups
psql gndb_test -c "
CREATE TEMP TABLE test_queries AS
SELECT name FROM canonicals ORDER BY random() LIMIT 1000;

\timing on
SELECT ns.* 
FROM test_queries tq
JOIN name_strings ns ON ns.name = tq.name;
\timing off
"
# Should complete in < 1 second for 1000 lookups
# Throughput: > 1000 queries/sec
```

### Step 6: Test Migration (Optional)

```bash
# Check migration status
./gndb migrate status \
  --url "postgres://localhost:5432/gndb_test?sslmode=disable"

# Expected output:
# Migration Status: OK
# Current version: 20231001120000
# Pending migrations: 0

# Dry-run a migration
atlas migrate apply \
  --url "postgres://localhost:5432/gndb_test?sslmode=disable" \
  --dir "file://migrations" \
  --dry-run

# Apply migrations (if any)
./gndb migrate apply \
  --url "postgres://localhost:5432/gndb_test?sslmode=disable"
```

---

## Cleanup

```bash
# Drop test database
dropdb gndb_test

# Remove test SFGA file
rm testdata/sample.sfga
```

---

## Success Criteria

- [x] Schema created with all tables and partitions
- [x] Extensions (uuid-ossp, pg_trgm) installed
- [x] Data imported successfully (10K names, 2K vernaculars)
- [x] Indexes created (30+ indexes)
- [x] Materialized views built
- [x] Exact matches: < 1ms latency
- [x] Fuzzy matches: < 50ms latency
- [x] Vernacular lookups: < 5ms latency
- [x] Throughput: > 1000 queries/sec
- [x] Index usage confirmed via EXPLAIN ANALYZE
- [x] No errors or warnings during any phase

---

## Troubleshooting

### Problem: "Extension uuid-ossp not available"
**Solution**: Install PostgreSQL contrib packages
```bash
apt-get install postgresql-contrib-15  # Linux
brew reinstall postgresql@15  # macOS
```

### Problem: "SFGA version incompatible"
**Solution**: Check SFGA version and update sflib
```bash
sqlite3 testdata/sample.sfga "SELECT version FROM version;"
go get github.com/sfborg/sflib@latest
```

### Problem: "Out of memory during populate"
**Solution**: Reduce batch size in config
```yaml
# gndb.yaml
populate:
  batch_size: 1000  # Reduce from default 5000
```

### Problem: "Index creation timeout"
**Solution**: Increase maintenance_work_mem
```sql
-- In postgresql.conf or SET
SET maintenance_work_mem = '4GB';
```

---

## Next Steps

After successful quickstart:
1. Test with full-size SFGA files (100M+ names)
2. Benchmark production workloads
3. Tune PostgreSQL configuration for production
4. Set up connection pooling (PgBouncer)
5. Configure backups and monitoring

---

**Quickstart Complete**: Database lifecycle validated end-to-end.
