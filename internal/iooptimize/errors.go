package iooptimize

import (
	"fmt"

	"github.com/gnames/gnlib"
)

// CacheNotOpenError is returned when cache operations are attempted on a closed database.
type CacheNotOpenError struct {
	error
	gnlib.MessageBase
}

// NewCacheNotOpenError creates a new cache not open error.
func NewCacheNotOpenError() error {
	userBase := gnlib.NewMessage(
		`<title>Cache Database Not Open</title>
<warning>Cannot perform cache operation - database is not open.</warning>

<em>How to fix:</em>
  1. Ensure CacheManager.Open() is called before StoreParsed() or GetParsed()
  2. Check that CacheManager.Close() was not called prematurely
  3. Verify cache initialization in the optimize workflow

This is likely an internal error. Please report if you see this message.`,
		nil,
	)

	return CacheNotOpenError{
		error:       fmt.Errorf("cache database is not open"),
		MessageBase: userBase,
	}
}

// NotImplementedError is returned when a function is called that hasn't been implemented yet.
// This is used in TDD workflow to make tests fail during the red phase.
type NotImplementedError struct {
	error
	gnlib.MessageBase
}

// errNotImplemented creates a not implemented error for TDD red phase.
func errNotImplemented(functionName string) error {
	userBase := gnlib.NewMessage(
		`<title>Function Not Yet Implemented</title>
<warning>The function <em>%s</em> has not been implemented yet.</warning>

This is expected during test-driven development (TDD).
The test should fail until the implementation is complete.`,
		[]any{functionName},
	)

	return NotImplementedError{
		error:       fmt.Errorf("%s is not yet implemented", functionName),
		MessageBase: userBase,
	}
}
