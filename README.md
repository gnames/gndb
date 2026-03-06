# GNdb

[![DOI](https://zenodo.org/badge/DOI/10.5281/zenodo.18895372.svg)](https://doi.org/10.5281/zenodo.18895372)

GNdb is a command-line tool for creating and managing a PostgreSQL database
for a local [GNverifier] instance.

<!-- vim-markdown-toc GFM -->

* [Introduction](#introduction)
* [Prerequisites](#prerequisites)
* [Installation](#installation)
  * [Download binary](#download-binary)
  * [Install with Go](#install-with-go)
  * [Build from source](#build-from-source)
* [Quick Start](#quick-start)
* [Next Steps: Running GNverifier](#next-steps-running-gnverifier)
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
  * [Standard sources](#standard-sources)
  * [Custom sources](#custom-sources)
  * [SFGA file formats](#sfga-file-formats)
  * [File naming convention](#file-naming-convention)
  * [Remote sources](#remote-sources)
* [Artificial Intelligence Policy](#artificial-intelligence-policy)
* [Authors](#authors)
* [License](#license)

<!-- vim-markdown-toc -->

## Introduction

[GNverifier] is a scientific name verification service that reconciles
scientific names against multiple biodiversity data sources. It detects
misspellings via fuzzy matching, identifies accepted names for taxa, and
retrieves vernacular/common names. GNdb is the tool that builds and
maintains the PostgreSQL database that a GNverifier server runs against.

GNverifier is available as a centralized service, but it does not always
have a particular data source or the most recent version of one. A local
instance gives you full control over which sources are included and when
they are updated.

**When to use a local instance:**

- You need a data source not available in the central service
- You have private or institutional data not suitable for a public service
- You need a specific version or snapshot of a data source
- You are deploying GNverifier in an offline or air-gapped environment
- You need a dedicated high-throughput server for your organization

## Prerequisites

GNdb stores data in a `gnames` PostgreSQL database. Install PostgreSQL for
your operating system, create the database, and make sure your user has
the necessary permissions. It is also very useful to tweak `postgresql.conf`
and optimize it according to CPU and memory available on the computer.

```bash
# Example: create the database
createdb gnames
```

Edit `~/.config/gndb/config.yaml` to provide `gndb` information how to
connect to the database. This file will be created after installing `gndb`
and running it for the first time without any subcommands, for example as
`gndb` or `gndb -V`.

## Installation

### Download binary

Download the latest pre-built binary for your platform from the [releases
page], unpack the archive, and place the `gndb` binary somewhere in your `PATH`
(e.g. `/usr/local/bin`).

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

This builds the binary and installs it to `$HOME/go/bin/gndb`.

## Quick Start

The typical workflow to set up a local GNverifier database:

```bash
# 1. Create the PostgreSQL database (run once)
createdb gnames
# or connect to PostgreSQL and run:
#   CREATE DATABASE gnames;

# 2. Create the GNverifier schema inside the database
gndb create

# 3. Populate the database with the data sources you need.
#    Standard sources (IDs < 1000) are pre-configured in sources.yaml
#    and downloaded automatically from opendata.globalnames.org/sfga.
gndb populate -s 1,11,4

# 4. Optimize the database for fast name verification
#    This step runs several optimization steps and denormalizes data to
#    a materialized view to speedup queries.
gndb optimize
```

Standard source IDs and their names are listed in
`~/.config/gndb/sources.yaml`, which is created automatically on the
first run. Run `gndb populate` without flags to import all configured
sources at once (it will take a long time).

`gndb populate` can be run multiple times to add sources incrementally,
including after a previous `gndb optimize`. Always run `gndb optimize`
once at the end, after all desired sources have been imported.

After step 4 the database is ready for GNverifier.

## Next Steps: Running GNverifier

Once the database is ready, install and configure the [GNverifier] server
to connect to your `gnames` database. See the [GNverifier] README for
installation and configuration instructions. GNverifier has a
server/client architecture: most users only need to run the server and
access verification results through its REST API or command-line client.

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
3. Creates word indexes for advanced name search
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

Most of the time the migration would run before populating new data.
In such cases there is no need to recreate materialized views, they
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
If you import your own data sources, make sure they have IDs from 1001 and
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

GNdb supports two kinds of sources: **standard** sources maintained by the
Global Names project, and **custom** sources you provide yourself.

### Standard sources

Standard sources have IDs below 1000. They are pre-configured in
`~/.config/gndb/sources.yaml` and their SFGA files are hosted on
`opendata.globalnames.org/sfga`. Running `gndb populate -s <id>` is all
that is needed — GNdb downloads the latest file for that source
automatically.

To see which standard sources are available, open
`~/.config/gndb/sources.yaml` after the first run of `gndb`.

### Custom sources

Custom sources have IDs of 1000 or higher. Use the [SF] tool to convert
your data (Darwin Core Archive, CoLDP, Excel spreadsheet, plain name
list, etc.) into an SFGA file. Place the resulting file on your local
computer or on a web server, then register it in
`~/.config/gndb/sources.yaml`:

```yaml
data_sources:
  - id: 1001
    parent: "/path/to/sfga/files/"
    title_short: "My Custom Source"
    home_url: "https://example.org/my-source"
    is_curated: true
    has_classification: true

  - id: 1002
    parent: "https://releases.example.org/sfga/"
    title_short: "VASCAN"
    is_curated: true
    has_classification: true
```

The `parent` field must point to the **immediate parent directory** (or
URL) of the SFGA file — GNdb searches only that location, not
subdirectories. Without the correct parent path the file will not be
found.

| Field | Description |
| ----- | ----------- |
| `id` | Unique integer ID. Use `< 1000` for standard sources, `>= 1000` for custom |
| `parent` | Immediate parent directory of the SFGA file (local path or URL) |
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

[Datasette]: https://datasette.io
[Dmitry Mozzherin]: https://github.com/dimus
[GNparser]: https://github.com/gnames/gnparser
[GNverifier]: https://github.com/gnames/gnverifier
[MIT License]: LICENSE
[SF]: https://github.com/sfborg/sf
[SFGA]: https://github.com/sfborg/sfga
[SQLite DB viewer]: https://sqlitebrowser.org/
[releases page]: https://github.com/gnames/gndb/releases/latest
