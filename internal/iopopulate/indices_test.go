package iopopulate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildOutlinkColumn(t *testing.T) {
	tests := []struct {
		name          string
		outlinkColumn string
		queryType     string
		expected      string
	}{
		// Empty column returns empty
		{
			name:          "empty column returns empty",
			outlinkColumn: "",
			queryType:     "taxa",
			expected:      "",
		},
		// Taxa query type
		{
			name:          "taxa: taxon.col__id",
			outlinkColumn: "taxon.col__id",
			queryType:     "taxa",
			expected:      "t.col__id",
		},
		{
			name:          "taxa: taxon.col__alternative_id",
			outlinkColumn: "taxon.col__alternative_id",
			queryType:     "taxa",
			expected:      "t.col__alternative_id",
		},
		{
			name:          "taxa: name.col__name_id",
			outlinkColumn: "name.col__name_id",
			queryType:     "taxa",
			expected:      "n.col__name_id",
		},
		{
			name:          "taxa: name.col__alternative_id",
			outlinkColumn: "name.col__alternative_id",
			queryType:     "taxa",
			expected:      "n.col__alternative_id",
		},
		{
			name:          "taxa: synonym table not available",
			outlinkColumn: "synonym.col__id",
			queryType:     "taxa",
			expected:      "",
		},
		// Synonyms query type
		{
			name:          "synonyms: taxon.col__id (accepted taxon)",
			outlinkColumn: "taxon.col__id",
			queryType:     "synonyms",
			expected:      "t.col__id",
		},
		{
			name:          "synonyms: synonym.col__id",
			outlinkColumn: "synonym.col__id",
			queryType:     "synonyms",
			expected:      "s.col__id",
		},
		{
			name:          "synonyms: synonym.col__name_id",
			outlinkColumn: "synonym.col__name_id",
			queryType:     "synonyms",
			expected:      "s.col__name_id",
		},
		{
			name:          "synonyms: name.col__alternative_id",
			outlinkColumn: "name.col__alternative_id",
			queryType:     "synonyms",
			expected:      "n.col__alternative_id",
		},
		// Bare names query type
		{
			name:          "bare_names: name.col__name_id",
			outlinkColumn: "name.col__name_id",
			queryType:     "bare_names",
			expected:      "name.col__name_id",
		},
		{
			name:          "bare_names: name.col__alternative_id",
			outlinkColumn: "name.col__alternative_id",
			queryType:     "bare_names",
			expected:      "name.col__alternative_id",
		},
		{
			name:          "bare_names: taxon table not available",
			outlinkColumn: "taxon.col__id",
			queryType:     "bare_names",
			expected:      "",
		},
		{
			name:          "bare_names: synonym table not available",
			outlinkColumn: "synonym.col__id",
			queryType:     "bare_names",
			expected:      "",
		},
		// Invalid formats
		{
			name:          "invalid format: no dot",
			outlinkColumn: "taxon_col__id",
			queryType:     "taxa",
			expected:      "",
		},
		{
			name:          "invalid format: too many dots",
			outlinkColumn: "taxon.col.id",
			queryType:     "taxa",
			expected:      "",
		},
		// Unknown query type
		{
			name:          "unknown query type",
			outlinkColumn: "taxon.col__id",
			queryType:     "unknown",
			expected:      "",
		},
		// Custom columns (any column name after table)
		{
			name:          "taxa: taxon.sf__custom_column",
			outlinkColumn: "taxon.sf__custom_column",
			queryType:     "taxa",
			expected:      "t.sf__custom_column",
		},
		{
			name:          "synonyms: name.gn__custom_field",
			outlinkColumn: "name.gn__custom_field",
			queryType:     "synonyms",
			expected:      "n.gn__custom_field",
		},
		{
			name:          "bare_names: name.col__local_id",
			outlinkColumn: "name.col__local_id",
			queryType:     "bare_names",
			expected:      "name.col__local_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildOutlinkColumn(tt.outlinkColumn, tt.queryType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildOutlinkColumn_AllTableColumnCombinations(t *testing.T) {
	// Test all valid table.column combinations from sources.yaml spec
	combinations := []struct {
		table     string
		column    string
		queryType string
		available bool // Whether this combination should be available
	}{
		// Taxa query: taxon and name tables
		{"taxon", "col__id", "taxa", true},
		{"taxon", "col__name_id", "taxa", true},
		{"taxon", "col__local_id", "taxa", true},
		{"taxon", "col__alternative_id", "taxa", true},
		{"name", "col__id", "taxa", true},
		{"name", "col__name_id", "taxa", true},
		{"name", "col__local_id", "taxa", true},
		{"name", "col__alternative_id", "taxa", true},
		{"synonym", "col__id", "taxa", false},

		// Synonyms query: taxon (accepted), synonym, and name tables
		{"taxon", "col__id", "synonyms", true},
		{"taxon", "col__alternative_id", "synonyms", true},
		{"synonym", "col__id", "synonyms", true},
		{"synonym", "col__name_id", "synonyms", true},
		{"synonym", "col__local_id", "synonyms", true},
		{"name", "col__id", "synonyms", true},
		{"name", "col__name_id", "synonyms", true},
		{"name", "col__alternative_id", "synonyms", true},

		// Bare names query: only name table
		{"name", "col__id", "bare_names", true},
		{"name", "col__name_id", "bare_names", true},
		{"name", "col__local_id", "bare_names", true},
		{"name", "col__alternative_id", "bare_names", true},
		{"taxon", "col__id", "bare_names", false},
		{"synonym", "col__id", "bare_names", false},
	}

	for _, combo := range combinations {
		t.Run(combo.queryType+"_"+combo.table+"."+combo.column, func(t *testing.T) {
			outlinkColumn := combo.table + "." + combo.column
			result := buildOutlinkColumn(outlinkColumn, combo.queryType)

			if combo.available {
				assert.NotEmpty(t, result, "Should return non-empty for available combination")
				assert.Contains(t, result, ".", "Should contain dot separator")
				assert.Contains(t, result, combo.column, "Should contain the column name")
			} else {
				assert.Empty(t, result, "Should return empty for unavailable combination")
			}
		})
	}
}
