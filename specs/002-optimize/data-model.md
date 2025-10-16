# Data Model for Optimize Database Performance

This feature does not introduce new tables or change the existing table schemas. Instead, it focuses on adding a performance-enhancing materialized view and word decomposition tables.

## Materialized View: `verification`

The `verification` materialized view denormalizes data from `name_string_indices` and `name_strings` tables for fast verification queries. This is the primary optimization artifact used by gnverifier.

### SQL Definition

```sql
CREATE MATERIALIZED VIEW verification AS
WITH taxon_names AS (
  SELECT nsi.data_source_id, nsi.record_id, nsi.name_string_id, ns.name
    FROM name_string_indices nsi
      JOIN name_strings ns
        ON nsi.name_string_id = ns.id
)
SELECT nsi.data_source_id, nsi.record_id, nsi.name_string_id,
  ns.name, nsi.name_id, nsi.code_id, ns.year, ns.cardinality, ns.canonical_id,
  ns.virus, ns.bacteria, ns.parse_quality, nsi.local_id, nsi.outlink_id,
  nsi.taxonomic_status, nsi.accepted_record_id, tn.name_string_id as
  accepted_name_id, tn.name as accepted_name, nsi.classification,
  nsi.classification_ranks, nsi.classification_ids
  FROM name_string_indices nsi
    JOIN name_strings ns ON ns.id = nsi.name_string_id
    LEFT JOIN taxon_names tn
      ON nsi.data_source_id = tn.data_source_id AND
         nsi.accepted_record_id = tn.record_id
  WHERE
    (
      ns.canonical_id is not NULL AND
      surrogate != TRUE AND
      (bacteria != TRUE OR parse_quality < 3)
    ) OR ns.virus = TRUE
```

### Indexes on Verification View

Three indexes are created to optimize common query patterns:

```sql
CREATE INDEX verification_canonical_id_idx ON verification (canonical_id);
CREATE INDEX verification_name_string_id_idx ON verification (name_string_id);
CREATE INDEX verification_year_idx ON verification (year);
```

## Word Decomposition Tables

The optimize process populates two existing tables for word-level fuzzy matching:

### `words` Table
Already defined in schema (pkg/schema/models.go). Populated during optimization with:
- `normalized`: Word normalized by gnparser (for sorting)
- `modified`: Heavy-normalized word (for matching)
- `type_id`: Word type from gnparser

### `word_name_strings` Table
Already defined in schema (pkg/schema/models.go). Junction table linking:
- `word_id`: Reference to words table
- `name_string_id`: Reference to name_strings table
- `canonical_id`: Reference to canonicals table

## Optimization Operations Summary

The optimize command performs these data operations:

1. **Reparse all name_strings**: Update canonical IDs using latest gnparser
2. **Normalize vernacular languages**: Convert to 3-letter ISO codes
3. **Remove orphan records**: Clean unreferenced name_strings and canonicals
4. **Populate words tables**: Extract and link words for fuzzy matching
5. **Create verification view**: Build materialized view with indexes
6. **VACUUM ANALYZE**: Update statistics and reclaim space

No schema changes are made - only data processing and view creation.