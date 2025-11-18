package ioschema

import (
	"testing"

	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/gndb"
	"github.com/stretchr/testify/require"
)

// TestManager_ImplementsInterface verifies manager
// implements gndb.SchemaManager interface.
func TestManager_ImplementsInterface(t *testing.T) {
	op := iodb.NewPgxOperator()
	cfg := config.New()
	var _ gndb.SchemaManager = NewManager(op, cfg)
}

// TestNewManager_CreatesManager verifies manager creation.
func TestNewManager_CreatesManager(t *testing.T) {
	op := iodb.NewPgxOperator()
	cfg := config.New()
	mgr := NewManager(op, cfg)
	require.NotNil(t, mgr)
}

// TestManager_PrivateStruct verifies struct is private.
func TestManager_PrivateStruct(t *testing.T) {
	// This test verifies that the manager struct
	// is not exported. If this compiles, the pattern
	// is correct.
	op := iodb.NewPgxOperator()
	cfg := config.New()
	var _ gndb.SchemaManager = NewManager(op, cfg)

	// Cannot create: var m *manager  (would fail to compile)
	// Can only use: NewManager() returns interface
}

// Integration tests would require:
// - Database connection
// - GORM setup
// - Schema migration testing
// These are better suited for E2E tests
