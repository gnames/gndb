package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetRootCmd_Exists verifies getRootCmd returns
// a valid command.
func TestGetRootCmd_Exists(t *testing.T) {
	cmd := getRootCmd()
	require.NotNil(t, cmd, "Root command should exist")
	assert.Equal(t, "gndb", cmd.Use,
		"Command name should be gndb")
}

// TestGetRootCmd_VersionFormat verifies version
// output format.
func TestGetRootCmd_VersionFormat(t *testing.T) {
	cmd := getRootCmd()

	// Set a test version
	cmd.Version = "version: v1.2.3\nbuild:   abc123"

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "v1.2.3",
		"Version output should contain version")
	assert.Contains(t, output, "abc123",
		"Version output should contain build")
}

// TestGetRootCmd_ShortVersionFlag verifies
// -V flag works.
func TestGetRootCmd_ShortVersionFlag(t *testing.T) {
	cmd := getRootCmd()
	cmd.Version = "version: v1.2.3\nbuild:   abc123"

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"-V"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "v1.2.3",
		"Version output should work with -V flag")
}

// TestGetRootCmd_HelpText verifies help text content.
func TestGetRootCmd_HelpText(t *testing.T) {
	cmd := getRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()
	assert.Contains(t, helpText, "gndb",
		"Help should mention gndb")
	assert.Contains(t, helpText, "GNdb",
		"Help should mention GNdb")
	assert.Contains(t, helpText, "database",
		"Help should mention database")
	assert.Contains(t, helpText, "GNverifier",
		"Help should mention GNverifier")
}

// TestGetRootCmd_ShortDescription verifies
// short description.
func TestGetRootCmd_ShortDescription(t *testing.T) {
	cmd := getRootCmd()

	assert.NotEmpty(t, cmd.Short,
		"Short description should not be empty")
	assert.Contains(t, cmd.Short, "GNdb",
		"Short description should mention GNdb")
	assert.Contains(t, cmd.Short, "lifecycle",
		"Short description should mention lifecycle")
}

// TestGetRootCmd_LongDescription verifies
// long description.
func TestGetRootCmd_LongDescription(t *testing.T) {
	cmd := getRootCmd()

	assert.NotEmpty(t, cmd.Long,
		"Long description should not be empty")
	assert.Contains(t, cmd.Long, "PostgreSQL",
		"Long description should mention PostgreSQL")
	assert.Contains(t, cmd.Long, "Schema Management",
		"Long description should mention features")
	assert.Contains(t, cmd.Long, "Data Population",
		"Long description should mention features")
	assert.Contains(t, cmd.Long, "Optimization",
		"Long description should mention features")
}

// TestGetRootCmd_HasPreRun verifies bootstrap
// function is set.
func TestGetRootCmd_HasPreRun(t *testing.T) {
	cmd := getRootCmd()

	assert.NotNil(t, cmd.PersistentPreRunE,
		"PersistentPreRunE should be set for bootstrap")
}

// TestGetRootCmd_HasRunE verifies root has a run function.
// This is needed to handle the version flag after bootstrap.
func TestGetRootCmd_HasRunE(t *testing.T) {
	cmd := getRootCmd()

	assert.NotNil(t, cmd.RunE,
		"RunE should be set to handle version flag")
}

// TestGetRootCmd_ErrorSilencing verifies error and
// usage silencing.
func TestGetRootCmd_ErrorSilencing(t *testing.T) {
	cmd := getRootCmd()

	assert.True(t, cmd.SilenceErrors,
		"Errors should be silenced")
	assert.True(t, cmd.SilenceUsage,
		"Usage should be silenced on errors")
}

// TestGetRootCmd_VersionTemplate verifies custom version template.
func TestGetRootCmd_VersionTemplate(t *testing.T) {
	cmd := getRootCmd()
	cmd.Version = "test-version"

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	// Should not have "gndb version" prefix due to
	// custom template
	assert.NotContains(t, output, "gndb version:",
		"Should use custom version template")
}

// TestGetRootCmd_IndependentInstances verifies each
// call returns independent instance.
func TestGetRootCmd_IndependentInstances(t *testing.T) {
	cmd1 := getRootCmd()
	cmd2 := getRootCmd()

	// Should be different instances
	assert.NotSame(t, cmd1, cmd2,
		"Each getRootCmd call should return new instance")

	// Modifying one shouldn't affect the other
	cmd1.Version = "version1"
	cmd2.Version = "version2"

	assert.Equal(t, "version1", cmd1.Version)
	assert.Equal(t, "version2", cmd2.Version)
}

// TestGetRootCmd_InvalidCommand verifies error on
// invalid command.
func TestGetRootCmd_InvalidCommand(t *testing.T) {
	cmd := getRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"nonexistent-command"})

	err := cmd.Execute()

	assert.Error(t, err,
		"Should error on invalid command")
	output := buf.String()
	assert.True(t,
		strings.Contains(output, "unknown") ||
			strings.Contains(output, "invalid") ||
			strings.Contains(err.Error(), "unknown"),
		"Error should indicate unknown command")
}
