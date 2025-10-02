package schema_test

import (
	"testing"

	"github.com/gnames/gndb/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAllModels tests that AllModels returns all schema models
func TestAllModels(t *testing.T) {
	models := schema.AllModels()

	// Should return 11 models
	require.Len(t, models, 11, "AllModels should return all 11 schema models")

	// Verify each model is present
	modelTypes := make(map[string]bool)
	for _, model := range models {
		switch model.(type) {
		case *schema.DataSource:
			modelTypes["DataSource"] = true
		case *schema.NameString:
			modelTypes["NameString"] = true
		case *schema.Canonical:
			modelTypes["Canonical"] = true
		case *schema.CanonicalFull:
			modelTypes["CanonicalFull"] = true
		case *schema.CanonicalStem:
			modelTypes["CanonicalStem"] = true
		case *schema.NameStringIndex:
			modelTypes["NameStringIndex"] = true
		case *schema.Word:
			modelTypes["Word"] = true
		case *schema.WordNameString:
			modelTypes["WordNameString"] = true
		case *schema.VernacularString:
			modelTypes["VernacularString"] = true
		case *schema.VernacularStringIndex:
			modelTypes["VernacularStringIndex"] = true
		case *schema.SchemaVersion:
			modelTypes["SchemaVersion"] = true
		}
	}

	// Verify all expected models are present
	expectedModels := []string{
		"DataSource", "NameString", "Canonical", "CanonicalFull",
		"CanonicalStem", "NameStringIndex", "Word", "WordNameString",
		"VernacularString", "VernacularStringIndex", "SchemaVersion",
	}
	for _, name := range expectedModels {
		assert.True(t, modelTypes[name], "Model %s should be in AllModels", name)
	}
}

// TestMigrate tests GORM AutoMigrate functionality
func TestMigrate(t *testing.T) {
	// Use SQLite for testing (in-memory)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to open test database")

	// Run migration
	err = schema.Migrate(db)
	require.NoError(t, err, "Migrate should succeed")

	// Verify all tables were created
	tables := []string{
		"data_sources", "name_strings", "canonicals", "canonical_fulls",
		"canonical_stems", "name_string_indices", "words", "word_name_strings",
		"vernacular_strings", "vernacular_string_indices", "schema_versions",
	}

	for _, tableName := range tables {
		has := db.Migrator().HasTable(tableName)
		assert.True(t, has, "Table %s should exist after migration", tableName)
	}
}

// TestDataSourceSchema tests DataSource GORM schema
func TestDataSourceSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.DataSource{})
	require.NoError(t, err)

	// Verify table exists
	assert.True(t, db.Migrator().HasTable("data_sources"))

	// Verify columns exist
	columns := []string{
		"id", "uuid", "title", "title_short", "version", "revision_date",
		"doi", "citation", "authors", "description", "website_url",
		"data_url", "outlink_url", "is_outlink_ready", "is_curated",
		"is_auto_curated", "has_taxon_data", "record_count",
		"vern_record_count", "updated_at",
	}

	for _, col := range columns {
		has := db.Migrator().HasColumn(&schema.DataSource{}, col)
		assert.True(t, has, "Column %s should exist in data_sources", col)
	}
}

// TestNameStringSchema tests NameString GORM schema
func TestNameStringSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.NameString{})
	require.NoError(t, err)

	// Verify table exists
	assert.True(t, db.Migrator().HasTable("name_strings"))

	// Verify key columns exist
	columns := []string{
		"id", "name", "year", "cardinality", "canonical_id",
		"canonical_full_id", "canonical_stem_id", "virus", "bacteria",
		"surrogate", "parse_quality",
	}

	for _, col := range columns {
		has := db.Migrator().HasColumn(&schema.NameString{}, col)
		assert.True(t, has, "Column %s should exist in name_strings", col)
	}
}

