// Package iopopulate implements Populator interface for importing
// SFGA data into PostgreSQL.
// This is an impure I/O package that reads SFGA files and performs
// bulk inserts.
package iopopulate

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/lifecycle"
	"github.com/gnames/gndb/pkg/populate"
)

// populator implements the Populator interface.
type populator struct {
	operator db.Operator
}

// NewPopulator creates a new Populator.
func NewPopulator(op db.Operator) lifecycle.Populator {
	return &populator{operator: op}
}

// Populate imports data from SFGA sources into the database.
// Orchestrates all phases: SFGA fetch, metadata, names, hierarchy,
// indices, and vernaculars.
func (p *populator) Populate(
	ctx context.Context,
	cfg *config.Config,
) error {
	pool := p.operator.Pool()
	if pool == nil {
		return NotConnectedError()
	}

	startTime := time.Now()
	slog.Info("Starting database population")

	// Load sources.yaml from config directory
	sourcesPath := config.SourcesFilePath(cfg.HomeDir)
	sourcesConfig, err := populate.LoadSourcesConfig(sourcesPath)
	if err != nil {
		return SourcesConfigError(sourcesPath, err)
	}

	// Filter to requested source IDs (or all if empty)
	var sourcesToProcess []populate.DataSourceConfig
	if len(cfg.Populate.SourceIDs) == 0 {
		// Empty means process all sources
		sourcesToProcess = sourcesConfig.DataSources
		slog.Info("Processing all sources",
			"count", len(sourcesToProcess))
	} else {
		// Filter to requested IDs
		sourceIDMap := make(map[int]bool)
		for _, id := range cfg.Populate.SourceIDs {
			sourceIDMap[id] = true
		}

		for _, src := range sourcesConfig.DataSources {
			if sourceIDMap[src.ID] {
				sourcesToProcess = append(sourcesToProcess, src)
			}
		}

		if len(sourcesToProcess) == 0 {
			return NoSourcesError(cfg.Populate.SourceIDs)
		}

		sources := "source"
		if len(sourcesToProcess) > 1 {
			sources += "s"
		}
		msg := fmt.Sprintf("Processing %d %s",
			len(sourcesToProcess), sources)
		gn.Info(msg)
	}

	// Process each source
	successCount := 0
	errorCount := 0

	for i, source := range sourcesToProcess {
		sourceStartTime := time.Now()

		fmt.Println() // Blank line between sources
		fmt.Println(strings.Repeat("─", 60))
		msg := fmt.Sprintf("Data Source [%d]: %s",
			source.ID, source.TitleShort)
		gn.Info(msg)
		fmt.Println(strings.Repeat("─", 60))

		slog.Debug("Processing source",
			"index", i+1,
			"total", len(sourcesToProcess),
			"data_source_id", source.ID,
			"title", source.TitleShort,
		)

		// Check context cancellation
		select {
		case <-ctx.Done():
			return CancelledError(ctx.Err())
		default:
		}

		// Process this source through all phases
		err := p.processSource(ctx, cfg, source)
		if err != nil {
			errorCount++
			slog.Error("Failed to process source",
				"data_source_id", source.ID,
				"title", source.TitleShort,
				"error", err,
			)
			// Continue with next source instead of failing
			continue
		}

		successCount++
		sourceDuration := time.Since(sourceStartTime)
		slog.Debug("Source processed successfully",
			"data_source_id", source.ID,
			"title", source.TitleShort,
			"duration", sourceDuration.Round(time.Second),
		)

		msg = fmt.Sprintf("Completed in %s",
			sourceDuration.Round(time.Second).String())
		gn.Info(msg)
	}

	// Summary
	totalDuration := time.Since(startTime)
	slog.Info("Population complete",
		"success", successCount,
		"errors", errorCount,
		"total", len(sourcesToProcess),
		"duration", totalDuration.Round(time.Second),
	)

	if errorCount > 0 && successCount == 0 {
		return AllSourcesFailedError(errorCount)
	}

	if errorCount > 0 {
		slog.Warn("Some sources failed to process",
			"failed", errorCount,
			"succeeded", successCount)
	}

	return nil
}

// processSource handles a single data source through all phases.
// This is a stub for Phase 2 - detailed implementation in Phase 3/4.
func (p *populator) processSource(
	ctx context.Context,
	cfg *config.Config,
	source populate.DataSourceConfig,
) error {
	// Resolve SFGA file location
	sfgaPath, metadata, warning, err := resolveSFGAPath(source)
	if err != nil {
		return SFGAFileNotFoundError(source.ID, source.Parent, err)
	}

	if warning != "" {
		slog.Warn(warning)
	}

	slog.Info("Resolved SFGA file",
		"source_id", source.ID,
		"path", sfgaPath,
		"version", metadata.Version,
		"date", metadata.RevisionDate)

	// TODO Phase 3: Implement data import phases
	// 1. Import metadata (DataSource records)
	// 2. Import name-strings (NameString, Canonical, etc.)
	// 3. Import vernacular names

	// TODO Phase 4: Implement additional data
	// 1. Build classification hierarchy
	// 2. Import name-string indices
	// 3. Optimize indexes

	slog.Info("Source processing complete (stub)",
		"source_id", source.ID)

	return nil
}
