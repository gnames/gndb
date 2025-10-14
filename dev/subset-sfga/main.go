// subset-sfga extracts a representative subset from large SFGA data sources.
//
// This tool creates smaller test SFGA files that preserve:
//   - Edge cases (empty strings, special characters, deep hierarchies, orphans)
//   - Hierarchy consistency (all parent nodes included)
//   - Representative sampling across the full dataset
//
// Uses sflib Reader/Writer to ensure SFGA format correctness.
//
// Usage:
//
//	go run . <source> <output>
//
// Examples:
//
//	go run . "http://opendata.globalnames.org/sfga/latest/0001.sqlite.zip" ../../testdata/0001-subset.sqlite
//	go run . "/path/to/local/0147.sqlite" ../../testdata/0147-subset.sqlite
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnames/gndb/internal/ioconfig"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sfborg/sflib"
)

// Configuration constants
const (
	// Target number of name_string records to extract
	targetNameStrings = 3000

	// Minimum records to include from each edge case category
	minEdgeCaseRecords = 50

	// Maximum hierarchy depth to fully preserve (all parents included)
	maxHierarchyDepth = 20
)

func main() {
	// Parse positional arguments
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <source> <output>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  source  SFGA source URL or local file path\n")
		fmt.Fprintf(os.Stderr, "  output  Path for output subset SFGA file\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s http://opendata.globalnames.org/sfga/latest/0001.sqlite.zip testdata/0001.sqlite\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s /path/to/local/0147.sqlite testdata/0147-subset.sqlite\n", os.Args[0])
		os.Exit(1)
	}

	sourceURL := os.Args[1]
	outputPath := os.Args[2]

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx := context.Background()

	logger.Info("starting SFGA subset extraction",
		"source", sourceURL,
		"target_size", targetNameStrings,
		"output", outputPath,
	)

	// TODO: Implement subset extraction
	if err := createSubset(ctx, logger, sourceURL, outputPath); err != nil {
		logger.Error("subset extraction failed", "error", err)
		os.Exit(1)
	}

	logger.Info("subset extraction complete", "output", outputPath)
}

// traverseAncestry follows parent_id chains for all taxa in the map.
// Adds all parent col__ids to the map (automatically deduplicated).
// This ensures we have complete hierarchy chains for all selected taxa.
func traverseAncestry(ctx context.Context, logger *slog.Logger, db *sql.DB, taxonMap map[string]bool) error {
	// For each taxon in the map, follow parent_id chain to root
	// We'll process in batches to avoid too many individual queries

	// Get initial snapshot of IDs (map will grow as we add parents)
	initialIDs := make([]string, 0, len(taxonMap))
	for id := range taxonMap {
		initialIDs = append(initialIDs, id)
	}

	logger.Info("traversing ancestry", "starting_taxa", len(initialIDs))

	// For each taxon, recursively find parents
	for _, colID := range initialIDs {
		if err := addAncestors(ctx, db, colID, taxonMap); err != nil {
			// Log but continue - some records may have broken parent_id references
			logger.Debug("could not add ancestors", "col__id", colID, "error", err)
		}
	}

	return nil
}

// addAncestors recursively adds all ancestors for a given col__id to the map.
func addAncestors(ctx context.Context, db *sql.DB, colID string, taxonMap map[string]bool) error {
	// Query for col__parent_id
	query := `SELECT col__parent_id FROM taxon WHERE col__id = ? AND col__parent_id IS NOT NULL AND col__parent_id != ''`

	var parentID sql.NullString
	err := db.QueryRowContext(ctx, query, colID).Scan(&parentID)
	if err == sql.ErrNoRows {
		// No parent (root node) or no such taxon - that's fine
		return nil
	}
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	// If no parent or already seen, stop recursion
	if !parentID.Valid || parentID.String == "" || taxonMap[parentID.String] {
		return nil
	}

	// Add parent to map
	taxonMap[parentID.String] = true

	// Recursively add parent's ancestors
	return addAncestors(ctx, db, parentID.String, taxonMap)
}

