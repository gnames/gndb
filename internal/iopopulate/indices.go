package iopopulate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gnuuid"
	"github.com/jackc/pgx/v5"
)

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
	sourceID int,
	hierarchy map[string]*hNode,
	cfg *config.Config,
) error {
	slog.Info("Processing name indices", "sourceID", sourceID)

	// Clean old data for this source
	err := cleanNameIndices(ctx, p, sourceID)
	if err != nil {
		return err
	}

	// Process taxa (accepted names with classification)
	err = processTaxa(ctx, p, sfgaDB, sourceID, hierarchy, cfg)
	if err != nil {
		return fmt.Errorf("failed to process taxa: %w", err)
	}

	// Process synonyms (linked to accepted taxa)
	err = processSynonyms(ctx, p, sfgaDB, sourceID, hierarchy, cfg)
	if err != nil {
		return fmt.Errorf("failed to process synonyms: %w", err)
	}

	// Process bare names (orphans not in taxon/synonym)
	err = processBareNames(ctx, p, sfgaDB, sourceID, cfg)
	if err != nil {
		return fmt.Errorf("failed to process bare names: %w", err)
	}

	slog.Info("Name indices processing complete", "sourceID", sourceID)
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

	slog.Info("Cleaned old name indices", "sourceID", sourceID)
	return nil
}

// processTaxa processes accepted taxon records from the SFGA taxon table.
// Each taxon gets full classification via hierarchy breadcrumbs.
func processTaxa(
	ctx context.Context,
	p *PopulatorImpl,
	sfgaDB *sql.DB,
	sourceID int,
	hierarchy map[string]*hNode,
	cfg *config.Config,
) error {
	slog.Info("Processing taxa (accepted names)")

	// Query taxa with flat classification fields
	query := `
		SELECT
			t.col__id, n.col__id, n.gn__scientific_name_string,
			n.col__code_id, n.col__rank_id, t.col__status_id,
			t.col__kingdom, t.sf__kingdom_id, t.col__phylum, t.sf__phylum_id,
			t.col__subphylum, t.sf__subphylum_id, t.col__class, t.sf__class_id,
			t.col__order, t.sf__order_id, t.col__family, t.sf__family_id,
			t.col__genus, t.sf__genus_id, t.col__species, t.sf__species_id
		FROM taxon t
		JOIN name n ON n.col__id = t.col__name_id
	`

	rows, err := sfgaDB.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query taxa: %w", err)
	}
	defer rows.Close()

	// Collect records for bulk insert
	var records [][]interface{}
	var count int

	for rows.Next() {
		var taxonID, nameID, nameString, codeID, rankID, statusID string
		var kingdom, kingdomID, phylum, phylumID, subphylum, subphylumID sql.NullString
		var class, classID, order, orderID, family, familyID sql.NullString
		var genus, genusID, species, speciesID sql.NullString

		err := rows.Scan(
			&taxonID, &nameID, &nameString,
			&codeID, &rankID, &statusID,
			&kingdom, &kingdomID, &phylum, &phylumID,
			&subphylum, &subphylumID, &class, &classID,
			&order, &orderID, &family, &familyID,
			&genus, &genusID, &species, &speciesID,
		)
		if err != nil {
			return fmt.Errorf("failed to scan taxon row: %w", err)
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

		// Create record for bulk insert
		record := []interface{}{
			sourceID,            // data_source_id
			taxonID,             // record_id
			nameStringID,        // name_string_id
			"",                  // outlink_id (not implemented yet)
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

		if count%100_000 == 0 {
			progressReport(count, "taxa")
		}

		// Bulk insert when batch is full
		if len(records) >= cfg.Import.BatchSize {
			err = insertNameIndices(ctx, p, records)
			if err != nil {
				return err
			}
			records = records[:0] // Clear batch
		}
	}

	// Insert remaining records
	if len(records) > 0 {
		err = insertNameIndices(ctx, p, records)
		if err != nil {
			return err
		}
	}

	if count > 0 {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 80))
		slog.Info("Processed taxa", "count", humanize.Comma(int64(count)))
	}

	return rows.Err()
}

