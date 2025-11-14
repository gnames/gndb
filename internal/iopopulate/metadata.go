package iopopulate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/gnames/gndb/pkg/sources"
	"github.com/google/uuid"
)

// updateDataSourceMetadata implements Phase 6: Data Source Metadata update.
// Creates or updates the data_sources table record by merging metadata from
// multiple sources.
//
// Metadata Sources:
//  1. SFGA metadata table (p.sfgaDB): title, description, doi, citation, authors
//  2. sources.yaml config: title_short, home_url, outlink_url, curation flags
//  3. Database counts: name_string_indices and vernacular_string_indices
//  4. SFGA filename: version, revision_date
//
// Uses DELETE + INSERT pattern for idempotency.
//
// Returns error if SFGA query, count query, or database insert fails.
func (p *populator) updateDataSourceMetadata(
	source sources.DataSourceConfig,
	sfgaFileMeta SFGAMetadata,
) error {
	slog.Info("Updating data source metadata", "data_source_id", source.ID)

	// Step 1: Read metadata from SFGA
	sfgaMetadata, err := p.readSFGAMetadata()
	if err != nil {
		return fmt.Errorf("failed to read SFGA metadata: %w", err)
	}

	// Step 2: Query record counts from database
	recordCount, err := p.queryNameStringIndicesCount(source.ID)
	if err != nil {
		return fmt.Errorf("failed to query name string indices count: %w", err)
	}

	vernRecordCount, err := p.queryVernacularIndicesCount(source.ID)
	if err != nil {
		return fmt.Errorf("failed to query vernacular indices count: %w", err)
	}

	// Step 3: Build DataSource record merging SFGA + sources.yaml metadata
	ds := buildDataSourceRecord(
		source, sfgaMetadata, sfgaFileMeta, recordCount, vernRecordCount,
	)

	// Step 4: Delete existing data source record (for idempotency)
	err = deleteDataSource(p, source.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing data source: %w", err)
	}

	// Step 5: Insert new data source record
	err = insertDataSource(p, ds)
	if err != nil {
		return fmt.Errorf("failed to insert data source: %w", err)
	}

	slog.Info("Data source metadata updated",
		"data_source_id", source.ID,
		"title_short", ds.TitleShort,
		"record_count", ds.RecordCount,
		"vern_record_count", ds.VernRecordCount,
	)

	// Print stats
	totalRecords := ds.RecordCount + ds.VernRecordCount
	gn.Message(
		"<em>Imported metadata and found %s total records</em>",
		humanize.Comma(int64(totalRecords)),
	)

	return nil
}

// sfgaMetadata holds metadata read from SFGA metadata table.
type sfgaMetadata struct {
	Title       string
	Description string
	DOI         string
}

