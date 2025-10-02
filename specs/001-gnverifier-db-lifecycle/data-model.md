# Data Model: GNverifier Database Schema

**Based on**: gnames/gnames migrations/gnames.hcl (existing production schema)  
**Date**: 2025-10-02  
**Status**: Adapted from existing schema

## Overview

The GNverifier database schema is based on the existing gnames production schema with optimizations for 100M+ scientific names. This document defines the PostgreSQL schema that will be created by `gndb create` and populated by `gndb populate`.

---

## Core Tables

### 1. canonicals

Stores canonical forms of scientific names (standardized name strings without authorship).

```sql
CREATE TABLE canonicals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE INDEX idx_canonicals_name ON canonicals (name);
CREATE INDEX idx_canonicals_name_trgm ON canonicals USING GIST (name gist_trgm_ops(siglen=256));
```

**Purpose**: Fast canonical name lookups for verification  
**Volume**: ~80-90M rows (deduplicated from 100M name strings)

### 2. canonical_fulls

Stores full canonical forms including infraspecific epithets and ranks.

```sql
CREATE TABLE canonical_fulls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE INDEX idx_canonical_fulls_name ON canonical_fulls (name);
CREATE INDEX idx_canonical_fulls_name_trgm ON canonical_fulls USING GIST (name gist_trgm_ops(siglen=256));
```

**Purpose**: Detailed canonical matching including subspecies/varieties  
**Volume**: ~90-95M rows

### 3. canonical_stems

Stores stemmed versions of canonical names for fuzzy matching.

```sql
CREATE TABLE canonical_stems (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE INDEX idx_canonical_stems_name ON canonical_stems (name);
CREATE INDEX idx_canonical_stems_name_trgm ON canonical_stems USING GIST (name gist_trgm_ops(siglen=128));
```

**Purpose**: Linguistic stemming for better fuzzy matches  
**Volume**: ~70-80M rows (fewer due to stemming collisions)

---

## 4. data_sources

Metadata about taxonomic data sources (e.g., Catalog of Life, GBIF, custom sources).

```sql
CREATE TABLE data_sources (
    id SMALLINT PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    title VARCHAR(255) NOT NULL,
    title_short VARCHAR(50),
    version VARCHAR(50),
    revision_date TEXT,
    doi VARCHAR(50),
    citation TEXT,
    authors TEXT,
    description TEXT,
    website_url VARCHAR(255),
    data_url VARCHAR(255),
    outlink_url TEXT,
    is_outlink_ready BOOLEAN DEFAULT false,
    is_curated BOOLEAN DEFAULT false,
    is_auto_curated BOOLEAN DEFAULT false,
    has_taxon_data BOOLEAN DEFAULT false,
    record_count INTEGER DEFAULT 0,
    vern_record_count INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_data_sources_title ON data_sources (title);
CREATE INDEX idx_data_sources_is_curated ON data_sources (is_curated) WHERE is_curated = true;
```

**Purpose**: Track origin of name data and enable source filtering  
**Volume**: 100-500 rows (number of taxonomic databases)

**Key Fields**:
- `is_curated`: Human-verified taxonomic data
- `is_auto_curated`: Algorithm-curated data
- `has_taxon_data`: Whether source includes full taxonomic hierarchy
- `record_count`: Number of scientific names from this source
- `vern_record_count`: Number of vernacular names

---

## 5. name_string_indices (Core Entity)

Links name strings to their occurrences in data sources with taxonomic context.

```sql
CREATE TABLE name_string_indices (
    data_source_id INTEGER NOT NULL,
    record_id VARCHAR(255) NOT NULL,
    name_string_id UUID NOT NULL,
    outlink_id VARCHAR(255),
    global_id VARCHAR(255),
    name_id VARCHAR(255),
    local_id VARCHAR(255),
    code_id SMALLINT,
    rank VARCHAR(255),
    taxonomic_status VARCHAR(255),
    accepted_record_id VARCHAR(255),
    classification TEXT,
    classification_ids TEXT,
    classification_ranks TEXT,
    
    -- Performance: denormalized fields added during restructure
    canonical_id UUID,
    canonical_full_id UUID,
    canonical_stem_id UUID,
    
    PRIMARY KEY (data_source_id, record_id)
);

-- Indexes
CREATE INDEX idx_nsi_name_string_id ON name_string_indices (name_string_id);
CREATE INDEX idx_nsi_accepted_record_id ON name_string_indices (accepted_record_id);
CREATE INDEX idx_nsi_canonical_id ON name_string_indices (canonical_id);
CREATE INDEX idx_nsi_data_source_id ON name_string_indices (data_source_id);
```

**Purpose**: Primary lookup table for name verification and reconciliation  
**Volume**: 200M rows (multiple occurrences per name across sources)

**Note**: No partitioning initially. Current production handles 60M occurrences at 2000 names/sec (exceeds 1000 names/sec requirement). Partitioning can be added later if needed at larger scale.

**Key Fields**:
- `record_id`: Unique identifier within data source
- `name_string_id`: Links to name_strings table
- `accepted_record_id`: For synonyms, points to accepted name
- `classification`: Pipe-delimited taxonomic hierarchy (Kingdom|Phylum|...|Species)
- `classification_ids`: Corresponding IDs for classification
- `classification_ranks`: Corresponding ranks

