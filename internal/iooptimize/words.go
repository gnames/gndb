package iooptimize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gnames/gndb/pkg/config"
	"github.com/jackc/pgx/v5"
)

// nameForWords holds name_string data needed for word extraction.
type nameForWords struct {
	ID          string
	Name        string // The actual name string to parse
	CanonicalID string
}

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

// truncateWordsTables clears the words and word_name_strings tables.
// This ensures a clean slate before populating word data.
//
// Reference: gnidump truncateTable() in db.go
//
//nolint:unused // Will be used in createWords orchestrator (T030)
func truncateWordsTables(ctx context.Context, conn *pgx.Conn) error {
	tables := []string{"words", "word_name_strings"}

	for _, table := range tables {
		sql := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		_, err := conn.Exec(ctx, sql)
		if err != nil {
			slog.Error("Cannot truncate table", "table", table, "error", err)
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
		slog.Info("Truncated table", "table", table)
	}

	return nil
}

// getNameStringsForWords queries all name_strings with canonical_id for word extraction.
// Only names with canonical forms are used for word extraction.
//
// Reference: gnidump getWordNames() in db.go
//
//nolint:unused // Will be used in createWords orchestrator (T030)
func getNameStringsForWords(ctx context.Context, conn *pgx.Conn) ([]nameForWords, error) {
	query := `
		SELECT id, name, canonical_id
		FROM name_strings
		WHERE canonical_id IS NOT NULL
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		slog.Error("Cannot query name_strings for word extraction", "error", err)
		return nil, fmt.Errorf("failed to query name_strings: %w", err)
	}
	defer rows.Close()

	var names []nameForWords
	for rows.Next() {
		var n nameForWords
		if err := rows.Scan(&n.ID, &n.Name, &n.CanonicalID); err != nil {
			slog.Error("Cannot scan name_string row", "error", err)
			return nil, fmt.Errorf("failed to scan name_string: %w", err)
		}
		names = append(names, n)
	}

	if err := rows.Err(); err != nil {
		slog.Error("Error iterating name_string rows", "error", err)
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	slog.Info("Loaded name_strings for word extraction", "count", len(names))
	return names, nil
}
