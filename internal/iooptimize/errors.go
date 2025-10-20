package iooptimize

import (
	"fmt"

	"github.com/gnames/gnlib"
)

// CacheNotOpenError is returned when cache operations are attempted on a closed database.
type CacheNotOpenError struct {
	error
	gnlib.MessageBase
}

// NewCacheNotOpenError creates a new cache not open error.
func NewCacheNotOpenError() error {
	userBase := gnlib.NewMessage(
		`<title>Cache Database Not Open</title>
<warning>Cannot perform cache operation - database is not open.</warning>

<em>How to fix:</em>
  1. Ensure CacheManager.Open() is called before StoreParsed() or GetParsed()
  2. Check that CacheManager.Close() was not called prematurely
  3. Verify cache initialization in the optimize workflow

This is likely an internal error. Please report if you see this message.`,
		nil,
	)

	return CacheNotOpenError{
		error:       fmt.Errorf("cache database is not open"),
		MessageBase: userBase,
	}
}

// ReparseQueryError is returned when querying name_strings for reparsing fails.
type ReparseQueryError struct {
	error
	gnlib.MessageBase
}

// NewReparseQueryError creates a new reparse query error.
func NewReparseQueryError(cause error) error {
	userBase := gnlib.NewMessage(
		`<title>Cannot Load Names for Reparsing</title>
<warning>Failed to query name_strings table for reparsing.</warning>

<em>How to fix:</em>
  1. Verify the database connection is active
  2. Check that name_strings table exists: <em>psql -d gndb_test -c "\d name_strings"</em>
  3. Ensure the database was populated: <em>gndb populate</em>
  4. Check PostgreSQL logs for query errors

<em>Technical details:</em> %v`,
		[]any{cause},
	)

	return ReparseQueryError{
		error:       fmt.Errorf("failed to query name_strings for reparsing: %w", cause),
		MessageBase: userBase,
	}
}

// ReparseScanError is returned when scanning a row from name_strings fails.
type ReparseScanError struct {
	error
	gnlib.MessageBase
}

// NewReparseScanError creates a new reparse scan error.
func NewReparseScanError(cause error) error {
	userBase := gnlib.NewMessage(
		`<title>Cannot Read Name String Data</title>
<warning>Failed to read name_strings row data during reparsing.</warning>

<em>Possible causes:</em>
  1. Database schema mismatch (try recreating schema)
  2. Data corruption in name_strings table
  3. Unexpected NULL values in required fields

<em>How to fix:</em>
  1. Recreate the database schema: <em>gndb create --drop</em>
  2. Repopulate the database: <em>gndb populate</em>

<em>Technical details:</em> %v`,
		[]any{cause},
	)

	return ReparseScanError{
		error:       fmt.Errorf("failed to scan name_strings row: %w", cause),
		MessageBase: userBase,
	}
}

// ReparseIterationError is returned when iterating over name_strings rows fails.
type ReparseIterationError struct {
	error
	gnlib.MessageBase
}

// NewReparseIterationError creates a new reparse iteration error.
func NewReparseIterationError(cause error) error {
	userBase := gnlib.NewMessage(
		`<title>Error Reading Name Strings</title>
<warning>Failed while iterating through name_strings table.</warning>

<em>Possible causes:</em>
  1. Network connection to database lost
  2. Database server encountered an error
  3. Transaction timeout or deadlock

<em>How to fix:</em>
  1. Check database connection: <em>pg_isready</em>
  2. Review PostgreSQL logs for errors
  3. Retry the optimize operation

<em>Technical details:</em> %v`,
		[]any{cause},
	)

	return ReparseIterationError{
		error:       fmt.Errorf("error iterating name_strings: %w", cause),
		MessageBase: userBase,
	}
}

// ReparseTransactionError is returned when beginning a transaction fails.
type ReparseTransactionError struct {
	error
	gnlib.MessageBase
}

// NewReparseTransactionError creates a new reparse transaction error.
func NewReparseTransactionError(cause error) error {
	userBase := gnlib.NewMessage(
		`<title>Cannot Start Database Transaction</title>
<warning>Failed to begin transaction for updating name_strings.</warning>

<em>Possible causes:</em>
  1. Too many concurrent connections to database
  2. Database server out of resources
  3. Connection pool exhausted

<em>How to fix:</em>
  1. Check PostgreSQL max_connections setting
  2. Reduce --jobs parameter to lower concurrency
  3. Check database server resource usage (CPU, memory)
  4. Review PostgreSQL logs for errors

<em>Technical details:</em> %v`,
		[]any{cause},
	)

	return ReparseTransactionError{
		error:       fmt.Errorf("failed to begin transaction: %w", cause),
		MessageBase: userBase,
	}
}

// ReparseUpdateError is returned when updating name_strings table fails.
type ReparseUpdateError struct {
	error
	gnlib.MessageBase
}

// NewReparseUpdateError creates a new reparse update error.
func NewReparseUpdateError(cause error) error {
	userBase := gnlib.NewMessage(
		`<title>Cannot Update Name String</title>
<warning>Failed to update name_strings table with reparsed data.</warning>

<em>Possible causes:</em>
  1. Database constraint violation
  2. Insufficient disk space
  3. Deadlock or lock timeout

<em>How to fix:</em>
  1. Check disk space: <em>df -h</em>
  2. Review PostgreSQL logs for constraint violations
  3. Retry the optimize operation

<em>Technical details:</em> %v`,
		[]any{cause},
	)

	return ReparseUpdateError{
		error:       fmt.Errorf("failed to update name_strings: %w", cause),
		MessageBase: userBase,
	}
}

// ReparseInsertError is returned when inserting canonical records fails.
type ReparseInsertError struct {
	error
	gnlib.MessageBase
}

// NewReparseInsertError creates a new reparse insert error.
func NewReparseInsertError(table string, cause error) error {
	userBase := gnlib.NewMessage(
		`<title>Cannot Insert Canonical Form</title>
<warning>Failed to insert record into <em>%s</em> table.</warning>

<em>Possible causes:</em>
  1. Database constraint violation
  2. Insufficient disk space
  3. Table does not exist (schema issue)

<em>How to fix:</em>
  1. Check disk space: <em>df -h</em>
  2. Verify table exists: <em>psql -d gndb_test -c "\d %s"</em>
  3. Recreate schema if needed: <em>gndb create --drop</em>

<em>Technical details:</em> %v`,
		[]any{table, table, cause},
	)

	return ReparseInsertError{
		error:       fmt.Errorf("failed to insert into %s: %w", table, cause),
		MessageBase: userBase,
	}
}

// OrphanRemovalError is returned when removing orphan records fails.
type OrphanRemovalError struct {
	error
	gnlib.MessageBase
}

// NewOrphanRemovalError creates a new orphan removal error.
func NewOrphanRemovalError(table string, cause error) error {
	userBase := gnlib.NewMessage(
		`<title>Cannot Remove Orphan Records</title>
<warning>Failed to delete orphan records from <em>%s</em> table.</warning>

<em>Possible causes:</em>
  1. Database constraint violation
  2. Insufficient permissions
  3. Table does not exist (schema issue)
  4. Foreign key constraints blocking deletion

<em>How to fix:</em>
  1. Verify table exists: <em>psql -d gndb_test -c "\d %s"</em>
  2. Check database permissions
  3. Review PostgreSQL logs for errors
  4. Retry the optimize operation

<em>Technical details:</em> %v`,
		[]any{table, table, cause},
	)

	return OrphanRemovalError{
		error:       fmt.Errorf("failed to remove orphans from %s: %w", table, cause),
		MessageBase: userBase,
	}
}
