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
	"context"
	"errors"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioexport"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/spf13/cobra"
)

// getExportCmd returns the export command.
func getExportCmd() *cobra.Command {
	var (
		sourceIDs []int
		outputDir string
		parentDir string
		withZip   bool
	)

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export database sources to SFGA files",
		Long: `Export gnames PostgreSQL data sources to SFGA SQLite format.

For each exported source the following files are written to the output
directory:

  {ID:04d}-{slug}-{date}.sqlite   — SQLite database
  {ID:04d}-{slug}-{date}.sql      — SQL dump
  {ID:04d}-{slug}-{date}.yaml     — companion sources.yaml entry
  {ID:04d}-{slug}-{date}.sqlite.zip  — (with --zip only)
  {ID:04d}-{slug}-{date}.sql.zip     — (with --zip only)

A consolidated sources-export.yaml listing all exported sources is also
written to the output directory.

The --parent flag sets the value of the 'parent' field in companion YAML
files. It should be set to the URL or path from which the exported files
will be served, so that the YAML is ready to use without editing.

Examples:
  # Export all sources to current directory
  gndb export

  # Export specific sources
  gndb export --source-ids 1,3

  # Export to a directory with zip archives
  gndb export --output-dir /tmp/sfga --zip

  # Set parent URL for re-import YAML
  gndb export -o /tmp/sfga -p http://myserver.org/sfga/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runExport(cmd, sourceIDs, outputDir, parentDir, withZip)
			if err != nil {
				gn.PrintErrorMessage(err)
			}
			return err
		},
	}

	exportCmd.Flags().IntSliceVarP(
		&sourceIDs, "source-ids", "s", []int{},
		"data source IDs to export (empty = all)",
	)
	exportCmd.Flags().StringVarP(
		&outputDir, "output-dir", "o", ".",
		"directory where exported files are written",
	)
	exportCmd.Flags().StringVarP(
		&parentDir, "parent", "p", "",
		`value for the 'parent' field (dir or URL) in companion YAML files
(default: output-dir)`,
	)
	exportCmd.Flags().BoolVarP(
		&withZip, "zip", "z", false,
		"also produce .zip compressed variants",
	)

	return exportCmd
}

func runExport(
	cmd *cobra.Command,
	sourceIDs []int,
	outputDir string,
	parentDir string,
	withZip bool,
) error {
	ctx := context.Background()

	var opts []config.Option

	if cmd.Flags().Changed("source-ids") {
		opts = append(opts, config.OptExportSourceIDs(sourceIDs))
	}
	if cmd.Flags().Changed("output-dir") {
		opts = append(opts, config.OptExportOutputDir(outputDir))
	}
	if cmd.Flags().Changed("parent") {
		opts = append(opts, config.OptExportParentDir(parentDir))
	}
	if cmd.Flags().Changed("zip") {
		opts = append(opts, config.OptExportWithZip(withZip))
	}

	if len(opts) > 0 {
		cfg.Update(opts)
	}

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
			Err: errors.New("cannot export from empty database"),
		}
	}

	exporter := ioexport.New(cfg, op)

	gn.Info("Starting export...")
	return exporter.Export()
}
