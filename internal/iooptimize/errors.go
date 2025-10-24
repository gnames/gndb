package iooptimize

import (
	"fmt"

	"github.com/gnames/gnlib"
)

// ReparseQueryError is returned when querying name_strings for reparsing fails.
type ReparseQueryError struct {
	error
	gnlib.MessageBase
}

// NewReparseQueryError creates a new reparse query error.
func NewReparseQueryError(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Cannot Load Names for Reparsing</title>
<warn>Failed to query name_strings table for reparsing.</warn>

<em>How to fix:</em>
  1. Verify the database connection is active
  2. Check that name_strings table exists: <em>psql -d gndb_test -c "\d name_strings"</em>
  3. Ensure the database was populated: <em>gndb populate</em>
  4. Check PostgreSQL logs for query errors
`,
		Vars: nil,
	}

	return ReparseQueryError{
		error:       fmt.Errorf("failed to query name_strings for reparsing: %w", err),
		MessageBase: msgBase,
	}
}

// ReparseScanError is returned when scanning a row from name_strings fails.
type ReparseScanError struct {
	error
	gnlib.MessageBase
}

// NewReparseScanError creates a new reparse scan error.
func NewReparseScanError(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Cannot Read Name String Data</title>
<warn>Failed to read name_strings row data during reparsing.</warn>

<em>Possible causes:</em>
  1. Database schema mismatch (try recreating schema)
  2. Data corruption in name_strings table
  3. Unexpected NULL values in required fields

<em>How to fix:</em>
  1. Recreate the database schema: <em>gndb create --drop</em>
  2. Repopulate the database: <em>gndb populate</em>

`,
		Vars: nil,
	}

	return ReparseScanError{
		error:       fmt.Errorf("failed to scan name_strings row: %w", err),
		MessageBase: msgBase,
	}
}

// ReparseIterationError is returned when iterating over name_strings rows fails.
type ReparseIterationError struct {
	error
	gnlib.MessageBase
}

// NewReparseIterationError creates a new reparse iteration error.
func NewReparseIterationError(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Error Reading Name Strings</title>
<warn>Failed while iterating through name_strings table.</warn>

<em>Possible causes:</em>
  1. Network connection to database lost
  2. Database server encountered an error
  3. Transaction timeout or deadlock

<em>How to fix:</em>
  1. Check database connection: <em>pg_isready</em>
  2. Review PostgreSQL logs for errors
  3. Retry the optimize operation

`,
		Vars: nil,
	}

	return ReparseIterationError{
		error:       fmt.Errorf("error iterating name_strings: %w", err),
		MessageBase: msgBase,
	}
}

// ReparseTransactionError is returned when beginning a transaction fails.
type ReparseTransactionError struct {
	error
	gnlib.MessageBase
}

// NewReparseTransactionError creates a new reparse transaction error.
func NewReparseTransactionError(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Cannot Start Database Transaction</title>
<warn>Failed to begin transaction for updating name_strings.</warn>

<em>Possible causes:</em>
  1. Too many concurrent connections to database
  2. Database server out of resources
  3. Connection pool exhausted

<em>How to fix:</em>
  1. Check PostgreSQL max_connections setting
  2. Reduce --jobs parameter to lower concurrency
  3. Check database server resource usage (CPU, memory)
  4. Review PostgreSQL logs for errors

`,
		Vars: nil,
	}

	return ReparseTransactionError{
		error:       fmt.Errorf("failed to begin transaction: %w", err),
		MessageBase: msgBase,
	}
}

// ReparseUpdateError is returned when updating name_strings table fails.
type ReparseUpdateError struct {
	error
	gnlib.MessageBase
}

