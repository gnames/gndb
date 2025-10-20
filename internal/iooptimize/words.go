package iooptimize

import (
	"context"
	"fmt"

	"github.com/gnames/gndb/pkg/config"
)

// createWords orchestrates the word extraction and insertion process.
// This is Step 4 of the optimization workflow from gnidump.
//
// Workflow:
//  1. Truncate words and word_name_strings tables
//  2. Load all name_strings with canonical_id
//  3. Extract words from cached parse results (no re-parsing)
//  4. Deduplicate words
//  5. Bulk insert words
//  6. Bulk insert word-name-string linkages
//
// Reference: gnidump createWords() in words.go
func createWords(ctx context.Context, o *OptimizerImpl, cfg *config.Config) error {
	return fmt.Errorf("createWords is not yet implemented")
}
