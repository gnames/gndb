package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetCreateCmd_Exists verifies getCreateCmd returns
// a valid command.
func TestGetCreateCmd_Exists(t *testing.T) {
	cmd := getCreateCmd()
	require.NotNil(t, cmd, "Create command should exist")
	assert.Equal(t, "create", cmd.Use,
		"Command name should be create")
}

// TestGetCreateCmd_ShortDescription verifies short
// description.
func TestGetCreateCmd_ShortDescription(t *testing.T) {
	cmd := getCreateCmd()

	assert.NotEmpty(t, cmd.Short,
		"Short description should not be empty")
	assert.Contains(t, cmd.Short, "schema",
		"Short description should mention schema")
}

// TestGetCreateCmd_LongDescription verifies long
// description.
func TestGetCreateCmd_LongDescription(t *testing.T) {
	cmd := getCreateCmd()

	assert.NotEmpty(t, cmd.Long,
		"Long description should not be empty")
	assert.Contains(t, cmd.Long, "PostgreSQL",
		"Long description should mention PostgreSQL")
	assert.Contains(t, cmd.Long, "GORM AutoMigrate",
		"Long description should mention GORM")
	assert.Contains(t, cmd.Long, "collation",
		"Long description should mention collation")
}

// TestGetCreateCmd_HasRunE verifies run function is set.
func TestGetCreateCmd_HasRunE(t *testing.T) {
	cmd := getCreateCmd()

	assert.NotNil(t, cmd.RunE,
		"RunE should be set")
}

// TestGetCreateCmd_ForceFlag verifies --force flag exists.
func TestGetCreateCmd_ForceFlag(t *testing.T) {
	cmd := getCreateCmd()

	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag,
		"--force flag should exist")

	assert.Equal(t, "f", forceFlag.Shorthand,
		"Short form should be -f")
	assert.Equal(t, "false", forceFlag.DefValue,
		"Default should be false")
	assert.Contains(t, forceFlag.Usage, "drop",
		"Usage should mention drop")
}

// TestGetCreateCmd_HelpText verifies help text content.
func TestGetCreateCmd_HelpText(t *testing.T) {
	cmd := getCreateCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()
	assert.Contains(t, helpText, "create",
		"Help should mention create")
	assert.Contains(t, helpText, "--force",
		"Help should mention --force flag")
	assert.Contains(t, helpText, "-f",
		"Help should mention -f short form")
	assert.Contains(t, helpText, "Examples:",
		"Help should include examples")
}

// TestGetCreateCmd_Examples verifies examples in help.
func TestGetCreateCmd_Examples(t *testing.T) {
	cmd := getCreateCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()
	assert.Contains(t, helpText, "gndb create",
		"Should show basic example")
	assert.Contains(t, helpText, "gndb create --force",
		"Should show force example")
	assert.Contains(t, helpText, "gndb create -f",
		"Should show short form example")
}

// TestGetCreateCmd_IndependentInstances verifies each
// call returns independent instance.
func TestGetCreateCmd_IndependentInstances(t *testing.T) {
	cmd1 := getCreateCmd()
	cmd2 := getCreateCmd()

	// Should be different instances
	assert.NotSame(t, cmd1, cmd2,
		"Each call should return new instance")

	// Modifying one shouldn't affect the other
	cmd1.Short = "test1"
	cmd2.Short = "test2"

	assert.Equal(t, "test1", cmd1.Short)
	assert.Equal(t, "test2", cmd2.Short)
}
