// Package contracts defines interfaces for SFGA data source operations.
package contracts

import (
	"context"
)

// SFGAReader defines operations for reading SFGA format data sources.
// Implemented by: internal/io/sfga/reader.go
// Used by: pkg/populate
type SFGAReader interface {
	// Open opens an SFGA SQLite file for reading.
	// Returns error if file doesn't exist or is not a valid SFGA archive.
	Open(ctx context.Context, filePath string) error

	// Close closes the SFGA file and releases resources.
	Close() error

	// GetVersion returns the SFGA format version from metadata.
	GetVersion(ctx context.Context) (string, error)

	// GetMetadata reads data source metadata (title, version, release date, etc.).
	// Maps to DataSource model.
	GetMetadata(ctx context.Context) (*DataSourceMetadata, error)

	// StreamReferences streams reference records via a channel.
	// Channel is closed when all records are read or error occurs.
	StreamReferences(ctx context.Context) (<-chan ReferenceRecord, <-chan error)

	// StreamNames streams name records via a channel.
	// Channel is closed when all records are read or error occurs.
	StreamNames(ctx context.Context) (<-chan NameRecord, <-chan error)

	// StreamTaxa streams taxon records via a channel.
	// Channel is closed when all records are read or error occurs.
	StreamTaxa(ctx context.Context) (<-chan TaxonRecord, <-chan error)

	// StreamSynonyms streams synonym records via a channel.
	// Channel is closed when all records are read or error occurs.
	StreamSynonyms(ctx context.Context) (<-chan SynonymRecord, <-chan error)

	// StreamVernaculars streams vernacular name records via a channel.
	// Channel is closed when all records are read or error occurs.
	StreamVernaculars(ctx context.Context) (<-chan VernacularRecord, <-chan error)

	// ValidateSchema validates SFGA file against expected schema version.
	// Returns error if schema is incompatible or missing required tables.
	ValidateSchema(ctx context.Context, expectedVersion string) error

	// RecordCount returns the total record count for a table.
	// Used for progress reporting during import.
	RecordCount(ctx context.Context, tableName string) (int64, error)
}

// DataSourceMetadata represents SFGA metadata table data.
type DataSourceMetadata struct {
	UUID           string
	Title          string
	TitleShort     string
	Version        string
	ReleaseDate    string // ISO 8601 format
	HomeURL        string
	Description    string
	DataSourceType string // "taxonomic" or "nomenclatural"
	RecordCount    int64
	SFGAVersion    string
}

// ReferenceRecord represents a row from SFGA reference table.
type ReferenceRecord struct {
	LocalID  string // col__id
	Citation string
	Author   string
	Title    string
	Year     string
	DOI      string
	Link     string
}

// NameRecord represents a row from SFGA name table.
type NameRecord struct {
	LocalID          string // col__id
	ScientificName   string // gn__scientific_name_string
	CanonicalSimple  string // gn__canonical_simple
	CanonicalFull    string // gn__canonical_full
	CanonicalStemmed string // gn__canonical_stemmed
	Authorship       string
	Year             string
	Rank             string // col__rank
}

// TaxonRecord represents a row from SFGA taxon table.
type TaxonRecord struct {
	LocalID         string // col__id
	NameID          string // col__name_id (FK to name)
	ParentID        string // col__parent_id
	AcceptedID      string // col__accepted_id
	Kingdom         string
	Phylum          string
	Class           string
	Order           string
	Family          string
	Genus           string
	Species         string
	TaxonomicStatus string
}

// SynonymRecord represents a row from SFGA synonym table.
type SynonymRecord struct {
	TaxonID string // col__taxon_id (FK to taxon)
	NameID  string // col__name_id (FK to name)
	Status  string
}

// VernacularRecord represents a row from SFGA vernacular table.
type VernacularRecord struct {
	TaxonID         string // col__taxon_id (FK to taxon)
	Name            string // col__name
	Language        string // col__language
	Country         string // col__country
	Locality        string // col__locality
	Transliteration string // col__transliteration
}
