# sources.yaml Specification

User-provided configuration file for `gndb populate` command.

## Purpose

Defines which SFGA data sources to import and provides metadata that supplements or overrides SFGA metadata.

## Command Usage

```bash
# Import all sources
gndb populate --config sources.yaml

# Import specific IDs
gndb populate --config sources.yaml --sources 1,3,5

# Exclude specific IDs  
gndb populate --config sources.yaml --exclude 2,4

# Import only official sources (ID < 1000)
gndb populate --config sources.yaml --sources main

# Import only custom sources (ID ≥ 1000)
gndb populate --config sources.yaml --exclude main

# Mix: official + specific custom
gndb populate --config sources.yaml --sources main,1001,1005
```

## File Structure

```yaml
data_sources:
  - file: <path-or-url>
    # Optional fields...

import:
  # Optional import settings...

logging:
  # Optional logging settings...
```

## Required Fields

### `file`
Path or URL to SFGA file (required for each data source)

```yaml
# Local path
file: /data/0001_col_2025-10-03_v2024.1.sqlite.zip

# URL
file: https://opendata.globalnames.org/sfga/latest/0001_col_2025-10-03_v2024.1.sqlite.zip

# Minimal filename
file: /data/1001.sql
```

### `id`
Data source ID (required, extracted from filename or specified explicitly)

```yaml
# From filename: 0001_col_2025-10-03.sqlite → id: 1
file: /data/0001_col_2025-10-03.sqlite

# Explicit
file: /data/my-data.sqlite
id: 1001
```

**ID Convention** (not enforced):
- `< 1000`: Official sources from main gnverifier site
- `≥ 1000`: Custom user sources
- Users free to use any IDs

## Optional Override Fields

These override SFGA metadata if provided:

```yaml
title: "My Custom Title"           # Override SFGA col__title
title_short: "CustomDS"             # Override SFGA col__alias
description: "Custom description"   # Override SFGA col__description  
home_url: "https://example.org"     # Override SFGA col__url
```

## Additional Fields (Not in SFGA)

### Data Download URL
```yaml
data_url: "https://example.org/download"
```

### Data Source Type
```yaml
data_source_type: "taxonomic"  # or "nomenclatural"
# If not provided, inferred from data structure
```

### Curation Flags
```yaml
is_curated: true              # Manually curated by experts
is_auto_curated: false        # Automatically validated
has_classification: true      # Has hierarchical taxonomy
```

### Outlink Configuration
```yaml
is_outlink_ready: true
outlink_url: "https://example.org/taxon/{}"
outlink_id_field: "record_id"  # or: local_id, global_id, name_id, canonical
```

## Fields NEVER in YAML

These are always extracted from SFGA or filename (too volatile for YAML):

- `version`: From SFGA `col__version` or filename `v{version}`
- `release_date`: From SFGA `col__issued` or filename `YYYY-MM-DD`

## Import Settings

```yaml
import:
  batch_size: 5000              # Records per batch insert (default: 5000)
  concurrent_jobs: 4            # Parallel processing jobs (default: 4)
  prefer_flat_classification: false  # Use flat vs hierarchical (default: false)
```

## Logging Settings

```yaml
logging:
  show_progress: true           # Show progress bars (default: true)
  log_level: "info"             # debug, info, warn, error (default: info)
```

## Complete Example

```yaml
# sources.yaml - Complete example

data_sources:
  # Official source - minimal config
  - file: /data/0001_col_2025-10-03_v2024.1.sqlite.zip
    title_short: "CoL"
  
  # Official source from URL
  - file: https://opendata.globalnames.org/sfga/latest/0042_itis.sqlite.zip
    title_short: "ITIS"
  
  # Custom source - full config
  - file: /data/1001_my-herbarium_2025-10-03_v1.0.sql.zip
    title: "My Institution Herbarium"
    title_short: "MyHerb"
    description: "Regional plant collection"
    home_url: "https://myinst.org/herbarium"
    data_url: "https://myinst.org/herbarium/download"
    data_source_type: "taxonomic"
    is_curated: true
    has_classification: true
    is_outlink_ready: true
    outlink_url: "https://myinst.org/specimen/{}"
    outlink_id_field: "record_id"
  
  # Minimal custom source
  - file: /data/1002.sql
    title_short: "LocalList"

import:
  batch_size: 5000
  concurrent_jobs: 4
  prefer_flat_classification: false

logging:
  show_progress: true
  log_level: "info"
```

## Template (Commented Out)

Users can duplicate and customize this template:

```yaml
# - file: /data/{id}_{name}_{date}_v{version}.{format}[.zip]
#   id: 1000                      # ≥ 1000 for custom sources
#   title: "Full Title"
#   title_short: "ShortName"
#   description: "Detailed description"
#   home_url: "https://example.org"
#   data_url: "https://example.org/download"
#   data_source_type: "taxonomic"  # or "nomenclatural"
#   is_curated: false
#   is_auto_curated: false
#   has_classification: false
#   is_outlink_ready: false
#   outlink_url: "https://example.org/{}"
#   outlink_id_field: "record_id"
```

## Validation Rules

1. **ID required**: From filename, SFGA, or YAML
2. **File must exist or be accessible URL**
3. **File format**: Must be `.sql`, `.sqlite`, `.sql.zip`, or `.sqlite.zip`
4. **Date format**: If provided, must be `YYYY-MM-DD`
5. **Outlink validation**: If `is_outlink_ready=true`, must have `outlink_url` with `{}`

## Edge Cases

### Empty SFGA Metadata
Some SFGA files have empty metadata table (created from simple name lists).
All fields must come from YAML in this case.

### Non-Standard Filenames
If filename doesn't match standard pattern:
- Try to extract ID
- Require missing fields in YAML
- Warn about non-standard format

### Title Short Fallback
If `title_short` not provided:
1. Use SFGA `col__alias` if exists
2. Truncate `col__title` with "..." if too long
3. Signals user to provide explicit `title_short`
