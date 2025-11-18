package iodb

import (
	"fmt"

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
  - Database does not exist
  - Database configuration is incorrect
  - Network connectivity issues

<em>How to fix:</em>
  1. Check if PostgreSQL is running: <em>pg_isready</em>
  2. Create database if needed: <em>createdb %s</em>
  3. Verify connection settings in ~/.config/gndb/config.yaml

<em>Note:</em> <em>gndb create</em> creates the schema (tables),
      but the database itself must be created manually`

	vars := []any{host, port, database, user, database}

	return &gn.Error{
		Code: errcode.DBConnectionError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf(
			"failed to connect to %s:%d/%s: %w",
			host, port, database, err),
	}
}

// TableCheckError creates an error for when checking
// table existence fails.
func TableCheckError(err error) error {
	msg := "Cannot check database tables"

	return &gn.Error{
		Code: errcode.DBTableCheckError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("failed to check tables: %w", err),
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

	return &gn.Error{
		Code: errcode.DBEmptyDatabaseError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("database %s has no tables", database),
	}
}

// NotConnectedError creates an error for when operation
// is attempted without connection.
func NotConnectedError() error {
	msg := "Database operation attempted without connection"

	return &gn.Error{
		Code: errcode.DBNotConnectedError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("not connected to database"),
	}
}

// TableExistsCheckError creates an error for when checking
// table existence fails.
func TableExistsCheckError(tableName string, err error) error {
	msg := "Cannot check if table <em>%s</em> exists"
	vars := []any{tableName}

	return &gn.Error{
		Code: errcode.DBTableExistsCheckError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to check table %s: %w", tableName, err),
	}
}

// QueryTablesError creates an error for when querying
// table list fails.
func QueryTablesError(err error) error {
	msg := "Cannot query database tables"

	return &gn.Error{
		Code: errcode.DBQueryTablesError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("failed to query tables: %w", err),
	}
}

// ScanTableError creates an error for when scanning
// table name fails.
func ScanTableError(err error) error {
	msg := "Cannot read table information"

	return &gn.Error{
		Code: errcode.DBScanTableError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("failed to scan table: %w", err),
	}
}

// DropTableError creates an error for when dropping
// table fails.
func DropTableError(tableName string, err error) error {
	msg := "Cannot drop table <em>%s</em>"
	vars := []any{tableName}

	return &gn.Error{
		Code: errcode.DBDropTableError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to drop table %s: %w", tableName, err),
	}
}

// QueryViewsError creates an error for when querying
// materialized view list fails.
func QueryViewsError(err error) error {
	msg := "Cannot query materialized views"

	return &gn.Error{
		Code: errcode.DBQueryViewsError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("failed to query views: %w", err),
	}
}

// ScanViewError creates an error for when scanning
// view name fails.
func ScanViewError(err error) error {
	msg := "Cannot read view information"

	return &gn.Error{
		Code: errcode.DBScanViewError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("failed to scan view: %w", err),
	}
}

// DropViewError creates an error for when dropping
// materialized view fails.
func DropViewError(viewName string, err error) error {
	msg := "Cannot drop materialized view <em>%s</em>"
	vars := []any{viewName}

	return &gn.Error{
		Code: errcode.DBDropViewError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to drop view %s: %w", viewName, err),
	}
}

// CreateViewError creates an error for when creating
// materialized view fails.
func CreateViewError(viewName string, err error) error {
	msg := "Cannot create materialized view <em>%s</em>"
	vars := []any{viewName}

	return &gn.Error{
		Code: errcode.DBCreateViewError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to create view %s: %w", viewName, err),
	}
}

// CreateViewIndexError creates an error for when creating
// index on materialized view fails.
func CreateViewIndexError(viewName string, err error) error {
	msg := "Cannot create index on view <em>%s</em>"
	vars := []any{viewName}

	return &gn.Error{
		Code: errcode.DBCreateViewIndexError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to create index on %s: %w", viewName, err),
	}
}
