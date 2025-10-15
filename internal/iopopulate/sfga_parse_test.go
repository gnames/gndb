package iopopulate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSFGAFilename(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectedVersion string
		expectedDate    string
	}{
		{
			name:            "full metadata with underscore separator",
			filename:        "1000_ruhoff_2023-08-22_v1.0.0.sqlite.zip",
			expectedVersion: "1.0.0",
			expectedDate:    "2023-08-22",
		},
		{
			name:            "full metadata with dash separator",
			filename:        "0147-vascan-2025-08-25.sqlite.zip",
			expectedVersion: "",
			expectedDate:    "2025-08-25",
		},
		{
			name:            "with version without v prefix",
			filename:        "1000_ruhoff_2023-08-22_1.0.0.sqlite",
			expectedVersion: "1.0.0",
			expectedDate:    "2023-08-22",
		},
		{
			name:            "minimal filename - just ID",
			filename:        "1000.sql",
			expectedVersion: "",
			expectedDate:    "",
		},
		{
			name:            "ID with name only",
			filename:        "1000_ruhoff.sqlite",
			expectedVersion: "",
			expectedDate:    "",
		},
		{
			name:            "ID with date only",
			filename:        "1000_2023-08-22.sqlite.zip",
			expectedVersion: "",
			expectedDate:    "2023-08-22",
		},
		{
			name:            "ID with version only",
			filename:        "1000_v1.5.2.sqlite",
			expectedVersion: "1.5.2",
			expectedDate:    "",
		},
		{
			name:            "version without patch number",
			filename:        "1000_ruhoff_v1.5.sqlite",
			expectedVersion: "1.5",
			expectedDate:    "",
		},
		{
			name:            "COL format with dash separators",
			filename:        "1001-col-small-2025-09-11.sqlite.zip",
			expectedVersion: "",
			expectedDate:    "2025-09-11",
		},
		{
			name:            "VASCAN format",
			filename:        "1002-vascan-2025-08-25.sqlite.zip",
			expectedVersion: "",
			expectedDate:    "2025-08-25",
		},
		{
			name:            "with spaces in name (edge case)",
			filename:        "1000_my database_2023-01-15_v2.0.sql",
			expectedVersion: "2.0",
			expectedDate:    "2023-01-15",
		},
		{
			name:            "version at end of filename",
			filename:        "1000_mydb-v1.2.3.sqlite.zip",
			expectedVersion: "1.2.3",
			expectedDate:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSFGAFilename(tt.filename)

			assert.Equal(t, tt.filename, result.Filename, "Filename should be preserved")
			assert.Equal(t, tt.expectedVersion, result.Version, "Version should match")
			assert.Equal(t, tt.expectedDate, result.RevisionDate, "Revision date should match")
		})
	}
}

func TestParseSFGAFilename_RealWorldExamples(t *testing.T) {
	// Test with actual testdata files
	tests := []struct {
		filename string
		version  string
		date     string
	}{
		{
			filename: "1000_ruhoff_2023-08-22_v1.0.0.sqlite.zip",
			version:  "1.0.0",
			date:     "2023-08-22",
		},
		{
			filename: "1001-col-small-2025-09-11.sqlite.zip",
			version:  "",
			date:     "2025-09-11",
		},
		{
			filename: "1002-vascan-2025-08-25.sqlite.zip",
			version:  "",
			date:     "2025-08-25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := parseSFGAFilename(tt.filename)
			assert.Equal(t, tt.version, result.Version)
			assert.Equal(t, tt.date, result.RevisionDate)
		})
	}
}
