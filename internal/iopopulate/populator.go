// Package populate implements Populator interface for importing SFGA data into PostgreSQL.
// This is an impure I/O package that reads SFGA files and performs bulk inserts.
package iopopulate

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gnames/gndb/internal/ioconfig"
	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/gnames/gndb/pkg/lifecycle"
	"github.com/gnames/gndb/pkg/populate"
	"github.com/gnames/gnlib"
)

// PopulatorImpl implements the Populator interface.
type PopulatorImpl struct {
	operator db.Operator
}

// NewPopulator creates a new Populator.
func NewPopulator(op db.Operator) lifecycle.Populator {
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

	// Preparation

	startTime := time.Now()
	slog.Info("Starting database population")

	// Pre 1. Load sources.yaml from default location
	configDir, err := ioconfig.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	sourcesYAMLPath := filepath.Join(configDir, "sources.yaml")

	sourcesConfig, err := populate.LoadSourcesConfig(sourcesYAMLPath)
	if err != nil {
		return fmt.Errorf("failed to load sources configuration from %s: %w", sourcesYAMLPath, err)
	}

	// Pre 2. Filter to requested source IDs (or all if empty)
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
			return fmt.Errorf(
				"no sources found matching requested IDs: %v",
				cfg.Populate.SourceIDs,
			)
		}

		sources := "source"
		if len(sourcesToProcess) > 1 {
			sources += "s"
		}
		msg := gnlib.FormatMessage(
			"<em>Processing %d %s</em>",
			[]any{len(sourcesToProcess), sources},
		)

		fmt.Println(msg)
	}

	// Pre 3. Validate release version/date constraints
	hasReleaseVersion := cfg.Populate.ReleaseVersion != ""
	hasReleaseDate := cfg.Populate.ReleaseDate != ""
	if (hasReleaseVersion || hasReleaseDate) && len(sourcesToProcess) > 1 {
		return fmt.Errorf(
			"release/version overrides can only be used with a single source (found %d sources)",
			len(sourcesToProcess),
		)
	}

	// Pre 4. Prepare cache directory (T034)
	cacheDir, err := prepareCacheDir()
	if err != nil {
		return fmt.Errorf("failed to prepare cache directory: %w", err)
	}
	slog.Info("Cache directory prepared", "path", cacheDir)

	// Pre 5: Process each source
	successCount := 0
	errorCount := 0

	for i, source := range sourcesToProcess {
		sourceStartTime := time.Now()

		fmt.Println() // Blank line between sources
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println(gnlib.FormatMessage(
			fmt.Sprintf("<title>Data Source [%d]: %s</title>", source.ID, source.TitleShort),
			nil,
		))
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
			return fmt.Errorf("population cancelled: %w", ctx.Err())
		default:
		}

		// Process this source through all phases
		err := p.processSource(ctx, cfg, source, cacheDir)
		if err != nil {
			errorCount++
			slog.Error("Failed to process source",
				"data_source_id", source.ID,
				"title", source.TitleShort,
				"error", err,
			)
			// Continue with next source instead of failing entire operation
			continue
		}

		successCount++
		sourceDuration := time.Since(sourceStartTime)
		slog.Debug("Source processed successfully",
			"data_source_id", source.ID,
			"title", source.TitleShort,
			"duration", sourceDuration.Round(time.Second),
		)

		// Print summary for this source
		msg := fmt.Sprintf("<em>Completed in %s</em>", sourceDuration.Round(time.Second).String())
		fmt.Println(gnlib.FormatMessage(msg, nil))
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

// sortFilesByDate sorts filenames in-place by extracted date (oldest first, newest last).
// Files without dates are placed at the beginning.
func sortFilesByDate(filenames []string) {
	type fileWithDate struct {
		name string
		date string
	}

	// Extract dates from filenames
	datePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	filesWithDates := make([]fileWithDate, len(filenames))

	for i, filename := range filenames {
		matches := datePattern.FindStringSubmatch(filename)
		date := ""
		if len(matches) > 1 {
			date = matches[1]
		}
		filesWithDates[i] = fileWithDate{name: filename, date: date}
	}

	// Sort by date (empty dates first, then chronological)
	sort.Slice(filesWithDates, func(i, j int) bool {
		if filesWithDates[i].date == "" && filesWithDates[j].date != "" {
			return true // Files without dates go first
		}
		if filesWithDates[i].date != "" && filesWithDates[j].date == "" {
			return false // Files with dates go after
		}
		return filesWithDates[i].date < filesWithDates[j].date // Chronological order
	})

	// Update original slice
	for i, f := range filesWithDates {
		filenames[i] = f.name
	}
}

