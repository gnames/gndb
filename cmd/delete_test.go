package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gnames/gndb/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDeleteCmd_Exists verifies getDeleteCmd returns
// a valid command.
func TestGetDeleteCmd_Exists(t *testing.T) {
	cmd := getDeleteCmd()
	require.NotNil(t, cmd, "Delete command should exist")
	assert.Equal(t, "delete", cmd.Use,
		"Command name should be delete")
}

// TestGetDeleteCmd_ShortDescription verifies the short
// description mentions deletion.
func TestGetDeleteCmd_ShortDescription(t *testing.T) {
	cmd := getDeleteCmd()

	assert.NotEmpty(t, cmd.Short,
		"Short description should not be empty")
	assert.Contains(t, strings.ToLower(cmd.Short), "delete",
		"Short description should mention delete")
}

// TestGetDeleteCmd_LongDescription verifies the long
// description covers key concepts.
func TestGetDeleteCmd_LongDescription(t *testing.T) {
	cmd := getDeleteCmd()

	assert.NotEmpty(t, cmd.Long,
		"Long description should not be empty")
	assert.Contains(t, cmd.Long, "confirmation",
		"Long description should mention confirmation")
	assert.Contains(t, cmd.Long, "name_string_indices",
		"Long description should mention tables affected")
	assert.Contains(t, cmd.Long, "gndb optimize",
		"Long description should mention optimize follow-up")
}

// TestGetDeleteCmd_HasRunE verifies run function is set.
func TestGetDeleteCmd_HasRunE(t *testing.T) {
	cmd := getDeleteCmd()

	assert.NotNil(t, cmd.RunE,
		"RunE should be set")
}

// TestGetDeleteCmd_SourceIDsFlag verifies -s / --source-ids
// flag exists with correct shorthand and usage text.
func TestGetDeleteCmd_SourceIDsFlag(t *testing.T) {
	cmd := getDeleteCmd()

	flag := cmd.Flags().Lookup("source-ids")
	require.NotNil(t, flag, "--source-ids flag should exist")
	assert.Equal(t, "s", flag.Shorthand,
		"Short form should be -s")
	assert.Contains(t, flag.Usage, "delete",
		"Usage should mention delete")
}

// TestGetDeleteCmd_SourceIDsDefaultEmpty verifies the
// default value of --source-ids is an empty slice.
func TestGetDeleteCmd_SourceIDsDefaultEmpty(t *testing.T) {
	cmd := getDeleteCmd()

	flag := cmd.Flags().Lookup("source-ids")
	require.NotNil(t, flag)
	assert.Equal(t, "[]", flag.DefValue,
		"Default source-ids should be empty slice")
}

// TestGetDeleteCmd_HelpText verifies help text content.
func TestGetDeleteCmd_HelpText(t *testing.T) {
	cmd := getDeleteCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	helpText := buf.String()
	assert.Contains(t, helpText, "--source-ids",
		"Help should mention --source-ids flag")
	assert.Contains(t, helpText, "-s",
		"Help should mention -s short form")
	assert.Contains(t, helpText, "Examples:",
		"Help should include examples section")
	assert.Contains(t, helpText, "gndb delete -s",
		"Help should show a usage example")
}

// TestGetDeleteCmd_IndependentInstances verifies each call
// returns an independent instance.
func TestGetDeleteCmd_IndependentInstances(t *testing.T) {
	cmd1 := getDeleteCmd()
	cmd2 := getDeleteCmd()

	assert.NotSame(t, cmd1, cmd2,
		"Each call should return a new instance")

	cmd1.Short = "test1"
	cmd2.Short = "test2"

	assert.Equal(t, "test1", cmd1.Short)
	assert.Equal(t, "test2", cmd2.Short)
}

// TestGetDeleteCmd_RegisteredOnRoot verifies the delete
// command is reachable from the root command.
func TestGetDeleteCmd_RegisteredOnRoot(t *testing.T) {
	root := getRootCmd()

	var found bool
	for _, sub := range root.Commands() {
		if sub.Use == "delete" {
			found = true
			break
		}
	}
	assert.True(t, found,
		"delete command should be registered on root")
}

