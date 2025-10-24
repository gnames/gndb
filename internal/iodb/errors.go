package iodb

import (
	"fmt"

	"github.com/gnames/gnlib"
)

// ConnectionError is returned when database connection fails.
type ConnectionError struct {
	error
	gnlib.MessageBase
}

// NewConnectionError creates a connection error with user-friendly message.
func NewConnectionError(host string, port int, database, user string, err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Database Connection Failed</title>

<warn>Could not connect to PostgreSQL database.</warn>

<em>Possible errs:</em>
  • PostgreSQL is not running
  • Database configuration is incorrect
  • Network connectivity issues

<em>How to fix:</em>
  1. Check if PostgreSQL is running:
     <em>pg_isready -h %s -p %d</em>

  2. Verify database exists:
     <em>psql -h %s -U %s -l</em>

  3. Check your configuration file:
     <em>~/.config/gndb/gndb.yaml</em>

  4. Review connection settings:
     Host: %s
     Port: %d
     Database: %s
     User: %s
`,
		Vars: []any{
			host, port,
			host, user,
			host, port, database, user,
		},
	}

	return ConnectionError{
		error:       fmt.Errorf("failed to connect to %s:%d/%s: %w", host, port, database, err),
		MessageBase: msgBase,
	}
}

// TableCheckError is returned when checking for tables fails.
type TableCheckError struct {
	error
	gnlib.MessageBase
}

// NewTableCheckError creates an error for when table existence check fails.
func NewTableCheckError(err error) error {
	msgBase := gnlib.MessageBase{
		Msg: `<title>Database Check Failed</title>
`,
		Vars: nil}

	return TableCheckError{
		error:       fmt.Errorf("failed to check database tables: %w", err),
		MessageBase: msgBase,
	}
}

// EmptyDatabaseError is returned when database has no tables.
type EmptyDatabaseError struct {
	error
	gnlib.MessageBase
}

// NewEmptyDatabaseError creates an error for unpopulated database.
func NewEmptyDatabaseError(host, database string) error {
	err := fmt.Errorf(
		"database has no tables - run 'gndb create' and 'gndb populate' first",
	)
	msgBase := gnlib.MessageBase{
		Msg: `<title>Database Not Ready</title>

<warn>The database appears to be empty or not populated.</warn>

<em>Required steps before optimization:</e>
  1. Create the database schema:
     <em>gndb create</em>

  2. Populate the database with data:
     <em>gndb populate</em>

  3. Then run optimization:
     <em>gndb optimize</em>

<em>Current database state:</em>
  Host: %s
  Database: %s
  Status: No tables found

<em>Tip:</em> Run <em>gndb populate --help</em> to see population options.

`,
		Vars: []any{host, database},
	}

	return EmptyDatabaseError{
		error:       err,
		MessageBase: msgBase,
	}
}
