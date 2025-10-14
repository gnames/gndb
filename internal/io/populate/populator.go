// Package populate implements Populator interface for importing SFGA data into PostgreSQL.
// This is an impure I/O package that reads SFGA files and performs bulk inserts.
package populate

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	internalconfig "github.com/gnames/gndb/internal/io/config"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/database"
	"github.com/gnames/gndb/pkg/lifecycle"
	"github.com/gnames/gndb/pkg/populate"
)

// PopulatorImpl implements the Populator interface.
type PopulatorImpl struct {
	operator database.Operator
}

// NewPopulator creates a new Populator.
func NewPopulator(op database.Operator) lifecycle.Populator {
	return &PopulatorImpl{operator: op}
}

// Populate imports data from SFGA sources into the database.
// Orchestrates all phases: cache setup, SFGA fetch, name strings, hierarchy,
// name indices, vernaculars, and metadata updates.
func (p *PopulatorImpl) Populate(ctx context.Context, cfg *config.Config) error {
	pool := p.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database not connected")
	}

	startTime := time.Now()
	slog.Info("Starting database population")

	// Step 1: Load sources.yaml from default location
	configDir, err := internalconfig.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	sourcesYAMLPath := filepath.Join(configDir, "sources.yaml")

	sourcesConfig, err := populate.LoadSourcesConfig(sourcesYAMLPath)
	if err != nil {
		return fmt.Errorf("failed to load sources configuration from %s: %w", sourcesYAMLPath, err)
	}

	// Step 2: Filter to requested source IDs (or all if empty)
	var sourcesToProcess []populate.DataSourceConfig
	if len(cfg.Populate.SourceIDs) == 0 {
		// Empty means process all sources
		sourcesToProcess = sourcesConfig.DataSources
		slog.Info("Processing all sources", "count", len(sourcesToProcess))
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
			return fmt.Errorf("no sources found matching requested IDs: %v", cfg.Populate.SourceIDs)
		}

		slog.Info("Processing filtered sources", "count", len(sourcesToProcess), "ids", cfg.Populate.SourceIDs)
	}

	// Step 3: Validate release version/date constraints
	hasReleaseVersion := cfg.Populate.ReleaseVersion != ""
	hasReleaseDate := cfg.Populate.ReleaseDate != ""
	if (hasReleaseVersion || hasReleaseDate) && len(sourcesToProcess) > 1 {
		return fmt.Errorf(
			"release version/date overrides can only be used with a single source (found %d sources)",
			len(sourcesToProcess),
		)
	}

	// Step 4: Prepare cache directory (T034)
	cacheDir, err := prepareCacheDir()
	if err != nil {
		return fmt.Errorf("failed to prepare cache directory: %w", err)
	}
	slog.Info("Cache directory prepared", "path", cacheDir)

	// Step 5: Process each source
	successCount := 0
	errorCount := 0

	for i, source := range sourcesToProcess {
		sourceStartTime := time.Now()
		slog.Info("Processing source",
			"index", i+1,
			"total", len(sourcesToProcess),
			"id", source.ID,
			"title", source.TitleShort,
		)

		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("population cancelled: %w", ctx.Err())
		default:
		}

		// Process this source through all phases
		err := p.processSource(ctx, cfg, source, cacheDir)
		if err != nil {
			errorCount++
			slog.Error("Failed to process source",
				"id", source.ID,
				"title", source.TitleShort,
				"error", err,
			)
			// Continue with next source instead of failing entire operation
			continue
		}

		successCount++
		sourceDuration := time.Since(sourceStartTime)
		slog.Info("Source processed successfully",
			"id", source.ID,
			"title", source.TitleShort,
			"duration", sourceDuration.Round(time.Second),
		)
	}

	// Step 6: Summary
	totalDuration := time.Since(startTime)
	slog.Info("Population complete",
		"success", successCount,
		"errors", errorCount,
		"total", len(sourcesToProcess),
		"duration", totalDuration.Round(time.Second),
	)

	if errorCount > 0 && successCount == 0 {
		return fmt.Errorf("all %d sources failed to process", errorCount)
	}

	if errorCount > 0 {
		slog.Warn("Some sources failed to process", "failed", errorCount, "succeeded", successCount)
	}

	return nil
}

// processSource handles a single data source through all 5 phases.
func (p *PopulatorImpl) processSource(
	ctx context.Context,
	cfg *config.Config,
	source populate.DataSourceConfig,
	cacheDir string,
) error {
	// Phase 0: Fetch SFGA (T037)
	slog.Info("Fetching SFGA", "source_id", source.ID)
	sqlitePath, err := fetchSFGA(ctx, source, cacheDir)
	if err != nil {
		return fmt.Errorf("failed to fetch SFGA: %w", err)
	}

	// Phase 0: Open SFGA database (T037)
	sfgaDB, err := openSFGA(sqlitePath)
	if err != nil {
		return fmt.Errorf("failed to open SFGA database: %w", err)
	}
	defer sfgaDB.Close()

	// Phase 1: Process name strings (T039)
	slog.Info("Phase 1: Processing name strings", "source_id", source.ID)
	err = processNameStrings(ctx, p, sfgaDB, source.ID)
	if err != nil {
		return fmt.Errorf("phase 1 failed (name strings): %w", err)
	}

	// Phase 1.5: Build hierarchy for classification (T041)
	slog.Info("Building hierarchy for classification", "source_id", source.ID)
	hierarchy, err := buildHierarchy(ctx, sfgaDB, cfg.JobsNumber)
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Phase 2: Process name indices with classification (T043)
	slog.Info("Phase 2: Processing name indices", "source_id", source.ID)
	err = processNameIndices(ctx, p, sfgaDB, source.ID, hierarchy, cfg)
	if err != nil {
		return fmt.Errorf("phase 2 failed (name indices): %w", err)
	}

	// Phase 3-4: Process vernaculars (T045)
	slog.Info("Phase 3-4: Processing vernaculars", "source_id", source.ID)
	err = processVernaculars(ctx, p, sfgaDB, source.ID)
	if err != nil {
		return fmt.Errorf("phase 3-4 failed (vernaculars): %w", err)
	}

	// Phase 5: Update data source metadata (T047)
	slog.Info("Phase 5: Updating data source metadata", "source_id", source.ID)
	err = updateDataSourceMetadata(ctx, p, source, sfgaDB)
	if err != nil {
		return fmt.Errorf("phase 5 failed (metadata): %w", err)
	}

	return nil
}
