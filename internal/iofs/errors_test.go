package iofs

import (
	"errors"
	"testing"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateDirError_Structure verifies error structure.
func TestCreateDirError_Structure(t *testing.T) {
	testDir := "/test/dir"
	originalErr := errors.New("permission denied")

	err := CreateDirError(testDir, originalErr)

	require.NotNil(t, err)

	// Check if it's a gn.Error
	gnErr, ok := err.(*gn.Error)
	require.True(t, ok,
		"Error should be of type *gn.Error")

	// Verify error code
	assert.Equal(t, errcode.CreateDirError, gnErr.Code,
		"Error code should be CreateDirError")

	// Verify user message
	assert.NotEmpty(t, gnErr.Msg,
		"User message should not be empty")
	assert.Contains(t, gnErr.Msg, "%s",
		"Message should contain format placeholder")

	// Verify vars for message formatting
	require.Len(t, gnErr.Vars, 1,
		"Should have one variable for message formatting")
	assert.Equal(t, testDir, gnErr.Vars[0],
		"Variable should be the directory path")

	// Verify wrapped error
	assert.NotNil(t, gnErr.Err,
		"Wrapped error should not be nil")
	assert.ErrorIs(t, gnErr.Err, originalErr,
		"Should wrap original error")
}

// TestCreateDirError_Message verifies error message.
func TestCreateDirError_Message(t *testing.T) {
	testDir := "/test/create"
	originalErr := errors.New("disk full")

	err := CreateDirError(testDir, originalErr)

	gnErr := err.(*gn.Error)

	// Verify the internal error contains useful info
	assert.Contains(t, gnErr.Err.Error(), "cannot create",
		"Error should mention creation failure")
	assert.Contains(t, gnErr.Err.Error(), originalErr.Error(),
		"Error should contain original error message")
}

// TestCopyFileError_Structure verifies error structure.
func TestCopyFileError_Structure(t *testing.T) {
	testFile := "/test/config.yaml"
	originalErr := errors.New("no space left")

	err := CopyFileError(testFile, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok,
		"Error should be of type *gn.Error")

	assert.Equal(t, errcode.CopyFileError, gnErr.Code,
		"Error code should be CopyFileError")

	assert.NotEmpty(t, gnErr.Msg,
		"User message should not be empty")
	assert.Contains(t, gnErr.Msg, "%s",
		"Message should contain format placeholder")

	require.Len(t, gnErr.Vars, 1,
		"Should have one variable")
	assert.Equal(t, testFile, gnErr.Vars[0],
		"Variable should be the file path")

	assert.NotNil(t, gnErr.Err)
	assert.ErrorIs(t, gnErr.Err, originalErr,
		"Should wrap original error")
}

// TestCopyFileError_Message verifies error message.
func TestCopyFileError_Message(t *testing.T) {
	testFile := "/test/file.txt"
	originalErr := errors.New("write failed")

	err := CopyFileError(testFile, originalErr)

	gnErr := err.(*gn.Error)

	assert.Contains(t, gnErr.Err.Error(), "cannot copy",
		"Error should mention copy failure")
	assert.Contains(t, gnErr.Err.Error(), originalErr.Error(),
		"Error should contain original error message")
}

// TestReadFileError_Structure verifies error structure.
func TestReadFileError_Structure(t *testing.T) {
	testPath := "/test/data.json"
	originalErr := errors.New("file not found")

	err := ReadFileError(testPath, originalErr)

	require.NotNil(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok,
		"Error should be of type *gn.Error")

	assert.Equal(t, errcode.ReadFileError, gnErr.Code,
		"Error code should be ReadFileError")

	assert.NotEmpty(t, gnErr.Msg,
		"User message should not be empty")
	assert.Contains(t, gnErr.Msg, "<em>",
		"Message should contain emphasis tags")
	assert.Contains(t, gnErr.Msg, "%s",
		"Message should contain format placeholder")

	require.Len(t, gnErr.Vars, 1,
		"Should have one variable")
	assert.Equal(t, testPath, gnErr.Vars[0],
		"Variable should be the file path")

	assert.NotNil(t, gnErr.Err)
	assert.ErrorIs(t, gnErr.Err, originalErr,
		"Should wrap original error")
}

// TestReadFileError_Message verifies error message.
func TestReadFileError_Message(t *testing.T) {
	testPath := "/important/config"
	originalErr := errors.New("access denied")

	err := ReadFileError(testPath, originalErr)

	gnErr := err.(*gn.Error)

	assert.Contains(t, gnErr.Err.Error(), "cannot read",
		"Error should mention read failure")
	assert.Contains(t, gnErr.Err.Error(), testPath,
		"Error should contain file path")
	assert.Contains(t, gnErr.Err.Error(), originalErr.Error(),
		"Error should contain original error message")
}

// TestErrorFunctions_CallerInfo verifies caller info
// is captured.
func TestErrorFunctions_CallerInfo(t *testing.T) {
	tests := []struct {
		name     string
		errorFn  func() error
		funcName string
	}{
		{
			name: "CreateDirError",
			errorFn: func() error {
				return CreateDirError("/test",
					errors.New("test"))
			},
			funcName: "CreateDirError",
		},
		{
			name: "CopyFileError",
			errorFn: func() error {
				return CopyFileError("/test.txt",
					errors.New("test"))
			},
			funcName: "CopyFileError",
		},
		{
			name: "ReadFileError",
			errorFn: func() error {
				return ReadFileError("/data",
					errors.New("test"))
			},
			funcName: "ReadFileError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFn()
			gnErr := err.(*gn.Error)

			// Verify error contains function context
			// (from runtime.Caller)
			assert.NotNil(t, gnErr.Err,
				"Should capture caller context")
			assert.Contains(t, gnErr.Err.Error(), "from",
				"Error should mention caller context")
		})
	}
}

// TestErrorFunctions_ErrorWrapping verifies proper
// error wrapping.
func TestErrorFunctions_ErrorWrapping(t *testing.T) {
	originalErr := errors.New("root cause")

	tests := []struct {
		name  string
		error error
	}{
		{
			name:  "CreateDirError",
			error: CreateDirError("/dir", originalErr),
		},
		{
			name:  "CopyFileError",
			error: CopyFileError("/file", originalErr),
		},
		{
			name:  "ReadFileError",
			error: ReadFileError("/path", originalErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify error unwrapping works
			gnErr := tt.error.(*gn.Error)
			assert.ErrorIs(t, gnErr.Err, originalErr,
				"Should be able to unwrap to original error")
		})
	}
}
