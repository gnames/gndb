package schema_test

import (
	"strings"
	"testing"

	"github.com/gnames/gndb/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDataSourceTableDDL tests DDL generation for DataSource model
func TestDataSourceTableDDL(t *testing.T) {
	ds := schema.DataSource{}
	ddl := ds.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE data_sources")

	// Should have primary key
	assert.Contains(t, ddl, "id BIGSERIAL PRIMARY KEY")

	// Should have required fields
	assert.Contains(t, ddl, "uuid TEXT UNIQUE NOT NULL")
	assert.Contains(t, ddl, "title TEXT NOT NULL")
	assert.Contains(t, ddl, "title_short TEXT NOT NULL")
	assert.Contains(t, ddl, "version TEXT NOT NULL")
	assert.Contains(t, ddl, "sfga_version TEXT NOT NULL")

	// Should have CHECK constraint for data_source_type
	assert.Contains(t, ddl, "CHECK (data_source_type IN ('taxonomic', 'nomenclatural'))")
}

// TestDataSourceTableName tests TableName method
func TestDataSourceTableName(t *testing.T) {
	ds := schema.DataSource{}
	assert.Equal(t, "data_sources", ds.TableName())
}

// TestNameStringTableDDL tests DDL generation for NameString model
func TestNameStringTableDDL(t *testing.T) {
	ns := schema.NameString{}
	ddl := ns.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE name_strings")

	// Should have primary key
	assert.Contains(t, ddl, "id BIGSERIAL PRIMARY KEY")

	// Should have canonical fields
	assert.Contains(t, ddl, "canonical_simple TEXT")
	assert.Contains(t, ddl, "canonical_full TEXT")
	assert.Contains(t, ddl, "canonical_stemmed TEXT")

	// Should have parse_quality CHECK constraint
	assert.Contains(t, ddl, "parse_quality SMALLINT CHECK (parse_quality BETWEEN 0 AND 4)")

	// Should have cardinality CHECK constraint
	assert.Contains(t, ddl, "cardinality SMALLINT CHECK (cardinality BETWEEN 0 AND 3)")

	// Should have boolean fields
	assert.Contains(t, ddl, "virus BOOLEAN DEFAULT FALSE")
	assert.Contains(t, ddl, "bacteria BOOLEAN DEFAULT FALSE")
}

// TestNameStringIndexDDL tests index generation for NameString model
func TestNameStringIndexDDL(t *testing.T) {
	ns := schema.NameString{}
	indexes := ns.IndexDDL()

	// Should return multiple indexes
	require.NotEmpty(t, indexes, "NameString should have secondary indexes")

	// Convert to single string for easier searching
	allIndexes := strings.Join(indexes, "\n")

	// Should have unique index on canonical_simple
	assert.Contains(t, allIndexes, "CREATE UNIQUE INDEX")
	assert.Contains(t, allIndexes, "canonical_simple")

	// Should have GiST trigram index for fuzzy matching
	assert.Contains(t, allIndexes, "USING GIST")
	assert.Contains(t, allIndexes, "gist_trgm_ops")
	assert.Contains(t, allIndexes, "siglen=256")
}

// TestTaxonTableDDL tests DDL generation for Taxon model
func TestTaxonTableDDL(t *testing.T) {
	taxon := schema.Taxon{}
	ddl := taxon.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE taxa")

	// Should have primary key
	assert.Contains(t, ddl, "id BIGSERIAL PRIMARY KEY")

	// Should have foreign keys
	assert.Contains(t, ddl, "REFERENCES data_sources(id)")
	assert.Contains(t, ddl, "REFERENCES name_strings(id)")

	// Should have taxonomic hierarchy fields
	assert.Contains(t, ddl, "kingdom TEXT")
	assert.Contains(t, ddl, "phylum TEXT")
	assert.Contains(t, ddl, "class TEXT")
	assert.Contains(t, ddl, "order_name TEXT") // 'order' is reserved keyword
	assert.Contains(t, ddl, "family TEXT")
	assert.Contains(t, ddl, "genus TEXT")
	assert.Contains(t, ddl, "species TEXT")
}

// TestSynonymTableDDL tests DDL generation for Synonym model
func TestSynonymTableDDL(t *testing.T) {
	syn := schema.Synonym{}
	ddl := syn.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE synonyms")

	// Should have primary key
	assert.Contains(t, ddl, "id BIGSERIAL PRIMARY KEY")

	// Should have foreign keys to data_sources, taxa, and name_strings
	assert.Contains(t, ddl, "REFERENCES data_sources(id)")
	assert.Contains(t, ddl, "REFERENCES taxa(id)")
	assert.Contains(t, ddl, "REFERENCES name_strings(id)")
}

// TestVernacularNameTableDDL tests DDL generation for VernacularName model
func TestVernacularNameTableDDL(t *testing.T) {
	vn := schema.VernacularName{}
	ddl := vn.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE vernacular_names")

	// Should have primary key
	assert.Contains(t, ddl, "id BIGSERIAL PRIMARY KEY")

	// Should have foreign keys
	assert.Contains(t, ddl, "REFERENCES data_sources(id)")
	assert.Contains(t, ddl, "REFERENCES taxa(id)")

	// Should have name and language fields
	assert.Contains(t, ddl, "name_string TEXT NOT NULL")
	assert.Contains(t, ddl, "language_code TEXT")
}

// TestSchemaVersionTableDDL tests DDL generation for SchemaVersion model
func TestSchemaVersionTableDDL(t *testing.T) {
	sv := schema.SchemaVersion{}
	ddl := sv.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE schema_versions")

	// Should have version as primary key (TEXT type)
	assert.Contains(t, ddl, "version TEXT PRIMARY KEY")

	// Should have timestamp field
	assert.Contains(t, ddl, "applied_at TIMESTAMP DEFAULT NOW()")
}

// TestReferenceTableDDL tests DDL generation for Reference model
func TestReferenceTableDDL(t *testing.T) {
	ref := schema.Reference{}
	ddl := ref.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE references")

	// Should have primary key
	assert.Contains(t, ddl, "id BIGSERIAL PRIMARY KEY")

	// Should have foreign key to data_sources
	assert.Contains(t, ddl, "REFERENCES data_sources(id)")

	// Should have citation fields
	assert.Contains(t, ddl, "citation TEXT NOT NULL")
	assert.Contains(t, ddl, "doi TEXT")
}

// TestNameStringOccurrenceTableDDL tests DDL generation for NameStringOccurrence model
func TestNameStringOccurrenceTableDDL(t *testing.T) {
	occ := schema.NameStringOccurrence{}
	ddl := occ.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE name_string_occurrences")

	// Should have primary key
	assert.Contains(t, ddl, "id BIGSERIAL PRIMARY KEY")

	// Should have foreign keys
	assert.Contains(t, ddl, "REFERENCES name_strings(id)")
	assert.Contains(t, ddl, "REFERENCES data_sources(id)")

	// Should have denormalized data_source_title field
	assert.Contains(t, ddl, "data_source_title TEXT NOT NULL")

	// Should have record_type CHECK constraint
	assert.Contains(t, ddl, "CHECK (record_type IN ('accepted', 'synonym', 'vernacular'))")
}

// TestAllModelsImplementDDLGenerator tests that all models implement the DDLGenerator interface
func TestAllModelsImplementDDLGenerator(t *testing.T) {
	models := []schema.DDLGenerator{
		&schema.DataSource{},
		&schema.NameString{},
		&schema.Taxon{},
		&schema.Synonym{},
		&schema.VernacularName{},
		&schema.Reference{},
		&schema.SchemaVersion{},
		&schema.NameStringOccurrence{},
	}

	for _, model := range models {
		// Each model should return valid DDL
		ddl := model.TableDDL()
		assert.NotEmpty(t, ddl, "TableDDL should return non-empty string")
		assert.Contains(t, ddl, "CREATE TABLE", "DDL should contain CREATE TABLE")

		// Each model should return a table name
		tableName := model.TableName()
		assert.NotEmpty(t, tableName, "TableName should return non-empty string")

		// IndexDDL should return a slice (may be empty for some models)
		indexes := model.IndexDDL()
		assert.NotNil(t, indexes, "IndexDDL should return non-nil slice")
	}
}
