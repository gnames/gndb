// Package populate provides configuration and operations for populating the GNdb
// database with data from SFGA (Standard Format for Global Archiving) files.
//
// The main entry point is the sources.yaml file which users provide to specify
// which SFGA data sources to import. See sources-yaml-spec.md for details.
package populate

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gnames/gndb/pkg/templates"
	"gopkg.in/yaml.v3"
)

// SourcesConfig represents the complete sources.yaml configuration file.
type SourcesConfig struct {
	// DataSources is the list of data sources to import.
	DataSources []DataSourceConfig `yaml:"data_sources"`

	// Populate contains settings for the Populate process.
	Populate PopulateConfig `yaml:"populate,omitempty"`

	// Logging contains logging and progress settings.
	Logging LoggingConfig `yaml:"logging,omitempty"`

	// Warnings holds non-fatal validation warnings (not serialized)
	Warnings []ValidationWarning `yaml:"-"`
}

// ValidationWarning represents a non-fatal configuration issue.
type ValidationWarning struct {
	DataSourceID int    // ID of the data source
	Field        string // Field name that has the issue
	Message      string // Description of the issue
	Suggestion   string // How to fix it
}

// DataSourceConfig represents configuration for a single data source.
//
// SFGA provides these fields (only override if needed):
//   - col__id (id)
//   - col__title (title)
//   - col__description (description)
//   - col__version (version) - NEVER in YAML, always from SFGA or filename
//   - col__issued (release_date) - NEVER in YAML, always from SFGA or filename
//   - col__url (home_url)
//   - col__doi, col__license, col__citation, etc.
//
// NOT in SFGA (can be provided here):
//   - title_short (optional, falls back to col__alias or truncated col__title)
//   - data_url (optional download link)
//   - data_source_type (optional, can be inferred from data structure)
//   - is_curated, is_auto_curated, has_classification (optional quality flags)
//   - outlink configuration (optional)
type DataSourceConfig struct {
	// Core identification (required)
	// ID identifies the data source. Convention: < 1000 = official, >= 1000 = custom
	ID int `yaml:"id"`

	// Parent is the directory or URL containing SFGA files for this source.
	// Auto-detected: starts with http:// or https:// = URL, otherwise = directory
	// SFGA files are matched by pattern: {4-digit-ID}*.zip or {ID}*.zip
	// Examples:
	//   - http://opendata.globalnames.org/sfga/latest/
	//   - /home/user/data/sfga/
	//   - ~/data/sfga/
	Parent string `yaml:"parent"`

	// Titles and description (override SFGA if needed)
	Title       string `yaml:"title,omitempty"`       // Override SFGA col__title
	TitleShort  string `yaml:"title_short,omitempty"` // Fallback: col__alias → truncate col__title
	Description string `yaml:"description,omitempty"` // Override SFGA col__description

	// URLs (override SFGA if needed)
	HomeURL string `yaml:"home_url,omitempty"` // Override SFGA col__url
	DataURL string `yaml:"data_url,omitempty"` // Download URL (not in SFGA)

	// Type classification (can be inferred)
	// Inferred: no classification & no accepted_record → nomenclatural
	DataSourceType string `yaml:"data_source_type,omitempty"` // "taxonomic" or "nomenclatural"

	// Curation level (quality indicators)
	IsCurated         bool `yaml:"is_curated,omitempty"`         // Manually curated by experts
	IsAutoCurated     bool `yaml:"is_auto_curated,omitempty"`    // Automatically validated
	HasClassification bool `yaml:"has_classification,omitempty"` // Has hierarchical taxonomy

	// Outlink configuration (for generating links to original records)
	IsOutlinkReady  bool   `yaml:"is_outlink_ready,omitempty"`  // Can generate outlinks
	OutlinkURL      string `yaml:"outlink_url,omitempty"`       // URL template with {} placeholder
	OutlinkIDColumn string `yaml:"outlink_id_column,omitempty"` // table.column format (e.g., "taxon.col__id", "name.col__alternative_id")
}

// PopulateConfig contains settings for the populate process.
type PopulateConfig struct {
	BatchSize                int  `yaml:"batch_size,omitempty"`                 // Records per batch insert (default: 5000)
	ConcurrentJobs           int  `yaml:"concurrent_jobs,omitempty"`            // Number of parallel jobs (default: 4)
	PreferFlatClassification bool `yaml:"prefer_flat_classification,omitempty"` // Use flat vs hierarchical classification
}

