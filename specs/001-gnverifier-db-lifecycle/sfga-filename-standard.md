# SFGA Filename Standard

Standard filename format for SFGA files to enable automatic metadata extraction.

## Format

```
{id}_{name}_{date}_v{version}.{format}[.zip]
```

## Components

### Required

**`{id}`** - Data source ID (4 digits with leading zeros)
- `0001-0999`: Official sources from main gnverifier site (stable IDs)
- `1000+`: Custom user sources (not globally stable)
- Examples: `0001`, `0042`, `1001`, `1234`

**`{format}`** - SFGA file format
- `sql`: SQLite dump (text SQL statements)
- `sqlite`: SQLite binary database

### Optional

**`{name}`** - Human-readable name (ignored by gndb, for user convenience only)
- Any characters except `_`
- Examples: `col`, `itis`, `my-herbarium`, `local-plants`

**`{date}`** - Release date in ISO 8601 format
- Format: `YYYY-MM-DD`
- Examples: `2025-10-03`, `2024-01-15`

**`v{version}`** - Version string
- Must start with `v`
- Any format after `v`: `v1.2.3`, `v2024.1`, `v1.0-beta`
- Examples: `v1.2.3`, `v2024.1`, `v3.5-rc1`

**`.zip`** - Compression (file may or may not be compressed)

## Location

File can be:
- **Local path**: `/data/0001_col_2025-10-03_v2024.1.sqlite.zip`
- **URL**: `https://example.org/data/0001_col_2025-10-03_v2024.1.sqlite.zip`

## Examples

### Full Format
```
0001_col_2025-10-03_v1.2.3.sqlite.zip
0001_col_2025-10-03_v1.2.3.sql.zip
```

### Compressed SQL Dump
```
0042_itis_2024-06-15_v2024.6.sql.zip
```

### Uncompressed SQLite Binary
```
0042_itis_2024-06-15_v2024.6.sqlite
```

### No Version
```
0001_col_2025-10-03.sqlite.zip
0001_col_2025-10-03.sql
```

### No Date
```
0001_col_v1.2.3.sqlite.zip
```

### Minimal (ID Only)
```
0001.sqlite.zip
0001.sql
```

### Custom User Sources (ID ≥ 1000)
```
1001_my-herbarium_2025-10-03_v2.0.sql.zip
1234_local-plants.sqlite
```

### URLs
```
https://example.org/sfga/0001_col_2025-10-03_v2024.1.sqlite.zip
http://data.myinst.edu/1001_herbarium_2025-01-15_v1.0.sql.zip
```

## Parsing Rules

1. **ID**: First 4 digits (required)
   - Match pattern: `^\d{4}`
   - Error if not found

2. **Format**: File extension before optional `.zip`
   - Match pattern: `\.(sql|sqlite)(\.zip)?$`
   - Error if not `sql` or `sqlite`

3. **Date**: First occurrence of ISO 8601 date
   - Match pattern: `\d{4}-\d{2}-\d{2}`
   - Optional (can be in SFGA metadata instead)

4. **Version**: Text after `_v` until next `_` or `.`
   - Match pattern: `_v([^_.]+)`
   - Optional (can be in SFGA metadata instead)

5. **Name**: Everything between first `_` and date/version/format
   - For user convenience only, ignored by gndb
   - Not used for data source title

## Fallback Order for Metadata

| Field | Priority |
|-------|----------|
| **ID** | filename → YAML → ERROR if missing |
| **Release Date** | filename → SFGA col__issued → YAML → current date |
| **Version** | filename → SFGA col__version → YAML → "unknown" |
| **Title** | SFGA col__title → YAML → ERROR if missing |

**Note**: Filename `{name}` component is NOT used for title - it's only for human readability.

## Validation

Files must:
- Start with 4-digit ID
- End with `.sql`, `.sqlite`, `.sql.zip`, or `.sqlite.zip`
- Have valid date format if date present (`YYYY-MM-DD`)
- Be accessible (local file exists OR URL is reachable)

## Non-Standard Filenames

If filename doesn't match standard:
- Extract what's possible (try to find ID pattern)
- Require missing fields in YAML
- Warn user about non-standard format
- Suggest renaming to standard format

Example:
```
col-2024.zip  # Non-standard

Warning: Non-standard filename 'col-2024.zip'
Error: Cannot extract data source ID from filename
Solution: Rename to '0001_col_2024-01-15_v2024.1.sqlite.zip' OR provide 'id' in YAML
```
