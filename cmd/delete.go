/*
Copyright © 2025 Dmitry Mozzherin <dmozzherin@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/spf13/cobra"
)

// getDeleteCmd returns the delete subcommand.
func getDeleteCmd() *cobra.Command {
	var sourceIDs []int

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete datasets from the database",
		Long: `Delete one or more datasets from the GNverifier database.

The command lists each dataset that will be deleted (ID and title) and asks
for confirmation before proceeding. No data is changed unless you confirm.

Records are removed from name_string_indices, vernacular_string_indices,
and data_sources. Orphaned name strings and canonical forms are cleaned up
by running 'gndb optimize' afterwards.

Examples:
  # Delete datasets 5 and 12
  gndb delete -s 5,12

  # Delete a single dataset
  gndb delete -s 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runDelete(sourceIDs)
			if err != nil {
				gn.PrintErrorMessage(err)
			}
			return err
		},
	}

	deleteCmd.Flags().IntSliceVarP(
		&sourceIDs, "source-ids", "s", []int{},
		"data source IDs to delete (required)",
	)

	return deleteCmd
}

func runDelete(sourceIDs []int) error {
	if len(sourceIDs) == 0 {
		gn.Info("No source IDs provided. Nothing to delete.")
		gn.Info("Use <em>-s</em> flag to specify dataset IDs, e.g.: " +
			"<em>gndb delete -s 1,2,3</em>")
		return nil
	}

	ctx := context.Background()

	op := iodb.NewPgxOperator()
	if err := op.Connect(ctx, &cfg.Database); err != nil {
		return err
	}
	defer op.Close()

	gn.Info("Connected to database: <em>%s@%s:%d/%s</em>",
		cfg.Database.User, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.Database)

	hasTables, err := op.HasTables(ctx)
	if err != nil {
		return err
	}
	if !hasTables {
		return &gn.Error{
			Code: errcode.DBEmptyDatabaseError,
			Msg: `<err>Database appears to be empty.</err>
   Run <em>'gndb create'</em> first to initialize the schema.`,
			Err: errors.New("cannot delete from empty database"),
		}
	}

	sources, err := op.GetDataSources(ctx, sourceIDs)
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		gn.Warn("<warn>None of the specified IDs were found in the database.</warn>")
		return nil
	}

	printDeletePlan(sources, sourceIDs)

	if !confirmDelete() {
		gn.Info("Aborted. No changes made.")
		return nil
	}

	// Build the IDs that actually exist.
	existingIDs := make([]int, 0, len(sources))
	for _, s := range sources {
		existingIDs = append(existingIDs, s.ID)
	}

	if err := op.DeleteDatasets(ctx, existingIDs); err != nil {
		return err
	}

	gn.Info("Deleted %d dataset(s) successfully.", len(sources))
	gn.Info("Run <em>'gndb optimize'</em> to clean up orphaned name strings.")
	return nil
}

// printDeletePlan prints the datasets that will be deleted and warns about
// any requested IDs that were not found.
func printDeletePlan(found []schema.DataSource, requested []int) {
	gn.Warn("<warn>The following datasets will be permanently deleted:</warn>")
	fmt.Println()
	for _, ds := range found {
		fmt.Printf("  [%d] %s\n", ds.ID, ds.Title)
	}
	fmt.Println()

	// Warn about IDs not found.
	foundSet := make(map[int]bool, len(found))
	for _, ds := range found {
		foundSet[ds.ID] = true
	}
	var missing []string
	for _, id := range requested {
		if !foundSet[id] {
			missing = append(missing, fmt.Sprintf("%d", id))
		}
	}
	if len(missing) > 0 {
		gn.Warn("<warn>The following IDs were not found and will be skipped: %s</warn>",
			strings.Join(missing, ", "))
		fmt.Println()
	}
}

// confirmDelete prompts the user for confirmation and returns true if
// confirmed.
func confirmDelete() bool {
	fmt.Print("Do you want to continue? (yes/No): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		gn.Warn("Failed to read user input")
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "yes" || response == "y"
}
