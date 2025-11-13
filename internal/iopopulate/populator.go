// Package iopopulate implements Populator interface for importing
// SFGA data into PostgreSQL.
// This is an impure I/O package that reads SFGA files and performs
// bulk inserts.
package iopopulate

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iosources"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/gndb"
	"github.com/gnames/gndb/pkg/sources"
	"github.com/gnames/gnfmt"
)

// populator implements the Populator interface.
type populator struct {
	cfg      *config.Config
	operator db.Operator
}

// New creates a new Populator.
func New(cfg *config.Config, op db.Operator) gndb.Populator {
	return &populator{cfg: cfg, operator: op}
}

// Populate imports data from SFGA sources into the database.
// Orchestrates all phases: SFGA fetch, metadata, names, hierarchy,
// indices, and vernaculars.
func (p *populator) Populate(
	ctx context.Context,
) error {
	pool := p.operator.Pool()
	if pool == nil {
		return NotConnectedError()
	}

	startTime := time.Now()
	slog.Info("Starting database population")

	// Load sources.yaml from config directory
	src := iosources.New(p.cfg)
	sourcesConfig, err := src.Load()
	if err != nil {
		return err
	}

	sourcesToProcess, err := p.collectSources(sourcesConfig)
	if err != nil {
		return err
	}

	if err = p.processSources(ctx, sourcesToProcess, startTime); err != nil {
		return err
	}

	return nil
}

func (p *populator) collectSources(
	sourcesConfig *sources.SourcesConfig,
) ([]sources.DataSourceConfig, error) {
	// Filter to requested source IDs (or all if empty)
	var sourcesToProcess []sources.DataSourceConfig
	if len(p.cfg.Populate.SourceIDs) == 0 {
		// Empty means process all sources
		sourcesToProcess = sourcesConfig.DataSources
		slog.Info("Processing all sources",
			"count", len(sourcesToProcess))
	} else {
		// Filter to requested IDs
		sourceIDMap := make(map[int]bool)
		for _, id := range p.cfg.Populate.SourceIDs {
			sourceIDMap[id] = true
		}

		for _, src := range sourcesConfig.DataSources {
			if sourceIDMap[src.ID] {
				sourcesToProcess = append(sourcesToProcess, src)
			}
		}

		if len(sourcesToProcess) == 0 {
			return nil, NoSourcesError(p.cfg.Populate.SourceIDs)
		}

		sources := "source"
		if len(sourcesToProcess) > 1 {
			sources += "s"
		}
		msg := fmt.Sprintf("Processing %d %s",
			len(sourcesToProcess), sources)
		gn.Info(msg)
	}
	return sourcesToProcess, nil
}

func (p *populator) processSources(
	ctx context.Context,
	sourcesToProcess []sources.DataSourceConfig,
	startTime time.Time,
) error {
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

		slog.Info("Processing source",
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
		err := p.processSource(ctx, source)
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
		slog.Info("Source processed successfully",
			"data_source_id", source.ID,
			"title", source.TitleShort,
			"duration", gnfmt.TimeString(sourceDuration.Seconds()),
		)

		msg = fmt.Sprintf("Completed in %s",
			gnfmt.TimeString(sourceDuration.Seconds()))
		gn.Info(msg)
	}

	// Summary
	totalDuration := time.Since(startTime)
	slog.Info("Population complete",
		"success", successCount,
		"errors", errorCount,
		"total", len(sourcesToProcess),
		"duration", gnfmt.TimeString(totalDuration.Seconds()),
	)
	gn.Info(`Population complete
Sources succeded: %d, failed %d, total %d.
		Elapsed time: <em>%s</em>
`,
		successCount,
		errorCount,
		len(sourcesToProcess),
		gnfmt.TimeString(totalDuration.Seconds()),
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
	source sources.DataSourceConfig,
) error {
	// Resolve SFGA file location
	sfgaPath, metadata, warning, err := resolveSFGAPath(source)
	if err != nil {
		return SFGAFileNotFoundError(source.ID, source.Parent, err)
	}

	if warning != "" {
		slog.Warn(warning)
	}
	file := filepath.Base(sfgaPath)
	gn.Info("(1/6) getting SFGA file <em>%s</em>", file)
	slog.Info("Resolved SFGA file",
		"source_id", source.ID,
		"path", sfgaPath,
		"version", metadata.Version,
		"date", metadata.RevisionDate)

	// Prepare cache directory
	cacheDir, err := prepareCacheDir(p.cfg.HomeDir)
	if err != nil {
		return CacheError("prepare cache directory", err)
	}

	// Fetch SFGA file to cache
	sqlitePath, err := fetchSFGA(ctx, sfgaPath, cacheDir)
	if err != nil {
		return SFGAReadError(sfgaPath, err)
	}

	// Open SFGA database
	sfgaDB, err := openSFGA(sqlitePath)
	if err != nil {
		return SFGAReadError(sqlitePath, err)
	}
	defer sfgaDB.Close()
	gn.Message("<em>Prepared SFGA file for import</em>")

	gn.Info("(2/6) Importing name-strings...")
	var nameStrNum int
	nameStrNum, err = processNameStrings(ctx, p, sfgaDB, source.ID)
	if err != nil {
		return NamesError(source.ID, err)
	}
	gn.Message(
		"<em>Imported %s name strings</em>",
		humanize.Comma(int64(nameStrNum)))

	gn.Info("(3/6) Importing vernacular names...")
	err = processVernaculars(ctx, p, sfgaDB, source.ID)
	if err != nil {
		// Vernaculars are optional, report error and continue
		slog.Error("Failed to import vernaculars",
			"source_id", source.ID,
			"error", err)
	}

	gn.Info("(4/6) Building classification hierarchy...")
	hierarchy, err := buildHierarchy(ctx, sfgaDB, p.cfg.JobsNumber)
	if err != nil {
		// Hierarchy is optional, log warning and continue
		slog.Warn("Failed to build hierarchy",
			"source_id", source.ID,
			"error", err)
	}

	gn.Info("(5/6) Importing name-string indices...")
	err = processNameIndices(ctx, p, sfgaDB, &source, hierarchy, p.cfg)
	if err != nil {
		return NamesError(source.ID, err)
	}

	gn.Info("(6/6) Importing metadata...")
	err = updateDataSourceMetadata(ctx, p, source, sfgaDB, metadata)
	if err != nil {
		return MetadataError(source.ID, err)
	}

	slog.Info("Source processing complete",
		"source_id", source.ID)

	return nil
}
