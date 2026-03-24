package ioexport

import (
	"errors"
	"testing"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotConnectedError verifies error structure.
func TestNotConnectedError(t *testing.T) {
	err := NotConnectedError()

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.DBNotConnectedError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Contains(t, gnErr.Err.Error(), "not connected")
}

// TestNoSourcesError verifies error structure with requested IDs in vars.
func TestNoSourcesError(t *testing.T) {
	requestedIDs := []int{1, 2, 3}

	err := NoSourcesError(requestedIDs)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.ExportNoSourcesError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Contains(t, gnErr.Err.Error(), "[1 2 3]")
}

// TestOutputDirError verifies error structure and wrapping.
func TestOutputDirError(t *testing.T) {
	dir := "/no/such/path"
	originalErr := errors.New("permission denied")

	err := OutputDirError(dir, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.ExportOutputDirError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, dir, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestSFGACreateError verifies error structure and wrapping.
func TestSFGACreateError(t *testing.T) {
	sourceID := 42
	originalErr := errors.New("network timeout")

	err := SFGACreateError(sourceID, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.ExportSFGACreateError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, sourceID, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestSFGAWriteError verifies error structure with stage and source ID.
func TestSFGAWriteError(t *testing.T) {
	sourceID := 11
	stage := "name-strings"
	originalErr := errors.New("disk full")

	err := SFGAWriteError(sourceID, stage, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.ExportSFGAWriteError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 2)
	assert.Equal(t, stage, gnErr.Vars[0])
	assert.Equal(t, sourceID, gnErr.Vars[1])
	assert.ErrorIs(t, gnErr.Err, originalErr)
	assert.Contains(t, gnErr.Err.Error(), stage)
}

// TestWorkDirError verifies error structure and code reuse.
func TestWorkDirError(t *testing.T) {
	sourceID := 7
	originalErr := errors.New("permission denied")

	err := WorkDirError(sourceID, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.CreateDirError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, sourceID, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
	assert.Contains(t, gnErr.Err.Error(), "7")
}

// TestCompanionYAMLError verifies error structure and code reuse.
func TestCompanionYAMLError(t *testing.T) {
	path := "/output/0001-col-2025-08-25.yaml"
	originalErr := errors.New("disk full")

	err := CompanionYAMLError(path, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.CopyFileError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, path, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestAllSourcesFailedError verifies plural/singular and wrapping.
func TestAllSourcesFailedError(t *testing.T) {
	tests := []struct {
		name        string
		count       int
		wantPlural  bool
	}{
		{"single failure", 1, false},
		{"multiple failures", 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AllSourcesFailedError(tt.count)

			require.NotNil(t, err)

			gnErr, ok := err.(*gn.Error)
			require.True(t, ok, "Error should be of type *gn.Error")

			assert.Equal(t, errcode.ExportAllSourcesFailedError, gnErr.Code)
			assert.NotEmpty(t, gnErr.Msg)
			assert.Len(t, gnErr.Vars, 1)
			assert.Equal(t, tt.count, gnErr.Vars[0])

			if tt.wantPlural {
				assert.Contains(t, gnErr.Err.Error(), "sources")
			} else {
				assert.Contains(t, gnErr.Err.Error(), "source")
				assert.NotContains(t, gnErr.Err.Error(), "sources")
			}
		})
	}
}

// TestAllErrors_ErrorWrapping verifies proper wrapping for all errors
// that accept an underlying error.
func TestAllErrors_ErrorWrapping(t *testing.T) {
	originalErr := errors.New("root cause")

	tests := []struct {
		name  string
		error error
	}{
		{
			name:  "OutputDirError",
			error: OutputDirError("/tmp/out", originalErr),
		},
		{
			name:  "SFGACreateError",
			error: SFGACreateError(1, originalErr),
		},
		{
			name:  "SFGAWriteError",
			error: SFGAWriteError(1, "metadata", originalErr),
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