// readSFGAMetadata reads metadata from SFGA metadata table.
// Returns zero values for missing/empty fields (graceful handling).
func (p *populator) readSFGAMetadata() (*sfgaMetadata, error) {
	query := `
		SELECT col__title, col__description, col__doi
		FROM metadata
		LIMIT 1
	`

	var meta sfgaMetadata
	err := p.sfgaDB.QueryRow(query).Scan(&meta.Title, &meta.Description, &meta.DOI)
	if err != nil {
		// If metadata table doesn't exist or is empty, return empty metadata
		if err == sql.ErrNoRows {
			slog.Warn("SFGA metadata table is empty, using empty metadata")
			return &sfgaMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to query SFGA metadata: %w", err)
	}

	return &meta, nil
}

// queryNameStringIndicesCount queries the count of name_string_indices
// for a given data source.
func (p *populator) queryNameStringIndicesCount(
	sourceID int,
) (int, error) {
	query := `SELECT COUNT(*) FROM name_string_indices WHERE data_source_id = $1`

	var count int
	err := p.operator.Pool().QueryRow(context.Background(), query, sourceID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count name string indices: %w", err)
	}

	return count, nil
}

// queryVernacularIndicesCount queries the count of vernacular_string_indices
// for a given data source.
func (p *populator) queryVernacularIndicesCount(
	sourceID int,
) (int, error) {
	query := `SELECT COUNT(*) FROM vernacular_string_indices WHERE data_source_id = $1`

	var count int
	err := p.operator.Pool().QueryRow(context.Background(), query, sourceID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count vernacular indices: %w", err)
	}

	return count, nil
}

// buildDataSourceRecord constructs a DataSource record by merging metadata
// from SFGA, sources.yaml config, and database counts.
//
// Priority:
//   - SFGA provides: title, description, doi
//   - sources.yaml provides: id, title_short, home_url, flags
//   - Database queries provide: record_count, vern_record_count
//   - SFGA filename provides: version, revision_date
//   - System provides: updated_at (current timestamp)
//
// Fields not currently populated (future enhancements):
//   - UUID: Would come from SFGA or sources.yaml (currently nil)
//   - Citation: Would come from SFGA metadata
//   - Authors: Would come from SFGA metadata
func buildDataSourceRecord(
	source sources.DataSourceConfig,
	sfgaMetadata *sfgaMetadata,
	sfgaFileMeta SFGAMetadata,
	recordCount int,
	vernRecordCount int,
) schema.DataSource {
	// Use nil UUID (could be enhanced to read from SFGA or sources.yaml)
	uuidStr := uuid.Nil.String()

	// Use Title from sources.yaml if provided, otherwise from SFGA
	title := sfgaMetadata.Title
	if source.Title != "" {
		title = source.Title
	}

	// Use Description from sources.yaml if provided, otherwise from SFGA
	description := sfgaMetadata.Description
	if source.Description != "" {
		description = source.Description
	}

	// Build the complete DataSource record
	ds := schema.DataSource{
		ID:              source.ID,
		UUID:            uuidStr,
		Title:           title,
		TitleShort:      source.TitleShort,
		Version:         sfgaFileMeta.Version,
		RevisionDate:    sfgaFileMeta.RevisionDate,
		DOI:             sfgaMetadata.DOI,
		Citation:        "", // Future: extract from SFGA
		Authors:         "", // Future: extract from SFGA
		Description:     description,
		WebsiteURL:      source.HomeURL,
		DataURL:         source.DataURL,
		OutlinkURL:      source.OutlinkURL,
		IsOutlinkReady:  source.IsOutlinkReady,
		IsCurated:       source.IsCurated,
		IsAutoCurated:   source.IsAutoCurated,
		HasTaxonData:    source.HasClassification,
		RecordCount:     recordCount,
		VernRecordCount: vernRecordCount,
		UpdatedAt:       time.Now(),
	}

	return ds
}

// deleteDataSource deletes an existing data source record by ID.
// This is part of the DELETE + INSERT pattern for idempotency.
// Does not return an error if the record doesn't exist.
func deleteDataSource(p *populator, sourceID int) error {
	query := `DELETE FROM data_sources WHERE id = $1`

	_, err := p.operator.Pool().Exec(context.Background(), query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to execute delete: %w", err)
	}

	return nil
}

// insertDataSource inserts a new data source record.
func insertDataSource(p *populator, ds schema.DataSource) error {
	query := `
		INSERT INTO data_sources (
			id, uuid, title, title_short, version, revision_date,
			doi, citation, authors, description,
			website_url, data_url, outlink_url, is_outlink_ready,
			is_curated, is_auto_curated, has_taxon_data,
			record_count, vern_record_count, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`

	_, err := p.operator.Pool().Exec(context.Background(), query,
		ds.ID,
		ds.UUID,
		ds.Title,
		ds.TitleShort,
		ds.Version,
		ds.RevisionDate,
		ds.DOI,
		ds.Citation,
		ds.Authors,
		ds.Description,
		ds.WebsiteURL,
		ds.DataURL,
		ds.OutlinkURL,
		ds.IsOutlinkReady,
		ds.IsCurated,
		ds.IsAutoCurated,
		ds.HasTaxonData,
		ds.RecordCount,
		ds.VernRecordCount,
		ds.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to execute insert: %w", err)
	}

	return nil
}
