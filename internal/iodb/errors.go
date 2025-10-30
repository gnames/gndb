package iodb

import (
	"fmt"
	"runtime"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
)

// ConnectionError creates an error for database
// connection failures.
func ConnectionError(
	host string,
	port int,
	database string,
	user string,
	err error,
) error {
	msg := `Cannot connect to PostgreSQL database

<em>Connection details:</em>
  Host:     %s
  Port:     %d
  Database: %s
  User:     %s

<em>Possible causes:</em>
  - PostgreSQL is not running
  - Database configuration is incorrect
  - Network connectivity issues

<em>How to fix:</em>
  1. Check if PostgreSQL is running
  2. Verify database exists
  3. Review ~/.config/gndb/config.yaml`

	vars := []any{host, port, database, user}
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	return &gn.Error{
		Code: errcode.DBConnectionError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf("from %s: failed to connect to %s:%d/%s: %w",
			fn, host, port, database, err),
	}
}

// TableCheckError creates an error for when checking
// table existence fails.
func TableCheckError(err error) error {
	msg := "Cannot check database tables"
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	return &gn.Error{
		Code: errcode.DBTableCheckError,
		Msg:  msg,
		Vars: nil,
		Err: fmt.Errorf("from %s: failed to check tables: %w",
			fn, err),
	}
}

// EmptyDatabaseError creates an error for when database
// has no tables.
func EmptyDatabaseError(host, database string) error {
	msg := `Database appears to be empty

<em>Database state:</em>
  Host:     %s
  Database: %s
  Status:   No tables found

<em>Required steps:</em>
  1. Create the schema: <em>gndb create</em>
  2. Populate data:     <em>gndb populate</em>
  3. Then optimize:     <em>gndb optimize</em>`

	vars := []any{host, database}
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	return &gn.Error{
		Code: errcode.DBEmptyDatabaseError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf("from %s: database %s has no tables",
			fn, database),
	}
}

// NotConnectedError creates an error for when operation
// is attempted without connection.
func NotConnectedError() error {
	msg := "Database operation attempted without connection"
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	return &gn.Error{
		Code: errcode.DBNotConnectedError,
		Msg:  msg,
		Vars: nil,
		Err: fmt.Errorf("from %s: not connected to database",
			fn),
	}
}

// TableExistsCheckError creates an error for when checking
// table existence fails.
func TableExistsCheckError(tableName string, err error) error {
	msg := "Cannot check if table <em>%s</em> exists"
	vars := []any{tableName}
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	return &gn.Error{
		Code: errcode.DBTableExistsCheckError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf(
			"from %s: failed to check table %s: %w",
			fn, tableName, err),
	}
}

// QueryTablesError creates an error for when querying
// table list fails.
func QueryTablesError(err error) error {
	msg := "Cannot query database tables"
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	return &gn.Error{
		Code: errcode.DBQueryTablesError,
		Msg:  msg,
		Vars: nil,
		Err: fmt.Errorf("from %s: failed to query tables: %w",
			fn, err),
	}
}

// ScanTableError creates an error for when scanning
// table name fails.
func ScanTableError(err error) error {
	msg := "Cannot read table information"
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	return &gn.Error{
		Code: errcode.DBScanTableError,
		Msg:  msg,
		Vars: nil,
		Err: fmt.Errorf("from %s: failed to scan table: %w",
			fn, err),
	}
}

// DropTableError creates an error for when dropping
// table fails.
func DropTableError(tableName string, err error) error {
	msg := "Cannot drop table <em>%s</em>"
	vars := []any{tableName}
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	return &gn.Error{
		Code: errcode.DBDropTableError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf("from %s: failed to drop table %s: %w",
			fn, tableName, err),
	}
}
