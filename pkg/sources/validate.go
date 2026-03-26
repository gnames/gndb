package sources

import (
	"fmt"
	"slices"
	"strings"
)

// Validate checks the configuration for errors and applies defaults.
func (c *SourcesConfig) Validate() error {
	if len(c.DataSources) == 0 {
		return fmt.Errorf("no data sources specified in configuration")
	}

	// Validate each data source
	for i := range c.DataSources {
		warnings, err := c.DataSources[i].Validate(i + 1)
		if err != nil {
			return fmt.Errorf("data source %d: %w", i+1, err)
		}
		c.Warnings = append(c.Warnings, warnings...)
	}

	return nil
}

// Validate checks a single data source configuration for data structure validity.
// File system validation (directory existence) is deferred to runtime (I/O layer).
// Returns a slice of warnings (non-fatal issues) and an error (fatal issues).
func (d *DataSourceConfig) Validate(index int) ([]ValidationWarning, error) {
	var warnings []ValidationWarning
	// ID is required
	if d.ID == 0 {
		return nil, fmt.Errorf("id is required")
	}

	// Parent is required
	if d.Parent == "" {
		return nil, fmt.Errorf("parent directory or URL is required")
	}

	// is_outlink_ready is computed purely from outlink_url and outlink_id_column
	// validity — any value set in the YAML is ignored.
	d.IsOutlinkReady = false
	if d.OutlinkURL != "" || d.OutlinkIDColumn != "" {
		outlinkValid := true
		var outlinkIssue string
		var outlinkSuggestion string

		if d.OutlinkURL == "" {
			outlinkValid = false
			outlinkIssue = "outlink_url is required for outlink configuration"
			outlinkSuggestion = "Set 'outlink_url' with a URL template containing {} placeholder"
		} else if !strings.Contains(d.OutlinkURL, "{}") {
			outlinkValid = false
			outlinkIssue = "outlink_url must contain {} placeholder for ID substitution"
			outlinkSuggestion = fmt.Sprintf("Update 'outlink_url: %s' to include {} where the ID should be inserted", d.OutlinkURL)
		} else if d.OutlinkIDColumn == "" {
			outlinkValid = false
			outlinkIssue = "outlink_id_column is required for outlink configuration"
			outlinkSuggestion = "Set 'outlink_id_column' to a valid table.column (e.g., 'taxon.col__id', 'name.col__alternative_id')"
		} else {
			// Validate outlink_id_column format: "table.column"
			parts := strings.Split(d.OutlinkIDColumn, ".")
			if len(parts) != 2 {
				outlinkValid = false
				outlinkIssue = fmt.Sprintf("invalid outlink_id_column format '%s': must be 'table.column'", d.OutlinkIDColumn)
				outlinkSuggestion = "Change to format 'table.column' (e.g., 'taxon.col__id', 'name.col__alternative_id')"
			} else {
				tableName := parts[0]
				columnName := parts[1]

				// Define valid table.column combinations based on SFGA schema
				// Note: Only name and taxon tables are supported (synonym table complicates import logic)
				validCombinations := map[string][]string{
					"name": {
						"col__id",
						"col__alternative_id",
						"col__link",
					},
					"taxon": {
						"col__id",
						"col__name_id",
						"col__alternative_id",
						"gn__local_id",
						"gn__global_id",
					},
				}

				// Validate table exists
				allowedColumns, validTable := validCombinations[tableName]
				if !validTable {
					outlinkValid = false
					var validTables []string
					for table := range validCombinations {
						validTables = append(validTables, table)
					}
					outlinkIssue = fmt.Sprintf("invalid table '%s' in outlink_id_column", tableName)
					outlinkSuggestion = fmt.Sprintf("Change table to one of: %v", validTables)
				} else {
					// Validate column exists for this table
					validColumn := false
					if slices.Contains(allowedColumns, columnName) {
						validColumn = true
					}
					if !validColumn {
						outlinkValid = false
						// Build list of all valid table.column combinations for error message
						var allValidCombinations []string
						for table, columns := range validCombinations {
							for _, col := range columns {
								allValidCombinations = append(allValidCombinations, fmt.Sprintf("%s.%s", table, col))
							}
						}
						outlinkIssue = fmt.Sprintf("invalid outlink_id_column '%s.%s': column not valid for this table", tableName, columnName)
						outlinkSuggestion = fmt.Sprintf("Use one of these valid combinations: %v", allValidCombinations)
					}
				}
			}
		}

		// Auto-set or clear is_outlink_ready based on validation result
		if outlinkValid {
			d.IsOutlinkReady = true
		} else {
			d.IsOutlinkReady = false
			warnings = append(warnings, ValidationWarning{
				DataSourceID: d.ID,
				Field:        "outlink configuration",
				Message:      outlinkIssue,
				Suggestion:   outlinkSuggestion,
			})
		}
	}

	return warnings, nil
}
