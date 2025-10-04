// Package templates provides embedded YAML configuration templates.
package templates

import _ "embed"

// SourcesYAML contains the default sources.yaml template for SFGA data sources.
//
//go:embed sources.yaml
var SourcesYAML string

// ConfigYAML contains the default gndb.yaml template for application configuration.
//
//go:embed gndb.yaml
var ConfigYAML string