// LoggingConfig contains logging and progress settings.
type LoggingConfig struct {
	ShowProgress bool   `yaml:"show_progress,omitempty"` // Show progress bars (default: true)
	LogLevel     string `yaml:"log_level,omitempty"`     // debug, info, warn, error (default: info)
}

// FileMetadata contains metadata extracted from SFGA filename.
type FileMetadata struct {
	ID          int    // Extracted from filename
	Version     string // Extracted from filename (if present)
	ReleaseDate string // Extracted from filename in YYYY-MM-DD format (if present)
	IsURL       bool   // True if file is a URL
}

// LoadSourcesConfig loads the data sources configuration from a YAML file.
func LoadSourcesConfig(path string) (*SourcesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read sources config file: %w", err)
	}

	var config SourcesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse sources config: %w", err)
	}

	// Validate and process configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate checks the configuration for errors and applies defaults.
func (c *SourcesConfig) Validate() error {
	if len(c.DataSources) == 0 {
		return fmt.Errorf("no data sources specified in configuration")
	}

	// Set import defaults
	if c.Populate.BatchSize == 0 {
		c.Populate.BatchSize = 50000
	}
	if c.Populate.ConcurrentJobs == 0 {
		c.Populate.ConcurrentJobs = 4
	}

	// Set logging defaults
	if c.Logging.LogLevel == "" {
		c.Logging.LogLevel = "info"
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

// Validate checks a single data source configuration.
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

	// Check if parent is URL or local directory
	isURL := IsValidURL(d.Parent)

	if !isURL {
		// For local directories, expand ~ if needed
		parentPath := d.Parent
		if strings.HasPrefix(parentPath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to expand ~: %w", err)
			}
			parentPath = filepath.Join(homeDir, parentPath[2:])
		}

		// Check if directory exists
		stat, err := os.Stat(parentPath)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("parent directory does not exist: %s", d.Parent)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to check parent directory: %w", err)
		}
		if !stat.IsDir() {
			return nil, fmt.Errorf("parent path is not a directory: %s", d.Parent)
		}
	}

	// Validate data source type if provided
	if d.DataSourceType != "" {
		if d.DataSourceType != "taxonomic" && d.DataSourceType != "nomenclatural" {
			return nil, fmt.Errorf(
				"invalid data_source_type: must be 'taxonomic' or 'nomenclatural'",
			)
		}
	}

	// Validate outlink configuration (generate warnings, not errors)
	if d.IsOutlinkReady {
		outlinkValid := true
		var outlinkIssue string
		var outlinkSuggestion string

		if d.OutlinkURL == "" {
			outlinkValid = false
			outlinkIssue = "outlink_url is required when is_outlink_ready is true"
			outlinkSuggestion = "Set 'outlink_url' with a URL template containing {} placeholder, or set 'is_outlink_ready: false'"
		} else if !strings.Contains(d.OutlinkURL, "{}") {
			outlinkValid = false
			outlinkIssue = "outlink_url must contain {} placeholder for ID substitution"
			outlinkSuggestion = fmt.Sprintf("Update 'outlink_url: %s' to include {} where the ID should be inserted", d.OutlinkURL)
		} else if d.OutlinkIDColumn == "" {
			outlinkValid = false
			outlinkIssue = "outlink_id_column is required when is_outlink_ready is true"
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
					for _, col := range allowedColumns {
						if columnName == col {
							validColumn = true
							break
						}
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

		// If outlink configuration is invalid, disable it and generate warning
		if !outlinkValid {
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

// ParseFilename extracts metadata from SFGA filename.
// Expected format: {id}_{name}_{date}_{version}.(sql|sqlite)[.zip]
// Examples:
//   - 0001_col_2025-10-03_v2024.1.sqlite.zip  → ID=1, Date=2025-10-03, Version=v2024.1
//   - 0002_gbif_2024-12-15_2024-12-15.sql.zip → ID=2, Date=2024-12-15, Version=2024-12-15
//   - 0003_worms_2025-01-01.sqlite            → ID=3, Date=2025-01-01, Version=""
//   - 1001.sql                                 → ID=1001, Date="", Version=""
//
// Version extraction rules:
//   - If release date is last segment before extension: no version
//   - If underscore + text after date: everything until .sql|.sqlite is version
func ParseFilename(path string) FileMetadata {
	var metadata FileMetadata

	// Get filename without directory
	filename := filepath.Base(path)

	// Strip .zip extension if present
	filename = strings.TrimSuffix(filename, ".zip")

	// Extract ID (first 4 digits)
	idPattern := regexp.MustCompile(`^(\d{4})`)
	if matches := idPattern.FindStringSubmatch(filename); len(matches) > 1 {
		if id, err := strconv.Atoi(matches[1]); err == nil {
			metadata.ID = id
		}
	}

	// Extract release date (YYYY-MM-DD pattern)
	datePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	if matches := datePattern.FindStringSubmatch(filename); len(matches) > 1 {
		metadata.ReleaseDate = matches[1]
	}

	// Extract version (everything after last underscore until .sql or .sqlite)
	// Only if there's content after the date
	if metadata.ReleaseDate != "" {
		// Find position of release date in filename
		dateIdx := strings.Index(filename, metadata.ReleaseDate)
		if dateIdx != -1 {
			// Get substring after the date
			afterDate := filename[dateIdx+len(metadata.ReleaseDate):]

			// Check if there's an underscore followed by content before extension
			versionPattern := regexp.MustCompile(`^_(.+?)\.(?:sql|sqlite)$`)
			if matches := versionPattern.FindStringSubmatch(afterDate); len(matches) > 1 {
				metadata.Version = matches[1]
			}
		}
	}

	return metadata
}

// IsValidURL checks if a string is a valid URL.
func IsValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

// FilterSources filters data sources based on the filter string.
// Supported filters:
//   - "main": Returns sources with ID < 1000 (official sources)
//   - "exclude main": Returns sources with ID >= 1000 (custom sources)
//   - "1,3,5": Returns sources with specified IDs (comma-separated)
//   - "180-208": Returns sources with IDs in range [180, 208] (inclusive)
//   - "-10": Returns sources with IDs from 1 to 10 (inclusive)
//   - "197-": Returns sources with IDs from 197 to end (inclusive)
//   - "1,5,10-20,50-": Mix of individual IDs and ranges
//   - "": Returns all sources (no filtering)
func FilterSources(sources []DataSourceConfig, filter string) ([]DataSourceConfig, error) {
	filter = strings.TrimSpace(filter)

	// No filter - return all sources
	if filter == "" {
		return sources, nil
	}

	// Handle "main" filter (ID < 1000)
	if filter == "main" {
		var filtered []DataSourceConfig
		for _, src := range sources {
			if src.ID < 1000 {
				filtered = append(filtered, src)
			}
		}
		return filtered, nil
	}

	// Handle "exclude main" filter (ID >= 1000)
	if filter == "exclude main" {
		var filtered []DataSourceConfig
		for _, src := range sources {
			if src.ID >= 1000 {
				filtered = append(filtered, src)
			}
		}
		return filtered, nil
	}

	// Parse comma-separated items (can be individual IDs or ranges)
	items := strings.Split(filter, ",")
	requestedIDs := make(map[int]bool) // All requested IDs
	explicitIDs := make(map[int]bool)  // Only explicitly specified IDs (not from ranges)
	var warnings []string

	for _, item := range items {
		item = strings.TrimSpace(item)

		// Check if this is a range (contains "-")
		if strings.Contains(item, "-") {
			start, end, err := parseRange(item, sources)
			if err != nil {
				return nil, fmt.Errorf("failed to parse range '%s': %w", item, err)
			}

			// Add all IDs in range (will silently skip non-existent ones)
			rangeHasMatches := false
			for id := start; id <= end; id++ {
				requestedIDs[id] = true
				// Check if this ID actually exists
				if sourceExists(sources, id) {
					rangeHasMatches = true
				}
			}

			// Warn if range matched no sources
			if !rangeHasMatches {
				warnings = append(warnings, fmt.Sprintf("range '%s' matched no sources", item))
			}
		} else {
			// Single explicit ID
			id, err := strconv.Atoi(item)
			if err != nil {
				return nil, fmt.Errorf("invalid source ID '%s': must be a number or range", item)
			}
			requestedIDs[id] = true
			explicitIDs[id] = true
		}
	}

	// Collect matching sources and check for missing explicit IDs
	var filtered []DataSourceConfig
	foundIDs := make(map[int]bool)

	for _, src := range sources {
		if requestedIDs[src.ID] {
			filtered = append(filtered, src)
			foundIDs[src.ID] = true
		}
	}

	// Warn about explicitly requested IDs that weren't found
	for id := range explicitIDs {
		if !foundIDs[id] {
			warnings = append(warnings, fmt.Sprintf("source ID %d not found in configuration", id))
		}
	}

	// Return error if no sources matched at all
	if len(filtered) == 0 {
		if len(warnings) > 0 {
			return nil, fmt.Errorf(
				"no sources matched filter '%s': %s",
				filter,
				strings.Join(warnings, "; "),
			)
		}
		return nil, fmt.Errorf("no sources matched filter '%s'", filter)
	}

	// Log warnings if any (using fmt.Fprintf to stderr since we're in pkg/)
	if len(warnings) > 0 {
		for _, warn := range warnings {
			fmt.Fprintf(os.Stderr, "WARNING: %s\n", warn)
		}
	}

	return filtered, nil
}

// parseRange parses a range string like "180-208", "-10", or "197-"
// Returns (start, end, error)
func parseRange(rangeStr string, sources []DataSourceConfig) (int, int, error) {
	parts := strings.Split(rangeStr, "-")

	// Handle different range formats
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format: expected 'X-Y', '-Y', or 'X-'")
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	var start, end int
	var err error

	// Handle "-10" (from 1 to 10)
	if startStr == "" {
		start = 1
		end, err = strconv.Atoi(endStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end value: %w", err)
		}
	} else if endStr == "" {
		// Handle "197-" (from 197 to max)
		start, err = strconv.Atoi(startStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start value: %w", err)
		}
		// Find maximum ID in sources
		end = findMaxSourceID(sources)
		if end == 0 {
			return 0, 0, fmt.Errorf("no sources available to determine end of range")
		}
	} else {
		// Handle "180-208" (from 180 to 208)
		start, err = strconv.Atoi(startStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start value: %w", err)
		}
		end, err = strconv.Atoi(endStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end value: %w", err)
		}
	}

	// Validate range
	if start > end {
		return 0, 0, fmt.Errorf("start (%d) must be <= end (%d)", start, end)
	}

	return start, end, nil
}

// findMaxSourceID returns the maximum ID among all sources
func findMaxSourceID(sources []DataSourceConfig) int {
	maxID := 0
	for _, src := range sources {
		if src.ID > maxID {
			maxID = src.ID
		}
	}
	return maxID
}

// sourceExists checks if a source with the given ID exists
func sourceExists(sources []DataSourceConfig, id int) bool {
	for _, src := range sources {
		if src.ID == id {
			return true
		}
	}
	return false
}

// ExtractOutlinkID extracts the outlink ID from a column value.
// If the column name ends with "col__alternative_id", it extracts the value
// after "gnoutlink:" from a comma-separated list of namespace:value pairs.
// Otherwise, it returns the value as-is.
//
// Examples:
//   - ExtractOutlinkID("taxon.col__id", "12345") → "12345"
//   - ExtractOutlinkID("taxon.col__alternative_id", "wikidata:Q123,gnoutlink:Homo_sapiens") → "Homo_sapiens"
//   - ExtractOutlinkID("name.col__alternative_id", "gnoutlink:url-encoded-name") → "url-encoded-name"
//   - ExtractOutlinkID("taxon.col__alternative_id", "wikidata:Q123") → "" (no gnoutlink namespace)
func ExtractOutlinkID(columnName, value string) string {
	// If column is not col__alternative_id, return value as-is
	if !strings.HasSuffix(columnName, "col__alternative_id") {
		return value
	}

	// Extract gnoutlink: namespace from comma-separated list
	parts := strings.Split(value, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "gnoutlink:") {
			return strings.TrimPrefix(part, "gnoutlink:")
		}
	}

	// gnoutlink namespace not found
	return ""
}

// GenerateExampleConfig creates an example configuration file with all official sources.
func GenerateExampleConfig(path string) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	}

	if err := os.WriteFile(path, []byte(templates.SourcesYAML), 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	return nil
}
