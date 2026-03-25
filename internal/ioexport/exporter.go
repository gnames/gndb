// Package ioexport implements the Exporter interface for exporting
// PostgreSQL gnames data to SFGA SQLite format files.
// This is an impure I/O package that reads from PostgreSQL and writes
// SFGA archives.
package ioexport

import (
	"context"
	"log/slog"
	"os"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/gndb"
	"github.com/gnames/gndb/pkg/schema"
)

// exporter implements the Exporter interface.
type exporter struct {
	cfg      *config.Config
	operator db.Operator

	// sources is the list of data sources to export, populated by Init.
	sources []schema.DataSource
}

// New creates a new Exporter.
func New(cfg *config.Config, op db.Operator) gndb.Exporter {
	return &exporter{cfg: cfg, operator: op}
}

// Init validates preconditions and prepares state for Export:
//  1. Checks that the database pool is connected.
//  2. Creates the output directory if it does not exist.
//  3. Loads the list of data sources to export from the data_sources table.
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

	return nil
}

// Export reads data from PostgreSQL for the configured source IDs
// and writes one SFGA .sqlite file per source into the output directory.
func (e *exporter) Export() error {
	ctx := context.Background()

	if err := e.Init(ctx); err != nil {
		return err
	}

	// TODO: process each source.
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

	rows, err := pool.Query(ctx,
		`SELECT id, title_short, title, version, revision_date
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
			&ds.ID,
			&ds.TitleShort,
			&ds.Title,
			&ds.Version,
			&ds.RevisionDate,
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

	// Filter to requested IDs.
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
