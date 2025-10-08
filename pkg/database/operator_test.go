package database_test

import (
	"testing"

	"github.com/gnames/gndb/internal/io/database"
	pkgdb "github.com/gnames/gndb/pkg/database"
)

// TestPgxOperatorImplementsInterface verifies that PgxOperator
// implements the database.Operator interface.
// This test ensures compile-time contract compliance.
func TestPgxOperatorImplementsInterface(t *testing.T) {
	// This will fail to compile if PgxOperator doesn't implement database.Operator
	var _ pkgdb.Operator = (*database.PgxOperator)(nil)
}
