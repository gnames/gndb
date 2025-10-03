# SFGA Metadata Research

Research on what metadata SFGA format provides vs what GNverifier needs for data sources.

## Usage Scenarios

**User provides `sources.yaml`** - This is the required configuration file for data import.

Location: User's choice (e.g., `~/my-data/sources.yaml`, `/data/gndb/sources.yaml`)

The file contains stable metadata for data sources. Users can:

1. **Edit source-by-source**: Update individual data sources as needed
2. **Update all at once**: Bulk update when upgrading
3. **Selective import**: Use CLI flags to include/exclude specific sources
   - `gndb populate --sources 1,3,5`: Import only specified IDs
   - `gndb populate --exclude 2,4`: Import all except specified IDs

**Version volatility problem**: Most metadata is stable (titles, URLs, etc.) but `version` and `release_date` change with each update. **Solution**: Always infer these from filename or SFGA metadata, never store in YAML.

## Command Usage

```bash
# Import all data sources from sources.yaml
gndb populate --config sources.yaml

# Import specific data sources by ID
gndb populate --config sources.yaml --sources 1,3,5

# Exclude specific data sources
gndb populate --config sources.yaml --exclude 2,4

# SFGA files can be local paths or URLs
# All specified in sources.yaml
```

## Avoiding Duplicate Databases

To prevent accidental database duplication:

1. **Provide example sources.yaml** with all official data sources pre-configured (commented out)
2. **Clear documentation** with examples for custom data sources  
3. **ID namespace separation**:
   - **Official sources**: ID < 1000 (stable IDs from main gnverifier site)
   - **Custom sources**: ID ≥ 1000 (user-assigned, not globally stable)
4. **Reusability**: Users maintain their sources.yaml across upgrades; use include/exclude flags for selective imports

## SFGA Metadata Table Schema

Source: https://github.com/sfborg/sfga/blob/6c4eec1caa68869bc640ff85a970ce58aeae5fb9/schema.sql#L17

SFGA provides the following metadata fields in the `metadata` table:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `col__id` | INTEGER | Primary key, auto-increment | 1, 2, 3... |
| `col__doi` | TEXT | Digital Object Identifier | "10.15468/abc123" |
| `col__title` | TEXT | Full title of the data source (NOT NULL) | "Catalogue of Life" |
| `col__alias` | TEXT | Alternative name/alias | "CoL" |
| `col__description` | TEXT | Detailed description | "Global catalog of..." |
| `col__issued` | TEXT | Release/publication date | "2024-01-15" |
| `col__version` | TEXT | Version string | "2024.1", "v3.5" |
| `col__keywords` | TEXT | Comma-separated keywords | "taxonomy,plants,animals" |
| `col__geographic_scope` | TEXT | Geographic coverage | "Global", "Europe" |
| `col__taxonomic_scope` | TEXT | Taxonomic coverage | "All life", "Plants only" |
| `col__temporal_scope` | TEXT | Temporal coverage | "Modern", "Fossil" |
| `col__confidence` | INTEGER | Confidence level (0-100?) | 95 |
| `col__completeness` | INTEGER | Completeness level (0-100?) | 80 |
| `col__license` | TEXT | License information | "CC BY 4.0" |
| `col__url` | TEXT | Homepage URL | "https://catalogueoflife.org" |
| `col__logo` | TEXT | Logo URL or path | "https://..." |
| `col__label` | TEXT | Display label | "CoL" |
| `col__citation` | TEXT | Citation text | "Cite as: ..." |
| `col__private` | INTEGER | Boolean flag (private data?) | 0 or 1 |

## What SFGA Provides ✅

Good coverage for:
- **Core identification**: `col__id`
- **Titles**: `col__title`, `col__alias`, `col__label`
- **Description**: `col__description`
- **Version**: `col__version`, `col__issued` (date)
- **URLs**: `col__url` (homepage)
- **Licensing**: `col__license`, `col__citation`
- **Scope**: Geographic, taxonomic, temporal
- **Quality indicators**: `col__confidence`, `col__completeness`
- **Publication info**: `col__doi`

## What SFGA Does NOT Provide ❌

Missing fields needed by GNverifier:

### 1. Global Identification
- **NOT NEEDED** - UUID and DOI concerns removed
- **ID namespace convention** (by convention only, not enforced):
  - ID < 1000: Official data sources from main gnverifier site
  - ID ≥ 1000: Custom user sources
  - Users are free to use any IDs they want

### 2. Short Title for UI Display
- **title_short**: Compact name for tables/dropdowns (e.g., "ITIS", "WoRMS", "CoL")
- **Optional field** with fallback strategy:
  1. Use YAML `title_short` if provided
  2. Use SFGA `col__alias` if exists
  3. Truncate `col__title` with "..." if too long (signals user to provide title_short)
- In to-gn, `TitleShort` is hardcoded in Go code for 50+ data sources

### 3. Data Download URL
- **data_url**: Where to download the data (optional)
- SFGA has `col__url` (homepage) but not download link
- Useful for users to find updates

### 4. Data Source Type
- **data_source_type**: "taxonomic" vs "nomenclatural"
- **Can be inferred** from data structure:
  - No classification AND no accepted_record_id → nomenclatural
  - Has classification OR has accepted_record_id → taxonomic
- Optional override in YAML if inference incorrect

### 5. Curation Quality Flags
- **is_curated**: Manually curated by experts (important quality signal)
- **is_auto_curated**: Automatically checked/validated
- **has_classification**: Contains hierarchical taxonomy
- SFGA has `col__confidence` and `col__completeness` (numeric) but not these semantic flags
- In to-gn, these are hardcoded in Go for specific data source IDs
- **Note**: `has_classification=true` always indicates taxonomic (but taxonomic can lack classification)

