// Package contracts defines interfaces for data import operations.
package contracts

import (
	"context"
)

// Importer defines operations for importing SFGA data into PostgreSQL.
// Implemented by: internal/io/database/importer.go
// Used by: pkg/populate
type Importer interface {
	// ImportDataSource inserts data source metadata.
	// Returns the assigned PostgreSQL ID for foreign key references.
	ImportDataSource(ctx context.Context, metadata *DataSourceMetadata) (int64, error)

	// ImportReferences batch inserts reference records using COPY protocol.
	// Returns count of inserted records and any error.
	ImportReferences(ctx context.Context, dataSourceID int64, records <-chan ReferenceRecord) (int64, error)

	// ImportNames batch inserts name string records using COPY protocol.
	// Deduplicates by canonical_simple (same canonical = same name_string_id).
	// Returns mapping of SFGA local_id → PostgreSQL name_string_id.
	ImportNames(ctx context.Context, dataSourceID int64, records <-chan NameRecord) (map[string]int64, int64, error)

	// ImportTaxa batch inserts taxon records using COPY protocol.
	// Returns mapping of SFGA local_id → PostgreSQL taxon_id.
	ImportTaxa(ctx context.Context, dataSourceID int64, nameIDMap map[string]int64, records <-chan TaxonRecord) (map[string]int64, int64, error)

	// ImportSynonyms batch inserts synonym records using COPY protocol.
	// Returns count of inserted records and any error.
	ImportSynonyms(ctx context.Context, dataSourceID int64, nameIDMap, taxonIDMap map[string]int64, records <-chan SynonymRecord) (int64, error)

	// ImportVernaculars batch inserts vernacular name records using COPY protocol.
	// Returns count of inserted records and any error.
	ImportVernaculars(ctx context.Context, dataSourceID int64, taxonIDMap map[string]int64, records <-chan VernacularRecord) (int64, error)

	// CreateOccurrences generates name_string_occurrences from imported data.
	// Called after all other imports complete.
	// Returns count of created occurrence records.
	CreateOccurrences(ctx context.Context, dataSourceID int64) (int64, error)

	// DisableIndexes drops all indexes except primary keys (speeds up bulk import).
	// Indexes recreated during restructure phase.
	DisableIndexes(ctx context.Context, tableName string) error

	// EnableIndexes recreates indexes after bulk import completes.
	EnableIndexes(ctx context.Context, tableName string) error

	// SetBatchSize configures the number of records per batch insert.
	// Optimal sizes: 5000 for names, 2000 for taxa, 1000 for references.
	SetBatchSize(recordType string, batchSize int)

	// GetImportStats returns statistics for the last import operation.
	GetImportStats(ctx context.Context, dataSourceID int64) (*ImportStats, error)
}

// ImportStats tracks import operation metrics.
type ImportStats struct {
	DataSourceID     int64
	ReferencesCount  int64
	NameStringsCount int64
	TaxaCount        int64
	SynonymsCount    int64
	VernacularsCount int64
	OccurrencesCount int64
	ImportDuration   int64 // milliseconds
	RecordsPerSecond int64
}

// ProgressReporter defines callbacks for import progress tracking.
// Implemented by: CLI output handlers
// Used by: internal/io/database/importer.go
type ProgressReporter interface {
	// OnStart called when import phase begins.
	OnStart(phase string, totalRecords int64)

	// OnProgress called periodically during import with current count.
	OnProgress(phase string, processedRecords int64)

	// OnComplete called when import phase finishes successfully.
	OnComplete(phase string, stats *ImportStats)

	// OnError called when import fails.
	OnError(phase string, err error)
}
