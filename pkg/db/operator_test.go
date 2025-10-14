package db_test

import (
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/pkg/db"
)

// TestPgxOperatorImplementsInterface verifies that PgxOperator
// implements the db.Operator interface.
// This test ensures compile-time contract compliance.
func TestPgxOperatorImplementsInterface(t *testing.T) {
	// This will fail to compile if PgxOperator doesn't implement db.Operator
	var _ db.Operator = (*iodb.PgxOperator)(nil)
}
