package errcode

import (
	"github.com/gnames/gn"
)

const (
	UnknownError gn.ErrorCode = iota

	// File System errors
	CreateDirError
	CopyFileError
	ReadFileError

	// Logging errors
	CreateLogFileError

	// Database errors
	DBConnectionError
	DBTableCheckError
	DBEmptyDatabaseError
	DBNotConnectedError
	DBTableExistsCheckError
	DBQueryTablesError
	DBScanTableError
	DBDropTableError
	DBQueryViewsError
	DBScanViewError
	DBDropViewError
	DBCreateViewError
	DBCreateViewIndexError

	// Schema errors
	SchemaGORMConnectionError
	SchemaCreateError
	SchemaMigrateError
	SchemaCollationError
	SchemaAtlasDevSchemaError
	SchemaAtlasDriverError
	SchemaAtlasInspectError
	SchemaAtlasDiffError
	SchemaAtlasPlanError

	// Populate errors
	PopulateSourcesConfigError
	PopulateSFGAFileNotFoundError
	PopulateSFGAReadError
	PopulateSFGAVersionError
	PopulateSFGAVersionTooOldError
	PopulateMetadataError
	PopulateNamesError
	PopulateVernacularsError
	PopulateHierarchyError
	PopulateIndicesError
	PopulateCacheError
	PopulateAllSourcesFailedError

	// Export errors
	ExportNoSourcesError
	ExportOutputDirError
	ExportSFGACreateError
	ExportSFGAWriteError
	ExportAllSourcesFailedError

	// Delete errors
	DeleteQuerySourcesError
	DeleteDatasetError

	// Optimizer errors
	OptimizerReparseError
	OptimizerTempTableError
	OptimizerCanonicalInsertError
	OptimizerVernacularNormalizeError
	OptimizerOrphanRemovalError
	OptimizerWordExtractionError
	OptimizerViewCreationError
	OptimizerVacuumError
)
