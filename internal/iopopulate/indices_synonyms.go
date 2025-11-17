package iopopulate

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/dustin/go-humanize"
	"github.com/gnames/gndb/pkg/sources"
	"github.com/gnames/gnuuid"
)

// processSynonyms processes synonym records from the SFGA synonym table.
// Synonyms link to accepted taxa and inherit their classification.
func (p *populator) processSynonyms(
	source *sources.DataSourceConfig,
	hierarchy map[string]*hNode,
) (int, error) {
	slog.Info("Processing synonyms", "data_source_id", source.ID)

	// Count total synonyms for progress bar
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM synonym`
	err := p.sfgaDB.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return 0, fmt.Errorf("failed to count synonyms: %w", err)
	}

	// Build outlink column expression if configured
	outlinkCol := buildOutlinkColumn(source.OutlinkIDColumn, "synonyms")

	rows, err := p.getSynonymData(outlinkCol)
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
		t, err := p.getSynonymDatum(rows, outlinkCol)
		if err != nil {
			return 0, fmt.Errorf("failed to scan synonym row: %w", err)
		}
		if t.statusID == "" {
			t.statusID = "SYNONYM"
		}

		flatClsf, useFlat := p.flatClassification(t)

		classification, classificationRanks, classificationIDs := getBreadcrumbs(
			t.taxonID, hierarchy, flatClsf, useFlat,
		)

		nameStringID := gnuuid.New(t.nameString).String()

		// Extract outlink ID if available
		outlinkID := ""
		if t.outlinkIDRaw.Valid {
			// Get the column name from the outlink column config
			parts := strings.Split(source.OutlinkIDColumn, ".")
			if len(parts) == 2 {
				columnName := parts[1]
				outlinkID = sources.ExtractOutlinkID(columnName, t.outlinkIDRaw.String)
			}
		}

		record := []any{
			source.ID,                   // data_source_id
			t.synonymID,                 // record_id (synonym's own ID)
			nameStringID,                // name_string_id
			outlinkID,                   // outlink_id
			t.globalID,                  // global_id
			t.nameID,                    // name_id
			t.localID,                   // local_id
			codeIDToInt(t.codeID),       // code_id
			strings.ToLower(t.rankID),   // rank
			strings.ToLower(t.statusID), // taxonomic_status
			t.taxonID,                   // accepted_record_id (points to accepted taxon)
			classification,              // classification
			classificationIDs,           // classification_ids
			classificationRanks,         // classification_ranks
		}

		records = append(records, record)
		count++

		// Update progress bar
		bar.Add(1)

		if len(records) >= p.cfg.Database.BatchSize {
			err = insertNameIndices(p, records)
			if err != nil {
				return 0, err
			}
			records = records[:0]
		}
	}

	if len(records) > 0 {
		err = insertNameIndices(p, records)
		if err != nil {
			return 0, err
		}
	}

	if count > 0 {
		slog.Info(
			"Processed synonyms",
			"data_source_id",
			source.ID,
			"count",
			humanize.Comma(int64(count)),
		)
	}

	return count, rows.Err()
}

func (p *populator) getSynonymData(outlinkCol string) (*sql.Rows, error) {
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

	return p.sfgaDB.Query(query)
}

func (p *populator) getSynonymDatum(
	rows *sql.Rows,
	outlinkCol string,
) (taxonDatum, error) {

	var t taxonDatum

	scanArgs := []any{
		&t.synonymID, &t.taxonID, &t.nameID, &t.nameString,
		&t.codeID, &t.rankID, &t.statusID,
		&t.kingdom, &t.kingdomID, &t.phylum, &t.phylumID,
		&t.subphylum, &t.subphylumID, &t.class, &t.classID,
		&t.order, &t.orderID, &t.family, &t.familyID,
		&t.genus, &t.genusID, &t.species, &t.speciesID,
	}

	// Add outlink ID to scan if column was selected
	if outlinkCol != "" {
		scanArgs = append(scanArgs, &t.outlinkIDRaw)
	}

	err := rows.Scan(scanArgs...)
	return t, err
}
