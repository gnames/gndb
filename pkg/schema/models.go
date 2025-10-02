// Package schema provides database schema models for GNdb.
// Models are defined as Go structs with DDL generation capabilities.
package schema

import (
	"time"
)

// DDLGenerator defines how Go models generate PostgreSQL DDL.
type DDLGenerator interface {
	// TableDDL returns the CREATE TABLE statement for this model.
	TableDDL() string

	// IndexDDL returns CREATE INDEX statements for this model.
	// Returns empty slice if no indexes needed.
	IndexDDL() []string

	// TableName returns the PostgreSQL table name for this model.
	TableName() string
}

// DataSource represents external nomenclature data sources in SFGA format.
type DataSource struct {
	ID             int64     `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
	UUID           string    `db:"uuid" ddl:"TEXT UNIQUE NOT NULL"`
	Title          string    `db:"title" ddl:"TEXT NOT NULL"`
	TitleShort     string    `db:"title_short" ddl:"TEXT NOT NULL"`
	Version        string    `db:"version" ddl:"TEXT NOT NULL"`
	ReleaseDate    time.Time `db:"release_date" ddl:"TIMESTAMP NOT NULL"`
	HomeURL        string    `db:"home_url" ddl:"TEXT"`
	Description    string    `db:"description" ddl:"TEXT"`
	DataSourceType string    `db:"data_source_type" ddl:"TEXT CHECK (data_source_type IN ('taxonomic', 'nomenclatural'))"`
	RecordCount    int64     `db:"record_count" ddl:"BIGINT DEFAULT 0"`
	SFGAVersion    string    `db:"sfga_version" ddl:"TEXT NOT NULL"`
	ImportedAt     time.Time `db:"imported_at" ddl:"TIMESTAMP DEFAULT NOW()"`
}

// NameString represents parsed scientific name strings.
type NameString struct {
	ID               int64     `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
	NameString       string    `db:"name_string" ddl:"TEXT NOT NULL"`
	CanonicalSimple  string    `db:"canonical_simple" ddl:"TEXT"`
	CanonicalFull    string    `db:"canonical_full" ddl:"TEXT"`
	CanonicalStemmed string    `db:"canonical_stemmed" ddl:"TEXT"`
	Authorship       string    `db:"authorship" ddl:"TEXT"`
	Year             string    `db:"year" ddl:"TEXT"`
	ParseQuality     int       `db:"parse_quality" ddl:"SMALLINT CHECK (parse_quality BETWEEN 0 AND 4)"`
	Cardinality      int       `db:"cardinality" ddl:"SMALLINT CHECK (cardinality BETWEEN 0 AND 3)"`
	Virus            bool      `db:"virus" ddl:"BOOLEAN DEFAULT FALSE"`
	Bacteria         bool      `db:"bacteria" ddl:"BOOLEAN DEFAULT FALSE"`
	ParserVersion    string    `db:"parser_version" ddl:"TEXT NOT NULL"`
	UpdatedAt        time.Time `db:"updated_at" ddl:"TIMESTAMP DEFAULT NOW()"`
}

// Taxon represents taxonomic hierarchy and accepted scientific names.
type Taxon struct {
	ID              int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
	DataSourceID    int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
	LocalID         string `db:"local_id" ddl:"TEXT NOT NULL"`
	NameID          int64  `db:"name_id" ddl:"BIGINT NOT NULL REFERENCES name_strings(id)"`
	ParentID        string `db:"parent_id" ddl:"TEXT"`
	AcceptedID      string `db:"accepted_id" ddl:"TEXT"`
	Rank            string `db:"rank" ddl:"TEXT"`
	Kingdom         string `db:"kingdom" ddl:"TEXT"`
	Phylum          string `db:"phylum" ddl:"TEXT"`
	Class           string `db:"class" ddl:"TEXT"`
	OrderName       string `db:"order_name" ddl:"TEXT"`
	Family          string `db:"family" ddl:"TEXT"`
	Genus           string `db:"genus" ddl:"TEXT"`
	Species         string `db:"species" ddl:"TEXT"`
	TaxonomicStatus string `db:"taxonomic_status" ddl:"TEXT"`
}

// Synonym maps alternative scientific names to accepted taxa.
type Synonym struct {
	ID           int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
	DataSourceID int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
	TaxonID      int64  `db:"taxon_id" ddl:"BIGINT NOT NULL REFERENCES taxa(id) ON DELETE CASCADE"`
	NameID       int64  `db:"name_id" ddl:"BIGINT NOT NULL REFERENCES name_strings(id)"`
	Status       string `db:"status" ddl:"TEXT"`
}

// VernacularName represents common names in various languages.
type VernacularName struct {
	ID              int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
	DataSourceID    int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
	TaxonID         int64  `db:"taxon_id" ddl:"BIGINT NOT NULL REFERENCES taxa(id) ON DELETE CASCADE"`
	NameString      string `db:"name_string" ddl:"TEXT NOT NULL"`
	LanguageCode    string `db:"language_code" ddl:"TEXT"`
	Country         string `db:"country" ddl:"TEXT"`
	Locality        string `db:"locality" ddl:"TEXT"`
	Transliteration string `db:"transliteration" ddl:"TEXT"`
}

// Reference represents bibliographic citations for taxonomic data.
type Reference struct {
	ID           int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
	DataSourceID int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
	LocalID      string `db:"local_id" ddl:"TEXT NOT NULL"`
	Citation     string `db:"citation" ddl:"TEXT NOT NULL"`
	Author       string `db:"author" ddl:"TEXT"`
	Title        string `db:"title" ddl:"TEXT"`
	Year         string `db:"year" ddl:"TEXT"`
	DOI          string `db:"doi" ddl:"TEXT"`
	Link         string `db:"link" ddl:"TEXT"`
}

// SchemaVersion tracks database schema migrations.
type SchemaVersion struct {
	Version     string    `db:"version" ddl:"TEXT PRIMARY KEY"`
	Description string    `db:"description" ddl:"TEXT"`
	AppliedAt   time.Time `db:"applied_at" ddl:"TIMESTAMP DEFAULT NOW()"`
}

// NameStringOccurrence tracks where name strings appear across data sources.
type NameStringOccurrence struct {
	ID              int64  `db:"id" ddl:"BIGSERIAL PRIMARY KEY"`
	NameStringID    int64  `db:"name_string_id" ddl:"BIGINT NOT NULL REFERENCES name_strings(id) ON DELETE CASCADE"`
	DataSourceID    int64  `db:"data_source_id" ddl:"BIGINT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE"`
	DataSourceTitle string `db:"data_source_title" ddl:"TEXT NOT NULL"`
	TaxonID         string `db:"taxon_id" ddl:"TEXT"`
	RecordType      string `db:"record_type" ddl:"TEXT CHECK (record_type IN ('accepted', 'synonym', 'vernacular'))"`
	LocalID         string `db:"local_id" ddl:"TEXT"`
	OutlinkID       string `db:"outlink_id" ddl:"TEXT"`
	AcceptedNameID  int64  `db:"accepted_name_id" ddl:"BIGINT REFERENCES name_strings(id)"`
}
