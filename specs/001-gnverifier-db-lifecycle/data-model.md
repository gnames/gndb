# Data Model: GNverifier Database Schema

**Date**: 2025-10-02  
**Status**: Design Complete  
**Approach**: Go models for schema creation and type-safe SQL mapping

---

## Overview

The GNverifier database schema is defined through Go struct models (inspired by gnidump's model.go pattern). This approach provides:
- **Type safety**: Go structs map directly to PostgreSQL tables
- **Schema generation**: DDL automatically generated from struct tags
- **Query mapping**: SQL results scan directly into Go types
- **Maintainability**: Single source of truth for schema definition

---

## Core Entities

### 1. DataSource

Represents external nomenclature data sources in SFGA format.

**Go Model**:
```go
type DataSource struct {
    ID              int64     `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
    UUID            string    `db:"uuid" ddl:"TEXT UNIQUE NOT NULL"`
    Title           string    `db:"title" ddl:"TEXT NOT NULL"`
    TitleShort      string    `db:"title_short" ddl:"TEXT NOT NULL"`
    Version         string    `db:"version" ddl:"TEXT NOT NULL"`
    ReleaseDate     time.Time `db:"release_date" ddl:"TIMESTAMP NOT NULL"`
    HomeURL         string    `db:"home_url" ddl:"TEXT"`
    Description     string    `db:"description" ddl:"TEXT"`
    DataSourceType  string    `db:"data_source_type" ddl:"TEXT CHECK (data_source_type IN ('taxonomic', 'nomenclatural'))"`
    RecordCount     int64     `db:"record_count" ddl:"BIGINT DEFAULT 0"`
    SFGAVersion     string    `db:"sfga_version" ddl:"TEXT NOT NULL"`
    ImportedAt      time.Time `db:"imported_at" ddl:"TIMESTAMP DEFAULT NOW()"`
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE TABLE data_sources (
    id BIGSERIAL PRIMARY KEY,
    uuid TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    title_short TEXT NOT NULL,
    version TEXT NOT NULL,
    release_date TIMESTAMP NOT NULL,
    home_url TEXT,
    description TEXT,
    data_source_type TEXT CHECK (data_source_type IN ('taxonomic', 'nomenclatural')),
    record_count BIGINT DEFAULT 0,
    sfga_version TEXT NOT NULL,
    imported_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_datasources_uuid ON data_sources(uuid);
CREATE INDEX idx_datasources_title_short ON data_sources(title_short);
```

**Relationships**:
- One-to-many with NameStringOccurrence (a data source has many name occurrences)

---

### 2. NameString

Represents parsed scientific name strings with canonical forms for matching.

**Go Model**:
```go
type NameString struct {
    ID                 int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
    NameString         string `db:"name_string" ddl:"TEXT NOT NULL"`
    CanonicalSimple    string `db:"canonical_simple" ddl:"TEXT"`
    CanonicalFull      string `db:"canonical_full" ddl:"TEXT"`
    CanonicalStemmed   string `db:"canonical_stemmed" ddl:"TEXT"`
    Authorship         string `db:"authorship" ddl:"TEXT"`
    Year               string `db:"year" ddl:"TEXT"`
    ParseQuality       int    `db:"parse_quality" ddl:"SMALLINT CHECK (parse_quality BETWEEN 0 AND 4)"`
    Cardinality        int    `db:"cardinality" ddl:"SMALLINT CHECK (cardinality BETWEEN 0 AND 3)"`
    Virus              bool   `db:"virus" ddl:"BOOLEAN DEFAULT FALSE"`
    Bacteria           bool   `db:"bacteria" ddl:"BOOLEAN DEFAULT FALSE"`
    ParserVersion      string `db:"parser_version" ddl:"TEXT NOT NULL"`
    UpdatedAt          time.Time `db:"updated_at" ddl:"TIMESTAMP DEFAULT NOW()"`
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE TABLE name_strings (
    id BIGSERIAL PRIMARY KEY,
    name_string TEXT NOT NULL,
    canonical_simple TEXT,
    canonical_full TEXT,
    canonical_stemmed TEXT,
    authorship TEXT,
    year TEXT,
    parse_quality SMALLINT CHECK (parse_quality BETWEEN 0 AND 4),
    cardinality SMALLINT CHECK (cardinality BETWEEN 0 AND 3),
    virus BOOLEAN DEFAULT FALSE,
    bacteria BOOLEAN DEFAULT FALSE,
    parser_version TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Created during populate phase (no indexes initially)
-- Added during restructure phase:
CREATE UNIQUE INDEX idx_namestrings_canonical_simple ON name_strings(canonical_simple);
CREATE INDEX idx_namestrings_canonical_full ON name_strings(canonical_full);
CREATE INDEX idx_namestrings_name_trgm ON name_strings USING GIST (name_string gist_trgm_ops(siglen=256));
CREATE INDEX idx_namestrings_cardinality ON name_strings(cardinality) WHERE cardinality > 0;
```

**Relationships**:
- One-to-many with NameStringOccurrence
- One-to-many with Taxon (via taxon.name_id)
- One-to-many with Synonym (via synonym.name_id)

**Canonical Forms**:
- `canonical_simple`: Uninomial/binomial without authorship (e.g., "Homo sapiens")
- `canonical_full`: Complete canonical with infraspecific (e.g., "Homo sapiens sapiens")
- `canonical_stemmed`: Stemmed for fuzzy matching (e.g., "hom sapien")

---

### 3. NameStringOccurrence

Tracks where and how often a name string appears across data sources (denormalized for performance).

**Go Model**:
```go
type NameStringOccurrence struct {
    ID                  int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
    NameStringID        int64  `db:"name_string_id" ddl:"BIGINT NOT NULL REFERENCES name_strings(id) ON DELETE CASCADE"`
    DataSourceID        int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
    DataSourceTitle     string `db:"data_source_title" ddl:"TEXT NOT NULL"` // Denormalized
    TaxonID             string `db:"taxon_id" ddl:"TEXT"` // SFGA taxon.col__id
    RecordType          string `db:"record_type" ddl:"TEXT CHECK (record_type IN ('accepted', 'synonym', 'vernacular'))"`
    LocalID             string `db:"local_id" ddl:"TEXT"` // Original ID from source
    OutlinkID           string `db:"outlink_id" ddl:"TEXT"` // External reference ID
    AcceptedNameID      int64  `db:"accepted_name_id" ddl:"BIGINT REFERENCES name_strings(id)"` // For synonyms
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE TABLE name_string_occurrences (
    id BIGSERIAL PRIMARY KEY,
    name_string_id BIGINT NOT NULL REFERENCES name_strings(id) ON DELETE CASCADE,
    data_source_id BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    data_source_title TEXT NOT NULL,
    taxon_id TEXT,
    record_type TEXT CHECK (record_type IN ('accepted', 'synonym', 'vernacular')),
    local_id TEXT,
    outlink_id TEXT,
    accepted_name_id BIGINT REFERENCES name_strings(id)
);

-- Restructure phase indexes:
CREATE INDEX idx_occurrences_namestring ON name_string_occurrences(name_string_id);
CREATE INDEX idx_occurrences_datasource ON name_string_occurrences(data_source_id);
CREATE INDEX idx_occurrences_taxon_id ON name_string_occurrences(taxon_id);
CREATE INDEX idx_occurrences_record_type ON name_string_occurrences(record_type);
```

**Design Decision**: Denormalized `data_source_title` avoids expensive joins during reconciliation.

---

### 4. Taxon

Represents taxonomic hierarchy and accepted scientific names.

**Go Model**:
```go
type Taxon struct {
    ID              int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
    DataSourceID    int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
    LocalID         string `db:"local_id" ddl:"TEXT NOT NULL"` // SFGA col__id
    NameID          int64  `db:"name_id" ddl:"BIGINT NOT NULL REFERENCES name_strings(id)"`
    ParentID        string `db:"parent_id" ddl:"TEXT"` // SFGA col__parent_id
    AcceptedID      string `db:"accepted_id" ddl:"TEXT"` // For provisional taxa
    Rank            string `db:"rank" ddl:"TEXT"`
    Kingdom         string `db:"kingdom" ddl:"TEXT"`
    Phylum          string `db:"phylum" ddl:"TEXT"`
    Class           string `db:"class" ddl:"TEXT"`
    OrderName       string `db:"order_name" ddl:"TEXT"` // 'order' is reserved
    Family          string `db:"family" ddl:"TEXT"`
    Genus           string `db:"genus" ddl:"TEXT"`
    Species         string `db:"species" ddl:"TEXT"`
    TaxonomicStatus string `db:"taxonomic_status" ddl:"TEXT"`
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE TABLE taxa (
    id BIGSERIAL PRIMARY KEY,
    data_source_id BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    local_id TEXT NOT NULL,
    name_id BIGINT NOT NULL REFERENCES name_strings(id),
    parent_id TEXT,
    accepted_id TEXT,
    rank TEXT,
    kingdom TEXT,
    phylum TEXT,
    class TEXT,
    order_name TEXT,
    family TEXT,
    genus TEXT,
    species TEXT,
    taxonomic_status TEXT
);

CREATE UNIQUE INDEX idx_taxa_datasource_localid ON taxa(data_source_id, local_id);
CREATE INDEX idx_taxa_name_id ON taxa(name_id);
CREATE INDEX idx_taxa_parent_id ON taxa(parent_id);
CREATE INDEX idx_taxa_rank ON taxa(rank);
```

**Relationships**:
- Many-to-one with DataSource
- Many-to-one with NameString
- Self-referential via parent_id (hierarchical taxonomy)
- One-to-many with Synonym
- One-to-many with VernacularName

---

### 5. Synonym

Maps alternative scientific names to accepted taxa.

**Go Model**:
```go
type Synonym struct {
    ID              int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
    DataSourceID    int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
    TaxonID         int64  `db:"taxon_id" ddl:"BIGINT NOT NULL REFERENCES taxa(id) ON DELETE CASCADE"`
    NameID          int64  `db:"name_id" ddl:"BIGINT NOT NULL REFERENCES name_strings(id)"`
    Status          string `db:"status" ddl:"TEXT"` // e.g., 'synonym', 'misapplied', 'homotypic'
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE TABLE synonyms (
    id BIGSERIAL PRIMARY KEY,
    data_source_id BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    taxon_id BIGINT NOT NULL REFERENCES taxa(id) ON DELETE CASCADE,
    name_id BIGINT NOT NULL REFERENCES name_strings(id),
    status TEXT
);

CREATE INDEX idx_synonyms_name_id ON synonyms(name_id);
CREATE INDEX idx_synonyms_taxon_id ON synonyms(taxon_id);
CREATE INDEX idx_synonyms_datasource ON synonyms(data_source_id);
```

**Relationships**:
- Many-to-one with DataSource
- Many-to-one with Taxon (accepted taxon)
- Many-to-one with NameString (synonym name)

---

### 6. VernacularName

Common names in various languages associated with scientific taxa.

**Go Model**:
```go
type VernacularName struct {
    ID              int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
    DataSourceID    int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
    TaxonID         int64  `db:"taxon_id" ddl:"BIGINT NOT NULL REFERENCES taxa(id) ON DELETE CASCADE"`
    NameString      string `db:"name_string" ddl:"TEXT NOT NULL"`
    LanguageCode    string `db:"language_code" ddl:"TEXT"` // ISO 639-1/2
    Country         string `db:"country" ddl:"TEXT"` // ISO 3166-1
    Locality        string `db:"locality" ddl:"TEXT"`
    Transliteration string `db:"transliteration" ddl:"TEXT"`
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE TABLE vernacular_names (
    id BIGSERIAL PRIMARY KEY,
    data_source_id BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    taxon_id BIGINT NOT NULL REFERENCES taxa(id) ON DELETE CASCADE,
    name_string TEXT NOT NULL,
    language_code TEXT,
    country TEXT,
    locality TEXT,
    transliteration TEXT
);

CREATE INDEX idx_vernacular_taxon_id ON vernacular_names(taxon_id);
CREATE INDEX idx_vernacular_name_trgm ON vernacular_names USING GIST (name_string gist_trgm_ops(siglen=256));

-- Partial indexes per language (created during restructure):
CREATE INDEX idx_vernacular_english ON vernacular_names(name_string) WHERE language_code = 'en';
CREATE INDEX idx_vernacular_spanish ON vernacular_names(name_string) WHERE language_code = 'es';
CREATE INDEX idx_vernacular_chinese ON vernacular_names(name_string) WHERE language_code IN ('zh', 'cmn');
-- Additional languages added based on data distribution
```

**Relationships**:
- Many-to-one with DataSource
- Many-to-one with Taxon

---

### 7. Reference

Bibliographic citations for taxonomic data.

**Go Model**:
```go
type Reference struct {
    ID              int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
    DataSourceID    int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
    LocalID         string `db:"local_id" ddl:"TEXT NOT NULL"` // SFGA col__id
    Citation        string `db:"citation" ddl:"TEXT NOT NULL"`
    Author          string `db:"author" ddl:"TEXT"`
    Title           string `db:"title" ddl:"TEXT"`
    Year            string `db:"year" ddl:"TEXT"`
    DOI             string `db:"doi" ddl:"TEXT"`
    Link            string `db:"link" ddl:"TEXT"`
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE TABLE references (
    id BIGSERIAL PRIMARY KEY,
    data_source_id BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    local_id TEXT NOT NULL,
    citation TEXT NOT NULL,
    author TEXT,
    title TEXT,
    year TEXT,
    doi TEXT,
    link TEXT
);

CREATE UNIQUE INDEX idx_references_datasource_localid ON references(data_source_id, local_id);
CREATE INDEX idx_references_doi ON references(doi) WHERE doi IS NOT NULL;
```

---

### 8. SchemaVersion

Tracks database schema migrations for Atlas.

**Go Model**:
```go
type SchemaVersion struct {
    Version     string    `db:"version" ddl:"TEXT PRIMARY KEY"`
    Description string    `db:"description" ddl:"TEXT"`
    AppliedAt   time.Time `db:"applied_at" ddl:"TIMESTAMP DEFAULT NOW()"`
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE TABLE schema_versions (
    version TEXT PRIMARY KEY,
    description TEXT,
    applied_at TIMESTAMP DEFAULT NOW()
);
```

---

## Materialized Views (Restructure Phase)

### 9. mv_name_lookup

Pre-joined view for fast name reconciliation (eliminates 3-table joins).

**Go Model**:
```go
type NameLookup struct {
    NameStringID        int64  `db:"name_string_id"`
    CanonicalSimple     string `db:"canonical_simple"`
    CanonicalFull       string `db:"canonical_full"`
    DataSourceID        int64  `db:"data_source_id"`
    DataSourceTitle     string `db:"data_source_title"`
    TaxonID             int64  `db:"taxon_id"`
    RecordType          string `db:"record_type"`
    AcceptedNameID      int64  `db:"accepted_name_id"`
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE MATERIALIZED VIEW mv_name_lookup AS
SELECT 
    ns.id AS name_string_id,
    ns.canonical_simple,
    ns.canonical_full,
    nso.data_source_id,
    nso.data_source_title,
    nso.taxon_id::BIGINT AS taxon_id,
    nso.record_type,
    nso.accepted_name_id
FROM name_strings ns
JOIN name_string_occurrences nso ON ns.id = nso.name_string_id;

CREATE UNIQUE INDEX idx_mv_name_lookup_composite ON mv_name_lookup(name_string_id, data_source_id);
CREATE INDEX idx_mv_name_lookup_canonical ON mv_name_lookup(canonical_simple);
```

---

### 10. mv_vernacular_by_language

Aggregated vernacular names by language for fast language-specific searches.

**Go Model**:
```go
type VernacularByLanguage struct {
    LanguageCode    string `db:"language_code"`
    NameString      string `db:"name_string"`
    TaxonIDs        []int64 `db:"taxon_ids"` // Array of matching taxon IDs
    RecordCount     int64  `db:"record_count"`
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE MATERIALIZED VIEW mv_vernacular_by_language AS
SELECT 
    language_code,
    name_string,
    array_agg(DISTINCT taxon_id) AS taxon_ids,
    count(*) AS record_count
FROM vernacular_names
GROUP BY language_code, name_string;

CREATE INDEX idx_mv_vernacular_lang_name ON mv_vernacular_by_language(language_code, name_string);
```

---

### 11. mv_synonym_map

Denormalized synonym resolution (synonym → accepted name).

**Go Model**:
```go
type SynonymMap struct {
    SynonymNameID       int64  `db:"synonym_name_id"`
    SynonymCanonical    string `db:"synonym_canonical"`
    AcceptedNameID      int64  `db:"accepted_name_id"`
    AcceptedCanonical   string `db:"accepted_canonical"`
    DataSourceID        int64  `db:"data_source_id"`
    DataSourceTitle     string `db:"data_source_title"`
}
```

**PostgreSQL DDL** (generated):
```sql
CREATE MATERIALIZED VIEW mv_synonym_map AS
SELECT 
    syn_ns.id AS synonym_name_id,
    syn_ns.canonical_simple AS synonym_canonical,
    acc_ns.id AS accepted_name_id,
    acc_ns.canonical_simple AS accepted_canonical,
    s.data_source_id,
    ds.title_short AS data_source_title
FROM synonyms s
JOIN name_strings syn_ns ON s.name_id = syn_ns.id
JOIN taxa t ON s.taxon_id = t.id
JOIN name_strings acc_ns ON t.name_id = acc_ns.id
JOIN data_sources ds ON s.data_source_id = ds.id;

CREATE INDEX idx_mv_synonym_map_synonym ON mv_synonym_map(synonym_name_id);
CREATE INDEX idx_mv_synonym_map_canonical ON mv_synonym_map(synonym_canonical);
```

---

## SFGA to PostgreSQL Mapping

| SFGA Table | Go Model | PostgreSQL Table | Import Order |
|------------|----------|------------------|--------------|
| metadata | DataSource | data_sources | 1 |
| reference | Reference | references | 2 |
| name | NameString | name_strings | 3 |
| taxon | Taxon | taxa | 4 |
| synonym | Synonym | synonyms | 5 |
| vernacular | VernacularName | vernacular_names | 6 |
| (derived) | NameStringOccurrence | name_string_occurrences | 7 |

**Import Order Rationale**: Foreign key dependencies dictate order (references before taxa, names before occurrences).

---

## Type Mappings

| Go Type | PostgreSQL Type | SFGA SQLite Type |
|---------|-----------------|------------------|
| `int64` | `BIGINT` | `INTEGER` |
| `string` | `TEXT` | `TEXT` |
| `time.Time` | `TIMESTAMP` | `TEXT` (parsed) |
| `bool` | `BOOLEAN` | `INTEGER` (0/1) |
| `[]int64` | `BIGINT[]` | - (aggregated) |

**Special Handling**:
- SQLite `INTEGER` → PostgreSQL `BIGINT` (IDs can exceed int32 range)
- SQLite `TEXT` dates → PostgreSQL `TIMESTAMP` (parse ISO 8601)
- Reserved keywords: `order` → `order_name`

---

## DDL Generation Pattern

**pkg/schema/ddl.go**:
```go
func (ns NameString) TableDDL() string {
    // Reflect on struct tags to generate CREATE TABLE
    return generateDDL(ns)
}

func (ns NameString) IndexDDL() []string {
    // Return slice of CREATE INDEX statements
    return []string{
        "CREATE UNIQUE INDEX idx_namestrings_canonical_simple ON name_strings(canonical_simple);",
        "CREATE INDEX idx_namestrings_name_trgm ON name_strings USING GIST (name_string gist_trgm_ops(siglen=256));",
    }
}
```

**Usage in gndb create**:
```go
models := []interface{}{
    DataSource{}, NameString{}, Taxon{}, Synonym{}, VernacularName{}, Reference{},
}

for _, model := range models {
    ddl := model.(DDLGenerator).TableDDL()
    db.Exec(ddl)
}
```

---

## Query Mapping Pattern

**Type-safe query result scanning**:
```go
func FindByCanonical(db *pgxpool.Pool, canonical string) ([]NameLookup, error) {
    query := `SELECT * FROM mv_name_lookup WHERE canonical_simple = $1`
    rows, err := db.Query(context.Background(), query, canonical)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []NameLookup
    for rows.Next() {
        var nl NameLookup
        err := rows.Scan(
            &nl.NameStringID, &nl.CanonicalSimple, &nl.CanonicalFull,
            &nl.DataSourceID, &nl.DataSourceTitle, &nl.TaxonID,
            &nl.RecordType, &nl.AcceptedNameID,
        )
        if err != nil {
            return nil, err
        }
        results = append(results, nl)
    }
    return results, nil
}
```

---

## Schema Evolution Strategy

**Atlas Migrations**:
1. Modify Go model structs (add/remove fields)
2. Generate migration: `atlas migrate diff --env gndb`
3. Review generated SQL in `migrations/{timestamp}_{name}.sql`
4. Apply: `gndb migrate` (calls Atlas SDK)

**Example Migration** (add index):
```sql
-- migrations/20251002120000_add_cardinality_index.sql
CREATE INDEX idx_namestrings_cardinality 
ON name_strings(cardinality) 
WHERE cardinality > 0;
```

---

## Performance Characteristics

| Operation | Index Used | Expected Time |
|-----------|-----------|---------------|
| Exact canonical match | B-tree on canonical_simple | <5ms (index-only scan) |
| Fuzzy name search | GiST trigram on name_string | <100ms (partial match) |
| Vernacular by language | Partial index on language_code | <10ms |
| Synonym resolution | mv_synonym_map canonical index | <5ms |
| Bulk import (100M records) | No indexes during import | ~3 hours |
| Index rebuild (100M records) | All indexes | ~2 hours |

---

## Validation Rules

**Enforced via CHECK constraints** (in Go model tags):
- `parse_quality`: 0-4 (0=unparsed, 4=perfect)
- `cardinality`: 0-3 (0=uninomial, 1=binomial, 2=trinomial, 3=quadrinomial+)
- `record_type`: 'accepted', 'synonym', 'vernacular'
- `data_source_type`: 'taxonomic', 'nomenclatural'

**Application-level validation** (in pkg/):
- SFGA version compatibility
- Foreign key existence before insert
- Required fields (e.g., canonical_simple for reconciliation)

---

## Next Steps

1. **Generate interface contracts** in `/contracts/`:
   - DatabaseOperator.go (schema creation, migrations)
   - SFGAReader.go (data source streaming)
   - Importer.go (batch insert operations)

2. **Create contract tests** verifying:
   - DDL generation produces valid PostgreSQL
   - Query scanning maps correctly to Go types
   - Materialized view definitions compile

3. **Document quickstart.md** with end-to-end integration test using small SFGA sample

---

**Data Model Complete**: Ready for contract generation (Phase 1, step 2).
