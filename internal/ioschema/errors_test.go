package ioschema

import (
	"errors"
	"testing"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotConnectedError_Structure verifies error structure.
func TestNotConnectedError_Structure(t *testing.T) {
	err := NotConnectedError()

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBNotConnectedError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
}

// TestGORMConnectionError_Structure verifies
// error structure.
func TestGORMConnectionError_Structure(t *testing.T) {
	originalErr := errors.New("connection failed")

	err := GORMConnectionError(originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.SchemaGORMConnectionError,
		gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestCreateSchemaError_Structure verifies error structure.
func TestCreateSchemaError_Structure(t *testing.T) {
	originalErr := errors.New("create failed")

	err := CreateSchemaError(originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.SchemaCreateError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestMigrateSchemaError_Structure verifies
// error structure.
func TestMigrateSchemaError_Structure(t *testing.T) {
	originalErr := errors.New("migrate failed")

	err := MigrateSchemaError(originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.SchemaMigrateError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestCollationError_Structure verifies error structure.
func TestCollationError_Structure(t *testing.T) {
	table := "test_table"
	column := "test_column"
	originalErr := errors.New("collation failed")

	err := CollationError(table, column, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.SchemaCollationError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 2)
	assert.Equal(t, table, gnErr.Vars[0])
	assert.Equal(t, column, gnErr.Vars[1])
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
			name:  "GORMConnectionError",
			error: GORMConnectionError(originalErr),
		},
		{
			name:  "CreateSchemaError",
			error: CreateSchemaError(originalErr),
		},
		{
			name:  "MigrateSchemaError",
			error: MigrateSchemaError(originalErr),
		},
		{
			name: "CollationError",
			error: CollationError("table", "column",
				originalErr),
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
