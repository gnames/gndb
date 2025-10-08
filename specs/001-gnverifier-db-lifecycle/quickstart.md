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

Populate the database with data from a sample SFGA file:

```bash
gndb populate testdata/sfga.sqlite
```

### 3.4. Optimize Database

Apply performance optimizations (indexes, materialized views):

```bash
gndb optimize
```

Note: This command is idempotent and always rebuilds optimizations from scratch to ensure algorithm improvements are applied.

## 4. Verification

After completing the above steps, you can connect to the `gnames_test` database and verify that the tables have been created and populated.

```sql
\c gnames_test

SELECT COUNT(*) FROM name_strings;
-- Expected: > 0

SELECT COUNT(*) FROM vernacular_names;
-- Expected: > 0
```