// handleMultipleFilesWarning displays a user-friendly warning when multiple SFGA files
// are found for a single data source ID and prompts for confirmation.
// The warning string format: "found N files matching ID X at/in <location>: [file1, file2, ...] - selected latest: <file>"
func handleMultipleFilesWarning(sourceID int, warning string) error {
	// Parse the warning to extract key information
	// Format: "found N files matching ID X at <url>: [files] - selected latest: <selected>"
	var allFiles, selectedFile string

	// Extract files list and selected file
	if idx := strings.Index(warning, ": ["); idx != -1 {
		rest := warning[idx+3:]
		if endIdx := strings.Index(rest, "] - selected latest: "); endIdx != -1 {
			allFiles = rest[:endIdx]
			selectedFile = rest[endIdx+len("] - selected latest: "):]
		}
	}

	// Display formatted warning
	fmt.Println()
	fmt.Println(gnlib.FormatMessage("<warn>Multiple Files Found</warn>", nil))
	fmt.Println(gnlib.FormatMessage(
		"Data Source ID <em>%d</em> has multiple SFGA files available:",
		[]any{sourceID},
	))
	fmt.Println()

	// Show available files (limit to most recent 3 to avoid clutter)
	if allFiles != "" {
		files := strings.Split(allFiles, " ")
		var validFiles []string
		for _, file := range files {
			file = strings.TrimSpace(file)
			if file != "" {
				validFiles = append(validFiles, file)
			}
		}

		// Sort files by date (oldest first, newest last for CLI convention)
		sortFilesByDate(validFiles)

		const maxFilesToShow = 3
		totalFiles := len(validFiles)

		// Show most recent files (take from end since newest is last)
		filesToShow := validFiles
		if len(validFiles) > maxFilesToShow {
			// Take last N files (most recent)
			filesToShow = validFiles[len(validFiles)-maxFilesToShow:]
			fmt.Printf("  (Showing %d most recent of %d files)\n", maxFilesToShow, totalFiles)
		}

		for _, file := range filesToShow {
			fmt.Printf("  • %s\n", file)
		}
		fmt.Println()
	}

	// Show selected file
	if selectedFile != "" {
		fmt.Println(gnlib.FormatMessage(
			"Auto-selected (latest): <em>%s</em>",
			[]any{selectedFile},
		))
	}
	fmt.Println()

	// Prompt for confirmation (defaults to yes)
	response, err := promptUser("Continue with this file? [Y/n]: ")
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	if response == "no" {
		return fmt.Errorf("import cancelled by user")
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
	// Phase 0: Clear cache and fetch SFGA (T037)
	// Clear cache before each source to avoid "too many database files" error
	if err := clearCache(cacheDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	slog.Info("Step 1/6: Fetching SFGA", "data_source_id", source.ID)

	// First, resolve the SFGA file path (without downloading)
	sfgaPath, sfgaMetadata, warning, err := resolveSFGAPath(source)
	if err != nil {
		return fmt.Errorf("failed to resolve SFGA path: %w", err)
	}

	// If multiple files found, prompt user before downloading
	if warning != "" {
		if err := handleMultipleFilesWarning(source.ID, warning); err != nil {
			return err
		}
	}

	// Now download and extract the file
	sqlitePath, err := fetchSFGA(ctx, sfgaPath, cacheDir)
	if err != nil {
		return fmt.Errorf("failed to fetch SFGA: %w", err)
	}

	// Open SFGA database (T037)
	sfgaDB, err := openSFGA(sqlitePath)
	if err != nil {
		return fmt.Errorf("failed to open SFGA database: %w", err)
	}
	defer sfgaDB.Close()

	// Process name strings (T039)
	slog.Info("Step 2/6: Processing name strings", "data_source_id", source.ID)
	err = processNameStrings(ctx, p, sfgaDB, source.ID)
	if err != nil {
		return fmt.Errorf("step 2 failed (name strings): %w", err)
	}

	// Build hierarchy for classification (T041)
	slog.Info("Step 3/6: Building taxa hierarchy", "data_source_id", source.ID)
	hierarchy, err := buildHierarchy(ctx, sfgaDB, cfg.JobsNumber)

	// badNodes are package accessible from hierarchy.go
	for k, v := range badNodes {
		switch v {
		case circularBadNode:
			slog.Warn("Circular reference detected in hierarchy", "id", k)
		case missingBadNode:
			slog.Warn("Hierarchy node not found, making short breadcrumbs", "id", k)
		}
	}
	if err != nil {
		return fmt.Errorf("Step 3 failed (hierarchy building): %w", err)
	}

	// Process name indices with classification (T043)
	slog.Info("Step 4/6: Processing name indices", "data_source_id", source.ID)
	err = processNameIndices(ctx, p, sfgaDB, &source, hierarchy, cfg)
	if err != nil {
		return fmt.Errorf("step 4 failed (name indices): %w", err)
	}

	// Process vernaculars (T045)
	slog.Info("Step 5/6: Processing vernaculars", "data_source_id", source.ID)
	err = processVernaculars(ctx, p, sfgaDB, source.ID)
	if err != nil {
		return fmt.Errorf("step 5 failed (vernaculars): %w", err)
	}

	// Phase 5: Update data source metadata (T047)
	slog.Info("Step 6/6: Updating data source metadata", "data_source_id", source.ID)
	err = updateDataSourceMetadata(ctx, p, source, sfgaDB, sfgaMetadata)
	if err != nil {
		return fmt.Errorf("Step 6 failed (metadata): %w", err)
	}

	return nil
}