// TestCanonicalSchema tests Canonical GORM schema
func TestCanonicalSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.Canonical{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable("canonicals"))
	assert.True(t, db.Migrator().HasColumn(&schema.Canonical{}, "id"))
	assert.True(t, db.Migrator().HasColumn(&schema.Canonical{}, "name"))
}

// TestCanonicalFullSchema tests CanonicalFull GORM schema
func TestCanonicalFullSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.CanonicalFull{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable("canonical_fulls"))
	assert.True(t, db.Migrator().HasColumn(&schema.CanonicalFull{}, "id"))
	assert.True(t, db.Migrator().HasColumn(&schema.CanonicalFull{}, "name"))
}

// TestCanonicalStemSchema tests CanonicalStem GORM schema
func TestCanonicalStemSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.CanonicalStem{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable("canonical_stems"))
	assert.True(t, db.Migrator().HasColumn(&schema.CanonicalStem{}, "id"))
	assert.True(t, db.Migrator().HasColumn(&schema.CanonicalStem{}, "name"))
}

// TestNameStringIndexSchema tests NameStringIndex GORM schema
func TestNameStringIndexSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.NameStringIndex{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable("name_string_indices"))

	columns := []string{
		"data_source_id", "record_id", "name_string_id", "outlink_id",
		"global_id", "name_id", "local_id", "code_id", "rank",
		"taxonomic_status", "accepted_record_id", "classification",
		"classification_ids", "classification_ranks",
	}

	for _, col := range columns {
		has := db.Migrator().HasColumn(&schema.NameStringIndex{}, col)
		assert.True(t, has, "Column %s should exist in name_string_indices", col)
	}
}

// TestWordSchema tests Word GORM schema
func TestWordSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.Word{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable("words"))
	assert.True(t, db.Migrator().HasColumn(&schema.Word{}, "id"))
	assert.True(t, db.Migrator().HasColumn(&schema.Word{}, "normalized"))
	assert.True(t, db.Migrator().HasColumn(&schema.Word{}, "modified"))
	assert.True(t, db.Migrator().HasColumn(&schema.Word{}, "type_id"))
}

// TestWordNameStringSchema tests WordNameString GORM schema
func TestWordNameStringSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.WordNameString{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable("word_name_strings"))
	assert.True(t, db.Migrator().HasColumn(&schema.WordNameString{}, "word_id"))
	assert.True(t, db.Migrator().HasColumn(&schema.WordNameString{}, "name_string_id"))
	assert.True(t, db.Migrator().HasColumn(&schema.WordNameString{}, "canonical_id"))
}

// TestVernacularStringSchema tests VernacularString GORM schema
func TestVernacularStringSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.VernacularString{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable("vernacular_strings"))
	assert.True(t, db.Migrator().HasColumn(&schema.VernacularString{}, "id"))
	assert.True(t, db.Migrator().HasColumn(&schema.VernacularString{}, "name"))
}

// TestVernacularStringIndexSchema tests VernacularStringIndex GORM schema
func TestVernacularStringIndexSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.VernacularStringIndex{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable("vernacular_string_indices"))

	columns := []string{
		"data_source_id", "record_id", "vernacular_string_id",
		"language_orig", "language", "lang_code", "locality",
		"country_code", "preferred",
	}

	for _, col := range columns {
		has := db.Migrator().HasColumn(&schema.VernacularStringIndex{}, col)
		assert.True(t, has, "Column %s should exist in vernacular_string_indices", col)
	}
}

// TestSchemaVersionSchema tests SchemaVersion GORM schema
func TestSchemaVersionSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&schema.SchemaVersion{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable("schema_versions"))
	assert.True(t, db.Migrator().HasColumn(&schema.SchemaVersion{}, "version"))
	assert.True(t, db.Migrator().HasColumn(&schema.SchemaVersion{}, "description"))
	assert.True(t, db.Migrator().HasColumn(&schema.SchemaVersion{}, "applied_at"))
}
