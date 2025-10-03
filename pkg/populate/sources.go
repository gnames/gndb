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

	"gopkg.in/yaml.v3"
)

// SourcesConfig represents the complete sources.yaml configuration file.
type SourcesConfig struct {
	// DataSources is the list of data sources to import.
	DataSources []DataSourceConfig `yaml:"data_sources"`

	// Import contains settings for the import process.
	Import ImportConfig `yaml:"import,omitempty"`

	// Logging contains logging and progress settings.
	Logging LoggingConfig `yaml:"logging,omitempty"`
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
	// File is the path or URL to the SFGA file (required).
	// Format: {id}_{name}_{date}_v{version}.(sql|sqlite)[.zip]
	// Examples:
	//   - /data/0001_col_2025-10-03_v2024.1.sqlite.zip
	//   - https://opendata.globalnames.org/sfga/latest/0001.sqlite.zip
	//   - /data/1001.sql (minimal)
	File string `yaml:"file"`

	// Core identification
	// ID extracted from filename, SFGA col__id, or explicit here
	// Convention (not enforced): < 1000 = official, >= 1000 = custom
	ID *int `yaml:"id,omitempty"`

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
	IsOutlinkReady bool   `yaml:"is_outlink_ready,omitempty"` // Can generate outlinks
	OutlinkURL     string `yaml:"outlink_url,omitempty"`      // URL template with {} placeholder
	OutlinkIDField string `yaml:"outlink_id_field,omitempty"` // record_id, local_id, global_id, name_id, canonical
}

// ImportConfig contains settings for the import process.
type ImportConfig struct {
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
	if c.Import.BatchSize == 0 {
		c.Import.BatchSize = 5000
	}
	if c.Import.ConcurrentJobs == 0 {
		c.Import.ConcurrentJobs = 4
	}

	// Set logging defaults
	if c.Logging.LogLevel == "" {
		c.Logging.LogLevel = "info"
	}

	// Validate each data source
	for i := range c.DataSources {
		if err := c.DataSources[i].Validate(); err != nil {
			return fmt.Errorf("data source %d: %w", i+1, err)
		}
	}

	return nil
}

// Validate checks a single data source configuration.
func (d *DataSourceConfig) Validate() error {
	// File is required
	if d.File == "" {
		return fmt.Errorf("file path is required")
	}

	// Check if file is URL or local path
	isURL := IsValidURL(d.File)

	if !isURL {
		// For local files, check if file exists
		if _, err := os.Stat(d.File); os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", d.File)
		}
	}

	// Extract metadata from filename
	metadata := ParseFilename(d.File)

	// Extract or validate ID
	if d.ID == nil {
		if metadata.ID > 0 {
			d.ID = &metadata.ID
		} else {
			return fmt.Errorf("cannot extract ID from filename '%s': must provide 'id' in YAML or use standard filename format", filepath.Base(d.File))
		}
	}

	// Validate data source type if provided
	if d.DataSourceType != "" {
		if d.DataSourceType != "taxonomic" && d.DataSourceType != "nomenclatural" {
			return fmt.Errorf("invalid data_source_type: must be 'taxonomic' or 'nomenclatural'")
		}
	}

	// Validate outlink configuration
	if d.IsOutlinkReady {
		if d.OutlinkURL == "" {
			return fmt.Errorf("outlink_url is required when is_outlink_ready is true")
		}
		if !strings.Contains(d.OutlinkURL, "{}") {
			return fmt.Errorf("outlink_url must contain {} placeholder for ID substitution")
		}
		if d.OutlinkIDField == "" {
			d.OutlinkIDField = "record_id" // Default to record_id
		} else {
			// Validate outlink_id_field
			validFields := []string{"record_id", "local_id", "global_id", "name_id", "canonical", "canonical_full"}
			valid := false
			for _, f := range validFields {
				if d.OutlinkIDField == f {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid outlink_id_field: must be one of %v", validFields)
			}
		}
	}

	return nil
}

