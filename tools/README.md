# Tools

Temporary development and testing tools for gndb. These are not part of the main application.

## compare_sources.go

Compares a data source between the `gnames` database (populated by to-gn) and the `gndb` database (populated by gndb).

### Usage

**Option 1: Using the shell script wrapper**

```bash
# Set environment variables
export GNDB_DB_PASSWORD=secret
export GNDB_DB_HOST=localhost  # optional, default: localhost
export GNDB_DB_PORT=5432       # optional, default: 5432
export GNDB_DB_USER=postgres   # optional, default: postgres

# Run comparison
tools/compare.sh 1              # Compare source ID 1
tools/compare.sh 1 200          # Compare source ID 1 with 200 sample records
```

**Option 2: Direct invocation**

```bash
go run tools/compare_sources.go \
  --source-id 1 \
  --host localhost \
  --port 5432 \
  --user postgres \
  --password secret \
  --sample-size 100
```

### Parameters

- `--source-id` (required): Data source ID to compare
- `--host`: PostgreSQL host (default: localhost)
- `--port`: PostgreSQL port (default: 5432)
- `--user`: PostgreSQL user (default: postgres)
- `--password`: PostgreSQL password
- `--sample-size`: Number of sample records to compare (default: 100)

### What it compares

1. **Record Counts**
   - Name string indices count
   - Vernacular string indices count

2. **Metadata**
   - Title, title_short, version
   - Record counts (name and vernacular)

3. **Sample Records**
   - Record ID, name string ID
   - Rank, taxonomic status
   - Classification paths
   - Classification IDs

4. **Taxonomic Status Distribution**
   - Count of records by taxonomic status (accepted, synonym, etc.)
   - Ensures the proportion of each status type matches

5. **Vernacular Records**
   - Sample of vernacular names with language metadata

### Example Output

```
Comparing data source ID 1
===

1. Record Counts
----------------
  Name String Indices:
    to-gn: 15234
    gndb:  15234
    ✓ Match

  Vernacular String Indices:
    to-gn: 8456
    gndb:  8456
    ✓ Match

2. Data Source Metadata
-----------------------
  Title:             ✓ Database of Vascular Plants of Canada
  Title Short:       ✓ VASCAN
  Version:           ✓ 2024-07-15
  Record Count:      ✓ 15234
  Vern Record Count: ✓ 8456

3. Sample Name String Indices
-----------------------------
  Sampled 100 records
  ✓ All sample records match
  ✓ All classifications match

4. Taxonomic Status Distribution
---------------------------------
  accepted: ✓ 12456
  synonym: ✓ 2778

  ✓ All taxonomic status counts match

5. Sample Vernacular String Indices
-----------------------------------
  Sampled 100 vernacular records
  ✓ All vernacular records match

6. Summary
----------
  ✓ All comparisons match!
  The imports are identical.
```

### Notes

- Both databases must exist on the same PostgreSQL server
- Database names are hardcoded: `gnames` (to-gn) and `gndb` (gndb)
- This tool is temporary and will be removed once gndb is production-ready