// NewReparseUpdateError creates a new reparse update error.
func NewReparseUpdateError(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Cannot Update Name String</title>
<warn>Failed to update name_strings table with reparsed data.</warn>

<em>Possible causes:</em>
  1. Database constraint violation
  2. Insufficient disk space
  3. Deadlock or lock timeout

<em>How to fix:</em>
  1. Check disk space: <em>df -h</em>
  2. Review PostgreSQL logs for constraint violations
  3. Retry the optimize operation
`,
		Vars: []any{err},
	}

	return ReparseUpdateError{
		error:       fmt.Errorf("failed to update name_strings: %w", err),
		MessageBase: msgBase,
	}
}

// ReparseInsertError is returned when inserting canonical records fails.
type ReparseInsertError struct {
	error
	gnlib.MessageBase
}

// NewReparseInsertError creates a new reparse insert error.
func NewReparseInsertError(table string, err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Cannot Insert Canonical Form</title>
<warn>Failed to insert record into <em>%s</em> table.</warn>

<em>Possible causes:</em>
  1. Database constraint violation
  2. Insufficient disk space
  3. Table does not exist (schema issue)

<em>How to fix:</em>
  1. Check disk space: <em>df -h</em>
  2. Verify table exists: <em>psql -d gndb_test -c "\d %s"</em>
  3. Recreate schema if needed: <em>gndb create --drop</em>

`,
		Vars: []any{table, table},
	}

	return ReparseInsertError{
		error:       fmt.Errorf("failed to insert into %s: %w", table, err),
		MessageBase: msgBase,
	}
}

// OrphanRemovalError is returned when removing orphan records fails.
type OrphanRemovalError struct {
	error
	gnlib.MessageBase
}

// NewOrphanRemovalError creates a new orphan removal error.
func NewOrphanRemovalError(table string, err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Cannot Remove Orphan Records</title>
<warn>Failed to delete orphan records from <em>%s</em> table.</warn>

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

`,
		Vars: []any{table, table},
	}

	return OrphanRemovalError{
		error:       fmt.Errorf("failed to remove orphans from %s: %w", table, err),
		MessageBase: msgBase,
	}
}

// OptimizeStepError is returned when an optimization step fails.
// Each step has a specific error type for better diagnostics.

// Step1Error is returned when Step 1 (reparse names) fails.
type Step1Error struct {
	error
	gnlib.MessageBase
}

// NewStep1Error creates an error for Step 1 failure.
func NewStep1Error(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Step 1 Failed: Reparse Names</title>
<warn>Failed to reparse name_strings with latest gnparser algorithms.</warn>

<em>This step updates all scientific names with the latest parsing logic.</em>

<em>Possible causes:</em>
  1. Database connection lost during processing
  2. Insufficient disk space for updates
  3. gnparser parsing errors
  4. Transaction conflicts or deadlocks

<em>How to fix:</em>
  1. Check database connection: <em>pg_isready</em>
  2. Check disk space: <em>df -h</em>
  3. Review PostgreSQL logs for errors
  4. Retry the optimize operation

`,
		Vars: nil,
	}

	return Step1Error{
		error:       fmt.Errorf("step 1 failed (reparse names): %w", err),
		MessageBase: msgBase,
	}
}

// Step2Error is returned when Step 2 (fix vernacular languages) fails.
type Step2Error struct {
	error
	gnlib.MessageBase
}

// NewStep2Error creates an error for Step 2 failure.
func NewStep2Error(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Step 2 Failed: Normalize Vernacular Languages</title>
<warn>Failed to normalize vernacular language codes.</warn>

<em>This step converts language codes to standard 3-letter ISO codes.</em>

<em>Possible causes:</em>
  1. Database connection lost during processing
  2. Insufficient disk space for updates
  3. Invalid language codes in data
  4. Transaction conflicts or deadlocks

<em>How to fix:</em>
  1. Check database connection: <em>pg_isready</em>
  2. Check disk space: <em>df -h</em>
  3. Review PostgreSQL logs for errors
  4. Retry the optimize operation

`,
		Vars: nil,
	}

	return Step2Error{
		error:       fmt.Errorf("step 2 failed (fix vernacular languages): %w", err),
		MessageBase: msgBase,
	}
}

// Step3Error is returned when Step 3 (remove orphans) fails.
type Step3Error struct {
	error
	gnlib.MessageBase
}

// NewStep3Error creates an error for Step 3 failure.
func NewStep3Error(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Step 3 Failed: Remove Orphaned Records</title>
<warn>Failed to remove orphaned records from database.</warn>

<em>This step cleans up unreferenced name_strings and canonical forms.</em>

<em>Possible causes:</em>
  1. Database connection lost during processing
  2. Foreign key constraint violations
  3. Insufficient permissions
  4. Deadlocks during deletion

<em>How to fix:</em>
  1. Check database connection: <em>pg_isready</em>
  2. Review PostgreSQL logs for constraint violations
  3. Check database permissions
  4. Retry the optimize operation

`,
		Vars: nil,
	}

	return Step3Error{
		error:       fmt.Errorf("step 3 failed (remove orphans): %w", err),
		MessageBase: msgBase,
	}
}

