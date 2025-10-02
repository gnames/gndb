// Package schema provides database schema models for GNdb.
// Models are aligned with gnidump for gnverifier compatibility.
package schema

import (
	"database/sql"
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

// DataSource stores metadata associated with a dataset.
type DataSource struct {
	// ID is a hard-coded identifier that aligns with historical IDs from
	// older versions of the resolver.
	ID int `db:"id" ddl:"SMALLINT PRIMARY KEY"`

	// UUID is a unique identifier assigned to the resource upon creation.
	UUID string `db:"uuid" ddl:"UUID DEFAULT '00000000-0000-0000-0000-000000000000'"`

	// Title is the full, descriptive title of the dataset.
	Title string `db:"title" ddl:"VARCHAR(255)"`

	// TitleShort is a concise or abbreviated version of the dataset title.
	TitleShort string `db:"title_short" ddl:"VARCHAR(50)"`

	// Version denotes the specific version of the dataset.
	Version string `db:"version" ddl:"VARCHAR(50)"`

	// RevisionDate indicates when the dataset was created or last revised.
	RevisionDate string `db:"revision_date" ddl:"VARCHAR(50)"`

	// DOI is the Digital Object Identifier for the dataset.
	DOI string `db:"doi" ddl:"VARCHAR(50)"`

	// Citation provides the recommended way to reference the dataset.
	Citation string `db:"citation" ddl:"TEXT"`

	// Authors lists the individuals or organizations responsible.
	Authors string `db:"authors" ddl:"TEXT"`

	// Description offers a summary of the dataset's content and purpose.
	Description string `db:"description" ddl:"TEXT"`

	// WebsiteURL is the primary web address associated with the dataset.
	WebsiteURL string `db:"website_url" ddl:"VARCHAR(255)"`

	// DataURL is the original URL from which the dataset was downloaded.
	DataURL string `db:"data_url" ddl:"VARCHAR(255)"`

	// OutlinkURL is a template for generating external links.
	OutlinkURL string `db:"outlink_url" ddl:"TEXT"`

	// IsOutlinkReady signifies if the data source is mature and reliable.
	IsOutlinkReady bool `db:"is_outlink_ready" ddl:"BOOLEAN"`

	// IsCurated is true when the dataset has undergone manual curation.
	IsCurated bool `db:"is_curated" ddl:"BOOLEAN"`

	// IsAutoCurated is true for automated curation.
	IsAutoCurated bool `db:"is_auto_curated" ddl:"BOOLEAN"`

	// HasTaxonData indicates if the dataset contains taxonomic data.
	HasTaxonData bool `db:"has_taxon_data" ddl:"BOOLEAN"`

	// RecordCount is the total number of name records.
	RecordCount int `db:"record_count" ddl:"INT"`

	// VernRecordCount is the number of vernacular string indices.
	VernRecordCount int `db:"vern_record_count" ddl:"INT"`

	// UpdatedAt records the timestamp of the dataset's last import.
	UpdatedAt time.Time `db:"updated_at" ddl:"TIMESTAMP WITHOUT TIME ZONE"`
}

// NameString is a name-string extracted from a dataset.
type NameString struct {
	// ID is UUID v5 generated from the name-string using DNS:"globalnames.org".
	ID string `db:"id" ddl:"UUID PRIMARY KEY"`

	// Name is the name-string with authorships and annotations.
	Name string `db:"name" ddl:"VARCHAR(255) NOT NULL"`

	// Year is the year when a name was published.
	Year sql.NullInt16 `db:"year" ddl:"INT"`

	// Cardinality: 0-unknown, 1-uninomial, 2-binomial, 3-trinomial.
	Cardinality sql.NullInt32 `db:"cardinality" ddl:"INT"`

	// CanonicalID is UUID v5 for simple canonical form.
	CanonicalID sql.NullString `db:"canonical_id" ddl:"UUID"`

	// CanonicalFullID is UUID v5 for full canonical form.
	CanonicalFullID sql.NullString `db:"canonical_full_id" ddl:"UUID"`

	// CanonicalStemID is UUID v5 for stemmed canonical form.
	CanonicalStemID sql.NullString `db:"canonical_stem_id" ddl:"UUID"`

	// Virus indicates if a name-string seems to be virus-like.
	Virus bool `db:"virus" ddl:"BOOLEAN"`

	// Bacteria is true if parser marks a name as from Bacterial Code.
	Bacteria bool `db:"bacteria" ddl:"BOOLEAN NOT NULL DEFAULT FALSE"`

	// Surrogate indicates if a name-string is a surrogate name.
	Surrogate bool `db:"surrogate" ddl:"BOOLEAN"`

	// ParseQuality: 0-no parse, 1-clear, 2-some problems, 3-big problems.
	ParseQuality int `db:"parse_quality" ddl:"INT NOT NULL DEFAULT 0"`
}

// Canonical is a 'simple' canonical form.
type Canonical struct {
	// ID is UUID v5 generated for simple canonical form.
	ID string `db:"id" ddl:"UUID PRIMARY KEY"`

	// Name is the canonical name-string.
	Name string `db:"name" ddl:"VARCHAR(255) NOT NULL"`
}

// CanonicalFull is a full canonical form.
type CanonicalFull struct {
	// ID is UUID v5 generated for full canonical form.
	ID string `db:"id" ddl:"UUID PRIMARY KEY"`

	// Name is the full canonical name-string with infraspecific ranks.
	Name string `db:"name" ddl:"VARCHAR(255) NOT NULL"`
}

// CanonicalStem is a stemmed derivative of a simple canonical form.
type CanonicalStem struct {
	// ID is UUID v5 for the stemmed derivative.
	ID string `db:"id" ddl:"UUID PRIMARY KEY"`

	// Name is the stemmed canonical name-string.
	Name string `db:"name" ddl:"VARCHAR(255) NOT NULL"`
}

// NameStringIndex represents name-string relations to datasets.
type NameStringIndex struct {
	// DataSourceID refers to a data-source ID.
	DataSourceID int `db:"data_source_id" ddl:"SMALLINT NOT NULL"`

	// RecordID is a unique ID for the record.
	RecordID string `db:"record_id" ddl:"VARCHAR(255) NOT NULL"`

	// NameStringID is UUID5 of a full name-string from the dataset.
	NameStringID string `db:"name_string_id" ddl:"UUID NOT NULL"`

	// OutlinkID is the id to create an outlink.
	OutlinkID string `db:"outlink_id" ddl:"VARCHAR(255)"`

	// GlobalID from the dataset.
	GlobalID string `db:"global_id" ddl:"VARCHAR(255)"`

	// NameID is an ID of a nomenclatural name provided by data source.
	NameID string `db:"name_id" ddl:"VARCHAR(255)"`

	// LocalID from the dataset.
	LocalID string `db:"local_id" ddl:"VARCHAR(255)"`

	// CodeID: 0-no info, 1-ICZN, 2-ICN, 3-ICNP, 4-ICTV.
	CodeID int `db:"code_id" ddl:"SMALLINT"`

	// Rank of the name.
	Rank string `db:"rank" ddl:"VARCHAR(255)"`

	// TaxonomicStatus: accepted, synonym, etc.
	TaxonomicStatus string `db:"taxonomic_status" ddl:"VARCHAR(255)"`

	// AcceptedRecordID of currently accepted name-string for the taxon.
	AcceptedRecordID string `db:"accepted_record_id" ddl:"VARCHAR(255)"`

	// Classification is pipe-delimited classification.
	Classification string `db:"classification" ddl:"TEXT"`

	// ClassificationIDs are RecordIDs of classification elements.
	ClassificationIDs string `db:"classification_ids" ddl:"TEXT"`

	// ClassificationRanks are ranks of classification elements.
	ClassificationRanks string `db:"classification_ranks" ddl:"TEXT"`
}

// Word is a word from a name-string.
type Word struct {
	// ID generated by combining modified word and type.
	ID string `db:"id" ddl:"UUID PRIMARY KEY"`

	// Normalized is the word normalized by GNparser.
	Normalized string `db:"normalized" ddl:"VARCHAR(250) NOT NULL"`

	// Modified is a heavy-normalized word used for matching.
	Modified string `db:"modified" ddl:"VARCHAR(250) NOT NULL"`

	// TypeID is the integer representation of parsed.WordType.
	TypeID int `db:"type_id" ddl:"INT"`
}

// WordNameString is the meaning of a word in a name-string.
type WordNameString struct {
	// WordID is the identifier of a word.
	WordID string `db:"word_id" ddl:"UUID NOT NULL"`

	// NameStringID is UUID5 of a full name-string.
	NameStringID string `db:"name_string_id" ddl:"UUID NOT NULL"`

	// CanonicalID is UUID5 of a simple canonical form.
	CanonicalID string `db:"canonical_id" ddl:"UUID NOT NULL"`
}

// VernacularString contains vernacular name-strings.
type VernacularString struct {
	// ID is UUID v5 generated from the name-string.
	ID string `db:"id" ddl:"UUID PRIMARY KEY"`

	// Name is a vernacular name as given by the dataset.
	Name string `db:"name" ddl:"VARCHAR(500) NOT NULL"`
}

// VernacularStringIndex links vernacular strings to datasets.
type VernacularStringIndex struct {
	// DataSourceID refers to a data-source ID.
	DataSourceID int `db:"data_source_id" ddl:"SMALLINT NOT NULL"`

	// RecordID is a unique ID for the record.
	RecordID string `db:"record_id" ddl:"VARCHAR(255) NOT NULL"`

	// VernacularStringID is UUID5 of the vernacular string.
	VernacularStringID string `db:"vernacular_string_id" ddl:"UUID NOT NULL"`

	// LanguageOrig is the vernacular name language verbatim.
	LanguageOrig string `db:"language_orig" ddl:"VARCHAR(255)"`

	// Language after normalization.
	Language string `db:"language" ddl:"VARCHAR(255)"`

	// LangCode is a three-letter code of the language.
	LangCode string `db:"lang_code" ddl:"VARCHAR(3)"`

	// Locality of the vernacular name.
	Locality string `db:"locality" ddl:"VARCHAR(255)"`

	// CountryCode of the vernacular name.
	CountryCode string `db:"country_code" ddl:"VARCHAR(50)"`

	// Preferred is true if this name is preferred for the language.
	Preferred bool `db:"preferred" ddl:"BOOLEAN"`
}

// SchemaVersion tracks database schema migrations.
type SchemaVersion struct {
	Version     string    `db:"version" ddl:"TEXT PRIMARY KEY"`
	Description string    `db:"description" ddl:"TEXT"`
	AppliedAt   time.Time `db:"applied_at" ddl:"TIMESTAMP DEFAULT NOW()"`
}