// processSynonyms processes synonym records from the SFGA synonym table.
// Synonyms link to accepted taxa and inherit their classification.
func processSynonyms(
	ctx context.Context,
	p *PopulatorImpl,
	sfgaDB *sql.DB,
	sourceID int,
	hierarchy map[string]*hNode,
	cfg *config.Config,
) error {
	slog.Info("Processing synonyms")

	// Check if synonym table exists
	var tableExists bool
	err := sfgaDB.QueryRow(`
		SELECT COUNT(*) > 0 FROM sqlite_master
		WHERE type='table' AND name='synonym'
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check synonym table: %w", err)
	}

	if !tableExists {
		slog.Info("No synonym table in SFGA, skipping synonyms")
		return nil
	}

	// Query synonyms with their accepted taxon info
	query := `
		SELECT
			s.col__id, s.col__taxon_id, n.col__id, n.gn__scientific_name_string,
			n.col__code_id, n.col__rank_id, s.col__status_id,
			t.col__kingdom, t.sf__kingdom_id, t.col__phylum, t.sf__phylum_id,
			t.col__subphylum, t.sf__subphylum_id, t.col__class, t.sf__class_id,
			t.col__order, t.sf__order_id, t.col__family, t.sf__family_id,
			t.col__genus, t.sf__genus_id, t.col__species, t.sf__species_id
		FROM synonym s
		JOIN name n ON n.col__id = s.col__name_id
		JOIN taxon t ON t.col__id = s.col__taxon_id
	`

	rows, err := sfgaDB.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query synonyms: %w", err)
	}
	defer rows.Close()

	var records [][]interface{}
	var count int

	for rows.Next() {
		var synonymID, taxonID, nameID, nameString, codeID, rankID, statusID string
		var kingdom, kingdomID, phylum, phylumID, subphylum, subphylumID sql.NullString
		var class, classID, order, orderID, family, familyID sql.NullString
		var genus, genusID, species, speciesID sql.NullString

		err := rows.Scan(
			&synonymID, &taxonID, &nameID, &nameString,
			&codeID, &rankID, &statusID,
			&kingdom, &kingdomID, &phylum, &phylumID,
			&subphylum, &subphylumID, &class, &classID,
			&order, &orderID, &family, &familyID,
			&genus, &genusID, &species, &speciesID,
		)
		if err != nil {
			return fmt.Errorf("failed to scan synonym row: %w", err)
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

		record := []interface{}{
			sourceID,
			synonymID, // record_id (synonym's own ID)
			nameStringID,
			"",
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

		if count%100_000 == 0 {
			progressReport(count, "synonyms")
		}

		if len(records) >= cfg.Import.BatchSize {
			err = insertNameIndices(ctx, p, records)
			if err != nil {
				return err
			}
			records = records[:0]
		}
	}

	if len(records) > 0 {
		err = insertNameIndices(ctx, p, records)
		if err != nil {
			return err
		}
	}

	if count > 0 {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 80))
		slog.Info("Processed synonyms", "count", humanize.Comma(int64(count)))
	}

	return rows.Err()
}

// processBareNames processes names that are not in taxon or synonym tables.
// These are "orphan" names with no taxonomic context.
func processBareNames(
	ctx context.Context,
	p *PopulatorImpl,
	sfgaDB *sql.DB,
	sourceID int,
	cfg *config.Config,
) error {
	slog.Info("Processing bare names")

	// Query names not in taxon or synonym
	query := `
		SELECT col__id, col__scientific_name, gn__scientific_name_string,
		       col__code_id, col__rank_id
		FROM name
		WHERE col__id NOT IN (
			SELECT col__name_id FROM taxon
			UNION
			SELECT col__name_id FROM synonym
		)
	`

	rows, err := sfgaDB.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query bare names: %w", err)
	}
	defer rows.Close()

	var records [][]interface{}
	var count int

	for rows.Next() {
		var nameID, sciName, gnName, codeID, rankID string

		err := rows.Scan(&nameID, &sciName, &gnName, &codeID, &rankID)
		if err != nil {
			return fmt.Errorf("failed to scan bare name row: %w", err)
		}

		// Use gn__scientific_name_string if available, fallback to col__scientific_name
		nameString := gnName
		if nameString == "" {
			nameString = sciName
		}

		nameStringID := gnuuid.New(nameString).String()
		recordID := "bare-name-" + nameID

		record := []interface{}{
			sourceID,
			recordID, // record_id with "bare-name-" prefix
			nameStringID,
			"",
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

		if count%100_000 == 0 {
			progressReport(count, "bare names")
		}

		if len(records) >= cfg.Import.BatchSize {
			err = insertNameIndices(ctx, p, records)
			if err != nil {
				return err
			}
			records = records[:0]
		}
	}

	if len(records) > 0 {
		err = insertNameIndices(ctx, p, records)
		if err != nil {
			return err
		}
	}

	if count > 0 {
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 80))
		slog.Info("Processed bare names", "count", humanize.Comma(int64(count)))
	}

	return rows.Err()
}

// insertNameIndices performs bulk insert using pgx CopyFrom.
func insertNameIndices(ctx context.Context, p *PopulatorImpl, records [][]interface{}) error {
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
