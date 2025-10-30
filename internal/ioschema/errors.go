package ioschema

import (
	"fmt"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
)

// NotConnectedError creates an error for when schema
// operation is attempted without database connection.
func NotConnectedError() error {
	msg := "Schema operation attempted without database connection"

	return &gn.Error{
		Code: errcode.DBNotConnectedError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("not connected to database"),
	}
}

// GORMConnectionError creates an error for GORM
// connection failures.
func GORMConnectionError(err error) error {
	msg := `Cannot connect to database with GORM

<em>Possible causes:</em>
  - Connection pool not initialized
  - Database configuration issue
  - GORM driver problem

<em>How to fix:</em>
  1. Ensure database operator is connected
  2. Check database configuration
  3. Verify GORM dependencies are installed`

	return &gn.Error{
		Code: errcode.SchemaGORMConnectionError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("failed to connect with GORM: %w", err),
	}
}

// CreateSchemaError creates an error for schema
// creation failures.
func CreateSchemaError(err error) error {
	msg := `Cannot create database schema

<em>Possible causes:</em>
  - Insufficient database permissions
  - Invalid schema definitions
  - Database constraint violations

<em>How to fix:</em>
  1. Check database user has CREATE permissions
  2. Review schema model definitions
  3. Check database logs for details`

	return &gn.Error{
		Code: errcode.SchemaCreateError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("failed to create schema: %w", err),
	}
}

// MigrateSchemaError creates an error for schema
// migration failures.
func MigrateSchemaError(err error) error {
	msg := `Cannot migrate database schema

<em>Possible causes:</em>
  - Incompatible schema changes
  - Insufficient database permissions
  - Data integrity issues

<em>How to fix:</em>
  1. Review migration compatibility
  2. Check database user permissions
  3. Backup data before migration`

	return &gn.Error{
		Code: errcode.SchemaMigrateError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("failed to migrate schema: %w", err),
	}
}

// CollationError creates an error for collation
// setting failures.
func CollationError(table, column string, err error) error {
	msg := `Cannot set collation on <em>%s.%s</em>

<em>Possible causes:</em>
  - Table or column does not exist
  - Insufficient database permissions
  - Incompatible data in column

<em>How to fix:</em>
  1. Ensure table was created successfully
  2. Check database user has ALTER permissions
  3. Review column data for compatibility`

	vars := []any{table, column}

	return &gn.Error{
		Code: errcode.SchemaCollationError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf(
			"failed to set collation on %s.%s: %w",
			table, column, err),
	}
}
