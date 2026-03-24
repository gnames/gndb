// Package ioexport implements the Exporter interface for exporting
// PostgreSQL gnames data to SFGA SQLite format files.
// This is an impure I/O package that reads from PostgreSQL and writes
// SFGA archives.
package ioexport

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/gndb"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/gnames/gnfmt"
	"github.com/gnames/gnlib/ent/nomcode"
	"github.com/gnames/gnparser"
)

// exporter implements the Exporter interface.
type exporter struct {
	cfg      *config.Config
	operator db.Operator

	// sources is the list of data sources to export, populated by Init.
	sources []schema.DataSource

	// parsers holds one gnparser instance per nomenclatural code.
	parsers map[nomcode.Code]gnparser.GNparser
}

// New creates a new Exporter.
func New(cfg *config.Config, op db.Operator) gndb.Exporter {
	return &exporter{cfg: cfg, operator: op}
}

// Init validates preconditions and prepares state for Export:
//  1. Checks that the database pool is connected.
//  2. Creates the output directory if it does not exist.
//  3. Loads the list of data sources to export from the data_sources table.
//  4. Initialises gnparser instances for each nomenclatural code.
func (e *exporter) Init(ctx context.Context) error {
	if e.operator.Pool() == nil {
		return NotConnectedError()
	}

	if err := e.ensureOutputDir(); err != nil {
		return err
	}

	sources, err := e.loadSources(ctx)
	if err != nil {
		return err
	}
	e.sources = sources

	e.parsers = buildParsers()

	return nil
}

// Export reads data from PostgreSQL for the configured source IDs
// and writes one SFGA .sqlite file per source into the output directory.
func (e *exporter) Export() error {
	ctx := context.Background()

	if err := e.Init(ctx); err != nil {
		return err
	}

	startTime := time.Now()
	successCount := 0
	errorCount := 0
	var exported []schema.DataSource

	for i, ds := range e.sources {
		sourceStart := time.Now()

		fmt.Println()
		fmt.Println(strings.Repeat("─", 60))
		gn.Info("Data Source [%d]: %s", ds.ID, ds.TitleShort)
		fmt.Println(strings.Repeat("─", 60))

		slog.Info("Exporting source",
			"index", i+1,
			"total", len(e.sources),
			"source_id", ds.ID,
			"title", ds.TitleShort,
		)

		if err := e.exportSource(ctx, ds); err != nil {
			errorCount++
			slog.Error("Failed to export source",
				"source_id", ds.ID,
				"title", ds.TitleShort,
				"error", err,
			)
			gn.PrintErrorMessage(err)
			continue
		}

		successCount++
		exported = append(exported, ds)

		dur := gnfmt.TimeString(time.Since(sourceStart).Seconds())
		slog.Info("Source exported successfully",
			"source_id", ds.ID,
			"title", ds.TitleShort,
			"duration", dur,
		)
		gn.Info("Completed in %s", dur)
	}

	// Write consolidated sources-export.yaml for all successful exports.
	if err := writeConsolidatedYAML(exported, e.cfg); err != nil {
		slog.Warn("Failed to write consolidated YAML", "error", err)
	}

	totalDur := gnfmt.TimeString(time.Since(startTime).Seconds())
	fmt.Println(strings.Repeat("─", 60))
	gn.Info(`Export complete
Sources succeeded: %d, failed: %d, total: %d.
Elapsed time: <em>%s</em>
`, successCount, errorCount, len(e.sources), totalDur)

	if errorCount > 0 && successCount == 0 {
		return AllSourcesFailedError(errorCount)
	}
	return nil
}

