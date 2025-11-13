package iosources

import (
	"fmt"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
)

// SourcesConfigError creates an error for when sources.yaml
// cannot be loaded.
func SourcesConfigError(path string, err error) error {
	msg := `Cannot load sources configuration

<em>Configuration file:</em> %s

<em>Possible causes:</em>
  - File does not exist
  - Invalid YAML format
  - Permission denied

<em>How to fix:</em>
  1. Check if file exists: <em>ls -l %s</em>
  2. Validate YAML syntax
  3. Generate example: <em>gndb config generate</em>`

	vars := []any{path, path}

	return &gn.Error{
		Code: errcode.PopulateSourcesConfigError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to load sources config: %w", err),
	}
}
