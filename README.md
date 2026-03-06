# GNdb

[![DOI](https://zenodo.org/badge/DOI/10.5281/zenodo.18895372.svg)](https://doi.org/10.5281/zenodo.18895372)

GNdb is a command-line tool for creating and managing a PostgreSQL database
for a local [GNverifier] instance.

<!-- vim-markdown-toc GFM -->

* [Introduction](#introduction)
* [Prerequisites](#prerequisites)
  * [PostgreSQL](#postgresql)
  * [SQLite](#sqlite)
* [Installation](#installation)
  * [Install with Go](#install-with-go)
  * [Build from source](#build-from-source)
* [Quick Start](#quick-start)
* [Commands](#commands)
  * [create](#create)
  * [populate](#populate)
  * [optimize](#optimize)
  * [migrate](#migrate)
* [Configuration](#configuration)
  * [Config file](#config-file)
  * [Environment variables](#environment-variables)
  * [CLI flags](#cli-flags)
* [Data Sources](#data-sources)
* [Architecture](#architecture)
* [Artificial Intelligence Policy](#artificial-intelligence-policy)
* [Authors](#authors)
* [License](#license)

<!-- vim-markdown-toc -->

## Introduction

[GNverifier] is a scientific name verification service that reconciles
taxonomic names against multiple biodiversity data sources. It detects
misspellings via fuzzy matching, identifies accepted names for taxa, and
retrieves vernacular/common names.

GNverifier is available as a centralized service, but it may lack a
particular data source or do not have the most recent version of a data source.
GNdb makes it possible to set up a local GNverifier instance and populate
it with whatever data sources a researcher needs.

Biodiversity data comes in many formats: Darwin Core Archives (DwCA),
Catalogue of Life Data Package (CoLDP), Excel spreadsheets, plain name
lists, etc. GNdb uses [SFGA] (Species File Group Archive), a normalized
SQLite-based format, as its input. The [SF] tool converts the most common
biodiversity formats into SFGA.

## Prerequisites

### PostgreSQL

GNdb stores data in a `gnames` PostgreSQL database. Install PostgreSQL for
your operating system, create the database, and make sure your user has
the necessary permissions. It is also very useful to tweak `postgresql.conf`
and optimize it according to CPU and memory available on the computer.

```bash
# Example: create the database
createdb gnames
```

Edit `~/.config/gndb/config.yaml` to provide `gndb` information how to connect
to the database. This file will be created after installing `gndb` and
running it for the first time without any subcommands for example as
`gndb` or `gndb -V`.

### SQLite

SQLite is not required by `GNdb` itself, but it is very useful for
examining and querying SFGA archives directly. We recommend the
[SQLite DB viewer] and the [Datasette] tool for this purpose.

## Installation

### Install with Go

If you have Go installed:

```bash
go install github.com/gnames/gndb@latest
```

Make sure `$HOME/go/bin` is in your `PATH`.

### Build from source

```bash
git clone https://github.com/gnames/gndb.git
cd gndb
just install
```

This builds the binary and installs it to `~/go/bin/gndb`.

## Quick Start

The typical workflow to set up a local GNverifier database:

```bash
# 1. Create the database schema
gndb create

# 2. Edit sources.yaml to point to your SFGA files
#    (created automatically at ~/.config/gndb/sources.yaml)

# 3. Populate the database from your SFGA sources
gndb populate
# most likely you would run this command several times providing
# specific data-sources IDs. The IDs must be present in
# ~/.config/gndb/sources.yaml file.
gndb populate -s 1,13,4,1001

# 4. Optimize the database for fast name verification
gndb optimize
```

After step 4 the database is ready to be used with GNverifier.

## Commands

### create

Creates the GNverifier database schema from scratch.

```bash
# Create schema (prompts for confirmation if tables already exist)
gndb create

# Drop existing tables without confirmation
gndb create --force
gndb create -f
```

**What it does:**

1. Connects to PostgreSQL using the configured credentials
2. Warns and prompts if the database already has tables
3. Creates all base tables using GORM AutoMigrate
4. Sets collation for correct scientific name sorting

### populate

Imports nomenclature data from SFGA sources into the database.

```bash
# Import all sources listed in sources.yaml
gndb populate

# Import specific sources by ID
gndb populate --source-ids 1,11,132
gndb populate -s 1,11,132

# Override release metadata for a single source
gndb populate -s 1 --release-version "2024.01" --release-date "2024-01-15"

# Use flat (non-hierarchical) classification
gndb populate --flat-classification
```

| Flag | Short | Description |
| ---- | ----- | ----------- |
| `--source-ids` | `-s` | Comma-separated source IDs to import (default: all) |
| `--release-version` | `-r` | Override version string (single source only) |
| `--release-date` | `-d` | Override date `YYYY-MM-DD` (single source only) |
| `--flat-classification` | `-f` | Use flat rather than hierarchical classification |

**What it does:**

1. Connects to PostgreSQL and verifies the schema exists
2. Reads `~/.config/gndb/sources.yaml` to discover SFGA files
3. Opens each SFGA SQLite file (local path or remote URL)
4. Imports data in phases: source metadata, name-strings, vernacular
   names, classification hierarchy, and name indices
5. Reports progress and final statistics

You can run `gndb populate` multiple times to add more sources. Run
`gndb optimize` after all desired sources are imported.

### optimize

Prepares the database for fast name verification queries.

```bash
gndb optimize
```

**What it does:**

1. Reparses all name-strings using the latest [GNparser]
2. Builds canonical forms (simple, full, stemmed)
3. Creates word indexes for fuzzy matching
4. Builds materialized views and runs VACUUM ANALYZE

Optimization may take 20–90 minutes depending on the dataset size.
Progress bars show the current status. You can re-run this command
any time to apply improvements from a newer version of GNparser.

### migrate

Updates the database schema to the latest version after a GNdb upgrade.
Run this command in case you already have older version of PostgreSQL
database, and in the most recent version the schema did change.

```bash
# Migrate schema only (drops materialized views)
gndb migrate

# Migrate and recreate materialized views immediately
gndb migrate --recreate-views
gndb migrate -v
```

Most of the time there migration would run before populating new data.
In such cases there is no need to recreated materialized views, they
will be restored after all data is imported during the `optimize` step.
GORM AutoMigrate adds new tables and columns but never removes existing
ones, making migrations safe. After migrating, run `gndb populate` and
then `gndb optimize` to rebuild views with fresh data.

## Configuration

Configuration is resolved in the following precedence order (highest first):

```
CLI flags  >  environment variables  >  config file  >  defaults
```

### Config file

The config file is created automatically at `~/.config/gndb/gndb.yaml` on
the first run. Edit it to set persistent settings:

```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: ""
  database: gnames
  ssl_mode: disable
  batch_size: 50000

log:
  level: info        # debug, info, warn, error
  format: json       # json, text
  destination: file  # file, stderr

jobs_number: 8
```

GNdb also creates `~/.config/gndb/sources.yaml` on first run. Edit it to
configure your SFGA data sources (see [Data Sources](#data-sources)).
If you import you own data sources, make sure they have IDs from 1001 and
higher.

Log files are written to `~/.local/share/gndb/logs/gndb.log`.

### Environment variables

All config file fields can be overridden with environment variables using
the `GNDB_` prefix:

```bash
export GNDB_DATABASE_HOST=localhost
export GNDB_DATABASE_PORT=5432
export GNDB_DATABASE_USER=postgres
export GNDB_DATABASE_PASSWORD=secret
export GNDB_DATABASE_DATABASE=gnames
export GNDB_DATABASE_SSL_MODE=disable
export GNDB_DATABASE_BATCH_SIZE=50000
export GNDB_LOG_LEVEL=info
export GNDB_LOG_FORMAT=json
export GNDB_LOG_DESTINATION=file
export GNDB_JOBS_NUMBER=8
```

### CLI flags

Run `gndb --help` or `gndb <command> --help` for the full list of flags.
The version flag follows the convention used across Global Names tools:

```bash
gndb -V   # print version and build timestamp
```

## Data Sources

SFGA data sources are configured in `~/.config/gndb/sources.yaml`. Each
entry points to an SFGA file and provides optional metadata overrides:

```yaml
data_sources:
  - id: 1001
    parent: "/path/to/sfga/files/"
    title_short: "Catalogue of Life"
    home_url: "https://catalogueoflife.org"
    is_curated: true
    has_classification: true

  - id: 1002
    parent: "https://releases.example.org/sfga/"
    title_short: "VASCAN"
    is_curated: true
    has_classification: true
```

| Field | Description |
| ----- | ----------- |
| `id` | Unique integer ID. Use `< 1000` for standard sources, `>= 1000` for custom |
| `parent` | Directory containing the SFGA file (local path or URL) |
| `title_short` | Short display name for the data source |
| `home_url` | URL of the source's home page |
| `is_curated` | Whether the source is expert-curated |
| `is_auto_curated` | Whether the source is algorithmically curated |
| `has_classification` | Whether the source includes taxonomic classification |

### SFGA file formats

GNdb recognizes four file formats:

| Format | Extension | Notes |
| ------ | --------- | ----- |
| SQLite binary | `.sqlite` | Fastest to process |
| Zipped SQLite | `.sqlite.zip` | Smallest download size; preferred over `.sql.zip` |
| SQL dump | `.sql` | Plain-text SQL statements |
| Zipped SQL dump | `.sql.zip` | Compressed SQL dump |

When multiple files match a source ID, GNdb selects the one with the
latest date embedded in the filename (`YYYY-MM-DD` format). On equal
dates the preference order is: `.sqlite.zip` > `.sql.zip` > `.sqlite`
> `.sql`.

### File naming convention

Files are matched to a source by their numeric ID prefix. GNdb tries
zero-padded variants in order: `0001`, `001`, `01`, `1`. The prefix
must be followed by `-`, `_`, or `.` to prevent false matches.

```
0147-vascan-2025-08-25.sqlite.zip       ← matched for id: 147
1000_ruhoff_2023-08-22_v1.0.0.sqlite   ← matched for id: 1000
0196.sql                                 ← matched for id: 196
```

Version (`v1.0.0`) and date (`YYYY-MM-DD`) embedded in the filename are
extracted automatically and stored as source metadata. They can be
overridden at import time with the `--release-version` and
`--release-date` flags.

### Remote sources

The `parent` field can be an HTTP/HTTPS URL pointing to a web directory
listing (Apache or nginx style). GNdb fetches the listing, identifies
the matching file by ID, downloads it to `~/.cache/gndb/sfga/`, and
imports it from there.

Use the [SF] tool to convert Darwin Core Archives, CoLDP packages, and
other biodiversity formats into SFGA.

## Architecture

GNdb follows Clean Architecture principles, with business logic separated
from framework and I/O concerns. See [ARCHITECTURE.md] for a detailed
description of the package structure, dependency graph, and design
patterns.

## Artificial Intelligence Policy

We use artificial intelligence to help find algorithms, decide on
implementation approaches, and generate code. All automatically generated
code is carefully reviewed, with inconsistencies fixed, superfluous
implementations removed, and optimizations improved. No code that we do
not understand or approve makes it into published versions of GNdb. We
primarily use Claude Code, with limited use of Gemini CLI.

## Authors

[Dmitry Mozzherin]

## License

Released under [MIT License]

[ARCHITECTURE.md]: ARCHITECTURE.md
[Claude Code]: https://claude.ai/code
[Datasette]: https://datasette.io
[Dmitry Mozzherin]: https://github.com/dimus
[GNparser]: https://github.com/gnames/gnparser
[GNverifier]: https://github.com/gnames/gnverifier
[MIT License]: LICENSE
[SF]: https://github.com/sfborg/sf
[SFGA]: https://github.com/sfborg/sfga
[SQLite DB viewer]: https://sqlitebrowser.org/
