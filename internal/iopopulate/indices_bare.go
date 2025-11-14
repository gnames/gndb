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

// processBareNames processes names that are not in taxon or synonym tables.
// These are "orphan" names with no taxonomic context.
func (p *populator) processBareNames(
	source *sources.DataSourceConfig,
) (int, error) {
	slog.Info("Processing bare names", "data_source_id", source.ID)

	// Count total bare names for progress bar
	totalCount, err := p.getTotalBareCount()
	if err != nil {
		return 0, fmt.Errorf("failed to count bare names: %w", err)
	}

	// Build outlink column expression if configured
	outlinkCol := buildOutlinkColumn(source.OutlinkIDColumn, "bare_names")

	// Query names not in taxon or synonym
	rows, err := p.getBareData(outlinkCol)
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
		t, err := p.getBareDatum(rows, outlinkCol)
		if err != nil {
			return 0, fmt.Errorf("failed to scan bare name row: %w", err)
		}

		// Use gn__scientific_name_string if available, fallback to
		// col__scientific_name
		nameString := t.gnName
		if nameString == "" {
			nameString = t.colName
		}

		nameStringID := gnuuid.New(nameString).String()
		recordID := "bare-name-" + t.nameID

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
			source.ID,                 // data_source_id
			recordID,                  // record_id with "bare-name-" prefix
			nameStringID,              // name_string_id
			outlinkID,                 // outlink_id
			"",                        // global_id
			t.nameID,                  // name_id
			"",                        // local_id
			codeIDToInt(t.codeID),     // code_id
			strings.ToLower(t.rankID), // rank
			"bare name",               // taxonomic_status
			recordID,                  // accepted_record_id (self)
			"",                        // classification (NULL)
			"",                        // classification_ids (NULL)
			"",                        // classification_ranks (NULL)
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
			"Processed bare names",
			"data_source_id",
			source.ID,
			"count",
			humanize.Comma(int64(count)),
		)
	}

	return count, rows.Err()
}

func (p *populator) getTotalBareCount() (int, error) {
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
	err := p.sfgaDB.QueryRow(countQuery).Scan(&totalCount)
	return totalCount, err
}

func (p *populator) getBareData(outlinkCol string) (*sql.Rows, error) {
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
	return p.sfgaDB.Query(query)
}

func (p *populator) getBareDatum(
	rows *sql.Rows,
	outlinkCol string,
) (taxonDatum, error) {
	var t taxonDatum

	scanArgs := []any{&t.nameID, &t.colName, &t.gnName, &t.codeID, &t.rankID}

	// Add outlink ID to scan if column was selected
	if outlinkCol != "" {
		scanArgs = append(scanArgs, &t.outlinkIDRaw)
	}
	err := rows.Scan(scanArgs...)

	return t, err
}
