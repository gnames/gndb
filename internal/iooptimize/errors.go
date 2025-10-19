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

// NotImplementedError is returned when a function is called that hasn't been implemented yet.
// This is used in TDD workflow to make tests fail during the red phase.
type NotImplementedError struct {
	error
	gnlib.MessageBase
}

// errNotImplemented creates a not implemented error for TDD red phase.
func errNotImplemented(functionName string) error {
	userBase := gnlib.NewMessage(
		`<title>Function Not Yet Implemented</title>
<warning>The function <em>%s</em> has not been implemented yet.</warning>

This is expected during test-driven development (TDD).
The test should fail until the implementation is complete.`,
		[]any{functionName},
	)

	return NotImplementedError{
		error:       fmt.Errorf("%s is not yet implemented", functionName),
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
