package iopopulate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/dustin/go-humanize"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/populate"
	"github.com/gnames/gnlib"
	"github.com/gnames/gnuuid"
	"github.com/jackc/pgx/v5"
)

// buildOutlinkColumn maps table.column format to query alias for SELECT.
// Returns column expression (e.g., "t.col__id", "n.col__name_id") or empty string if table not available.
//
// Parameters:
//   - outlinkColumn: Format "table.column" (e.g., "taxon.col__id", "name.col__alternative_id")
//   - queryType: One of "taxa", "synonyms", "bare_names"
//
// Table availability by query type:
//   - "taxa":       taxon→t, name→n
//   - "synonyms":   taxon→t (accepted), synonym→s, name→n
//   - "bare_names": name→name (no alias)
//
// Returns empty string if:
//   - outlinkColumn is empty
//   - table not available for the query type
func buildOutlinkColumn(outlinkColumn, queryType string) string {
	if outlinkColumn == "" {
		return ""
	}

	// Parse table.column format
	parts := strings.Split(outlinkColumn, ".")
	if len(parts) != 2 {
		return "" // Invalid format
	}

	tableName := parts[0]
	columnName := parts[1]

	// Map table to query alias based on query type
	var alias string
	switch queryType {
	case "taxa":
		switch tableName {
		case "taxon":
			alias = "t"
		case "name":
			alias = "n"
		default:
			return "" // Table not available in taxa query
		}
	case "synonyms":
		switch tableName {
		case "taxon":
			alias = "t" // Accepted taxon
		case "synonym":
			alias = "s"
		case "name":
			alias = "n"
		default:
			return "" // Table not available in synonyms query
		}
	case "bare_names":
		switch tableName {
		case "name":
			alias = "name" // No alias in bare names query
		default:
			return "" // Only name table available in bare names query
		}
	default:
		return "" // Unknown query type
	}

	return alias + "." + columnName
}

// processNameIndices imports name indices from SFGA into the database.
// It handles three scenarios:
//  1. Taxa (accepted names) - linked to taxon table with full classification
//  2. Synonyms - linked to accepted taxa via synonym table
//  3. Bare names - names not in taxon or synonym tables (orphans)
//
// This follows the to-gn pattern of processing each scenario separately.
func processNameIndices(
	ctx context.Context,
	p *PopulatorImpl,
	sfgaDB *sql.DB,
	source *populate.DataSourceConfig,
	hierarchy map[string]*hNode,
	cfg *config.Config,
) error {
	slog.Debug("Processing name indices", "data_source_id", source.ID)

	// Clean old data for this source
	err := cleanNameIndices(ctx, p, source.ID)
	if err != nil {
		return err
	}

	// Process taxa (accepted names with classification)
	taxaCount, err := processTaxa(ctx, p, sfgaDB, source, hierarchy, cfg)
	if err != nil {
		return fmt.Errorf("failed to process taxa: %w", err)
	}

	// Process synonyms (linked to accepted taxa)
	synonymCount, err := processSynonyms(ctx, p, sfgaDB, source, hierarchy, cfg)
	if err != nil {
		return fmt.Errorf("failed to process synonyms: %w", err)
	}

	// Process bare names (orphans not in taxon/synonym)
	bareCount, err := processBareNames(ctx, p, sfgaDB, source, cfg)
	if err != nil {
		return fmt.Errorf("failed to process bare names: %w", err)
	}

	totalCount := taxaCount + synonymCount + bareCount
	slog.Debug("Name indices processing complete", "data_source_id", source.ID, "total", totalCount)

	// Print stats
	msg := fmt.Sprintf(
		"<em>Imported %s name indices (%s taxa, %s synonyms, %s bare names)</em>",
		humanize.Comma(int64(totalCount)),
		humanize.Comma(int64(taxaCount)),
		humanize.Comma(int64(synonymCount)),
		humanize.Comma(int64(bareCount)),
	)
	fmt.Println(gnlib.FormatMessage(msg, nil))

	return nil
}

// cleanNameIndices deletes existing name indices for the given data source.
// This ensures clean re-imports without duplicates.
func cleanNameIndices(ctx context.Context, p *PopulatorImpl, sourceID int) error {
	query := "DELETE FROM name_string_indices WHERE data_source_id = $1"

	_, err := p.operator.Pool().Exec(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to clean name indices: %w", err)
	}

	slog.Debug("Cleaned old name indices", "data_source_id", sourceID)
	return nil
}