// TestRunDelete_NoIDs verifies that calling runDelete with
// an empty slice is a no-op that returns nil.
func TestRunDelete_NoIDs(t *testing.T) {
	err := runDelete([]int{})
	assert.NoError(t, err,
		"Empty ID list should return nil without error")
}

// TestRunDelete_NilIDs verifies that a nil slice is treated
// the same as an empty slice.
func TestRunDelete_NilIDs(t *testing.T) {
	err := runDelete(nil)
	assert.NoError(t, err,
		"Nil ID list should return nil without error")
}

// TestPrintDeletePlan_AllFound verifies that every dataset
// is listed when all requested IDs are present.
func TestPrintDeletePlan_AllFound(t *testing.T) {
	sources := []schema.DataSource{
		{ID: 1, Title: "Catalogue of Life"},
		{ID: 3, Title: "ITIS"},
	}
	requested := []int{1, 3}

	out := captureStdout(t, func() {
		printDeletePlan(sources, requested)
	})

	assert.Contains(t, out, "[1]",
		"Output should list ID 1")
	assert.Contains(t, out, "Catalogue of Life",
		"Output should list title for ID 1")
	assert.Contains(t, out, "[3]",
		"Output should list ID 3")
	assert.Contains(t, out, "ITIS",
		"Output should list title for ID 3")
	assert.NotContains(t, out, "skipped",
		"Should not mention skipped IDs when all are found")
}

// TestPrintDeletePlan_PartialMatch verifies that IDs not
// found in the database are reported as skipped.
func TestPrintDeletePlan_PartialMatch(t *testing.T) {
	sources := []schema.DataSource{
		{ID: 2, Title: "GBIF Backbone"},
	}
	requested := []int{2, 99}

	out := captureStdout(t, func() {
		printDeletePlan(sources, requested)
	})

	assert.Contains(t, out, "[2]",
		"Output should list found ID 2")
	assert.Contains(t, out, "GBIF Backbone",
		"Output should list title for ID 2")
	assert.Contains(t, out, "99",
		"Output should report missing ID 99")
	assert.Contains(t, out, "skipped",
		"Output should mention skipped IDs")
}

// TestPrintDeletePlan_SingleDataset verifies formatting
// for a single dataset entry.
func TestPrintDeletePlan_SingleDataset(t *testing.T) {
	sources := []schema.DataSource{
		{ID: 5, Title: "MyDataset"},
	}

	out := captureStdout(t, func() {
		printDeletePlan(sources, []int{5})
	})

	assert.Contains(t, out, "[5]",
		"Output should list the dataset ID")
	assert.Contains(t, out, "MyDataset",
		"Output should list the dataset title")
}

// TestPrintDeletePlan_MultipleMissing verifies all missing
// IDs are reported when several are absent.
func TestPrintDeletePlan_MultipleMissing(t *testing.T) {
	sources := []schema.DataSource{
		{ID: 1, Title: "Species 2000"},
	}
	requested := []int{1, 50, 100}

	out := captureStdout(t, func() {
		printDeletePlan(sources, requested)
	})

	assert.Contains(t, out, "50",
		"Output should report missing ID 50")
	assert.Contains(t, out, "100",
		"Output should report missing ID 100")
}

// captureStdout redirects os.Stdout and os.Stderr to pipes
// for the duration of fn and returns the combined output.
// This is necessary because printDeletePlan writes dataset
// lines via fmt.Printf (stdout) and warnings via gn.Warn
// (stderr).
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	rOut, wOut, err := os.Pipe()
	require.NoError(t, err)
	rErr, wErr, err := os.Pipe()
	require.NoError(t, err)

	origStdout := os.Stdout
	origStderr := os.Stderr
	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	var buf bytes.Buffer
	_, err = io.Copy(&buf, rOut)
	require.NoError(t, err)
	_, err = io.Copy(&buf, rErr)
	require.NoError(t, err)

	return buf.String()
}
