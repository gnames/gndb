
package lifecycle_test

import (
	"testing"

	"github.com/gnames/gndb/internal/io/optimize"
	"github.com/gnames/gndb/pkg/lifecycle"
	"github.com/stretchr/testify/assert"
)

// TestOptimizerContract ensures that the optimize.OptimizerImpl implementation
// satisfies the lifecycle.Optimizer interface.
// This is a compile-time check, and the test will not run if the contract
// is broken.
func TestOptimizerContract(t *testing.T) {
	// The following line is a compile-time check.
	// If optimize.OptimizerImpl does not implement lifecycle.Optimizer,
	// this code will fail to compile.
	var _ lifecycle.Optimizer = &optimize.OptimizerImpl{}

	// This assertion is a runtime check to confirm the test was executed.
	assert.True(t, true, "optimize.OptimizerImpl should implement lifecycle.Optimizer")
}