// ParseFilename extracts metadata from SFGA filename.
// Expected format: {id}_{name}_{date}_v{version}.(sql|sqlite)[.zip]
// Examples:
//   - 0001_col_2025-10-03_v2024.1.sqlite.zip
//   - 1001.sql (minimal)
func ParseFilename(path string) FileMetadata {
	var metadata FileMetadata

	// Get filename without directory
	filename := filepath.Base(path)

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

	// Extract version (text after _v until .sql or .sqlite)
	versionPattern := regexp.MustCompile(`_v([^_]+?)\.(?:sql|sqlite)`)
	if matches := versionPattern.FindStringSubmatch(filename); len(matches) > 1 {
		metadata.Version = matches[1]
	}

	return metadata
}

// IsValidURL checks if a string is a valid URL.
func IsValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

// GenerateExampleConfig creates an example configuration file with all official sources.
func GenerateExampleConfig(path string) error {
	example := `# sources.yaml - Data Source Configuration for gndb populate
#
# This file defines which SFGA data sources to import.
# See: specs/001-gnverifier-db-lifecycle/sources-yaml-spec.md
#
# Usage:
#   gndb populate --config sources.yaml
#   gndb populate --config sources.yaml --sources 1,3,5
#   gndb populate --config sources.yaml --sources main      # ID < 1000 only
#   gndb populate --config sources.yaml --exclude main      # ID >= 1000 only

# ============================================================================
# Data Sources
# ============================================================================

data_sources:
  # Example: Official source (minimal configuration)
  - file: https://opendata.globalnames.org/sfga/latest/0001_col.sqlite.zip
    title_short: "CoL"

  # Example: Official source from local path
  # - file: /data/0042_itis_2024-06-15_v2024.6.sqlite.zip
  #   title_short: "ITIS"

  # Example: Custom source with full configuration
  # - file: /data/1001_my-herbarium_2025-10-03_v1.0.sql.zip
  #   title: "My Institution Herbarium"
  #   title_short: "MyHerb"
  #   description: "Regional plant collection with taxonomic data"
  #   home_url: "https://myinst.org/herbarium"
  #   data_url: "https://myinst.org/herbarium/download"
  #   data_source_type: "taxonomic"
  #   is_curated: true
  #   has_classification: true
  #   is_outlink_ready: true
  #   outlink_url: "https://myinst.org/specimen/{}"
  #   outlink_id_field: "record_id"

  # Example: Minimal custom source
  # - file: /data/1002.sql
  #   title_short: "LocalList"

# ============================================================================
# Import Settings (optional)
# ============================================================================

import:
  batch_size: 5000              # Records per batch insert (default: 5000)
  concurrent_jobs: 4            # Parallel processing jobs (default: 4)
  prefer_flat_classification: false  # Use flat vs hierarchical (default: false)

# ============================================================================
# Logging Settings (optional)
# ============================================================================

logging:
  show_progress: true           # Show progress bars (default: true)
  log_level: "info"             # debug, info, warn, error (default: info)

# ============================================================================
# Official Data Sources (commented out - uncomment as needed)
# ============================================================================
# ID convention: < 1000 = official, >= 1000 = custom (not enforced)
#
# Official sources from https://opendata.globalnames.org/sfga/latest/
#
# - file: https://opendata.globalnames.org/sfga/latest/0001_col.sqlite.zip
#   title_short: "CoL"
#
# - file: https://opendata.globalnames.org/sfga/latest/0003_itis.sqlite.zip
#   title_short: "ITIS"
#
# - file: https://opendata.globalnames.org/sfga/latest/0009_worms.sqlite.zip
#   title_short: "WoRMS"
#
# ... (add more official sources as needed)

# ============================================================================
# Template for Custom Data Sources
# ============================================================================
# Copy and customize this template for your own data sources (ID >= 1000):
#
# - file: /data/1000_{name}_{date}_v{version}.(sql|sqlite)[.zip]
#   id: 1000                      # >= 1000 for custom sources
#   title: "Full Title"
#   title_short: "ShortName"      # Required for display
#   description: "Detailed description"
#   home_url: "https://example.org"
#   data_url: "https://example.org/download"
#   data_source_type: "taxonomic"  # or "nomenclatural"
#   is_curated: false
#   is_auto_curated: false
#   has_classification: false
#   is_outlink_ready: false
#   outlink_url: "https://example.org/{}"
#   outlink_id_field: "record_id"
`

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	}

	if err := os.WriteFile(path, []byte(example), 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	return nil
}
