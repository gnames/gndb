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

	// Should have SMALLINT primary key (historical hard-coded IDs)
	assert.Contains(t, ddl, "id SMALLINT PRIMARY KEY")

	// Should have UUID field
	assert.Contains(t, ddl, "uuid UUID")

	// Should have required fields
	assert.Contains(t, ddl, "title VARCHAR(255)")
	assert.Contains(t, ddl, "title_short VARCHAR(50)")
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

	// Should have UUID primary key
	assert.Contains(t, ddl, "id UUID PRIMARY KEY")

	// Should have name field
	assert.Contains(t, ddl, "name VARCHAR(255) NOT NULL")

	// Should have canonical ID references
	assert.Contains(t, ddl, "canonical_id UUID")
	assert.Contains(t, ddl, "canonical_full_id UUID")
	assert.Contains(t, ddl, "canonical_stem_id UUID")

	// Should have parse_quality with default
	assert.Contains(t, ddl, "parse_quality INT NOT NULL DEFAULT 0")
}

// TestNameStringIndexDDL tests index generation for NameString model
func TestNameStringIndexDDL(t *testing.T) {
	ns := schema.NameString{}
	indexes := ns.IndexDDL()

	// Should return indexes
	require.NotEmpty(t, indexes, "NameString should have secondary indexes")

	// Convert to single string for easier searching
	allIndexes := strings.Join(indexes, "\n")

	// Should have indexes on canonical IDs
	assert.Contains(t, allIndexes, "canonical_id")
	assert.Contains(t, allIndexes, "canonical_full_id")
	assert.Contains(t, allIndexes, "canonical_stem_id")
}

// TestCanonicalTableDDL tests DDL generation for Canonical model
func TestCanonicalTableDDL(t *testing.T) {
	c := schema.Canonical{}
	ddl := c.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE canonicals")

	// Should have UUID primary key
	assert.Contains(t, ddl, "id UUID PRIMARY KEY")

	// Should have name field
	assert.Contains(t, ddl, "name VARCHAR(255) NOT NULL")
}

// TestCanonicalFullTableDDL tests DDL generation for CanonicalFull model
func TestCanonicalFullTableDDL(t *testing.T) {
	cf := schema.CanonicalFull{}
	ddl := cf.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE canonical_fulls")

	// Should have UUID primary key
	assert.Contains(t, ddl, "id UUID PRIMARY KEY")

	// Should have name field
	assert.Contains(t, ddl, "name VARCHAR(255) NOT NULL")
}

// TestCanonicalStemTableDDL tests DDL generation for CanonicalStem model
func TestCanonicalStemTableDDL(t *testing.T) {
	cs := schema.CanonicalStem{}
	ddl := cs.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE canonical_stems")

	// Should have UUID primary key
	assert.Contains(t, ddl, "id UUID PRIMARY KEY")

	// Should have name field
	assert.Contains(t, ddl, "name VARCHAR(255) NOT NULL")
}

// TestNameStringIndexTableDDL tests DDL generation for NameStringIndex model
func TestNameStringIndexTableDDL(t *testing.T) {
	nsi := schema.NameStringIndex{}
	ddl := nsi.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE name_string_indices")

	// Should have data_source_id
	assert.Contains(t, ddl, "data_source_id SMALLINT NOT NULL")

	// Should have name_string_id as UUID
	assert.Contains(t, ddl, "name_string_id UUID NOT NULL")

	// Should have accepted_record_id
	assert.Contains(t, ddl, "accepted_record_id VARCHAR(255)")

	// Should have classification fields
	assert.Contains(t, ddl, "classification TEXT")
}

// TestWordTableDDL tests DDL generation for Word model
func TestWordTableDDL(t *testing.T) {
	w := schema.Word{}
	ddl := w.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE words")

	// Should have UUID primary key
	assert.Contains(t, ddl, "id UUID PRIMARY KEY")

	// Should have normalized and modified fields
	assert.Contains(t, ddl, "normalized VARCHAR(250) NOT NULL")
	assert.Contains(t, ddl, "modified VARCHAR(250) NOT NULL")
}

// TestWordNameStringTableDDL tests DDL generation for WordNameString model
func TestWordNameStringTableDDL(t *testing.T) {
	wns := schema.WordNameString{}
	ddl := wns.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE word_name_strings")

	// Should have word_id
	assert.Contains(t, ddl, "word_id UUID NOT NULL")

	// Should have name_string_id
	assert.Contains(t, ddl, "name_string_id UUID NOT NULL")

	// Should have canonical_id
	assert.Contains(t, ddl, "canonical_id UUID NOT NULL")
}

// TestVernacularStringTableDDL tests DDL generation for VernacularString model
func TestVernacularStringTableDDL(t *testing.T) {
	vs := schema.VernacularString{}
	ddl := vs.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE vernacular_strings")

	// Should have UUID primary key
	assert.Contains(t, ddl, "id UUID PRIMARY KEY")

	// Should have name field
	assert.Contains(t, ddl, "name VARCHAR(500) NOT NULL")
}

// TestVernacularStringIndexTableDDL tests DDL generation for VernacularStringIndex model
func TestVernacularStringIndexTableDDL(t *testing.T) {
	vsi := schema.VernacularStringIndex{}
	ddl := vsi.TableDDL()

	// Should create table with correct name
	assert.Contains(t, ddl, "CREATE TABLE vernacular_string_indices")

	// Should have data_source_id
	assert.Contains(t, ddl, "data_source_id SMALLINT NOT NULL")

	// Should have vernacular_string_id
	assert.Contains(t, ddl, "vernacular_string_id UUID NOT NULL")

	// Should have language fields
	assert.Contains(t, ddl, "lang_code VARCHAR(3)")
	assert.Contains(t, ddl, "language VARCHAR(255)")
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

// TestAllModelsImplementDDLGenerator tests that all models implement the DDLGenerator interface
func TestAllModelsImplementDDLGenerator(t *testing.T) {
	models := []schema.DDLGenerator{
		&schema.DataSource{},
		&schema.NameString{},
		&schema.Canonical{},
		&schema.CanonicalFull{},
		&schema.CanonicalStem{},
		&schema.NameStringIndex{},
		&schema.Word{},
		&schema.WordNameString{},
		&schema.VernacularString{},
		&schema.VernacularStringIndex{},
		&schema.SchemaVersion{},
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
