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
func NewConnectionError(host string, port int, database, user string, cause error) error {
	userBase := gnlib.NewMessage(
		`<title>Database Connection Failed</title>

<warning>Could not connect to PostgreSQL database.</warning>

<em>Possible causes:</em>
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
		[]any{
			host, port,
			host, user,
			host, port, database, user,
		},
	)

	return ConnectionError{
		error:       fmt.Errorf("failed to connect to %s:%d/%s: %w", host, port, database, cause),
		MessageBase: userBase,
	}
}

// TableCheckError is returned when checking for tables fails.
type TableCheckError struct {
	error
	gnlib.MessageBase
}

// NewTableCheckError creates an error for when table existence check fails.
func NewTableCheckError(cause error) error {
	userBase := gnlib.NewMessage(
		`<title>Database Check Failed</title>

<warning>Could not verify database state.</warning>
`,
		nil,
	)

	return TableCheckError{
		error:       fmt.Errorf("failed to check database tables: %w", cause),
		MessageBase: userBase,
	}
}

// EmptyDatabaseError is returned when database has no tables.
type EmptyDatabaseError struct {
	error
	gnlib.MessageBase
}

// NewEmptyDatabaseError creates an error for unpopulated database.
func NewEmptyDatabaseError(host, database string) error {
	userBase := gnlib.NewMessage(
		`<title>Database Not Ready for Optimization</title>

<warning>The database appears to be empty or not populated.</warning>

<em>Required steps before optimization:</em>
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
		[]any{host, database},
	)

	return EmptyDatabaseError{
		error:       fmt.Errorf("database has no tables - run 'gndb create' and 'gndb populate' first"),
		MessageBase: userBase,
	}
}
