# Quickstart: GNverifier Database Lifecycle

This quickstart guide demonstrates the end-to-end lifecycle of creating, populating, and optimizing a local GNverifier database using the `gndb` CLI.

## Prerequisites

*   Go 1.25 or later
*   PostgreSQL 14 or later
*   `just` command-line tool

## 1. Installation

Install `gndb` from the root of the repository:

```bash
just install
```

## 2. Configuration

Create a `gndb.yaml` file in the root of the project with the following content:

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  database: "gnames_test"
  ssl_mode: "disable"
```

## 3. Database Lifecycle

### 3.1. Create Schema

Create the database schema:

```bash
gndb create --force
```

The `--force` flag will drop any existing tables in the database before creating the new schema.

### 3.2. Migrate Schema (Optional)

If the schema needs to be updated to a newer version:

```bash
gndb migrate
```

Note: GORM AutoMigrate handles schema versioning automatically.

### 3.3. Populate Database

Populate the database with nomenclature data from SFGA sources.

#### Configuration

The populate command uses `sources.yaml` to define which data sources to import. By default, it looks for this file at `~/.config/gndb/sources.yaml` (same directory as `config.yaml`).

To generate a template `sources.yaml`:

```bash
# Copy the template to your config directory
cp pkg/templates/sources.yaml ~/.config/gndb/sources.yaml
```

The `sources.yaml` file defines data sources with their IDs, URLs, and metadata. Official sources have IDs < 1000, custom sources use IDs >= 1000.

#### Basic Usage

```bash
# Import all sources defined in sources.yaml
gndb populate

# Use a custom sources.yaml location
gndb populate --sources-yaml /path/to/custom-sources.yaml
```

#### Filtering Sources

You can selectively import specific sources using the `--sources` flag:

```bash
# Import only official sources (ID < 1000)
gndb populate --sources main

# Import only custom sources (ID >= 1000)
gndb populate --sources "exclude main"

# Import specific sources by ID (comma-separated)
gndb populate --sources 1,3,9

# Import a single source
gndb populate --sources 206
```

#### Override Flags (Single Source Only)

When importing a single source, you can override its release version or date:

```bash
# Override release version
gndb populate --sources 1 --release-version "2024.1"

# Override release date (YYYY-MM-DD format)
gndb populate --sources 2 --release-date "2024-12-15"

# Override both
gndb populate --sources 3 --release-version "2024.2" --release-date "2024-12-20"
```

**Note**: Override flags only work when importing a single source. They will return an error if used with multiple sources.

#### Cache Location

Downloaded SFGA files are cached at `~/.cache/gndb/sfga/` to avoid redundant downloads. The cache structure is:

```
~/.cache/gndb/sfga/
├── 0001.sqlite        # Catalogue of Life
├── 0003.sqlite        # ITIS
├── 0009.sqlite        # WoRMS
└── ...
```

To force re-download of a source, delete its cached file before running populate.

### 3.4. Optimize Database

Apply performance optimizations (indexes, materialized views):

```bash
gndb optimize
```

Note: This command is idempotent and always rebuilds optimizations from scratch to ensure algorithm improvements are applied.

## 4. Troubleshooting

### Populate Issues

**Problem**: "failed to load sources configuration"

**Solution**: Ensure `sources.yaml` exists at `~/.config/gndb/sources.yaml` or provide a custom path with `--sources-yaml`.

```bash
# Generate sources.yaml from template
cp pkg/templates/sources.yaml ~/.config/gndb/sources.yaml
```

**Problem**: "no sources selected for import"

**Solution**: Your `--sources` filter matched no sources. Check your filter syntax or `sources.yaml` content.

```bash
# List all sources in your config
grep "^  - id:" ~/.config/gndb/sources.yaml

# Try without filter
gndb populate
```

**Problem**: Download fails or times out

**Solution**: Check your internet connection and the `parent` URL in `sources.yaml`. If the issue persists, the SFGA file may be temporarily unavailable.

```bash
# Clear cache and retry
rm -rf ~/.cache/gndb/sfga/
gndb populate --sources 1
```

**Problem**: "cannot override release version with multiple sources"

**Solution**: Override flags (`--release-version`, `--release-date`) only work with a single source. Use `--sources` to select one source.

```bash
# Correct: single source
gndb populate --sources 1 --release-version "2024.1"

# Incorrect: multiple sources
gndb populate --sources 1,2,3 --release-version "2024.1"  # Error!
```

### Cache Debugging

To inspect or clear the SFGA cache:

```bash
# View cached files
ls -lh ~/.cache/gndb/sfga/

# Clear entire cache (force re-download)
rm -rf ~/.cache/gndb/sfga/

# Clear specific source cache (e.g., source ID 206)
rm ~/.cache/gndb/sfga/0206.sqlite
```

### Database Connection Issues

**Problem**: "failed to connect to database"

**Solution**: Verify PostgreSQL is running and `config.yaml` has correct credentials.

```bash
# Test PostgreSQL connection
psql -h localhost -U postgres -d gnames_test

# Check config
cat ~/.config/gndb/config.yaml
```

## 5. Verification

After completing the above steps, you can connect to the `gnames_test` database and verify that the tables have been created and populated.

```sql
\c gnames_test

SELECT COUNT(*) FROM name_strings;
-- Expected: > 0

SELECT COUNT(*) FROM vernacular_names;
-- Expected: > 0
```