package ioexport

import (
	"fmt"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
)

// NotConnectedError creates an error for when export
// operation is attempted without database connection.
func NotConnectedError() error {
	msg := "Export operation attempted without database connection"

	return &gn.Error{
		Code: errcode.DBNotConnectedError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("not connected to database"),
	}
}

// NoSourcesError creates an error for when no sources
// are found in the data_sources table.
func NoSourcesError(requestedIDs []int) error {
	msg := `No sources found matching requested IDs

<em>Requested IDs:</em> %v

<em>How to fix:</em>
  1. Check available sources in the data_sources table
  2. Verify source IDs are correct`

	vars := []any{requestedIDs}

	return &gn.Error{
		Code: errcode.ExportNoSourcesError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("no sources found matching IDs: %v", requestedIDs),
	}
}

// OutputDirError creates an error for when the output
// directory cannot be created or accessed.
func OutputDirError(dir string, err error) error {
	msg := `Cannot create or access output directory

<em>Directory:</em> %s

<em>How to fix:</em>
  1. Check that the parent directory exists
  2. Verify write permissions`

	vars := []any{dir}

	return &gn.Error{
		Code: errcode.ExportOutputDirError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("output directory error: %w", err),
	}
}

// SFGACreateError creates an error for when the SFGA
// archive cannot be created.
func SFGACreateError(sourceID int, err error) error {
	msg := `Failed to create SFGA archive for data source <em>%d</em>

<em>Note:</em> This step requires internet access to fetch the SFGA schema.
Check your connection and try again.`

	vars := []any{sourceID}

	return &gn.Error{
		Code: errcode.ExportSFGACreateError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to create SFGA archive: %w", err),
	}
}

// SFGAWriteError creates an error for when data cannot
// be written to the SFGA archive.
func SFGAWriteError(sourceID int, stage string, err error) error {
	msg := `Failed to write <em>%s</em> to SFGA archive for data source <em>%d</em>`

	vars := []any{stage, sourceID}

	return &gn.Error{
		Code: errcode.ExportSFGAWriteError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to write %s: %w", stage, err),
	}
}

// WorkDirError creates an error for when the temporary work directory
// for a source cannot be created or cleared.
func WorkDirError(sourceID int, err error) error {
	msg := `Cannot prepare work directory for data source <em>%d</em>`

	vars := []any{sourceID}

	return &gn.Error{
		Code: errcode.CreateDirError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("work directory error for source %d: %w", sourceID, err),
	}
}

// CompanionYAMLError creates an error for when the companion .yaml file
// cannot be written alongside the exported SFGA archive.
func CompanionYAMLError(path string, err error) error {
	msg := `Cannot write companion YAML file <em>%s</em>`

	vars := []any{path}

	return &gn.Error{
		Code: errcode.CopyFileError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("cannot write companion YAML %s: %w", path, err),
	}
}

// AllSourcesFailedError creates an error for when all
// sources fail to export.
func AllSourcesFailedError(count int) error {
	msg := `Failed number of sources: <em>%d</em>`

	vars := []any{count}

	plural := "s"
	if count == 1 {
		plural = ""
	}

	return &gn.Error{
		Code: errcode.ExportAllSourcesFailedError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("%d source%s failed to export", count, plural),
	}
}
