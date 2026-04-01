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

// AtlasDevSchemaError creates an error for dev schema operations.
func AtlasDevSchemaError(err error) error {
	msg := `Cannot create or use temporary dev schema for migration planning

<em>Possible causes:</em>
  - Insufficient database permissions to create schemas
  - Schema name conflict

<em>How to fix:</em>
  1. Ensure database user has CREATE SCHEMA permissions
  2. Retry the migration`

	return &gn.Error{
		Code: errcode.SchemaAtlasDevSchemaError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("atlas dev schema error: %w", err),
	}
}

// AtlasDriverError creates an error for Atlas driver initialization.
func AtlasDriverError(err error) error {
	msg := `Cannot initialize Atlas database driver`

	return &gn.Error{
		Code: errcode.SchemaAtlasDriverError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("atlas driver error: %w", err),
	}
}

// AtlasInspectError creates an error for schema inspection failures.
func AtlasInspectError(schemaName string, err error) error {
	msg := `Cannot inspect schema <em>%s</em>

<em>Possible causes:</em>
  - Schema does not exist
  - Insufficient permissions

<em>How to fix:</em>
  1. Ensure the schema exists and is accessible
  2. Check database user permissions`

	return &gn.Error{
		Code: errcode.SchemaAtlasInspectError,
		Msg:  msg,
		Vars: []any{schemaName},
		Err:  fmt.Errorf("atlas inspect schema %s: %w", schemaName, err),
	}
}

// AtlasDiffError creates an error for schema diff computation failures.
func AtlasDiffError(err error) error {
	msg := `Cannot compute schema diff`

	return &gn.Error{
		Code: errcode.SchemaAtlasDiffError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("atlas diff error: %w", err),
	}
}

// AtlasPlanError creates an error for migration plan generation failures.
func AtlasPlanError(err error) error {
	msg := `Cannot generate migration plan from schema diff`

	return &gn.Error{
		Code: errcode.SchemaAtlasPlanError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("atlas plan error: %w", err),
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
