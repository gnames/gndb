package contracts

import "context"

// SFGAReader reads data from SFGA files
type SFGAReader interface {
	// Open opens an SFGA file for reading
	Open(path string) error

	// Close closes the SFGA file
	Close() error

	// LoadMetadata loads SFGA metadata
	LoadMetadata(ctx context.Context) (*SFGAMetadata, error)

	// StreamNames streams scientific names from SFGA
	StreamNames(ctx context.Context) (<-chan NameRecord, <-chan error)

	// StreamVernaculars streams vernacular names from SFGA
	StreamVernaculars(ctx context.Context) (<-chan VernacularRecord, <-chan error)

	// StreamReferences streams references from SFGA
	StreamReferences(ctx context.Context) (<-chan ReferenceRecord, <-chan error)
}

// SFGAMetadata represents SFGA file metadata
type SFGAMetadata struct {
	Version       string
	Title         string
	Description   string
	Issued        string
	Contact       string
	License       string
	RecordCount   int64
	VernacularCount int64
}

// NameRecord represents a scientific name from SFGA
type NameRecord struct {
	ID                string
	ScientificName    string
	CanonicalSimple   string
	CanonicalFull     string
	Authorship        string
	Year              int
	Rank              string
	ParseQuality      int
	Code              string
	TaxonomicStatus   string
	AcceptedNameID    string
	Classification    string
}

// VernacularRecord represents a vernacular name from SFGA
type VernacularRecord struct {
	TaxonID         string
	Name            string
	Language        string
	Country         string
	Locality        string
	Preferred       bool
}

// ReferenceRecord represents a bibliographic reference from SFGA
type ReferenceRecord struct {
	ID      string
	Citation string
	DOI     string
	Author  string
	Year    string
}