---

## 6. name_strings

Complete scientific name strings with parsed components and metadata.

```sql
CREATE TABLE name_strings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    cardinality SMALLINT,
    canonical_id UUID,
    canonical_full_id UUID,
    canonical_stem_id UUID,
    virus BOOLEAN DEFAULT false,
    bacteria BOOLEAN DEFAULT false,
    surrogate BOOLEAN DEFAULT false,
    parse_quality SMALLINT,
    
    -- Parsed components (from gnparser)
    parsed_data JSONB,
    year INTEGER,
    authors TEXT,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (canonical_id) REFERENCES canonicals(id),
    FOREIGN KEY (canonical_full_id) REFERENCES canonical_fulls(id),
    FOREIGN KEY (canonical_stem_id) REFERENCES canonical_stems(id)
);

CREATE INDEX idx_name_strings_name ON name_strings (name);
CREATE INDEX idx_name_strings_name_trgm ON name_strings USING GIST (name gist_trgm_ops(siglen=256));
CREATE INDEX idx_name_strings_canonical_id ON name_strings (canonical_id);
CREATE INDEX idx_name_strings_year ON name_strings (year) WHERE year IS NOT NULL;
CREATE INDEX idx_name_strings_parse_quality ON name_strings (parse_quality);
```

**Purpose**: Master table of all unique scientific name strings  
**Volume**: 100M rows

**Key Fields**:
- `name`: Full scientific name string (with authorship)
- `cardinality`: Number of name parts (1=uninomial, 2=binomial, 3=trinomial, etc.)
- `canonical_id`, `canonical_full_id`, `canonical_stem_id`: Links to canonical tables
- `virus`, `bacteria`: Taxonomic group flags
- `surrogate`: Whether name is placeholder/surrogate
- `parse_quality`: gnparser quality score (1-3, higher is better)
- `parsed_data`: Full gnparser JSON output

---

## 7. name_strings_alphas

Alphabetical index for name string browsing and pagination.

```sql
CREATE TABLE name_strings_alphas (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    name_string_id UUID NOT NULL,
    
    FOREIGN KEY (name_string_id) REFERENCES name_strings(id)
);

CREATE INDEX idx_name_strings_alphas_name ON name_strings_alphas (name);
```

**Purpose**: Fast alphabetical navigation (A-Z browsing)  
**Volume**: 100M rows (same as name_strings)

---

## 8. verification_runs

Tracks batch name verification requests.

```sql
CREATE TABLE verification_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    input_hash VARCHAR(64) UNIQUE NOT NULL,
    data_source_ids SMALLINT[] NOT NULL,
    name_strings TEXT[] NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    result_count INTEGER
);

CREATE INDEX idx_verification_runs_input_hash ON verification_runs (input_hash);
CREATE INDEX idx_verification_runs_created_at ON verification_runs (created_at);
```

**Purpose**: Cache verification results, track API usage  
**Volume**: Variable (grows with usage)

---

## 9. vernacular_strings

Common/vernacular names in various languages.

```sql
CREATE TABLE vernacular_strings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    language VARCHAR(10) NOT NULL,
    name_string_id UUID NOT NULL,
    data_source_id SMALLINT NOT NULL,
    
    FOREIGN KEY (name_string_id) REFERENCES name_strings(id),
    FOREIGN KEY (data_source_id) REFERENCES data_sources(id)
);

-- Language-specific partial indexes
CREATE INDEX idx_vernacular_english ON vernacular_strings (name, name_string_id) 
WHERE language = 'en';

CREATE INDEX idx_vernacular_spanish ON vernacular_strings (name, name_string_id) 
WHERE language = 'es';

CREATE INDEX idx_vernacular_french ON vernacular_strings (name, name_string_id) 
WHERE language = 'fr';

CREATE INDEX idx_vernacular_german ON vernacular_strings (name, name_string_id) 
WHERE language = 'de';

-- Trigram indexes per major language
CREATE INDEX idx_vernacular_english_trgm ON vernacular_strings USING GIST (name gist_trgm_ops(siglen=128))
WHERE language = 'en';

CREATE INDEX idx_vernacular_spanish_trgm ON vernacular_strings USING GIST (name gist_trgm_ops(siglen=128))
WHERE language = 'es';

-- General language index
CREATE INDEX idx_vernacular_language ON vernacular_strings (language);
CREATE INDEX idx_vernacular_name_string_id ON vernacular_strings (name_string_id);
```

**Purpose**: Vernacular name search by language  
**Volume**: 20M rows

---

## 10. vernacular_string_indices

Links vernacular names to their occurrences in data sources.

