# Quickstart for Optimize Database Performance

This guide provides the steps to use the `optimize` command to prepare your database for gnverifier.

## Prerequisites

- ✅ PostgreSQL database must be created (run `gndb create` first)
- ✅ Database must be populated with data sources (run `gndb populate` first)

## What Does Optimize Do?

The `gndb optimize` command applies 6 performance optimizations:

1. **Reparse Names**: Updates all scientific names with latest gnparser algorithms
2. **Normalize Vernacular Languages**: Standardizes language codes to ISO 639-3
3. **Remove Orphans**: Cleans up unreferenced records (names, canonicals, stems)
4. **Create Words Tables**: Extracts searchable word indices for fast matching
5. **Create Verification View**: Builds materialized view for gnverifier queries
6. **VACUUM ANALYZE**: Updates PostgreSQL statistics for optimal query planning

## Basic Usage

Optimize with default settings (uses all CPU cores):

```bash
gndb optimize
```

**Expected Output:**
```
Starting Database Optimization

Optimizing your database for gnverifier...

This may take several hours for large databases.
Workers: 16

# Progress logs appear on STDERR (colored):
2:45PM INF Starting database optimization workflow
2:45PM INF Step 1/6: Reparsing name strings
2:46PM INF Loaded 100000 names for reparsing
...
2:47PM INF Step 1/6: Complete - Name strings reparsed
...

Database Optimization Complete!

✓ Your database is now optimized and ready for gnverifier.

You can re-run gndb optimize anytime to apply the latest algorithm updates.
```

## Advanced Usage

### Custom Worker Count

Use more workers for faster optimization on powerful servers:

```bash
gndb optimize --jobs=100
```

Adjust workers based on your system:
- **Small servers** (2-4 cores): `--jobs=4`
- **Medium servers** (8-16 cores): `--jobs=16` (default)
- **Large servers** (32+ cores): `--jobs=100`

### Custom Batch Size

Adjust batch size for memory-constrained systems:

```bash
gndb optimize --batch-size=10000
```

Default batch size is 50,000. Lower values use less memory but run
slower.

### Batch Size Tuning

The batch size controls how many name records are processed together
during the reparsing step. Tuning this parameter balances memory usage
with performance.

**Performance Results** (from 100K row test):
- **Batch size 50,000** (default):
  - Throughput: 18,000-26,000 rows/sec
  - Memory: ~20 MB per batch
  - Best for most systems

**Recommended Settings by System:**

| System RAM | Recommended Batch Size | Expected Memory |
|------------|------------------------|-----------------|
| < 8 GB     | 10,000                 | ~4 MB           |
| 8-16 GB    | 25,000                 | ~10 MB          |
| 16-32 GB   | 50,000 (default)       | ~20 MB          |
| > 32 GB    | 100,000                | ~40 MB          |

**Quick Formula**: `batch_size = available_memory_gb * 1000`

**Memory Considerations:**
- Each batch uses approximately 200-400 bytes per name record
- PostgreSQL also needs memory for connections, temp tables, and caching
- Reserve at least 2GB for PostgreSQL's base operations
- The filter-then-batch strategy only processes *changed* names, so
  actual memory usage depends on how many names need updating

**When to Tune Batch Size:**
1. **Lower batch size** if you experience:
   - Out of memory errors
   - System swapping/thrashing
   - Database connection timeouts

2. **Higher batch size** for:
   - Systems with abundant RAM (> 32 GB)
   - First-time optimization (100% of names need parsing)
   - Faster completion on powerful servers

**Example Configurations:**

```bash
# Low memory server (4 GB RAM)
gndb optimize --batch-size=10000 --jobs=4

# Standard server (16 GB RAM) - uses defaults
gndb optimize

# High memory server (64 GB RAM)
gndb optimize --batch-size=100000 --jobs=50
```

**Configuration File Alternative:**

Instead of using command-line flags, you can set the batch size in your
config file (`~/.config/gndb/config.yaml`):

```yaml
optimization:
  reparse_batch_size: 50000
```

Or via environment variable:

```bash
export GNDB_OPTIMIZATION_REPARSE_BATCH_SIZE=50000
gndb optimize
```

