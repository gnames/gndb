package lifecycle_test

import (
	"testing"

	"github.com/gnames/gndb/internal/ioschema"
	"github.com/gnames/gndb/pkg/lifecycle"
	"github.com/stretchr/testify/assert"
)

// TestSchemaManagerContract ensures that the schema.Manager implementation
// satisfies the lifecycle.SchemaManager interface.
// This is a compile-time check, and the test will not run if the contract
// is broken.
func TestSchemaManagerContract(t *testing.T) {
	// The following line is a compile-time check.
	// If schema.Manager does not implement lifecycle.SchemaManager,
	// this code will fail to compile.
	var _ lifecycle.SchemaManager = &ioschema.Manager{}

	// This assertion is a runtime check to confirm the test was executed.
	assert.True(t, true, "schema.Manager should implement lifecycle.SchemaManager")
}