// processTaxa processes accepted taxon records from the SFGA taxon table.
// Each taxon gets full classification via hierarchy breadcrumbs.
func processTaxa(
	ctx context.Context,
	p *PopulatorImpl,
	sfgaDB *sql.DB,
	source *populate.DataSourceConfig,
	hierarchy map[string]*hNode,
	cfg *config.Config,
) (int, error) {
	slog.Debug("Processing taxa (accepted names)", "data_source_id", source.ID)

	// Count total taxa for progress bar
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM taxon`
	err := sfgaDB.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return 0, fmt.Errorf("failed to count taxa: %w", err)
	}

	// Build outlink column expression if configured
	outlinkCol := buildOutlinkColumn(source.OutlinkIDColumn, "taxa")

	// Query taxa with flat classification fields
	query := `
		SELECT
			t.col__id, n.col__id, n.gn__scientific_name_string,
			n.col__code_id, n.col__rank_id, t.col__status_id,
			t.col__kingdom, t.sf__kingdom_id, t.col__phylum, t.sf__phylum_id,
			t.col__subphylum, t.sf__subphylum_id, t.col__class, t.sf__class_id,
			t.col__order, t.sf__order_id, t.col__family, t.sf__family_id,
			t.col__genus, t.sf__genus_id, t.col__species, t.sf__species_id`

	// Add outlink column if available
	if outlinkCol != "" {
		query += `,
			` + outlinkCol
	}

	query += `
		FROM taxon t
		JOIN name n ON n.col__id = t.col__name_id
	`

	rows, err := sfgaDB.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query taxa: %w", err)
	}
	defer rows.Close()

	// Collect records for bulk insert
	var records [][]any
	var count int

	// Create progress bar with known total
	bar := pb.Full.Start(totalCount)
	bar.Set("prefix", "Processing taxa: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	for rows.Next() {
		var taxonID, nameID, nameString, codeID, rankID, statusID string
		var kingdom, kingdomID, phylum, phylumID, subphylum, subphylumID sql.NullString
		var class, classID, order, orderID, family, familyID sql.NullString
		var genus, genusID, species, speciesID sql.NullString
		var outlinkIDRaw sql.NullString

		scanArgs := []any{
			&taxonID, &nameID, &nameString,
			&codeID, &rankID, &statusID,
			&kingdom, &kingdomID, &phylum, &phylumID,
			&subphylum, &subphylumID, &class, &classID,
			&order, &orderID, &family, &familyID,
			&genus, &genusID, &species, &speciesID,
		}

		// Add outlink ID to scan if column was selected
		if outlinkCol != "" {
			scanArgs = append(scanArgs, &outlinkIDRaw)
		}

		err := rows.Scan(scanArgs...)
		if err != nil {
			return 0, fmt.Errorf("failed to scan taxon row: %w", err)
		}

		// Build flat classification map
		flatClsf := make(map[string]string)
		if kingdom.Valid {
			flatClsf["kingdom"] = kingdom.String
			flatClsf["kingdom_id"] = kingdomID.String
		}
		if phylum.Valid {
			flatClsf["phylum"] = phylum.String
			flatClsf["phylum_id"] = phylumID.String
		}
		if subphylum.Valid {
			flatClsf["subphylum"] = subphylum.String
			flatClsf["subphylum_id"] = subphylumID.String
		}
		if class.Valid {
			flatClsf["class"] = class.String
			flatClsf["class_id"] = classID.String
		}
		if order.Valid {
			flatClsf["order"] = order.String
			flatClsf["order_id"] = orderID.String
		}
		if family.Valid {
			flatClsf["family"] = family.String
			flatClsf["family_id"] = familyID.String
		}
		if genus.Valid {
			flatClsf["genus"] = genus.String
			flatClsf["genus_id"] = genusID.String
		}
		if species.Valid {
			flatClsf["species"] = species.String
			flatClsf["species_id"] = speciesID.String
		}

		// Get classification breadcrumbs
		classification, classificationRanks, classificationIDs := getBreadcrumbs(
			taxonID, hierarchy, flatClsf, cfg.Import.WithFlatClassification,
		)

		// Generate UUID for name string
		nameStringID := gnuuid.New(nameString).String()

		// Extract outlink ID if available
		outlinkID := ""
		if outlinkIDRaw.Valid {
			// Get the column name from the outlink column config
			parts := strings.Split(source.OutlinkIDColumn, ".")
			if len(parts) == 2 {
				columnName := parts[1]
				outlinkID = populate.ExtractOutlinkID(columnName, outlinkIDRaw.String)
			}
		}

		// Create record for bulk insert
		record := []any{
			source.ID,           // data_source_id
			taxonID,             // record_id
			nameStringID,        // name_string_id
			outlinkID,           // outlink_id
			"",                  // global_id (not in SFGA)
			nameID,              // name_id
			nameID,              // local_id
			codeIDToInt(codeID), // code_id
			rankID,              // rank
			statusID,            // taxonomic_status
			taxonID,             // accepted_record_id (self for accepted taxa)
			classification,      // classification
			classificationIDs,   // classification_ids
			classificationRanks, // classification_ranks
		}

		records = append(records, record)
		count++

		// Update progress bar
		bar.Add(1)

		// Bulk insert when batch is full
		if len(records) >= cfg.Database.BatchSize {
			err = insertNameIndices(ctx, p, records)
			if err != nil {
				return 0, err
			}
			records = records[:0] // Clear batch
		}
	}

	// Insert remaining records
	if len(records) > 0 {
		err = insertNameIndices(ctx, p, records)
		if err != nil {
			return 0, err
		}
	}

	if count > 0 {
		slog.Debug("Processed taxa", "data_source_id", source.ID, "count", humanize.Comma(int64(count)))
	}

	return count, rows.Err()
}

// processSynonyms processes synonym records from the SFGA synonym table.
// Synonyms link to accepted taxa and inherit their classification.
func processSynonyms(
	ctx context.Context,
	p *PopulatorImpl,
	sfgaDB *sql.DB,
	source *populate.DataSourceConfig,
	hierarchy map[string]*hNode,
	cfg *config.Config,
) (int, error) {
	slog.Debug("Processing synonyms", "data_source_id", source.ID)

	// Check if synonym table exists
	var tableExists bool
	err := sfgaDB.QueryRow(`
		SELECT COUNT(*) > 0 FROM sqlite_master
		WHERE type='table' AND name='synonym'
	`).Scan(&tableExists)
	if err != nil {
		return 0, fmt.Errorf("failed to check synonym table: %w", err)
	}

	if !tableExists {
		slog.Debug("No synonym table in SFGA, skipping synonyms", "data_source_id", source.ID)
		return 0, nil
	}

	// Count total synonyms for progress bar
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM synonym`
	err = sfgaDB.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return 0, fmt.Errorf("failed to count synonyms: %w", err)
	}

	// Build outlink column expression if configured
	outlinkCol := buildOutlinkColumn(source.OutlinkIDColumn, "synonyms")

	// Query synonyms with their accepted taxon info
	query := `
		SELECT
			s.col__id, s.col__taxon_id, n.col__id, n.gn__scientific_name_string,
			n.col__code_id, n.col__rank_id, s.col__status_id,
			t.col__kingdom, t.sf__kingdom_id, t.col__phylum, t.sf__phylum_id,
			t.col__subphylum, t.sf__subphylum_id, t.col__class, t.sf__class_id,
			t.col__order, t.sf__order_id, t.col__family, t.sf__family_id,
			t.col__genus, t.sf__genus_id, t.col__species, t.sf__species_id`

	// Add outlink column if available
	if outlinkCol != "" {
		query += `,
			` + outlinkCol
	}

	query += `
		FROM synonym s
		JOIN name n ON n.col__id = s.col__name_id
		JOIN taxon t ON t.col__id = s.col__taxon_id
	`

	rows, err := sfgaDB.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query synonyms: %w", err)
	}
	defer rows.Close()

	var records [][]any
	var count int

	// Create progress bar with known total
	bar := pb.Full.Start(totalCount)
	bar.Set("prefix", "Processing synonyms: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	for rows.Next() {
		var synonymID, taxonID, nameID, nameString, codeID, rankID, statusID string
		var kingdom, kingdomID, phylum, phylumID, subphylum, subphylumID sql.NullString
		var class, classID, order, orderID, family, familyID sql.NullString
		var genus, genusID, species, speciesID sql.NullString
		var outlinkIDRaw sql.NullString

		scanArgs := []any{
			&synonymID, &taxonID, &nameID, &nameString,
			&codeID, &rankID, &statusID,
			&kingdom, &kingdomID, &phylum, &phylumID,
			&subphylum, &subphylumID, &class, &classID,
			&order, &orderID, &family, &familyID,
			&genus, &genusID, &species, &speciesID,
		}

		// Add outlink ID to scan if column was selected
		if outlinkCol != "" {
			scanArgs = append(scanArgs, &outlinkIDRaw)
		}

		err := rows.Scan(scanArgs...)
		if err != nil {
			return 0, fmt.Errorf("failed to scan synonym row: %w", err)
		}

		// Build flat classification from accepted taxon
		flatClsf := make(map[string]string)
		if kingdom.Valid {
			flatClsf["kingdom"] = kingdom.String
			flatClsf["kingdom_id"] = kingdomID.String
		}
		if phylum.Valid {
			flatClsf["phylum"] = phylum.String
			flatClsf["phylum_id"] = phylumID.String
		}
		if subphylum.Valid {
			flatClsf["subphylum"] = subphylum.String
			flatClsf["subphylum_id"] = subphylumID.String
		}
		if class.Valid {
			flatClsf["class"] = class.String
			flatClsf["class_id"] = classID.String
		}
		if order.Valid {
			flatClsf["order"] = order.String
			flatClsf["order_id"] = orderID.String
		}
		if family.Valid {
			flatClsf["family"] = family.String
			flatClsf["family_id"] = familyID.String
		}
		if genus.Valid {
			flatClsf["genus"] = genus.String
			flatClsf["genus_id"] = genusID.String
		}
		if species.Valid {
			flatClsf["species"] = species.String
			flatClsf["species_id"] = speciesID.String
		}

		// Get classification from accepted taxon
		classification, classificationRanks, classificationIDs := getBreadcrumbs(
			taxonID, hierarchy, flatClsf, cfg.Import.WithFlatClassification,
		)

		nameStringID := gnuuid.New(nameString).String()

		// Extract outlink ID if available
		outlinkID := ""
		if outlinkIDRaw.Valid {
			// Get the column name from the outlink column config
			parts := strings.Split(source.OutlinkIDColumn, ".")
			if len(parts) == 2 {
				columnName := parts[1]
				outlinkID = populate.ExtractOutlinkID(columnName, outlinkIDRaw.String)
			}
		}

		record := []any{
			source.ID,
			synonymID, // record_id (synonym's own ID)
			nameStringID,
			outlinkID,
			"",
			nameID,
			nameID,
			codeIDToInt(codeID),
			rankID,
			statusID,
			taxonID, // accepted_record_id (points to accepted taxon)
			classification,
			classificationIDs,
			classificationRanks,
		}

		records = append(records, record)
		count++

		// Update progress bar
		bar.Add(1)

		if len(records) >= cfg.Database.BatchSize {
			err = insertNameIndices(ctx, p, records)
			if err != nil {
				return 0, err
			}
			records = records[:0]
		}
	}

	if len(records) > 0 {
		err = insertNameIndices(ctx, p, records)
		if err != nil {
			return 0, err
		}
	}

	if count > 0 {
		slog.Debug("Processed synonyms", "data_source_id", source.ID, "count", humanize.Comma(int64(count)))
	}

	return count, rows.Err()
}