// exportSource exports a single data source through all 5 stages.
func (e *exporter) exportSource(ctx context.Context, ds schema.DataSource) error {
	pool := e.operator.Pool()
	batchSize := e.cfg.Database.BatchSize

	// Prepare temp work directory.
	workDir, err := prepareWorkDir(e.cfg.HomeDir, ds.ID)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(workDir); err != nil {
			slog.Warn("Failed to remove work directory",
				"dir", workDir, "error", err)
		}
	}()

	// Initialise SFGA archive in work directory.
	arc, err := initSfga(workDir)
	if err != nil {
		return SFGACreateError(ds.ID, err)
	}
	defer arc.Close()

	// (1/5) Metadata
	t := time.Now()
	gn.Info("(1/5) writing metadata...")
	if err := arc.InsertMeta(dataSourceToMeta(ds)); err != nil {
		return SFGAWriteError(ds.ID, "metadata", err)
	}
	gn.Message("<em>Metadata written</em> %s",
		gnfmt.TimeString(time.Since(t).Seconds()))

	// (2/5) Names
	if _, err := exportNames(ctx, pool, arc, e.parsers, ds, batchSize); err != nil {
		return err
	}

	// (3/5) Taxa
	if _, err := exportTaxa(ctx, pool, arc, ds, batchSize); err != nil {
		return err
	}

	// (4/5) Synonyms
	if _, err := exportSynonyms(ctx, pool, arc, ds, batchSize); err != nil {
		return err
	}

	// (5/5) Vernaculars
	if _, err := exportVernaculars(ctx, pool, arc, ds, batchSize); err != nil {
		return err
	}

	// Export SFGA to output files (.sqlite + .sql, optional .zip).
	outputDir := e.cfg.Export.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	basePath := buildOutputBase(ds, outputDir)

	gn.Info("Writing SFGA files to %s...", basePath)
	if err := arc.Export(basePath, e.cfg.Export.WithZip); err != nil {
		return SFGAWriteError(ds.ID, "export", err)
	}

	// Write companion .yaml
	if err := writeCompanionYAML(ds, basePath, e.cfg); err != nil {
		slog.Warn("Failed to write companion YAML",
			"source_id", ds.ID, "error", err)
	}

	return nil
}

// ensureOutputDir creates the output directory when it does not exist.
func (e *exporter) ensureOutputDir() error {
	dir := e.cfg.Export.OutputDir
	if dir == "" {
		dir = "."
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return OutputDirError(dir, err)
	}

	slog.Info("Output directory ready", "path", dir)
	return nil
}

// loadSources reads data_sources from PostgreSQL and filters to the
// requested IDs (or returns all rows when SourceIDs is empty).
func (e *exporter) loadSources(ctx context.Context) ([]schema.DataSource, error) {
	pool := e.operator.Pool()

	rows, err := pool.Query(ctx, `
		SELECT id, title, title_short, version, revision_date,
		       doi, citation, description, website_url, data_url,
		       outlink_url, is_outlink_ready, is_curated,
		       is_auto_curated, has_taxon_data, updated_at
		  FROM data_sources
		 ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []schema.DataSource
	for rows.Next() {
		var ds schema.DataSource
		if err := rows.Scan(
			&ds.ID, &ds.Title, &ds.TitleShort, &ds.Version, &ds.RevisionDate,
			&ds.DOI, &ds.Citation, &ds.Description, &ds.WebsiteURL, &ds.DataURL,
			&ds.OutlinkURL, &ds.IsOutlinkReady, &ds.IsCurated,
			&ds.IsAutoCurated, &ds.HasTaxonData, &ds.UpdatedAt,
		); err != nil {
			return nil, err
		}
		all = append(all, ds)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(e.cfg.Export.SourceIDs) == 0 {
		slog.Info("Exporting all sources", "count", len(all))
		return all, nil
	}

	wanted := make(map[int]bool, len(e.cfg.Export.SourceIDs))
	for _, id := range e.cfg.Export.SourceIDs {
		wanted[id] = true
	}

	var filtered []schema.DataSource
	for _, ds := range all {
		if wanted[ds.ID] {
			filtered = append(filtered, ds)
		}
	}

	if len(filtered) == 0 {
		return nil, NoSourcesError(e.cfg.Export.SourceIDs)
	}

	slog.Info("Exporting requested sources",
		"requested", len(e.cfg.Export.SourceIDs),
		"found", len(filtered),
	)
	return filtered, nil
}

// buildParsers creates one gnparser.GNparser per nomenclatural code.
func buildParsers() map[nomcode.Code]gnparser.GNparser {
	codes := []nomcode.Code{
		nomcode.Unknown,
		nomcode.Zoological,
		nomcode.Botanical,
		nomcode.Bacterial,
		nomcode.Virus,
		nomcode.Cultivars,
	}
	parsers := make(map[nomcode.Code]gnparser.GNparser, len(codes))
	for _, code := range codes {
		cfg := gnparser.NewConfig(
			gnparser.OptWithDetails(true),
			gnparser.OptCode(code),
		)
		parsers[code] = gnparser.New(cfg)
	}
	return parsers
}
