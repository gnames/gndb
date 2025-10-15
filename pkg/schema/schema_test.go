package schema_test

import (
	"testing"

	"github.com/gnames/gndb/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllModels tests that AllModels returns all schema models
func TestAllModels(t *testing.T) {
	models := schema.AllModels()

	// Should return 10 models
	require.Len(t, models, 10, "AllModels should return all 10 schema models")

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
		}
	}

	// Verify all expected models are present
	expectedModels := []string{
		"DataSource", "NameString", "Canonical", "CanonicalFull",
		"CanonicalStem", "NameStringIndex", "Word", "WordNameString",
		"VernacularString", "VernacularStringIndex",
	}
	for _, name := range expectedModels {
		assert.True(t, modelTypes[name], "Model %s should be in AllModels", name)
	}
}

// TestMigrate tests GORM AutoMigrate functionality
func TestMigrate(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestDataSourceSchema tests DataSource GORM schema
func TestDataSourceSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestNameStringSchema tests NameString GORM schema
func TestNameStringSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestCanonicalSchema tests Canonical GORM schema
func TestCanonicalSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestCanonicalFullSchema tests CanonicalFull GORM schema
func TestCanonicalFullSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestCanonicalStemSchema tests CanonicalStem GORM schema
func TestCanonicalStemSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestNameStringIndexSchema tests NameStringIndex GORM schema
func TestNameStringIndexSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestWordSchema tests Word GORM schema
func TestWordSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestWordNameStringSchema tests WordNameString GORM schema
func TestWordNameStringSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestVernacularStringSchema tests VernacularString GORM schema
func TestVernacularStringSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}

// TestVernacularStringIndexSchema tests VernacularStringIndex GORM schema
func TestVernacularStringIndexSchema(t *testing.T) {
	t.Skip("Skipped: requires database connection, tested via integration tests")
}
