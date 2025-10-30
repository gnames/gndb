package iodb

import (
	"errors"
	"testing"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectionError_Structure verifies error structure.
func TestConnectionError_Structure(t *testing.T) {
	host := "localhost"
	port := 5432
	database := "test"
	user := "postgres"
	originalErr := errors.New("connection refused")

	err := ConnectionError(host, port, database, user,
		originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBConnectionError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 5,
		"Should have 5 vars: host, port, database, user, database")
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestTableCheckError_Structure verifies error structure.
func TestTableCheckError_Structure(t *testing.T) {
	originalErr := errors.New("query failed")

	err := TableCheckError(originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBTableCheckError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestEmptyDatabaseError_Structure verifies error structure.
func TestEmptyDatabaseError_Structure(t *testing.T) {
	host := "localhost"
	database := "test_db"

	err := EmptyDatabaseError(host, database)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBEmptyDatabaseError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 2)
	assert.Equal(t, host, gnErr.Vars[0])
	assert.Equal(t, database, gnErr.Vars[1])
}

// TestNotConnectedError_Structure verifies error structure.
func TestNotConnectedError_Structure(t *testing.T) {
	err := NotConnectedError()

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBNotConnectedError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
}

// TestTableExistsCheckError_Structure verifies
// error structure.
func TestTableExistsCheckError_Structure(t *testing.T) {
	tableName := "test_table"
	originalErr := errors.New("check failed")

	err := TableExistsCheckError(tableName, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBTableExistsCheckError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, tableName, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestQueryTablesError_Structure verifies error structure.
func TestQueryTablesError_Structure(t *testing.T) {
	originalErr := errors.New("query failed")

	err := QueryTablesError(originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBQueryTablesError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestScanTableError_Structure verifies error structure.
func TestScanTableError_Structure(t *testing.T) {
	originalErr := errors.New("scan failed")

	err := ScanTableError(originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBScanTableError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestDropTableError_Structure verifies error structure.
func TestDropTableError_Structure(t *testing.T) {
	tableName := "test_table"
	originalErr := errors.New("drop failed")

	err := DropTableError(tableName, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBDropTableError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, tableName, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestAllErrors_ErrorWrapping verifies proper error
// wrapping.
func TestAllErrors_ErrorWrapping(t *testing.T) {
	originalErr := errors.New("root cause")

	tests := []struct {
		name  string
		error error
	}{
		{
			name: "ConnectionError",
			error: ConnectionError("host", 5432, "db", "user",
				originalErr),
		},
		{
			name:  "TableCheckError",
			error: TableCheckError(originalErr),
		},
		{
			name:  "TableExistsCheckError",
			error: TableExistsCheckError("table", originalErr),
		},
		{
			name:  "QueryTablesError",
			error: QueryTablesError(originalErr),
		},
		{
			name:  "ScanTableError",
			error: ScanTableError(originalErr),
		},
		{
			name:  "DropTableError",
			error: DropTableError("table", originalErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gnErr := tt.error.(*gn.Error)
			assert.ErrorIs(t, gnErr.Err, originalErr,
				"Should wrap original error")
		})
	}
}
