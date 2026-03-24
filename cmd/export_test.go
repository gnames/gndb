package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetExportCmd_Exists verifies getExportCmd returns
// a valid command.
func TestGetExportCmd_Exists(t *testing.T) {
	cmd := getExportCmd()
	require.NotNil(t, cmd, "Export command should exist")
	assert.Equal(t, "export", cmd.Use,
		"Command name should be export")
}

// TestGetExportCmd_ShortDescription verifies short
// description mentions SFGA.
func TestGetExportCmd_ShortDescription(t *testing.T) {
	cmd := getExportCmd()

	assert.NotEmpty(t, cmd.Short,
		"Short description should not be empty")
	assert.Contains(t, cmd.Short, "SFGA",
		"Short description should mention SFGA")
}

// TestGetExportCmd_LongDescription verifies long description
// covers key concepts.
func TestGetExportCmd_LongDescription(t *testing.T) {
	cmd := getExportCmd()

	assert.NotEmpty(t, cmd.Long,
		"Long description should not be empty")
	assert.Contains(t, cmd.Long, "sqlite",
		"Long description should mention sqlite output")
	assert.Contains(t, cmd.Long, "sources-export.yaml",
		"Long description should mention consolidated YAML")
	assert.Contains(t, cmd.Long, "--parent",
		"Long description should explain --parent flag")
}

// TestGetExportCmd_HasRunE verifies run function is set.
func TestGetExportCmd_HasRunE(t *testing.T) {
	cmd := getExportCmd()

	assert.NotNil(t, cmd.RunE,
		"RunE should be set")
}

// TestGetExportCmd_SourceIDsFlag verifies --source-ids flag.
func TestGetExportCmd_SourceIDsFlag(t *testing.T) {
	cmd := getExportCmd()

	flag := cmd.Flags().Lookup("source-ids")
	require.NotNil(t, flag, "--source-ids flag should exist")
	assert.Equal(t, "s", flag.Shorthand,
		"Short form should be -s")
	assert.Contains(t, flag.Usage, "source IDs",
		"Usage should mention source IDs")
}

// TestGetExportCmd_OutputDirFlag verifies --output-dir flag.
func TestGetExportCmd_OutputDirFlag(t *testing.T) {
	cmd := getExportCmd()

	flag := cmd.Flags().Lookup("output-dir")
	require.NotNil(t, flag, "--output-dir flag should exist")
	assert.Equal(t, "o", flag.Shorthand,
		"Short form should be -o")
	assert.Equal(t, ".", flag.DefValue,
		"Default output dir should be current directory")
}

// TestGetExportCmd_ParentFlag verifies --parent flag.
func TestGetExportCmd_ParentFlag(t *testing.T) {
	cmd := getExportCmd()

	flag := cmd.Flags().Lookup("parent")
	require.NotNil(t, flag, "--parent flag should exist")
	assert.Equal(t, "p", flag.Shorthand,
		"Short form should be -p")
	assert.Equal(t, "", flag.DefValue,
		"Default parent should be empty (falls back to output-dir)")
}

// TestGetExportCmd_ZipFlag verifies --zip flag.
func TestGetExportCmd_ZipFlag(t *testing.T) {
	cmd := getExportCmd()

	flag := cmd.Flags().Lookup("zip")
	require.NotNil(t, flag, "--zip flag should exist")
	assert.Equal(t, "z", flag.Shorthand,
		"Short form should be -z")
	assert.Equal(t, "false", flag.DefValue,
		"Zip should be disabled by default")
}

// TestGetExportCmd_IndependentInstances verifies each call
// returns an independent instance.
func TestGetExportCmd_IndependentInstances(t *testing.T) {
	cmd1 := getExportCmd()
	cmd2 := getExportCmd()

	assert.NotSame(t, cmd1, cmd2,
		"Each call should return a new instance")

	cmd1.Short = "test1"
	cmd2.Short = "test2"

	assert.Equal(t, "test1", cmd1.Short)
	assert.Equal(t, "test2", cmd2.Short)
}

// TestGetExportCmd_RegisteredOnRoot verifies the export
// command is reachable from the root command.
func TestGetExportCmd_RegisteredOnRoot(t *testing.T) {
	root := getRootCmd()

	var found bool
	for _, sub := range root.Commands() {
		if sub.Use == "export" {
			found = true
			break
		}
	}
	assert.True(t, found,
		"export command should be registered on root")
}
