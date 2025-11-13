// Package sources provides configuration and validation for SFGA data sources.
//
// This package defines the schema for sources.yaml, which users provide to
// specify which SFGA (Standard Format for Global Archiving) data sources to
// import. It handles source configuration validation, filtering, and metadata
// extraction from SFGA filenames.
//
// See sources-yaml-spec.md for the complete sources.yaml specification.
package sources

type Sources interface {
	Load() (*SourcesConfig, error)
}

// SourcesConfig represents the complete sources.yaml configuration file.
type SourcesConfig struct {
	// DataSources is the list of data sources to import.
	DataSources []DataSourceConfig `yaml:"data_sources"`

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

// FileMetadata contains metadata extracted from SFGA filename.
type FileMetadata struct {
	ID          int    // Extracted from filename
	Version     string // Extracted from filename (if present)
	ReleaseDate string // Extracted from filename in YYYY-MM-DD format (if present)
	IsURL       bool   // True if file is a URL
}