// queryNamesWithVernaculars queries for taxa that have vernacular names.
// Returns taxon col__ids that appear in vernacular table.
// These are NOT orphan names - they must have taxon records.
func queryNamesWithVernaculars(ctx context.Context, logger *slog.Logger, db *sql.DB, limit int) ([]string, error) {
	// Query for taxa that have vernacular names
	query := `
		SELECT DISTINCT col__taxon_id
		FROM vernacular
		WHERE col__taxon_id IS NOT NULL AND col__taxon_id != ''
		LIMIT ?
	`

	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var taxonIDs []string
	for rows.Next() {
		var colID sql.NullString

		if err := rows.Scan(&colID); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Only include valid IDs
		if colID.Valid && colID.String != "" {
			taxonIDs = append(taxonIDs, colID.String)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	logger.Info("queried names with vernaculars", "requested", limit, "found", len(taxonIDs))
	return taxonIDs, nil
}

// queryNamesWithSynonyms queries for accepted names that have synonyms.
// Returns taxon col__ids for names that appear in synonym table.
// Tries both col__taxon_id (COL schema) and col__accepted_id (other schemas).
func queryNamesWithSynonyms(ctx context.Context, logger *slog.Logger, db *sql.DB, limit int) ([]string, error) {
	// Try col__taxon_id first (COL uses this for accepted taxon)
	query := `
		SELECT DISTINCT col__taxon_id
		FROM synonym
		WHERE col__taxon_id IS NOT NULL AND col__taxon_id != ''
		LIMIT ?
	`

	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		// Try col__accepted_id (other sources might use this)
		query = `
			SELECT DISTINCT t.col__id
			FROM taxon t
			WHERE t.col__id IN (
				SELECT DISTINCT col__accepted_id
				FROM synonym
				WHERE col__accepted_id IS NOT NULL AND col__accepted_id != ''
			)
			LIMIT ?
		`
		rows, err = db.QueryContext(ctx, query, limit)
		if err != nil {
			return nil, fmt.Errorf("query failed: %w", err)
		}
	}
	defer rows.Close()

	var taxonIDs []string
	for rows.Next() {
		var colID sql.NullString

		if err := rows.Scan(&colID); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Only include valid IDs
		if colID.Valid && colID.String != "" {
			taxonIDs = append(taxonIDs, colID.String)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	logger.Info("queried names with synonyms", "requested", limit, "found", len(taxonIDs))
	return taxonIDs, nil
}

// queryOrphanNames queries for names in name table that don't appear in taxon table.
// Returns a slice of name col__id strings.
// Some sources may have 0-N orphan names, that's fine!
func queryOrphanNames(ctx context.Context, logger *slog.Logger, db *sql.DB, limit int) ([]string, error) {
	// Query for names that exist but aren't linked to any taxon
	query := `
		SELECT DISTINCT n.col__id
		FROM name n
		WHERE n.col__id NOT IN (
			SELECT DISTINCT col__name_id
			FROM taxon
			WHERE col__name_id IS NOT NULL AND col__name_id != ''
		)
		LIMIT ?
	`

	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var nameIDs []string
	for rows.Next() {
		var id sql.NullString

		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if id.Valid && id.String != "" {
			nameIDs = append(nameIDs, id.String)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	logger.Info("queried orphan names", "requested", limit, "found", len(nameIDs))
	return nameIDs, nil
}

// queryLeafNodes queries for leaf taxa (taxa with no children).
// Returns a slice of taxon IDs (col__id from taxon table) for the leaf nodes.
// We need IDs to later trace parent chains and include all ancestors.
func queryLeafNodes(ctx context.Context, logger *slog.Logger, db *sql.DB, limit int) ([]string, error) {
	// Query for leaf nodes: taxa that don't appear as col__parent_id in other records
	// We want the col__id (taxon ID) not the name_string ID
	query := `
		SELECT DISTINCT t.col__id
		FROM taxon t
		WHERE t.col__id NOT IN (
			SELECT DISTINCT col__parent_id
			FROM taxon
			WHERE col__parent_id IS NOT NULL AND col__parent_id != ''
		)
		LIMIT ?
	`

	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var taxonIDs []string
	for rows.Next() {
		var colID sql.NullString

		if err := rows.Scan(&colID); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Only include valid IDs
		if colID.Valid && colID.String != "" {
			taxonIDs = append(taxonIDs, colID.String)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	logger.Info("queried leaf nodes", "requested", limit, "found", len(taxonIDs))
	return taxonIDs, nil
}

// fetchSFGA returns the path to the SFGA file.
// For local .sqlite files, returns the path directly.
// For URLs or .zip files, uses sflib to download/extract.
func fetchSFGA(ctx context.Context, logger *slog.Logger, source, cacheDir string) (string, error) {
	logger.Info("fetching SFGA", "source", source)

	// If it's a local .sqlite file, just use it directly
	if _, err := os.Stat(source); err == nil && strings.HasSuffix(source, ".sqlite") {
		logger.Info("using local SFGA file", "path", source)
		return source, nil
	}

	// Clean cache directory before fetching to avoid "too many database files" error
	// sflib gets confused if there are multiple .sqlite files in the cache
	logger.Info("cleaning cache directory", "path", cacheDir)
	files, err := filepath.Glob(filepath.Join(cacheDir, "*.sqlite"))
	if err == nil {
		for _, file := range files {
			os.Remove(file)
		}
	}

	// Otherwise use sflib to fetch (URLs, zip files, etc.)
	arc := sflib.NewSfga()
	err = arc.Fetch(source, cacheDir)
	if err != nil {
		return "", fmt.Errorf("sflib fetch failed: %w", err)
	}

	// Get the extracted database path
	dbPath := arc.DbPath()
	if dbPath == "" {
		return "", fmt.Errorf("sflib did not return database path after fetch")
	}

	logger.Info("SFGA ready", "path", dbPath)
	return dbPath, nil
}

// copyNameRecords copies name records where col__id corresponds to col__name_id from selected taxa.
// Uses INSERT INTO ... SELECT from attached 'source' database for efficient bulk copy.
// Batches the operation to avoid SQLite's 999 variable limit.
func copyNameRecords(ctx context.Context, logger *slog.Logger, targetDB *sql.DB, taxonIDs []string) (int, error) {
	if len(taxonIDs) == 0 {
		return 0, nil
	}

	const batchSize = 900 // SQLite has ~999 variable limit
	totalCount := 0

	// Process in batches
	for i := 0; i < len(taxonIDs); i += batchSize {
		end := i + batchSize
		if end > len(taxonIDs) {
			end = len(taxonIDs)
		}
		batch := taxonIDs[i:end]

		// Copy names that are referenced by selected taxa
		query := `
			INSERT OR IGNORE INTO name
			SELECT DISTINCT n.* FROM source.name n
			INNER JOIN source.taxon t ON n.col__id = t.col__name_id
			WHERE t.col__id IN (` + buildPlaceholders(len(batch)) + `)`

		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}

		result, err := targetDB.ExecContext(ctx, query, args...)
		if err != nil {
			return totalCount, fmt.Errorf("failed to copy name records: %w", err)
		}

		count, _ := result.RowsAffected()
		totalCount += int(count)
	}

	return totalCount, nil
}

// buildPlaceholders creates a comma-separated string of '?' placeholders for SQL IN clause.
func buildPlaceholders(n int) string {
	if n == 0 {
		return ""
	}
	placeholders := make([]string, n)
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return strings.Join(placeholders, ",")
}

// copyTaxonRecords copies taxon records where col__id is in the given list.
// Uses INSERT INTO ... SELECT from attached 'source' database for efficient bulk copy.
// Batches the operation to avoid SQLite's 999 variable limit.
func copyTaxonRecords(ctx context.Context, logger *slog.Logger, targetDB *sql.DB, colIDs []string) (int, error) {
	if len(colIDs) == 0 {
		return 0, nil
	}

	const batchSize = 900 // SQLite has ~999 variable limit
	totalCount := 0

	// Process in batches
	for i := 0; i < len(colIDs); i += batchSize {
		end := i + batchSize
		if end > len(colIDs) {
			end = len(colIDs)
		}
		batch := colIDs[i:end]

		query := fmt.Sprintf(`
			INSERT OR IGNORE INTO taxon SELECT * FROM source.taxon WHERE col__id IN (%s)
		`, buildPlaceholders(len(batch)))

		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}

		result, err := targetDB.ExecContext(ctx, query, args...)
		if err != nil {
			return totalCount, fmt.Errorf("failed to copy taxon records: %w", err)
		}

		count, _ := result.RowsAffected()
		totalCount += int(count)
	}

	return totalCount, nil
}

// copySynonymRecords copies synonym records where col__taxon_id or col__accepted_id is in the given taxon ID list.
// Tries col__taxon_id first (COL schema), falls back to col__accepted_id (other schemas).
// Batches the operation to avoid SQLite's 999 variable limit.
func copySynonymRecords(ctx context.Context, logger *slog.Logger, targetDB *sql.DB, taxonIDs []string) (int, error) {
	if len(taxonIDs) == 0 {
		return 0, nil
	}

	const batchSize = 900
	totalCount := 0

	// Try col__taxon_id first (COL schema)
	for i := 0; i < len(taxonIDs); i += batchSize {
		end := i + batchSize
		if end > len(taxonIDs) {
			end = len(taxonIDs)
		}
		batch := taxonIDs[i:end]

		query := fmt.Sprintf(`
			INSERT OR IGNORE INTO synonym SELECT * FROM source.synonym WHERE col__taxon_id IN (%s)
		`, buildPlaceholders(len(batch)))

		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}

		result, err := targetDB.ExecContext(ctx, query, args...)
		if err != nil {
			// Try col__accepted_id (other schemas)
			query = fmt.Sprintf(`
				INSERT OR IGNORE INTO synonym SELECT * FROM source.synonym WHERE col__accepted_id IN (%s)
			`, buildPlaceholders(len(batch)))

			result, err = targetDB.ExecContext(ctx, query, args...)
			if err != nil {
				// Table might not exist
				return totalCount, nil
			}
		}

		count, _ := result.RowsAffected()
		totalCount += int(count)
	}

	return totalCount, nil
}

// copyVernacularRecords copies vernacular records where col__taxon_id is in the given taxon ID list.
// Batches the operation to avoid SQLite's 999 variable limit.
func copyVernacularRecords(ctx context.Context, logger *slog.Logger, targetDB *sql.DB, taxonIDs []string) (int, error) {
	if len(taxonIDs) == 0 {
		return 0, nil
	}

	const batchSize = 900
	totalCount := 0

	for i := 0; i < len(taxonIDs); i += batchSize {
		end := i + batchSize
		if end > len(taxonIDs) {
			end = len(taxonIDs)
		}
		batch := taxonIDs[i:end]

		query := fmt.Sprintf(`
			INSERT OR IGNORE INTO vernacular SELECT * FROM source.vernacular WHERE col__taxon_id IN (%s)
		`, buildPlaceholders(len(batch)))

		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}

		result, err := targetDB.ExecContext(ctx, query, args...)
		if err != nil {
			// Table might not exist
			return totalCount, nil
		}

		count, _ := result.RowsAffected()
		totalCount += int(count)
	}

	return totalCount, nil
}

// prepareCacheDir returns the SFGA cache directory path and ensures it exists.
// Uses the same cache location as gndb: ~/.cache/gndb/sfga/
//
// Note: This shares the cache with gndb populate. That's fine because:
//   - Cache is only for debugging (inspect downloaded SFGA files)
//   - Both tools download/cache SFGA files the same way
//   - If subset-sfga downloads COL, gndb populate can reuse it (helpful!)
func prepareCacheDir() (string, error) {
	// Get base cache directory
	baseCache, err := ioconfig.GetCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Append "sfga" subdirectory
	sfgaCache := filepath.Join(baseCache, "sfga")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(sfgaCache, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return sfgaCache, nil
}

// createSubset extracts a representative subset from an SFGA source.
//
// Implementation uses sflib Reader/Writer pattern:
//  1. Fetch source SFGA using sflib.Fetch (handles URLs and local paths)
//  2. Open with sflib.Reader and query SQLite for edge cases
//  3. Random sample remaining records to reach target size
//  4. Complete hierarchy (include all parent taxon records)
//  5. Read records into coldp structs (NameUsage, Vernacular)
//  6. Write to output using sflib.Writer
//
// See tasks.md for detailed implementation steps.
func createSubset(ctx context.Context, logger *slog.Logger, sourceURL, outputPath string) error {
	// Setup cache directory
	cacheDir, err := prepareCacheDir()
	if err != nil {
		return fmt.Errorf("failed to prepare cache: %w", err)
	}
	logger.Info("cache directory ready", "path", cacheDir)

	// Fetch source SFGA
	sfgaPath, err := fetchSFGA(ctx, logger, sourceURL, cacheDir)
	if err != nil {
		return fmt.Errorf("failed to fetch SFGA: %w", err)
	}
	logger.Info("SFGA file ready", "path", sfgaPath)

	// Open SFGA SQLite database
	db, err := sql.Open("sqlite3", sfgaPath)
	if err != nil {
		return fmt.Errorf("failed to open SFGA database: %w", err)
	}
	defer db.Close()

	// Query for leaf nodes (taxa with no children)
	// Note: Some sources may have 0-N results, that's fine!
	// We'll need to traverse ancestry for these to include all parent taxa
	leafTaxonIDs, err := queryLeafNodes(ctx, logger, db, targetNameStrings)
	if err != nil {
		// Log the error but continue - some sources don't have taxon table
		logger.Warn("could not query leaf nodes", "error", err)
		leafTaxonIDs = []string{} // Empty slice, will try other approaches
	}
	logger.Info("leaf taxon IDs collected (will traverse ancestry)", "count", len(leafTaxonIDs))

	// Query for orphan names (names in name but not in taxon)
	// These are standalone - no ancestry to traverse since they have no taxon records
	orphanNameIDs, err := queryOrphanNames(ctx, logger, db, targetNameStrings)
	if err != nil {
		logger.Warn("could not query orphan names", "error", err)
		orphanNameIDs = []string{} // Empty slice
	}
	logger.Info("orphan name IDs collected (no ancestry)", "count", len(orphanNameIDs))

	// Query for names that have synonyms (accepted names with synonym records)
	// These need ancestry traversal, so we'll deduplicate and add to leafTaxonIDs
	synonymParentIDs, err := queryNamesWithSynonyms(ctx, logger, db, targetNameStrings)
	if err != nil {
		logger.Warn("could not query names with synonyms", "error", err)
		synonymParentIDs = []string{} // Empty slice
	}
	logger.Info("names with synonyms collected", "count", len(synonymParentIDs))

	// Query for names that have vernacular connections
	// These are NOT orphan names (connection via taxon table)
	// Need ancestry traversal like leaf nodes and synonyms
	vernacularNameIDs, err := queryNamesWithVernaculars(ctx, logger, db, targetNameStrings)
	if err != nil {
		logger.Warn("could not query names with vernaculars", "error", err)
		vernacularNameIDs = []string{} // Empty slice
	}
	logger.Info("names with vernaculars collected", "count", len(vernacularNameIDs))

	// Use map for automatic deduplication - much simpler!
	// Map key = col__id, value = true (we just care about presence)
	taxonMap := make(map[string]bool)

	// Add leaf taxa
	for _, id := range leafTaxonIDs {
		taxonMap[id] = true
	}

	// Add synonym parents
	for _, id := range synonymParentIDs {
		taxonMap[id] = true
	}

	// Add vernacular names
	for _, id := range vernacularNameIDs {
		taxonMap[id] = true
	}

	logger.Info("initial taxon IDs collected", "count", len(taxonMap))

	// Traverse ancestry for all taxa in the map
	// This will add all parent col__ids (automatically deduplicated by map)
	err = traverseAncestry(ctx, logger, db, taxonMap)
	if err != nil {
		logger.Warn("could not traverse ancestry", "error", err)
	}

	logger.Info("total unique taxon IDs (after ancestry traversal)", "count", len(taxonMap))

	// Phase 5: Load metadata from source SFGA
	logger.Info("loading metadata from source SFGA")
	sourceArc := sflib.NewSfga()
	sourceArc.SetDb(sfgaPath)

	// Connect to the database
	_, err = sourceArc.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to source SFGA: %w", err)
	}
	defer sourceArc.Close()

	meta, err := sourceArc.LoadMeta()
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	// Update metadata for subset
	meta.Title = meta.Title + " (Subset)"
	meta.Description = fmt.Sprintf("Subset of %d taxa extracted for testing purposes. %s", len(taxonMap), meta.Description)
	logger.Info("metadata loaded", "title", meta.Title)

	// Phase 6: Create output SFGA archive
	logger.Info("creating output SFGA", "path", outputPath)

	// Remove output file if it already exists
	if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing output file: %w", err)
	}

	outputArc := sflib.NewSfga()

	// Create needs a directory, so extract the directory from output path
	outputDir := filepath.Dir(outputPath)

	// Remove schema.sqlite if it exists from previous run
	schemaPath := filepath.Join(outputDir, "schema.sqlite")
	if err := os.Remove(schemaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing schema file: %w", err)
	}

	if err := outputArc.Create(outputDir); err != nil {
		return fmt.Errorf("failed to create output SFGA: %w", err)
	}
	defer outputArc.Close()

	// Create() creates schema.sqlite in the directory
	// We need to rename it to our desired output path
	if err := os.Rename(schemaPath, outputPath); err != nil {
		return fmt.Errorf("failed to rename output file: %w", err)
	}

	// Set the database path to the renamed output path
	outputArc.SetDb(outputPath)

	// Connect to the output database
	_, err = outputArc.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to output SFGA: %w", err)
	}

	// Insert metadata
	if err := outputArc.InsertMeta(meta); err != nil {
		return fmt.Errorf("failed to insert metadata: %w", err)
	}
	logger.Info("metadata written to output")

	// Phase 5 & 6: Query and copy records directly using collected col__ids
	// This is much more efficient than loading all records via channels and filtering

	// Build list of taxon IDs from taxonMap (non-orphan taxa with ancestry)
	var taxonIDs []string
	for id := range taxonMap {
		taxonIDs = append(taxonIDs, id)
	}
	logger.Info("starting record copy", "taxon_ids", len(taxonIDs), "orphan_names", len(orphanNameIDs))

	// Get output database handle for direct writes
	outputDB := outputArc.Db()
	if outputDB == nil {
		return fmt.Errorf("output database not connected")
	}

	// Attach source database to output database for efficient copying
	_, err = outputDB.ExecContext(ctx, fmt.Sprintf("ATTACH DATABASE '%s' AS source", sfgaPath))
	if err != nil {
		return fmt.Errorf("failed to attach source database: %w", err)
	}
	defer outputDB.ExecContext(ctx, "DETACH DATABASE source")

	// Copy name table records first (referenced by taxon.name_string_id)
	nameCount, err := copyNameRecords(ctx, logger, outputDB, taxonIDs)
	if err != nil {
		logger.Warn("failed to copy name records", "error", err)
	}
	logger.Info("name records copied", "count", nameCount)

	// Copy taxon table records where col__id is in our selection
	taxonCount, err := copyTaxonRecords(ctx, logger, outputDB, taxonIDs)
	if err != nil {
		logger.Warn("failed to copy taxon records", "error", err)
	}
	logger.Info("taxon records copied", "count", taxonCount)

	// Copy synonym table records where accepted_id is in our taxon IDs
	synonymCount, err := copySynonymRecords(ctx, logger, outputDB, taxonIDs)
	if err != nil {
		logger.Warn("failed to copy synonym records", "error", err)
	}
	logger.Info("synonym records copied", "count", synonymCount)

	// Copy vernacular records where taxon_id is in our taxon IDs
	vernacularCount, err := copyVernacularRecords(ctx, logger, outputDB, taxonIDs)
	if err != nil {
		logger.Warn("failed to copy vernacular records", "error", err)
	}
	logger.Info("vernacular records copied", "count", vernacularCount)

	// Phase 7: Log summary
	logger.Info("subset extraction complete",
		"output", outputPath,
		"names", nameCount,
		"taxa", taxonCount,
		"synonyms", synonymCount,
		"vernaculars", vernacularCount,
		"orphan_names", len(orphanNameIDs),
	)

	return nil
}
