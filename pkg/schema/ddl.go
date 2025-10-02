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
		"CREATE INDEX idx_name_strings_canonical ON name_strings(canonical_id);",
		"CREATE INDEX idx_name_strings_canonical_full ON name_strings(canonical_full_id);",
		"CREATE INDEX idx_name_strings_canonical_stem ON name_strings(canonical_stem_id);",
	}
}

func (ns NameString) TableName() string {
	return "name_strings"
}

// Canonical DDL methods
func (c Canonical) TableDDL() string {
	return generateDDL(c, "canonicals")
}

func (c Canonical) IndexDDL() []string {
	return []string{}
}

func (c Canonical) TableName() string {
	return "canonicals"
}

// CanonicalFull DDL methods
func (cf CanonicalFull) TableDDL() string {
	return generateDDL(cf, "canonical_fulls")
}

func (cf CanonicalFull) IndexDDL() []string {
	return []string{}
}

func (cf CanonicalFull) TableName() string {
	return "canonical_fulls"
}

// CanonicalStem DDL methods
func (cs CanonicalStem) TableDDL() string {
	return generateDDL(cs, "canonical_stems")
}

func (cs CanonicalStem) IndexDDL() []string {
	return []string{}
}

func (cs CanonicalStem) TableName() string {
	return "canonical_stems"
}

// NameStringIndex DDL methods
func (nsi NameStringIndex) TableDDL() string {
	return generateDDL(nsi, "name_string_indices")
}

func (nsi NameStringIndex) IndexDDL() []string {
	return []string{
		"CREATE INDEX idx_name_string_indices_idx ON name_string_indices(data_source_id, record_id, name_string_id);",
		"CREATE INDEX idx_name_string_indices_name_string_id ON name_string_indices(name_string_id);",
		"CREATE INDEX idx_name_string_indices_accepted_record_id ON name_string_indices(accepted_record_id);",
	}
}

func (nsi NameStringIndex) TableName() string {
	return "name_string_indices"
}

// Word DDL methods
func (w Word) TableDDL() string {
	return generateDDL(w, "words")
}

func (w Word) IndexDDL() []string {
	return []string{
		"CREATE INDEX idx_words_modified ON words(modified);",
	}
}

func (w Word) TableName() string {
	return "words"
}

// WordNameString DDL methods
func (wns WordNameString) TableDDL() string {
	return generateDDL(wns, "word_name_strings")
}

func (wns WordNameString) IndexDDL() []string {
	return []string{
		"CREATE INDEX idx_word_name_strings_word ON word_name_strings(word_id);",
		"CREATE INDEX idx_word_name_strings_name ON word_name_strings(name_string_id);",
	}
}

func (wns WordNameString) TableName() string {
	return "word_name_strings"
}

// VernacularString DDL methods
func (vs VernacularString) TableDDL() string {
	return generateDDL(vs, "vernacular_strings")
}

func (vs VernacularString) IndexDDL() []string {
	return []string{
		"CREATE INDEX idx_vern_str_name_idx ON vernacular_strings(name);",
	}
}

func (vs VernacularString) TableName() string {
	return "vernacular_strings"
}

// VernacularStringIndex DDL methods
func (vsi VernacularStringIndex) TableDDL() string {
	return generateDDL(vsi, "vernacular_string_indices")
}

func (vsi VernacularStringIndex) IndexDDL() []string {
	return []string{
		"CREATE INDEX idx_vernacular_string_idx_idx ON vernacular_string_indices(data_source_id, record_id, lang_code);",
		"CREATE INDEX idx_vernacular_string_id ON vernacular_string_indices(vernacular_string_id);",
	}
}

func (vsi VernacularStringIndex) TableName() string {
	return "vernacular_string_indices"
}

// SchemaVersion DDL methods
func (sv SchemaVersion) TableDDL() string {
	return generateDDL(sv, "schema_versions")
}

func (sv SchemaVersion) IndexDDL() []string {
	return []string{}
}

func (sv SchemaVersion) TableName() string {
	return "schema_versions"
}
