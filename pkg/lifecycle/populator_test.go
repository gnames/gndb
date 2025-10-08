
package lifecycle_test

import (
	"testing"

	"github.com/gnames/gndb/internal/io/populate"
	"github.com/gnames/gndb/pkg/lifecycle"
	"github.com/stretchr/testify/assert"
)

// TestPopulatorContract ensures that the populate.PopulatorImpl implementation
// satisfies the lifecycle.Populator interface.
// This is a compile-time check, and the test will not run if the contract
// is broken.
func TestPopulatorContract(t *testing.T) {
	// The following line is a compile-time check.
	// If populate.PopulatorImpl does not implement lifecycle.Populator,
	// this code will fail to compile.
	var _ lifecycle.Populator = &populate.PopulatorImpl{}

	// This assertion is a runtime check to confirm the test was executed.
	assert.True(t, true, "populate.PopulatorImpl should implement lifecycle.Populator")
}
