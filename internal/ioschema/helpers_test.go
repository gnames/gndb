package ioschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFormatCollationSQL_FormatsCorrectly verifies SQL
// formatting.
func TestFormatCollationSQL_FormatsCorrectly(t *testing.T) {
	template := `ALTER TABLE %s ALTER COLUMN %s ` +
		`TYPE VARCHAR(%d) COLLATE "C"`
	table := "test_table"
	column := "test_column"
	varchar := 255

	result := formatCollationSQL(template, table, column,
		varchar)

	expected := `ALTER TABLE test_table ALTER COLUMN ` +
		`test_column TYPE VARCHAR(255) COLLATE "C"`
	assert.Equal(t, expected, result)
}

// TestFormatCollationSQL_DifferentValues verifies
// formatting with different inputs.
func TestFormatCollationSQL_DifferentValues(t *testing.T) {
	tests := []struct {
		name     string
		table    string
		column   string
		varchar  int
		expected string
	}{
		{
			name:    "name_strings table",
			table:   "name_strings",
			column:  "name",
			varchar: 500,
			expected: `ALTER TABLE name_strings ` +
				`ALTER COLUMN name ` +
				`TYPE VARCHAR(500) COLLATE "C"`,
		},
		{
			name:    "canonicals table",
			table:   "canonicals",
			column:  "name",
			varchar: 255,
			expected: `ALTER TABLE canonicals ` +
				`ALTER COLUMN name ` +
				`TYPE VARCHAR(255) COLLATE "C"`,
		},
	}

	template := `ALTER TABLE %s ALTER COLUMN %s ` +
		`TYPE VARCHAR(%d) COLLATE "C"`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCollationSQL(template,
				tt.table, tt.column, tt.varchar)
			assert.Equal(t, tt.expected, result)
		})
	}
}
