package iopopulate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/sources"
)

// processNameIndices implements Phase 4: Name Indices import from SFGA.
// It handles three scenarios:
//  1. Taxa (accepted names) - linked to taxon table with full classification
//  2. Synonyms - linked to accepted taxa via synonym table
//  3. Bare names - names not in taxon or synonym tables (orphans)
//
// Each scenario is processed separately with its own batch insert logic.
// The hierarchy map (built in Phase 3) provides classification paths for taxa and synonyms.
func (p *populator) processNameIndices(
	source *sources.DataSourceConfig,
	hierarchy map[string]*hNode,
) error {
	slog.Info("Processing name indices", "data_source_id", source.ID)

	// Clean old data for this source
	err := p.cleanNameIndices(source.ID)
	if err != nil {
		return err
	}

	// Process taxa (accepted names with classification)
	taxaCount, err := p.processTaxa(source, hierarchy)
	if err != nil {
		return fmt.Errorf("failed to process taxa: %w", err)
	}

	// Process synonyms (linked to accepted taxa)
	synonymCount, err := p.processSynonyms(source, hierarchy)
	if err != nil {
		return fmt.Errorf("failed to process synonyms: %w", err)
	}

	// Process bare names (orphans not in taxon/synonym)
	bareCount, err := p.processBareNames(source)
	if err != nil {
		return fmt.Errorf("failed to process bare names: %w", err)
	}

	totalCount := taxaCount + synonymCount + bareCount
	slog.Info("Name indices processing complete",
		"data_source_id", source.ID, "total", totalCount)

	// Print stats
	gn.Message(
		"<em>Imported %s name indices (%s taxa, %s synonyms, %s bare names)</em>",
		humanize.Comma(int64(totalCount)),
		humanize.Comma(int64(taxaCount)),
		humanize.Comma(int64(synonymCount)),
		humanize.Comma(int64(bareCount)),
	)

	return nil
}

// cleanNameIndices deletes existing name indices for the given data source.
// This ensures clean re-imports without duplicates.
func (p *populator) cleanNameIndices(sourceID int) error {
	query := "DELETE FROM name_string_indices WHERE data_source_id = $1"

	_, err := p.operator.Pool().Exec(context.Background(), query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to clean name indices: %w", err)
	}

	slog.Info("Cleaned old name indices", "data_source_id", sourceID)
	return nil
}
