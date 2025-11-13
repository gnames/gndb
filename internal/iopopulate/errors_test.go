package iopopulate

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

// TestNoSourcesError verifies error structure.
func TestNoSourcesError(t *testing.T) {
	requestedIDs := []int{1, 2, 3}

	err := NoSourcesError(requestedIDs)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateSourcesConfigError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Contains(t, gnErr.Err.Error(), "[1 2 3]")
}

// TestSFGAFileNotFoundError verifies error structure.
func TestSFGAFileNotFoundError(t *testing.T) {
	sourceID := 42
	parent := "/data/sfga"
	originalErr := errors.New("no matching files")

	err := SFGAFileNotFoundError(sourceID, parent, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateSFGAFileNotFoundError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 3)
	assert.Equal(t, sourceID, gnErr.Vars[0])
	assert.Equal(t, parent, gnErr.Vars[1])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestSFGAReadError verifies error structure.
func TestSFGAReadError(t *testing.T) {
	path := "/data/0001.sqlite"
	originalErr := errors.New("corrupted file")

	err := SFGAReadError(path, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateSFGAReadError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, path, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestAllSourcesFailedError verifies error structure.
func TestAllSourcesFailedError(t *testing.T) {
	count := 5

	err := AllSourcesFailedError(count)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateAllSourcesFailedError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, count, gnErr.Vars[0])
	assert.Contains(t, gnErr.Err.Error(), "5 sources")
}

// TestMetadataError verifies error structure.
func TestMetadataError(t *testing.T) {
	sourceID := 10
	originalErr := errors.New("constraint violation")

	err := MetadataError(sourceID, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateMetadataError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, sourceID, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestNamesError verifies error structure.
func TestNamesError(t *testing.T) {
	sourceID := 20
	originalErr := errors.New("parsing failed")

	err := NamesError(sourceID, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateNamesError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, sourceID, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestCacheError verifies error structure.
func TestCacheError(t *testing.T) {
	operation := "initialize cache"
	originalErr := errors.New("disk full")

	err := CacheError(operation, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateCacheError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 1)
	assert.Equal(t, operation, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}

// TestCancelledError verifies error structure.
func TestCancelledError(t *testing.T) {
	originalErr := errors.New("context cancelled")

	err := CancelledError(originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.UnknownError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.ErrorIs(t, gnErr.Err, originalErr)
	assert.Contains(t, gnErr.Err.Error(), "cancelled")
}
