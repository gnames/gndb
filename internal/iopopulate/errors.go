package iopopulate

import (
	"fmt"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/errcode"
)

// NotConnectedError creates an error for when populate
// operation is attempted without database connection.
func NotConnectedError() error {
	msg := "Populate operation attempted without database connection"

	return &gn.Error{
		Code: errcode.DBNotConnectedError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("not connected to database"),
	}
}

// NoSourcesError creates an error for when no matching
// sources are found.
func NoSourcesError(requestedIDs []int) error {
	msg := `No sources found matching requested IDs

<em>Requested IDs:</em> %v

<em>How to fix:</em>
  1. Check available sources: review sources.yaml
  2. Verify source IDs are correct`

	vars := []any{requestedIDs}

	return &gn.Error{
		Code: errcode.PopulateSourcesConfigError,
		Msg:  msg,
		Vars: vars,
		Err: fmt.Errorf(
			"no sources found matching IDs: %v",
			requestedIDs),
	}
}

// SFGAFileNotFoundError creates an error for when SFGA file
// cannot be found.
func SFGAFileNotFoundError(
	sourceID int,
	parent string,
	err error,
) error {
	msg := `SFGA file not found for data source

<em>Data Source ID:</em> %d
<em>Parent location:</em> %s

<em>Possible causes:</em>
  - SFGA file not downloaded
  - Incorrect parent directory/URL
  - File naming doesn't match ID pattern

<em>How to fix:</em>
  1. Check parent directory/URL exists
  2. Verify SFGA file naming: %04d*.{sql,sqlite}{,.zip}
  3. Download SFGA file if missing`

	vars := []any{sourceID, parent, sourceID}

	return &gn.Error{
		Code: errcode.PopulateSFGAFileNotFoundError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("SFGA file not found: %w", err),
	}
}

// SFGAReadError creates an error for when SFGA file
// cannot be read.
func SFGAReadError(path string, err error) error {
	msg := `Cannot read SFGA file

<em>File path:</em> %s

<em>Possible causes:</em>
  - File is corrupted
  - Unsupported format
  - Permission denied
  - SQLite error

<em>How to fix:</em>
  1. Verify file integrity
  2. Check file permissions
  3. Re-download SFGA file if corrupted`

	vars := []any{path}

	return &gn.Error{
		Code: errcode.PopulateSFGAReadError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to read SFGA file: %w", err),
	}
}

// AllSourcesFailedError creates an error for when all
// sources fail to process.
func AllSourcesFailedError(count int) error {
	msg := `Failed number of sources: <em>%d</em>`

	vars := []any{count}

	plural := "s"
	if count == 1 {
		plural = ""
	}

	return &gn.Error{
		Code: errcode.PopulateAllSourcesFailedError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("%d source%s failed to process", count, plural),
	}
}

// SfgaGetVersionError creates an error for SFGA version reading error.
func SfgaGetVersionError(sourceID int, err error) error {
	msg := `Failed to get version of SFGA data source <em>%d</em>`
	vars := []any{sourceID}

	return &gn.Error{
		Code: errcode.PopulateSFGAVersionError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to get SFGA version: %w", err),
	}
}

func NotSfgaVersion(sourceID int, version string) error {
	msg := `Cannot parse SFGA version <em>%s</em> for data source <em>%d</em>`
	vars := []any{version, sourceID}

	return &gn.Error{
		Code: errcode.PopulateSFGAVersionError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("string '%s' is not a semantic version", version),
	}
}

func SFGAVersionTooOld(sourceID int, version string) error {
	msg :=
		`The SFGA <em>%s</em> is not supported (data source <em>#%d</em>).
Supported SFGA versions are equal or greater than <em>%s</em>`
	vars := []any{version, sourceID, config.MinVersionSFGA}

	return &gn.Error{
		Code: errcode.PopulateSFGAVersionTooOldError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("too old SFGA version '%s'", version),
	}
}

// MetadataError creates an error for metadata import failures.
func MetadataError(sourceID int, err error) error {
	msg := `Failed to import metadata for data source <em>%d</em>`
	vars := []any{sourceID}

	return &gn.Error{
		Code: errcode.PopulateMetadataError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to import metadata: %w", err),
	}
}

// NamesError creates an error for name-string import failures.
func NamesError(sourceID int, err error) error {
	msg := `Failed to import name-strings for data source <em>%d</em>`
	vars := []any{sourceID}

	return &gn.Error{
		Code: errcode.PopulateNamesError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("failed to import names: %w", err),
	}
}

// CacheError creates an error for cache-related failures.
func CacheError(operation string, err error) error {
	msg := `Cache operation failed: <em>%s</em>`
	vars := []any{operation}

	return &gn.Error{
		Code: errcode.PopulateCacheError,
		Msg:  msg,
		Vars: vars,
		Err:  fmt.Errorf("cache operation failed: %w", err),
	}
}

// CancelledError creates an error for when populate
// operation is cancelled.
func CancelledError(err error) error {
	msg := "Population operation was cancelled"

	return &gn.Error{
		Code: errcode.UnknownError,
		Msg:  msg,
		Vars: nil,
		Err:  fmt.Errorf("population cancelled: %w", err),
	}
}
