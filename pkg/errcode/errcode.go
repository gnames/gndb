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

	// Schema errors
	SchemaGORMConnectionError
	SchemaCreateError
	SchemaMigrateError
	SchemaCollationError
)