```sql
CREATE TABLE vernacular_string_indices (
    id BIGSERIAL PRIMARY KEY,
    data_source_id SMALLINT NOT NULL,
    record_id VARCHAR(255) NOT NULL,
    vernacular_string_id UUID NOT NULL,
    name_string_id UUID NOT NULL,
    language VARCHAR(10),
    locality VARCHAR(255),
    country_code VARCHAR(3),
    
    FOREIGN KEY (vernacular_string_id) REFERENCES vernacular_strings(id),
    FOREIGN KEY (name_string_id) REFERENCES name_strings(id),
    FOREIGN KEY (data_source_id) REFERENCES data_sources(id),
    
    UNIQUE (data_source_id, record_id)
);

CREATE INDEX idx_vsi_vernacular_string_id ON vernacular_string_indices (vernacular_string_id);
CREATE INDEX idx_vsi_name_string_id ON vernacular_string_indices (name_string_id);
CREATE INDEX idx_vsi_language ON vernacular_string_indices (language);
CREATE INDEX idx_vsi_country_code ON vernacular_string_indices (country_code);
```

**Purpose**: Track vernacular name occurrences across sources  
**Volume**: 20M rows

---

## Support Tables

### 11. schema_migrations

Tracks Atlas migration versions.

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT NOW()
);
```

### 12. version

Stores current database/data version.

```sql
CREATE TABLE version (
    id SERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## Materialized Views (Created During Restructure)

### mv_name_lookup

Denormalized view for fast name verification.

```sql
CREATE MATERIALIZED VIEW mv_name_lookup AS
SELECT 
    ns.id AS name_string_id,
    ns.name AS name_string,
    c.name AS canonical,
    cf.name AS canonical_full,
    cs.name AS canonical_stem,
    ns.year,
    ns.authors,
    ns.parse_quality,
    nsi.data_source_id,
    nsi.taxonomic_status,
    nsi.rank,
    nsi.accepted_record_id,
    ds.title AS data_source_title,
    ds.is_curated
FROM name_strings ns
LEFT JOIN canonicals c ON ns.canonical_id = c.id
LEFT JOIN canonical_fulls cf ON ns.canonical_full_id = cf.id
LEFT JOIN canonical_stems cs ON ns.canonical_stem_id = cs.id
JOIN name_string_indices nsi ON ns.id = nsi.name_string_id
JOIN data_sources ds ON nsi.data_source_id = ds.id;

CREATE UNIQUE INDEX idx_mv_name_lookup_id ON mv_name_lookup (name_string_id, data_source_id);
CREATE INDEX idx_mv_name_lookup_canonical ON mv_name_lookup (canonical);
CREATE INDEX idx_mv_name_lookup_canonical_trgm ON mv_name_lookup USING GIST (canonical gist_trgm_ops(siglen=256));
```

---

## Data Relationships

```
data_sources (1) ─── (M) name_string_indices
                              │
                              └─ (M) name_strings (1) ─┬─ (1) canonicals
                                      │                 ├─ (1) canonical_fulls
                                      │                 └─ (1) canonical_stems
                                      │
                                      └─ (M) vernacular_strings
                                              │
                                              └─ (M) vernacular_string_indices
```

---

## Extensions Required

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
```

---

## Performance Characteristics

| Table | Rows | Indexes | Est. Size |
|-------|------|---------|-----------|
| name_strings | 100M | 6 | 50GB |
| canonicals | 90M | 2 | 20GB |
| canonical_fulls | 95M | 2 | 22GB |
| canonical_stems | 80M | 2 | 18GB |
| name_string_indices | 200M | 4 | 120GB |
| vernacular_strings | 20M | 8 | 5GB |
| vernacular_string_indices | 20M | 4 | 4GB |
| data_sources | 500 | 2 | <1MB |
| **Total** | **605M+** | **30** | **~240GB** |

---

## Schema Creation Order

1. Extensions (`uuid-ossp`, `pg_trgm`)
2. Core lookup tables (`data_sources`)
3. Canonical tables (`canonicals`, `canonical_fulls`, `canonical_stems`)
4. Name tables (`name_strings`, `name_strings_alphas`)
5. Index tables (`name_string_indices`)
6. Vernacular tables (`vernacular_strings`, `vernacular_string_indices`)
7. Support tables (`schema_migrations`, `version`)
8. Primary key indexes (auto-created)
9. Secondary indexes (deferred to restructure phase)
10. Materialized views (deferred to restructure phase)

---

## Population Order (Foreign Key Dependencies)

1. `data_sources`
2. `canonicals`, `canonical_fulls`, `canonical_stems` (parallel)
3. `name_strings` (references canonical tables)
4. `name_strings_alphas` (references name_strings)
5. `name_string_indices` (references name_strings, data_sources)
6. `vernacular_strings` (references name_strings, data_sources)
7. `vernacular_string_indices` (references vernacular_strings, name_strings, data_sources)

---

## Index Creation Strategy

### During Create Phase
- Primary keys only
- UNIQUE constraints on critical columns

### During Restructure Phase
- All B-tree indexes for foreign keys
- GiST trigram indexes for fuzzy matching
- Partial indexes for language-specific queries
- Materialized views
- Statistics updates (`ANALYZE`)

**Rationale**: Disabling indexes during bulk load improves insert performance by 10x.

---

## Next Steps

1. Generate Atlas migration files from this schema
2. Create Go models matching table structures
3. Define interfaces for database operations
4. Write contract tests for each table/operation

