package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetMigrateCmd_Exists verifies getMigrateCmd returns
// a valid command.
func TestGetMigrateCmd_Exists(t *testing.T) {
	cmd := getMigrateCmd()
	require.NotNil(t, cmd, "Migrate command should exist")
	assert.Equal(t, "migrate", cmd.Use,
		"Command name should be migrate")
}

// TestGetMigrateCmd_ShortDescription verifies short
// description.
func TestGetMigrateCmd_ShortDescription(t *testing.T) {
	cmd := getMigrateCmd()

	assert.NotEmpty(t, cmd.Short,
		"Short description should not be empty")
	assert.Contains(t, cmd.Short, "schema",
		"Short description should mention schema")
}

// TestGetMigrateCmd_LongDescription verifies long
// description.
func TestGetMigrateCmd_LongDescription(t *testing.T) {
	cmd := getMigrateCmd()

	assert.NotEmpty(t, cmd.Long,
		"Long description should not be empty")
	assert.Contains(t, cmd.Long, "GORM AutoMigrate",
		"Long description should mention GORM")
	assert.Contains(t, cmd.Long, "Drops materialized views",
		"Long description should mention view handling")
	assert.Contains(t, cmd.Long, "gndb optimize",
		"Long description should mention optimize step")
}

// TestGetMigrateCmd_HasRunE verifies run function is set.
func TestGetMigrateCmd_HasRunE(t *testing.T) {
	cmd := getMigrateCmd()

	assert.NotNil(t, cmd.RunE,
		"RunE should be set")
}

// TestGetMigrateCmd_HasRecreateViewsFlag verifies the
// --recreate-views flag exists.
func TestGetMigrateCmd_HasRecreateViewsFlag(t *testing.T) {
	cmd := getMigrateCmd()

	// Check for recreate-views flag
	flag := cmd.Flags().Lookup("recreate-views")
	require.NotNil(t, flag,
		"Should have recreate-views flag")
	assert.Equal(t, "v", flag.Shorthand,
		"Short flag should be -v")
	assert.Equal(t, "false", flag.DefValue,
		"Default should be false")
}

// TestGetMigrateCmd_HelpText verifies help text content.
func TestGetMigrateCmd_HelpText(t *testing.T) {
	cmd := getMigrateCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()
	assert.Contains(t, helpText, "migrate",
		"Help should mention migrate")
	assert.Contains(t, helpText, "schema",
		"Help should mention schema")
	assert.Contains(t, helpText, "Examples:",
		"Help should include examples")
}

// TestGetMigrateCmd_Examples verifies examples in help.
func TestGetMigrateCmd_Examples(t *testing.T) {
	cmd := getMigrateCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()
	assert.Contains(t, helpText, "gndb migrate",
		"Should show basic example")
}

// TestGetMigrateCmd_SafetyMentioned verifies safety is
// documented.
func TestGetMigrateCmd_SafetyMentioned(t *testing.T) {
	cmd := getMigrateCmd()

	// Check that help text mentions it's safe
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()
	assert.Contains(t, helpText, "safe",
		"Help should mention safety")
	assert.Contains(t, helpText, "Does NOT delete",
		"Help should mention what it doesn't do")
}

// TestGetMigrateCmd_IndependentInstances verifies each
// call returns independent instance.
func TestGetMigrateCmd_IndependentInstances(t *testing.T) {
	cmd1 := getMigrateCmd()
	cmd2 := getMigrateCmd()

	// Should be different instances
	assert.NotSame(t, cmd1, cmd2,
		"Each call should return new instance")

	// Modifying one shouldn't affect the other
	cmd1.Short = "test1"
	cmd2.Short = "test2"

	assert.Equal(t, "test1", cmd1.Short)
	assert.Equal(t, "test2", cmd2.Short)
}
