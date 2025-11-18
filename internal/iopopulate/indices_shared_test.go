package iopopulate

import (
	"database/sql"
	"testing"

	"github.com/gnames/gndb/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestCodeIDToInt(t *testing.T) {
	tests := []struct {
		name     string
		codeID   string
		expected int
	}{
		{
			name:     "zoological",
			codeID:   "ZOOLOGICAL",
			expected: 1,
		},
		{
			name:     "botanical",
			codeID:   "BOTANICAL",
			expected: 2,
		},
		{
			name:     "bacterial",
			codeID:   "BACTERIAL",
			expected: 3,
		},
		{
			name:     "virus",
			codeID:   "VIRUS",
			expected: 4,
		},
		{
			name:     "lowercase zoological",
			codeID:   "zoological",
			expected: 1,
		},
		{
			name:     "mixed case botanical",
			codeID:   "Botanical",
			expected: 2,
		},
		{
			name:     "unknown code",
			codeID:   "UNKNOWN",
			expected: 0,
		},
		{
			name:     "empty string",
			codeID:   "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codeIDToInt(tt.codeID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildOutlinkColumn(t *testing.T) {
	tests := []struct {
		name          string
		outlinkColumn string
		queryType     string
		expected      string
	}{
		// Taxa query tests
		{
			name:          "taxa - taxon table",
			outlinkColumn: "taxon.col__id",
			queryType:     "taxa",
			expected:      "t.col__id",
		},
		{
			name:          "taxa - name table",
			outlinkColumn: "name.col__name_id",
			queryType:     "taxa",
			expected:      "n.col__name_id",
		},
		{
			name:          "taxa - invalid table",
			outlinkColumn: "synonym.col__id",
			queryType:     "taxa",
			expected:      "",
		},
		// Synonyms query tests
		{
			name:          "synonyms - taxon table",
			outlinkColumn: "taxon.col__id",
			queryType:     "synonyms",
			expected:      "t.col__id",
		},
		{
			name:          "synonyms - synonym table",
			outlinkColumn: "synonym.col__id",
			queryType:     "synonyms",
			expected:      "s.col__id",
		},
		{
			name:          "synonyms - name table",
			outlinkColumn: "name.col__name_id",
			queryType:     "synonyms",
			expected:      "n.col__name_id",
		},
		// Bare names query tests
		{
			name:          "bare_names - name table",
			outlinkColumn: "name.col__id",
			queryType:     "bare_names",
			expected:      "name.col__id",
		},
		{
			name:          "bare_names - invalid table",
			outlinkColumn: "taxon.col__id",
			queryType:     "bare_names",
			expected:      "",
		},
		// Edge cases
		{
			name:          "empty column",
			outlinkColumn: "",
			queryType:     "taxa",
			expected:      "",
		},
		{
			name:          "invalid format - no dot",
			outlinkColumn: "taxon_col__id",
			queryType:     "taxa",
			expected:      "",
		},
		{
			name:          "invalid format - too many dots",
			outlinkColumn: "taxon.col.id",
			queryType:     "taxa",
			expected:      "",
		},
		{
			name:          "unknown query type",
			outlinkColumn: "taxon.col__id",
			queryType:     "unknown",
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildOutlinkColumn(tt.outlinkColumn, tt.queryType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFlatClassification(t *testing.T) {
	tests := []struct {
		name        string
		datum       taxonDatum
		wantKeys    []string
		wantNotKeys []string
	}{
		{
			name: "all fields valid",
			datum: taxonDatum{
				kingdom:   sql.NullString{String: "Plantae", Valid: true},
				kingdomID: sql.NullString{String: "1", Valid: true},
				phylum:    sql.NullString{String: "Magnoliophyta", Valid: true},
				phylumID:  sql.NullString{String: "2", Valid: true},
				class:     sql.NullString{String: "Magnoliopsida", Valid: true},
				classID:   sql.NullString{String: "3", Valid: true},
			},
			wantKeys:    []string{"kingdom", "kingdom_id", "phylum", "phylum_id", "class", "class_id"},
			wantNotKeys: []string{"order", "family"},
		},
		{
			name: "some fields null",
			datum: taxonDatum{
				kingdom:   sql.NullString{String: "Animalia", Valid: true},
				kingdomID: sql.NullString{String: "1", Valid: true},
				phylum:    sql.NullString{String: "", Valid: false},
				phylumID:  sql.NullString{String: "", Valid: false},
			},
			wantKeys:    []string{"kingdom", "kingdom_id"},
			wantNotKeys: []string{"phylum", "phylum_id"},
		},
		{
			name:        "all fields null",
			datum:       taxonDatum{},
			wantKeys:    []string{},
			wantNotKeys: []string{"kingdom", "phylum", "class"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.New()
			p := &populator{cfg: cfg}
			result, _ := p.flatClassification(tt.datum)

			for _, key := range tt.wantKeys {
				assert.Contains(t, result, key)
			}
			for _, key := range tt.wantNotKeys {
				assert.NotContains(t, result, key)
			}
		})
	}
}