**Note**: Batch size primarily affects memory usage during Step 1
(Reparse Names). Other optimization steps use different memory
patterns.

### Combined Options

```bash
gndb optimize --jobs=50 --batch-size=25000
```

## Performance Expectations

Typical optimization times (approximate):

| Database Size | Name Count | Time (16 workers) | Time (100 workers) |
|---------------|------------|-------------------|--------------------|
| Small         | 100K names | 5-10 minutes      | 2-5 minutes        |
| Medium        | 1M names   | 30-60 minutes     | 10-20 minutes      |
| Large         | 10M names  | 5-10 hours        | 2-4 hours          |
| Very Large    | 100M+ names| 1-2 days          | 8-12 hours         |

**Note**: Times vary based on CPU, disk speed, and PostgreSQL configuration.

## Idempotency

The optimize command is **safe to run multiple times**:

```bash
# First run - applies all optimizations
gndb optimize

# Second run - safe, ensures latest algorithms applied
gndb optimize
```

Each run:
- Truncates and rebuilds words tables (no duplicates)
- Drops and recreates verification view (no duplicates)
- Updates all records with latest parsing logic
- Removes any new orphaned records

## Troubleshooting

### Database Not Populated

**Error:**
```
Database Not Populated

Cannot optimize an empty database.
Please populate the database first with data sources.
```

**Solution:**
```bash
# Populate database first
gndb populate

# Then optimize
gndb optimize
```

### Out of Memory

**Symptoms:**
- Process killed by OS
- "out of memory" errors in logs

**Solutions:**
1. Reduce worker count: `gndb optimize --jobs=4`
2. Reduce batch size: `gndb optimize --batch-size=10000`
3. Increase PostgreSQL `work_mem` setting
4. Add more system RAM

### Disk Space Issues

**Error:**
```
Step 4 Failed: Create Words

Failed to populate words tables.
Check disk space and PostgreSQL logs.
```

**Solution:**
```bash
# Check available disk space
df -h

# PostgreSQL data directory typically needs:
# - 10-20% of database size for temp files
# - Additional space for indexes and materialized views
```

### Connection Timeout

**Error:**
```
Database connection lost during processing
```

**Solutions:**
1. Check PostgreSQL is running: `pg_isready`
2. Increase PostgreSQL timeouts:
   ```sql
   ALTER DATABASE gndb SET statement_timeout = '0';
   ```
3. Check network stability (if remote database)

### Slow Performance

**Symptoms:**
- Optimization taking much longer than expected
- Low CPU utilization

**Solutions:**
1. Increase worker count: `gndb optimize --jobs=100`
2. Check PostgreSQL configuration:
   - `max_parallel_workers` should be high (e.g., 32)
   - `shared_buffers` should be 25% of RAM
   - `work_mem` should be adequate (e.g., 256MB)
3. Use faster storage (SSD recommended)
4. Disable unnecessary PostgreSQL logging during optimization

## Monitoring Progress

### View Logs (STDERR)

Progress logs appear on STDERR with colored output:

```bash
# View logs in real-time
gndb optimize 2>&1 | tee optimize.log

# Filter to info messages only
gndb optimize 2>&1 | grep INF
```

### Monitor PostgreSQL

```bash
# Watch active queries
watch -n 1 "psql -d gndb_test -c 'SELECT pid, state, query FROM pg_stat_activity WHERE datname = current_database()'"

# Monitor table sizes during optimization
watch -n 5 "psql -d gndb_test -c \"SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size FROM pg_tables WHERE schemaname = 'public' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC\""
```

## Next Steps

After optimization completes successfully:

1. **Verify optimization**: Check verification view exists
   ```sql
   SELECT COUNT(*) FROM verification;
   ```

2. **Check words tables**: Ensure words are populated
   ```sql
   SELECT COUNT(*) FROM words;
   SELECT COUNT(*) FROM word_name_strings;
   ```

3. **Use with gnverifier**: Your database is now ready!
   ```bash
   gnverifier -s "Homo sapiens"
   ```

4. **Re-optimize periodically**: Run `gndb optimize` after:
   - Adding new data sources (`gndb populate`)
   - Updating to newer gnparser versions
   - Every 6-12 months for algorithm updates
