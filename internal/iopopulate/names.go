package iopopulate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gnuuid"
)

type nameRecord struct {
	gnScientificName  sql.NullString // May be NULL or empty
	colScientificName string
}

// processNameStrings implements Phase 2: Name Strings import from SFGA.
// It reads names from the SFGA name table, generates UUID v5 identifiers,
// and inserts them into the name_strings table using batch inserts with
// ON CONFLICT DO NOTHING for idempotency.
//
// Uses p.sfgaDB for SQLite queries and p.operator.Pool() for PostgreSQL
// inserts. Prompts user if gn__scientific_name_string is empty, falling
// back to col__scientific_name.
//
// Returns error if:
//   - SFGA query fails
//   - User aborts when gn__scientific_name_string is empty
//   - Database insert fails
func (p *populator) processNameStrings(
	sourceID int,
) (string, error) {
	slog.Info("Step 2/6: Processing name strings", "data_source_id", sourceID)

	names, emptyGNameStr, err := p.getNames()
	if err != nil {
		return "", err
	}

	err = handleEmptyGNameStr(emptyGNameStr, sourceID)
	if err != nil {
		return "", err
	}

	var totalInserted int
	totalInserted, err = p.insertNames(names)
	if err != nil {
		return "", err
	}

	// Final log with total count
	slog.Info("Name strings imported",
		"data_source_id", sourceID,
		"inserted", totalInserted,
		"total_records", len(names),
	)

	msg := "<em>All names strings are in the database already</em>"
	if totalInserted > 0 {
		msg = fmt.Sprintf("<em>Inserted %d name strings</em>", totalInserted)
	}

	return msg, nil
}

func (p *populator) insertNames(
	names []nameRecord,
) (int, error) {
	// Batch insert configuration
	// PostgreSQL has a limit of 65535 parameters per query.
	// With 2 parameters per row (id, name), max is 32767 rows.
	// Use 30000 to stay safely under the limit.
	const batchSize = 30000

	var totalInserted int

	// Create progress bar for processing names
	bar := pb.Full.Start(len(names))
	bar.Set("prefix", "Processing names: ")
	bar.Set(pb.CleanOnFinish, true)

	// Process names in batches
	for i := 0; i < len(names); i += batchSize {
		end := i + batchSize
		end = slices.Min([]int{end, len(names)})

		batch := names[i:end]

		// Build parameterized INSERT with ON CONFLICT DO NOTHING
		// This handles duplicate UUIDs gracefully (same name from multiple sources)
		var valueStrings []string
		var valueArgs []any
		argIdx := 1

		for _, rec := range batch {
			// Determine which name to use
			nameString := rec.colScientificName
			if rec.gnScientificName.Valid &&
				strings.TrimSpace(rec.gnScientificName.String) != "" {
				nameString = strings.TrimSpace(rec.gnScientificName.String)
			}

			// Generate UUID v5 using gnuuid (deterministic)
			uuid := gnuuid.New(nameString).String()

			// Add to batch
			// ($1, $2), ($3, $4), ...
			valueStrings = append(
				valueStrings,
				fmt.Sprintf("($%d, $%d)", argIdx, argIdx+1),
			)
			valueArgs = append(valueArgs, uuid, nameString)
			argIdx += 2
		}

		// Build and execute INSERT statement
		insertQuery := fmt.Sprintf(
			`INSERT INTO name_strings (id, name) VALUES %s
			 ON CONFLICT (id) DO NOTHING`,
			strings.Join(valueStrings, ", "),
		)

		result, err := p.operator.Pool().Exec(
			context.Background(),
			insertQuery,
			valueArgs...,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to insert name strings batch: %w", err)
		}

		rowsAffected := result.RowsAffected()
		totalInserted += int(rowsAffected)

		// Update progress bar
		bar.Add(len(batch))
	}

	// Finish progress bar before printing final stats
	bar.Finish()

	return totalInserted, nil
}

func handleEmptyGNameStr(emptyGNameStr, sourceID int) error {
	// If there are empty gn__scientific_name_string values, prompt user
	if emptyGNameStr > 0 {
		fmt.Println()
		gn.Warn(
			"<em>Warning</em>: gn__scientific_name_string is empty "+
				"for %s records.\n",
			humanize.Comma(int64(emptyGNameStr)),
		)
		fmt.Println(
			"Falling back to col__scientific_name may lose authorship data.",
		)
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  [Y]es    - Continue with fallback (default)")
		fmt.Println("  [N]o     - Skip this data source")
		fmt.Println("  [A]bort  - Cancel entire import")
		fmt.Println()

		response, err := promptUserMulti(
			"Your choice [Y/n/a]: ",
			[]string{"yes", "no", "abort"},
		)
		if err != nil {
			return fmt.Errorf("failed to get user response: %w", err)
		}

		switch response {
		case "yes":
			slog.Info(
				"User chose to continue with fallback to col__scientific_name",
				"data_source_id",
				sourceID,
			)
		case "no":
			slog.Info("User chose to skip this source", "data_source_id", sourceID)
			return nil // Skip this source, continue with next
		case "abort":
			return fmt.Errorf("user aborted populate run")
		}
	}
	return nil
}

func (p *populator) getNames() ([]nameRecord, int, error) {
	// Query SFGA name table
	// gn__scientific_name_string is preferred (always includes authorship)
	// col__scientific_name is fallback (might keep authorship in
	// col__authorship)
	query := `
		SELECT gn__scientific_name_string, col__scientific_name
		FROM name
	`

	rows, err := p.sfgaDB.Query(query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query SFGA name table: %w", err)
	}
	defer rows.Close()

	// First pass: collect all names and check for empty gn__scientific_name_string
	var names []nameRecord
	emptyCount := 0

	for rows.Next() {
		var rec nameRecord
		err := rows.Scan(&rec.gnScientificName, &rec.colScientificName)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan SFGA name row: %w", err)
		}

		// Check if gn__scientific_name_string is empty
		if !rec.gnScientificName.Valid ||
			strings.TrimSpace(rec.gnScientificName.String) == "" {
			emptyCount++
		}

		names = append(names, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating SFGA name rows: %w", err)
	}
	return names, emptyCount, nil
}

// promptUserMulti displays a message and reads user input with multiple
// options. The first option in validOptions is the default (used on empty
// input). Accepts both full words and single-letter shortcuts
// (e.g., "yes"/"y", "no"/"n", "abort"/"a").
func promptUserMulti(message string, validOptions []string) (string, error) {
	if len(validOptions) == 0 {
		return "", fmt.Errorf("no valid options provided")
	}

	fmt.Print(message)

	var response string
	// Scanln returns error on empty input, but we want to allow that as default
	_, err := fmt.Scanln(&response)
	if err != nil && err.Error() != "unexpected newline" {
		// Empty input - use first option as default
		return validOptions[0], nil
	}
	if err != nil {
		// Real error
		return "", err
	}

	response = strings.ToLower(strings.TrimSpace(response))

	// Check if response matches any valid option (full word or first letter)
	for _, opt := range validOptions {
		if response == opt ||
			(len(response) == 1 && len(opt) > 0 && response[0] == opt[0]) {
			return opt, nil
		}
	}

	// Invalid response
	return "", fmt.Errorf(
		"invalid response: %s (expected one of: %v)",
		response,
		validOptions,
	)
}