// Step4Error is returned when Step 4 (create words) fails.
type Step4Error struct {
	error
	gnlib.MessageBase
}

// NewStep4Error creates an error for Step 4 failure.
func NewStep4Error(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Step 4 Failed: Create Words Tables</title>
<warn>Failed to extract and link words for fuzzy matching.</warn>

<em>This step creates the words and word_name_strings tables for search.</em>

<em>Possible causes:</em>
  1. Database connection lost during processing
  2. Insufficient disk space for word tables
  3. Memory exhaustion during word extraction
  4. Bulk insert failures

<em>How to fix:</em>
  1. Check database connection: <em>pg_isready</em>
  2. Check disk space: <em>df -h</em>
  3. Check available memory: <em>free -h</em>
  4. Review PostgreSQL logs for errors
  5. Retry the optimize operation

`,
		Vars: nil,
	}

	return Step4Error{
		error:       fmt.Errorf("step 4 failed (create words): %w", err),
		MessageBase: msgBase,
	}
}

// Step5Error is returned when Step 5 (create verification view) fails.
type Step5Error struct {
	error
	gnlib.MessageBase
}

// NewStep5Error creates an error for Step 5 failure.
func NewStep5Error(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Step 5 Failed: Create Verification View</title>
<warn>Failed to create verification materialized view.</warn>

<em>This step creates the verification view and indexes for gnverifier.</em>

<em>Possible causes:</em>
  1. Database connection lost during processing
  2. Insufficient disk space for materialized view
  3. Missing required tables (name_strings, name_string_indices)
  4. Index creation failures

<em>How to fix:</em>
  1. Check database connection: <em>pg_isready</em>
  2. Check disk space: <em>df -h</em>
  3. Verify required tables exist
  4. Review PostgreSQL logs for errors
  5. Retry the optimize operation

`,
		Vars: nil,
	}

	return Step5Error{
		error:       fmt.Errorf("step 5 failed (create verification view): %w", err),
		MessageBase: msgBase,
	}
}

// Step6Error is returned when Step 6 (vacuum analyze) fails.
type Step6Error struct {
	error
	gnlib.MessageBase
}

// NewStep6Error creates an error for Step 6 failure.
func NewStep6Error(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Step 6 Failed: VACUUM ANALYZE</title>
<warn>Failed to run VACUUM ANALYZE on database.</warn>

<em>This step reclaims storage and updates query planner statistics.</em>

<em>Possible causes:</em>
  1. Database connection lost during VACUUM
  2. Insufficient disk space for temporary files
  3. Long-running transactions blocking VACUUM
  4. PostgreSQL configuration issues

<em>How to fix:</em>
  1. Check database connection: <em>pg_isready</em>
  2. Check disk space: <em>df -h</em>
  3. Check for long-running transactions: <em>SELECT * FROM pg_stat_activity</em>
  4. Review PostgreSQL logs for errors
  5. Retry the optimize operation

`,
		Vars: nil,
	}

	return Step6Error{
		error:       fmt.Errorf("step 6 failed (vacuum analyze): %w", err),
		MessageBase: msgBase,
	}
}
