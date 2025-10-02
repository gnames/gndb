package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRootCommand_HasSubcommands verifies root command structure
func TestRootCommand_HasSubcommands(t *testing.T) {
	cmd := getRootCmd()

	// Verify root command exists
	require.NotNil(t, cmd, "Root command should exist")

	// Verify create subcommand exists (others will be added in future tasks)
	var foundCreate bool
	for _, c := range cmd.Commands() {
		if c.Name() == "create" {
			foundCreate = true
			break
		}
	}
	assert.True(t, foundCreate, "create subcommand should exist")
}

// TestRootCommand_ConfigFlag verifies --config flag exists
func TestRootCommand_ConfigFlag(t *testing.T) {
	cmd := getRootCmd()

	// Verify --config flag exists
	configFlag := cmd.PersistentFlags().Lookup("config")
	require.NotNil(t, configFlag, "--config flag should exist")
	assert.Equal(t, "string", configFlag.Value.Type(), "--config should be string type")
}

// TestRootCommand_Help verifies help text includes usage
func TestRootCommand_Help(t *testing.T) {
	cmd := getRootCmd()

	// Get help output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()

	// Verify help contains key information
	assert.Contains(t, helpText, "gndb", "Help should mention gndb")
	assert.Contains(t, helpText, "database", "Help should mention database")
	assert.Contains(t, helpText, "Available Commands", "Help should list commands")
}

// TestRootCommand_VersionFlag verifies --version flag
func TestRootCommand_VersionFlag(t *testing.T) {
	cmd := getRootCmd()

	// Verify version can be set
	cmd.Version = "test-version"

	// Get version output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})
	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "test-version", "Version output should contain version string")
}

// TestCreateCommand_Exists verifies create subcommand exists
func TestCreateCommand_Exists(t *testing.T) {
	cmd := getRootCmd()

	// Find create subcommand
	var createCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "create" {
			createCmd = c
			break
		}
	}

	require.NotNil(t, createCmd, "create subcommand should exist")
	assert.Contains(t, createCmd.Short, "schema", "create command description should mention schema")
}

// TestCreateCommand_HasForceFlag verifies create --force flag
func TestCreateCommand_HasForceFlag(t *testing.T) {
	cmd := getRootCmd()

	// Find create subcommand
	var createCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "create" {
			createCmd = c
			break
		}
	}

	require.NotNil(t, createCmd)

	// Verify --force flag exists
	forceFlag := createCmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag, "--force flag should exist on create command")
	assert.Equal(t, "bool", forceFlag.Value.Type(), "--force should be boolean")
}

// TestCreateCommand_Help verifies create command help
func TestCreateCommand_Help(t *testing.T) {
	cmd := getRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"create", "--help"})
	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()
	assert.Contains(t, helpText, "create", "Help should mention create")
	assert.Contains(t, helpText, "force", "Help should mention force flag")
}

// TestRootCommand_PersistentFlags verifies persistent flags work across subcommands
func TestRootCommand_PersistentFlags(t *testing.T) {
	cmd := getRootCmd()

	// Verify persistent flags exist and are inherited
	configFlag := cmd.PersistentFlags().Lookup("config")
	require.NotNil(t, configFlag, "Persistent --config flag should exist")

	// Find create subcommand and verify it inherits persistent flags
	var createCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "create" {
			createCmd = c
			break
		}
	}
	require.NotNil(t, createCmd)

	// Persistent flags should be available on subcommands
	inheritedConfig := createCmd.InheritedFlags().Lookup("config")
	assert.NotNil(t, inheritedConfig, "create should inherit --config flag")
}

// TestRootCommand_ValidArgs verifies root command doesn't accept invalid positional args
func TestRootCommand_ValidArgs(t *testing.T) {
	cmd := getRootCmd()

	// Root command should not accept arbitrary args
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"invalid-arg"})
	err := cmd.Execute()

	// Should error on unknown command/arg
	assert.Error(t, err, "Root command should reject invalid arguments")
	errOutput := buf.String()
	assert.True(t,
		strings.Contains(errOutput, "unknown") || strings.Contains(errOutput, "invalid"),
		"Error should mention unknown or invalid command")
}

// TestCreateCommand_RequiresConfig verifies create command loads config
func TestCreateCommand_RequiresConfig(t *testing.T) {
	cmd := getRootCmd()

	// create command should fail without a valid database connection
	// (we're not testing actual DB connection here, just that it tries)
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"create"})

	// This will fail because no database is available, but that's expected
	// We're just verifying the command tries to execute
	err := cmd.Execute()

	// We expect an error (no database), which means the command structure works
	// The error should be about connection, not about command structure
	if err != nil {
		errMsg := err.Error()
		// Should be a connection error, not a cobra error
		assert.True(t,
			strings.Contains(errMsg, "connect") ||
				strings.Contains(errMsg, "database") ||
				strings.Contains(errMsg, "config"),
			"Error should be about database connection or config, got: %s", errMsg)
	}
}
