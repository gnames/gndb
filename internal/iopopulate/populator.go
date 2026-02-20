// Package iopopulate implements Populator interface for importing
// SFGA data into PostgreSQL.
// This is an impure I/O package that reads SFGA files and performs
// bulk inserts.
package iopopulate

import (
	"database/sql"
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
	sfgaDB   *sql.DB
}

// New creates a new Populator.
func New(cfg *config.Config, op db.Operator) gndb.Populator {
	return &populator{cfg: cfg, operator: op}
}

// Populate imports data from SFGA sources into the database.
// Orchestrates all phases: SFGA fetch, metadata, names, hierarchy,
// indices, and vernaculars.
func (p *populator) Populate() error {
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

	if err = p.processSources(sourcesToProcess, startTime); err != nil {
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

		// Process this source through all phases
		err := p.processSource(source)
		if err != nil {
			errorCount++
			slog.Error("Failed to process source",
				"data_source_id", source.ID,
				"title", source.TitleShort,
				"error", err,
			)
			gn.PrintErrorMessage(err)
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
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println() // Blank line between sources
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

// processSource handles a single data source through all phases:
// 1. SFGA file resolution and caching
// 2. Name strings import
// 3. Classification hierarchy construction
// 4. Name indices import (taxa, synonyms, bare names)
// 5. Vernacular names import
// 6. Data source metadata update
func (p *populator) processSource(
	source sources.DataSourceConfig,
) error {
	var t time.Time
	var msg string

	// Stage 1: Resolve and fetch SFGA file
	t = time.Now()
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
	sqlitePath, err := fetchSFGA(sfgaPath, cacheDir)
	if err != nil {
		return SFGAReadError(sfgaPath, err)
	}

	// Open SFGA database
	p.sfgaDB, err = openSFGA(sqlitePath)
	if err != nil {
		return SFGAReadError(sqlitePath, err)
	}
	defer p.sfgaDB.Close()
	gn.Message(
		"<em>Prepared SFGA file for import</em> %s",
		gnfmt.TimeString(time.Since(t).Seconds()),
	)

	// Stage 2: Import name-strings
	t = time.Now()
	gn.Info("(2/6) Importing name-strings...")

	err = p.checkSfgaVersion(source.ID)
	if err != nil {
		return err
	}

	msg, err = p.processNameStrings(source.ID)
	if err != nil {
		return NamesError(source.ID, err)
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(t).Seconds()))

	// Stage 3: Build classification hierarchy
	t = time.Now()
	gn.Info("(3/6) Building classification hierarchy...")
	hierarchy, err := p.buildHierarchy()
	if err != nil {
		// Hierarchy is optional, log warning and continue
		slog.Warn("Failed to build hierarchy",
			"source_id", source.ID,
			"error", err)
	}
	msg = "<em>Did not detect hierarchy existance</em>"
	if len(hierarchy) > 0 {
		msg = fmt.Sprintf(
			"<em>Finished building hierarchy with %s nodes</em>",
			humanize.Comma(int64(len(hierarchy))),
		)
	}
	gn.Message(
		"%s %s", msg, gnfmt.TimeString(time.Since(t).Seconds()),
	)

	// Stage 4: Import name-string indices
	t = time.Now()
	gn.Info("(4/6) Importing name-string indices...")
	msg, err = p.processNameIndices(&source, hierarchy)
	if err != nil {
		return NamesError(source.ID, err)
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(t).Seconds()))

	// Stage 5: Import vernacular names
	t = time.Now()
	gn.Info("(5/6) Importing vernacular names...")
	msg, err = p.processVernaculars(source.ID)
	if err != nil {
		// Vernaculars are optional, report error and continue
		slog.Error("Failed to import vernaculars",
			"source_id", source.ID,
			"error", err)
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(t).Seconds()))

	// Stage 6: Update data source metadata
	t = time.Now()
	gn.Info("(6/6) Importing metadata...")
	msg, err = p.updateDataSourceMetadata(source, metadata)
	if err != nil {
		return MetadataError(source.ID, err)
	}
	gn.Message("%s %s", msg, gnfmt.TimeString(time.Since(t).Seconds()))

	slog.Info("Source processing complete",
		"source_id", source.ID)

	return nil
}
