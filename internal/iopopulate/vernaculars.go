package iopopulate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gnlib"
	"github.com/gnames/gnuuid"
	"github.com/jackc/pgx/v5"
)

// vernIndex is a helper struct for vernacular string indices.
type vernIndex struct {
	recordID           string
	vernacularStringID string
	language           string
	locality           string
	countryCode        string
}

// processVernaculars implements vernacular name import from SFGA.
// This includes both vernacular strings and vernacular indices.
//
// Phase 1: Vernacular Strings
//   - Read unique vernacular names from SFGA vernacular table
//   - Generate UUID v5 for each unique name
//   - Insert into vernacular_strings with ON CONFLICT DO NOTHING
//
// Phase 2: Vernacular Indices
//   - Clean old vernacular indices for this data source
//   - Read vernacular records with metadata (language, locality, country)
//   - Link to vernacular strings via UUID v5
//   - Bulk insert using pgx.CopyFrom
//
// Parameters:
//   - ctx: Context for cancellation
//   - p: Populator instance with database connection
//   - sfgaDB: Open SQLite database handle for SFGA source
//   - sourceID: Data source ID
//
// Returns error if:
//   - SFGA query fails
//   - Database insert fails
func processVernaculars(
	ctx context.Context,
	p *populator,
	sfgaDB *sql.DB,
	sourceID int,
) error {
	slog.Info("Processing vernacular names", "data_source_id", sourceID)

	// Phase 1: Process vernacular strings (unique names)
	vernStrNum, err := processVernacularStrings(ctx, p, sfgaDB)
	if err != nil {
		return fmt.Errorf("failed to process vernacular strings: %w", err)
	}

	// Phase 2: Process vernacular indices (links to data source with metadata)
	vernIdxNum, err := processVernacularIndices(ctx, p, sfgaDB, sourceID)
	if err != nil {
		return fmt.Errorf("failed to process vernacular indices: %w", err)
	}

	slog.Info("Vernacular processing complete",
		"data_source_id", sourceID,
		"strings", vernStrNum,
		"indices", vernIdxNum)

	if vernStrNum == 0 && vernIdxNum == 0 {
		gn.Message("<em>No vernacular names found</em>")
	} else {
		gn.Message(
			"<em>Imported %s vernacular strings and %s vernacular indices</em>",
			humanize.Comma(int64(vernStrNum)),
			humanize.Comma(int64(vernIdxNum)),
		)
	}
	return nil
}

// processVernacularStrings reads unique vernacular names from SFGA and
// inserts them into vernacular_strings table with UUID v5 identifiers.
// Uses ON CONFLICT DO NOTHING for deduplication across data sources.
func processVernacularStrings(ctx context.Context, p *populator, sfgaDB *sql.DB) (int, error) {
	slog.Info("Phase 1: Processing vernacular strings")

	// Query unique vernacular names from SFGA
	query := `SELECT DISTINCT col__name FROM vernacular`

	rows, err := sfgaDB.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query SFGA vernacular table: %w", err)
	}
	defer rows.Close()

	// Collect vernacular strings with UUIDs
	type vernString struct {
		id   string
		name string
	}

	var vernStrings []vernString
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return 0, fmt.Errorf("failed to scan vernacular name: %w", err)
		}

		// Truncate if too long (vernacular_strings.name is varchar(500))
		if len(name) > 500 {
			name = name[:500]
		}

		// Fix UTF-8 encoding issues
		nameFixed := gnlib.FixUtf8(name)

		// Generate UUID v5 using gnuuid (deterministic)
		uuid := gnuuid.New(nameFixed).String()

		vernStrings = append(vernStrings, vernString{
			id:   uuid,
			name: nameFixed,
		})

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
	}

	if err = rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating vernacular rows: %w", err)
	}

	// If no vernaculars, nothing to do
	if len(vernStrings) == 0 {
		slog.Info("No vernacular names found in SFGA")
		return 0, nil
	}

	// Batch insert configuration
	// PostgreSQL has a limit of 65535 parameters per query.
	// With 2 parameters per row (id, name), max is 32767 rows.
	const batchSize = 30000

	// Process in batches
	for i := 0; i < len(vernStrings); i += batchSize {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		end := i + batchSize
		if end > len(vernStrings) {
			end = len(vernStrings)
		}

		batch := vernStrings[i:end]

		// Build parameterized INSERT with ON CONFLICT DO NOTHING
		var valueStrings []string
		var valueArgs []any
		argIdx := 1

		for _, vs := range batch {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", argIdx, argIdx+1))
			valueArgs = append(valueArgs, vs.id, vs.name)
			argIdx += 2
		}

		// Build and execute INSERT statement
		insertQuery := fmt.Sprintf(
			`INSERT INTO vernacular_strings (id, name) VALUES %s ON CONFLICT (id) DO NOTHING`,
			joinStrings(valueStrings, ","),
		)

		_, err := p.operator.Pool().Exec(ctx, insertQuery, valueArgs...)
		if err != nil {
			return 0, fmt.Errorf("failed to insert vernacular strings batch: %w", err)
		}
	}

	slog.Info("Processed vernacular strings", "count", len(vernStrings))
	return len(vernStrings), nil
}

