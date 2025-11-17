package ioschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFormatCollationSQL_FormatsCorrectly verifies SQL
// formatting.
func TestFormatCollationSQL_FormatsCorrectly(t *testing.T) {
	template := `ALTER TABLE %s ALTER COLUMN %s ` +
		`TYPE TEXT COLLATE "C"`
	table := "test_table"
	column := "test_column"

	result := formatCollationSQL(template, table, column)

	expected := `ALTER TABLE test_table ALTER COLUMN ` +
		`test_column TYPE TEXT COLLATE "C"`
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
				`TYPE TEXT COLLATE "C"`,
		},
		{
			name:    "canonicals table",
			table:   "canonicals",
			column:  "name",
			varchar: 255,
			expected: `ALTER TABLE canonicals ` +
				`ALTER COLUMN name ` +
				`TYPE TEXT COLLATE "C"`,
		},
	}

	template := `ALTER TABLE %s ALTER COLUMN %s ` +
		`TYPE TEXT COLLATE "C"`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCollationSQL(template,
				tt.table, tt.column)
			assert.Equal(t, tt.expected, result)
		})
	}
}
