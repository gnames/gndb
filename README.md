# GNdb

GNdb provides tools to create and populate database for a local gnverifier app.

## Quick Start

### Prerequisites

1. **Install PostgreSQL** (version 15 or later recommended):
   ```bash
   # macOS
   brew install postgresql@15
   brew services start postgresql@15
   
   # Ubuntu/Debian
   sudo apt install postgresql-15
   sudo systemctl start postgresql
   ```

2. **Create a PostgreSQL database** (REQUIRED before running `gndb create`):
   ```bash
   createdb gndb_local
   ```
   
   > **Important**: The `gndb create` command creates tables/schema inside an existing database. 
   > It does NOT create the database itself. You must create the database first using `createdb` 
   > or your preferred PostgreSQL tool.

### Installation

```bash
go install github.com/gnames/gndb/cmd/gndb@latest
```

### Configuration

GNdb can be configured using environment variables, a config file, or command-line flags.

**Option A: Using environment variables with direnv (recommended for development)**

```bash
# Copy the example file
cp .envrc.example .envrc

# Edit with your database settings
vim .envrc

# Allow direnv to load the file
direnv allow .
```

**Option B: Using the config file**

On first run, `gndb` automatically generates a config file at `~/.config/gndb/gndb.yaml` with all options commented out. Uncomment and edit the values you want to override:

```yaml
# Uncomment the "database:" line and the settings you want to override:
# database:
#   host: localhost
#   port: 5432
#   user: postgres
#   password: postgres
#   database: gndb_local
#   ssl_mode: disable
```

**Option C: Using command-line flags**

```bash
gndb create --user myuser --database gndb_local
```

**Configuration Precedence** (highest to lowest):
1. CLI flags (`--host`, `--port`, etc.)
2. Environment variables (`GNDB_*`)
3. Config file (`~/.config/gndb/gndb.yaml`)
4. Built-in defaults

### Create Database Schema

```bash
# Create schema in the database
gndb create

# Or force recreate (drops existing tables - DESTRUCTIVE)
gndb create --force
```

**Expected output:**
```
Connected to database: myuser@localhost:5432/gndb_local
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
```

### Next Steps

```bash
# Import data from SFGA files (coming soon)
gndb populate

# Create indexes and optimize (coming soon)
gndb optimize
```

## Troubleshooting

### Error: "database does not exist"

```
Error: failed to connect to database: ... database "mydb" does not exist
```

**Solution**: Create the database first using `createdb mydb`

### Error: "role does not exist"

```
Error: failed to connect to database: ... role "postgres" does not exist
```

**Solution**: Set the correct database user with:
- Environment variable: `export GNDB_DATABASE_USER=yourusername`
- Config file: Uncomment `user:` in `~/.config/gndb/gndb.yaml`
- CLI flag: `--user yourusername`

### Verify environment variables are loaded

If using direnv:
```bash
env | grep GNDB_
```

## Development

See [specs/001-gnverifier-db-lifecycle/](specs/001-gnverifier-db-lifecycle/) for implementation details.