// processVernacularIndices reads vernacular records from SFGA with metadata
// and inserts them into vernacular_string_indices table, linking to data source.
func processVernacularIndices(
	ctx context.Context,
	p *populator,
	sfgaDB *sql.DB,
	sourceID int,
) (int, error) {
	slog.Info("Phase 2: Processing vernacular indices", "data_source_id", sourceID)

	// Clean old vernacular indices for this data source
	if err := cleanVernacularIndices(ctx, p, sourceID); err != nil {
		return 0, fmt.Errorf("failed to clean old vernacular indices: %w", err)
	}

	// Query SFGA vernacular table with all metadata
	query := `
		SELECT DISTINCT
			col__taxon_id, col__name, col__language,
			col__area, col__country
		FROM vernacular
	`

	rows, err := sfgaDB.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query SFGA vernacular indices: %w", err)
	}
	defer rows.Close()

	// Collect vernacular indices
	var indices []vernIndex
	for rows.Next() {
		var recordID, name, language, locality, countryCode string

		err := rows.Scan(&recordID, &name, &language, &locality, &countryCode)
		if err != nil {
			return 0, fmt.Errorf("failed to scan vernacular index row: %w", err)
		}

		// Truncate name if too long (to match vernacular_strings processing)
		if len(name) > 500 {
			name = name[:500]
		}

		// Fix UTF-8 encoding
		nameFixed := gnlib.FixUtf8(name)

		// Generate UUID v5 for vernacular string (matches Phase 1)
		uuid := gnuuid.New(nameFixed).String()

		// Truncate fields to fit database constraints
		if len(language) > 255 {
			language = language[:253] + "…"
		}
		if len(locality) > 255 {
			locality = locality[:253] + "…"
		}
		locality = gnlib.FixUtf8(locality)

		if len(countryCode) > 50 {
			countryCode = countryCode[:48] + "…"
		}

		indices = append(indices, vernIndex{
			recordID:           recordID,
			vernacularStringID: uuid,
			language:           language,
			locality:           locality,
			countryCode:        countryCode,
		})

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
	}

	if err = rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating vernacular index rows: %w", err)
	}

	// If no indices, nothing to do
	if len(indices) == 0 {
		slog.Info("No vernacular indices to process", "data_source_id", sourceID)
		return 0, nil
	}

	// Bulk insert using pgx.CopyFrom
	err = bulkInsertVernacularIndices(ctx, p, sourceID, indices)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk insert vernacular indices: %w", err)
	}

	slog.Info("Processed vernacular indices", "data_source_id", sourceID, "count", len(indices))
	return len(indices), nil
}

// cleanVernacularIndices removes old vernacular indices for a data source.
func cleanVernacularIndices(ctx context.Context, p *populator, sourceID int) error {
	slog.Info("Cleaning old vernacular indices", "data_source_id", sourceID)

	query := `DELETE FROM vernacular_string_indices WHERE data_source_id = $1`

	_, err := p.operator.Pool().Exec(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete old vernacular indices: %w", err)
	}

	return nil
}

// bulkInsertVernacularIndices performs efficient bulk insert of vernacular indices
// using pgx.CopyFrom.
func bulkInsertVernacularIndices(
	ctx context.Context,
	p *populator,
	sourceID int,
	indices []vernIndex,
) error {
	// Convert to format required by CopyFrom
	rows := make([][]any, len(indices))
	for i, idx := range indices {
		rows[i] = []any{
			sourceID,               // data_source_id
			idx.recordID,           // record_id
			idx.vernacularStringID, // vernacular_string_id
			"",                     // language_orig (not in SFGA v0.3.33)
			idx.language,           // language
			"",                     // lang_code (would need language detection library)
			idx.locality,           // locality
			idx.countryCode,        // country_code
			false,                  // preferred (not in SFGA v0.3.33)
		}
	}

	// Use CopyFrom for efficient bulk insert
	_, err := p.operator.Pool().CopyFrom(
		ctx,
		pgx.Identifier{"vernacular_string_indices"},
		[]string{
			"data_source_id",
			"record_id",
			"vernacular_string_id",
			"language_orig",
			"language",
			"lang_code",
			"locality",
			"country_code",
			"preferred",
		},
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return fmt.Errorf("CopyFrom failed: %w", err)
	}

	return nil
}

// joinStrings joins a slice of strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