// processBareNames processes names that are not in taxon or synonym tables.
// These are "orphan" names with no taxonomic context.
func processBareNames(
	ctx context.Context,
	p *PopulatorImpl,
	sfgaDB *sql.DB,
	source *populate.DataSourceConfig,
	cfg *config.Config,
) (int, error) {
	slog.Debug("Processing bare names", "data_source_id", source.ID)

	// Count total bare names for progress bar
	var totalCount int
	countQuery := `
		SELECT COUNT(*)
		FROM name
		WHERE name.col__id NOT IN (
			SELECT col__name_id FROM taxon
			UNION
			SELECT col__name_id FROM synonym
		)
	`
	err := sfgaDB.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return 0, fmt.Errorf("failed to count bare names: %w", err)
	}

	// Build outlink column expression if configured
	outlinkCol := buildOutlinkColumn(source.OutlinkIDColumn, "bare_names")

	// Query names not in taxon or synonym
	query := `
		SELECT name.col__id, name.col__scientific_name, name.gn__scientific_name_string,
		       name.col__code_id, name.col__rank_id`

	// Add outlink column if available
	if outlinkCol != "" {
		query += `,
			` + outlinkCol
	}

	query += `
		FROM name
		WHERE name.col__id NOT IN (
			SELECT col__name_id FROM taxon
			UNION
			SELECT col__name_id FROM synonym
		)
	`

	rows, err := sfgaDB.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query bare names: %w", err)
	}
	defer rows.Close()

	var records [][]any
	var count int

	// Create progress bar with known total
	bar := pb.Full.Start(totalCount)
	bar.Set("prefix", "Processing bare names: ")
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	for rows.Next() {
		var nameID, sciName, gnName, codeID, rankID string
		var outlinkIDRaw sql.NullString

		scanArgs := []any{&nameID, &sciName, &gnName, &codeID, &rankID}

		// Add outlink ID to scan if column was selected
		if outlinkCol != "" {
			scanArgs = append(scanArgs, &outlinkIDRaw)
		}

		err := rows.Scan(scanArgs...)
		if err != nil {
			return 0, fmt.Errorf("failed to scan bare name row: %w", err)
		}

		// Use gn__scientific_name_string if available, fallback to col__scientific_name
		nameString := gnName
		if nameString == "" {
			nameString = sciName
		}

		nameStringID := gnuuid.New(nameString).String()
		recordID := "bare-name-" + nameID

		// Extract outlink ID if available
		outlinkID := ""
		if outlinkIDRaw.Valid {
			// Get the column name from the outlink column config
			parts := strings.Split(source.OutlinkIDColumn, ".")
			if len(parts) == 2 {
				columnName := parts[1]
				outlinkID = populate.ExtractOutlinkID(columnName, outlinkIDRaw.String)
			}
		}

		record := []any{
			source.ID,
			recordID, // record_id with "bare-name-" prefix
			nameStringID,
			outlinkID,
			"",
			nameID,
			nameID,
			codeIDToInt(codeID),
			rankID,
			"bare name", // taxonomic_status
			recordID,    // accepted_record_id (self)
			"",          // classification (NULL)
			"",          // classification_ids (NULL)
			"",          // classification_ranks (NULL)
		}

		records = append(records, record)
		count++

		// Update progress bar
		bar.Add(1)

		if len(records) >= cfg.Database.BatchSize {
			err = insertNameIndices(ctx, p, records)
			if err != nil {
				return 0, err
			}
			records = records[:0]
		}
	}

	if len(records) > 0 {
		err = insertNameIndices(ctx, p, records)
		if err != nil {
			return 0, err
		}
	}

	if count > 0 {
		slog.Debug("Processed bare names", "data_source_id", source.ID, "count", humanize.Comma(int64(count)))
	}

	return count, rows.Err()
}

// insertNameIndices performs bulk insert using pgx CopyFrom.
func insertNameIndices(ctx context.Context, p *PopulatorImpl, records [][]any) error {
	// Column names for CopyFrom
	columns := []string{
		"data_source_id", "record_id", "name_string_id",
		"outlink_id", "global_id", "name_id", "local_id",
		"code_id", "rank", "taxonomic_status", "accepted_record_id",
		"classification", "classification_ids", "classification_ranks",
	}

	_, err := p.operator.Pool().CopyFrom(
		ctx,
		pgx.Identifier{"name_string_indices"},
		columns,
		pgx.CopyFromRows(records),
	)

	return err
}

// codeIDToInt converts SFGA code ID string to integer.
// 0 - no info, 1 - ICZN (zoological), 2 - ICN (botanical), 3 - ICNP (bacterial), 4 - ICTV (viral)
func codeIDToInt(codeID string) int {
	switch strings.ToUpper(codeID) {
	case "ZOOLOGICAL", "ICZN":
		return 1
	case "BOTANICAL", "ICN", "ICNAFP":
		return 2
	case "BACTERIAL", "ICNP":
		return 3
	case "VIRAL", "VIRUS", "ICTV":
		return 4
	default:
		return 0
	}
}
