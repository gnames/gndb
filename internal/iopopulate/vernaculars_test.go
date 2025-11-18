package iopopulate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		strs     []string
		sep      string
		expected string
	}{
		{
			name:     "empty slice",
			strs:     []string{},
			sep:      ",",
			expected: "",
		},
		{
			name:     "single element",
			strs:     []string{"one"},
			sep:      ",",
			expected: "one",
		},
		{
			name:     "two elements",
			strs:     []string{"one", "two"},
			sep:      ",",
			expected: "one,two",
		},
		{
			name:     "multiple elements",
			strs:     []string{"a", "b", "c", "d"},
			sep:      "-",
			expected: "a-b-c-d",
		},
		{
			name:     "empty separator",
			strs:     []string{"a", "b", "c"},
			sep:      "",
			expected: "abc",
		},
		{
			name:     "multi-char separator",
			strs:     []string{"one", "two"},
			sep:      ", ",
			expected: "one, two",
		},
		{
			name:     "SQL values format",
			strs:     []string{"($1, $2)", "($3, $4)", "($5, $6)"},
			sep:      ",",
			expected: "($1, $2),($3, $4),($5, $6)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.strs, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}
