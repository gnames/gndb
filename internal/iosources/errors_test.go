package iosources_test

import (
	"errors"
	"testing"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iosources"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSourcesConfigError verifies error structure.
func TestSourcesConfigError(t *testing.T) {
	path := "/test/sources.yaml"
	originalErr := errors.New("file not found")

	err := iosources.SourcesConfigError(path, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateSourcesConfigError, gnErr.Code)
	assert.NotEmpty(t, gnErr.Msg)
	assert.Len(t, gnErr.Vars, 2)
	assert.Equal(t, path, gnErr.Vars[0])
	assert.ErrorIs(t, gnErr.Err, originalErr)
}
