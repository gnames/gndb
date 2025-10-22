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
	"github.com/gnames/gnuuid"
)

// processNameStrings implements Phase 1: Name Strings import from SFGA.
// It reads names from the SFGA name table, generates UUID v5 identifiers,
// and inserts them into the name_strings table using batch inserts with
// ON CONFLICT DO NOTHING for idempotency.
//
// Parameters:
//   - ctx: Context for cancellation
//   - p: Populator instance with database connection
//   - sfgaDB: Open SQLite database handle for SFGA source
//   - sourceID: Data source ID for logging
//
// Returns error if:
//   - SFGA query fails
//   - User aborts when gn__scientific_name_string is empty
//   - Database insert fails
func processNameStrings(ctx context.Context, p *PopulatorImpl, sfgaDB *sql.DB, sourceID int) error {
	slog.Debug("Phase 1: Processing name strings", "source_id", sourceID)

	// Query SFGA name table
	// gn__scientific_name_string is preferred (includes authorship)
	// col__scientific_name is fallback
	query := `
		SELECT col__id, gn__scientific_name_string, col__scientific_name
		FROM name
	`

	rows, err := sfgaDB.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query SFGA name table: %w", err)
	}
	defer rows.Close()

	// First pass: collect all names and check for empty gn__scientific_name_string
	type nameRecord struct {
		colID             string
		gnScientificName  sql.NullString // May be NULL or empty
		colScientificName string
	}

	var names []nameRecord
	emptyCount := 0

	for rows.Next() {
		var rec nameRecord
		err := rows.Scan(&rec.colID, &rec.gnScientificName, &rec.colScientificName)
		if err != nil {
			return fmt.Errorf("failed to scan SFGA name row: %w", err)
		}

		// Check if gn__scientific_name_string is empty
		if !rec.gnScientificName.Valid || strings.TrimSpace(rec.gnScientificName.String) == "" {
			emptyCount++
		}

		names = append(names, rec)
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating SFGA name rows: %w", err)
	}

	// If there are empty gn__scientific_name_string values, prompt user
	if emptyCount > 0 {
		slog.Warn("Empty gn__scientific_name_string detected",
			"count", emptyCount,
			"total", len(names),
		)

		response, err := promptUser(fmt.Sprintf(
			"gn__scientific_name_string is empty for %s records.\n"+
				"Falling back to col__scientific_name may lose authorship data.\n"+
				"Continue? (yes/no/abort): ",
			humanize.Comma(int64(emptyCount)),
		))
		if err != nil {
			return fmt.Errorf("failed to get user response: %w", err)
		}

		switch response {
		case "yes":
			slog.Info("User chose to continue with fallback to col__scientific_name")
		case "no":
			slog.Info("User chose to skip this source")
			return nil // Skip this source, continue with next
		case "abort":
			return fmt.Errorf("user aborted populate run")
		default:
			return fmt.Errorf("invalid response: %s (expected yes/no/abort)", response)
		}
	}

	// Batch insert configuration
	// PostgreSQL has a limit of 65535 parameters per query.
	// With 2 parameters per row (id, name), max is 32767 rows.
	// Use 30000 to stay safely under the limit.
	const batchSize = 30000
	totalInserted := 0

	// Create progress bar for processing names
	bar := pb.Full.Start(len(names))
	bar.Set("prefix", "Processing names: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	// Process names in batches
	for i := 0; i < len(names); i += batchSize {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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
			if rec.gnScientificName.Valid && strings.TrimSpace(rec.gnScientificName.String) != "" {
				nameString = strings.TrimSpace(rec.gnScientificName.String)
			}

			// Generate UUID v5 using gnuuid (deterministic)
			uuid := gnuuid.New(nameString).String()

			// Add to batch
			// ($1, $2), ($3, $4), ...
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", argIdx, argIdx+1))
			valueArgs = append(valueArgs, uuid, nameString)
			argIdx += 2
		}

		// Build and execute INSERT statement
		insertQuery := fmt.Sprintf(
			`INSERT INTO name_strings (id, name) VALUES %s ON CONFLICT (id) DO NOTHING`,
			strings.Join(valueStrings, ", "),
		)

		result, err := p.operator.Pool().Exec(ctx, insertQuery, valueArgs...)
		if err != nil {
			return fmt.Errorf("failed to insert name strings batch: %w", err)
		}

		rowsAffected := result.RowsAffected()
		totalInserted += int(rowsAffected)

		// Update progress bar
		bar.Add(len(batch))
	}

	// Final log with total count
	slog.Info("Phase 1 complete: Name strings imported",
		"inserted", totalInserted,
		"total_records", len(names),
	)

	return nil
}

// promptUser displays a message and reads user input from stdin.
// Returns the trimmed, lowercase response.
func promptUser(message string) (string, error) {
	fmt.Print(message)

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return "", err
	}

	return strings.ToLower(strings.TrimSpace(response)), nil
}
