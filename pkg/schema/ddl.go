package schema

import (
	"fmt"
	"reflect"
	"strings"
)

// generateDDL creates a CREATE TABLE statement from struct tags.
func generateDDL(model interface{}, tableName string) string {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	var columns []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		ddlTag := field.Tag.Get("ddl")

		if dbTag != "" && ddlTag != "" {
			columns = append(columns, fmt.Sprintf("    %s %s", dbTag, ddlTag))
		}
	}

	ddl := fmt.Sprintf("CREATE TABLE %s (\n%s\n);",
		tableName,
		strings.Join(columns, ",\n"))

	return ddl
}

// DataSource DDL methods
func (ds DataSource) TableDDL() string {
	return generateDDL(ds, "data_sources")
}

func (ds DataSource) IndexDDL() []string {
	return []string{
		"CREATE INDEX idx_datasources_uuid ON data_sources(uuid);",
		"CREATE INDEX idx_datasources_title_short ON data_sources(title_short);",
	}
}

func (ds DataSource) TableName() string {
	return "data_sources"
}

// NameString DDL methods
func (ns NameString) TableDDL() string {
	return generateDDL(ns, "name_strings")
}

func (ns NameString) IndexDDL() []string {
	return []string{
		"CREATE UNIQUE INDEX idx_namestrings_canonical_simple ON name_strings(canonical_simple);",
		"CREATE INDEX idx_namestrings_canonical_full ON name_strings(canonical_full);",
		"CREATE INDEX idx_namestrings_name_trgm ON name_strings USING GIST (name_string gist_trgm_ops(siglen=256));",
		"CREATE INDEX idx_namestrings_cardinality ON name_strings(cardinality) WHERE cardinality > 0;",
	}
}

func (ns NameString) TableName() string {
	return "name_strings"
}

// Taxon DDL methods
func (t Taxon) TableDDL() string {
	return generateDDL(t, "taxa")
}

func (t Taxon) IndexDDL() []string {
	return []string{
		"CREATE UNIQUE INDEX idx_taxa_datasource_localid ON taxa(data_source_id, local_id);",
		"CREATE INDEX idx_taxa_name_id ON taxa(name_id);",
		"CREATE INDEX idx_taxa_parent_id ON taxa(parent_id);",
		"CREATE INDEX idx_taxa_rank ON taxa(rank);",
	}
}

func (t Taxon) TableName() string {
	return "taxa"
}

// Synonym DDL methods
func (s Synonym) TableDDL() string {
	return generateDDL(s, "synonyms")
}

func (s Synonym) IndexDDL() []string {
	return []string{
		"CREATE INDEX idx_synonyms_name_id ON synonyms(name_id);",
		"CREATE INDEX idx_synonyms_taxon_id ON synonyms(taxon_id);",
		"CREATE INDEX idx_synonyms_datasource ON synonyms(data_source_id);",
	}
}

func (s Synonym) TableName() string {
	return "synonyms"
}

// VernacularName DDL methods
func (vn VernacularName) TableDDL() string {
	return generateDDL(vn, "vernacular_names")
}

func (vn VernacularName) IndexDDL() []string {
	return []string{
		"CREATE INDEX idx_vernacular_taxon_id ON vernacular_names(taxon_id);",
		"CREATE INDEX idx_vernacular_name_trgm ON vernacular_names USING GIST (name_string gist_trgm_ops(siglen=256));",
	}
}

func (vn VernacularName) TableName() string {
	return "vernacular_names"
}

// Reference DDL methods
func (r Reference) TableDDL() string {
	return generateDDL(r, "references")
}

func (r Reference) IndexDDL() []string {
	return []string{
		"CREATE UNIQUE INDEX idx_references_datasource_localid ON references(data_source_id, local_id);",
		"CREATE INDEX idx_references_doi ON references(doi) WHERE doi IS NOT NULL;",
	}
}

func (r Reference) TableName() string {
	return "references"
}

// SchemaVersion DDL methods
func (sv SchemaVersion) TableDDL() string {
	return generateDDL(sv, "schema_versions")
}

func (sv SchemaVersion) IndexDDL() []string {
	return []string{} // No secondary indexes needed
}

func (sv SchemaVersion) TableName() string {
	return "schema_versions"
}

// NameStringOccurrence DDL methods
func (occ NameStringOccurrence) TableDDL() string {
	return generateDDL(occ, "name_string_occurrences")
}

func (occ NameStringOccurrence) IndexDDL() []string {
	return []string{
		"CREATE INDEX idx_occurrences_namestring ON name_string_occurrences(name_string_id);",
		"CREATE INDEX idx_occurrences_datasource ON name_string_occurrences(data_source_id);",
		"CREATE INDEX idx_occurrences_taxon_id ON name_string_occurrences(taxon_id);",
		"CREATE INDEX idx_occurrences_record_type ON name_string_occurrences(record_type);",
	}
}

func (occ NameStringOccurrence) TableName() string {
	return "name_string_occurrences"
}