### 6. Outlink Configuration
- **Outlink URL template**: How to construct links back to original records
  - Example: `"https://www.catalogueoflife.org/data/taxon/{}"`
- **Outlink ID field**: Which field to use (record_id, local_id, global_id, name_id, canonical)
- **Outlink ready flag**: Whether outlinks can be generated
- Not in SFGA - must be configured per data source
- In to-gn, includes Go functions for complex URL construction
- **Challenge**: Need to support both simple YAML templates AND Go functions for complex cases

### 7. Field Explanations (Meta)
- **OUT OF SCOPE** for data source metadata
- These fields explain what IDs mean but are not part of data source configuration
- Will not be included in YAML 

## Comparison with to-gn Implementation

In to-gn (see `/Users/dimus/code/golang/to-gn/pkg/ds/ds.go`):

### Hardcoded in Go for 50+ Data Sources:
- `Title` (sometimes overrides SFGA)
- `TitleShort` (ALWAYS needed, not in SFGA)
- `UUID` (not in SFGA)
- `HomeURL` (may override SFGA `col__url`)
- `DataURL` (not in SFGA)
- `IsOutlinkReady`, `OutlinkURL`, `OutlinkID` function (not in SFGA)
- `Meta` explanations (not in SFGA)

### Hardcoded in Arrays by ID:
- `curatedAry`: List of manually curated data source IDs
- `autoCuratedAry`: List of auto-curated data source IDs  
- `hasClassifAry`: List of data sources with classification

## Recommendations for gndb populate

### Approach: YAML Configuration File

Allow users to provide missing metadata via YAML since:

1. **Users can edit YAML** but may not be comfortable with Go code
2. **Local/custom data sources** won't be in to-gn's hardcoded list
3. **Flexibility** to override SFGA values when needed
4. **Documentation** via comments in the YAML template

### Required Fields in sources.yaml:
- `file`: Path or URL to SFGA file (REQUIRED)
  - Local: `/data/0001_col_2025-10-03_v2024.1.sqlite.zip`
  - URL: `https://example.org/data/0001_col_2025-10-03_v2024.1.sqlite.zip`
- `id`: Data source ID (REQUIRED - from SFGA col__id, filename pattern, or YAML)
  - **ID namespace**: Official sources (< 1000) vs custom sources (≥ 1000)
  - Official IDs must be stable across gnverifier installations
  - Custom IDs need not be globally stable

### Optional Fields (with Smart Defaults):
- `title_short`: Short display name
  - **Fallback**: SFGA col__alias → truncate col__title with "..." (signals user to provide it)
- `title`: Override SFGA `col__title`
- `description`: Override SFGA `col__description`
- `home_url`: Override SFGA `col__url`

### Fields NEVER in YAML (Always Inferred):
- `version`: Always from SFGA `col__version` or filename (too volatile for YAML)
- `release_date`: Always from SFGA `col__issued` or filename (too volatile for YAML)

### Important Edge Case:
- **Empty SFGA metadata**: Some SFGA files created from simple name lists have completely empty metadata tables
- Must handle gracefully - all fields come from YAML in this case

### Additional Fields (Not in SFGA):
- `uuid`: UUID for global identification
- `data_url`: Download URL
- `data_source_type`: "taxonomic" or "nomenclatural"
- `is_curated`: Manual curation flag
- `is_auto_curated`: Auto-validation flag
- `has_classification`: Hierarchical classification flag
- `is_outlink_ready`: Can generate outlinks
- `outlink_url`: URL template with `{}`
- `outlink_id_field`: Which field to use
- `meta`: Field explanations (record_id, local_id, global_id, name_id)

## Implementation Strategy

1. **Extract ID first** (required):
   - From SFGA `col__id`
   - From filename pattern `{id}-{name}.{ext}`
   - From YAML (error if missing from all sources)

2. **Read SFGA metadata** - Get all available fields (handle empty metadata gracefully)

3. **Apply YAML configuration** - Override SFGA values or provide missing fields

4. **Infer version/date** - Always from SFGA or filename, never from YAML

5. **Generate/derive optional fields**:
   - `title_short`: YAML → col__alias → truncate col__title with "..."
   - `uuid`: YAML → col__doi (prefer DOI) → generate new UUID
   - `data_source_type`: YAML → infer from data (has classification/accepted_record)

6. **Validate required fields** - Ensure ID exists and is in correct namespace

## Example YAML (Minimal)

```yaml
data_sources:
  - file: /data/001-col-2024.zip
    # ID extracted from filename: 001
    # Everything else from SFGA
    title_short: "CoL"  # Only this needed if SFGA has rest
```

## Example YAML (Custom Local Source)

```yaml
data_sources:
  - file: /data/my-herbarium.zip
    # No ID
    # DIMUS: we would try to infer it from the file name or break.
    title: "My Herbarium Collection"
    title_short: "MyHerb"
    description: "Local plant specimens"
    version: "1.0"
    release_date: "2024-10-03"
    uuid: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    data_source_type: "taxonomic"
    is_curated: true
    has_classification: true
    is_outlink_ready: true
    outlink_url: "https://myherb.org/specimen/{}"
    outlink_id_field: "record_id"
```

In the end of the file we leave commented-out template that people can duplicate as many sometimes
as they need, uncomment and populate.

## Next Steps

1. Design YAML schema with clear documentation
2. Implement loader that reads SFGA first, then applies YAML overrides
3. Create validation logic to ensure required fields exist
4. Generate example YAML template with extensive comments
5. Support both new data sources and overrides of existing ones
