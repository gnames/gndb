package iopopulate

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gnames/gndb/pkg/populate"
	"github.com/sfborg/sflib"
	_ "modernc.org/sqlite" // Pure Go SQLite driver (no CGo)
)

// resolveSFGAFile finds the SFGA file in the parent directory matching the given ID.
// Matches patterns: {ID}-, {ID}_, or {ID}.ext with varying digit lengths (0001, 001, 01, 1)
// SFGA extensions: .sql, .sql.zip, .sqlite, .sqlite.zip
// If multiple files match, selects the one with the latest date.
// Returns (filePath, warningMessage, error). Warning is non-empty when multiple files found.
func resolveSFGAFile(parentDir string, id int) (string, string, error) {
	// Generate ID patterns to try: 0001, 001, 01, 1 (descending order)
	idPatterns := generateIDPatterns(id)

	// Read directory
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to read parent directory %s: %w", parentDir, err)
	}

	// Find matching files
	var matches []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		// Check if this file matches any of our ID patterns
		if matchesIDPattern(filename, idPatterns) && isSFGAFile(filename) {
			matches = append(matches, filename)
		}
	}

	// Handle no matches
	if len(matches) == 0 {
		return "", "", fmt.Errorf(
			"no files found matching ID %d (patterns: %v) in %s",
			id,
			idPatterns,
			parentDir,
		)
	}

	// Handle single match
	if len(matches) == 1 {
		return filepath.Join(parentDir, matches[0]), "", nil
	}

	// Handle multiple matches - select latest by date
	selected := selectLatestFile(matches)
	warning := fmt.Sprintf("found %d files matching ID %d in %s: %v - selected latest: %s",
		len(matches), id, parentDir, matches, selected)

	return filepath.Join(parentDir, selected), warning, nil
}

// resolveRemoteSFGAFile finds the SFGA file at a remote URL by listing the directory.
// Matches patterns: {ID}-, {ID}_, or {ID}.ext with varying digit lengths (0001, 001, 01, 1)
// If multiple files match, selects the one with the latest date.
// Returns (fullURL, warningMessage, error). Warning is non-empty when multiple files found.
func resolveRemoteSFGAFile(baseURL string, id int) (string, string, error) {
	// Generate ID patterns to try: 0001, 001, 01, 1 (descending order)
	idPatterns := generateIDPatterns(id)

	// Fetch directory listing
	resp, err := http.Get(baseURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch directory listing from %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf(
			"failed to fetch directory listing from %s: status %d",
			baseURL,
			resp.StatusCode,
		)
	}

	// Read HTML content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read directory listing from %s: %w", baseURL, err)
	}

	// Parse HTML for href links (simple pattern matching for Apache/nginx directory listings)
	// Pattern matches: href="0196_something.sqlite.zip" or href='0196_something.sql'
	hrefPattern := regexp.MustCompile(`href=["']([^"']+)["']`)
	hrefs := hrefPattern.FindAllStringSubmatch(string(body), -1)

	// Find matching files
	var matches []string
	for _, match := range hrefs {
		if len(match) < 2 {
			continue
		}
		filename := match[1]

		// Skip parent directory links and non-files
		if filename == "../" || strings.HasSuffix(filename, "/") {
			continue
		}

		// Check if this file matches any of our ID patterns
		if matchesIDPattern(filename, idPatterns) && isSFGAFile(filename) {
			matches = append(matches, filename)
		}
	}

	// Handle no matches
	if len(matches) == 0 {
		return "", "", fmt.Errorf(
			"no files found matching ID %d (patterns: %v) at %s",
			id,
			idPatterns,
			baseURL,
		)
	}

	// Handle single match
	if len(matches) == 1 {
		fullURL := strings.TrimSuffix(baseURL, "/") + "/" + matches[0]
		return fullURL, "", nil
	}

	// Handle multiple matches - select latest by date
	selected := selectLatestFile(matches)
	warning := fmt.Sprintf("found %d files matching ID %d at %s: %v - selected latest: %s",
		len(matches), id, baseURL, matches, selected)
	fullURL := strings.TrimSuffix(baseURL, "/") + "/" + selected

	return fullURL, warning, nil
}

// selectLatestFile selects the file with the latest date from a list of filenames.
// Extracts dates in YYYY-MM-DD format from filenames and picks the latest.
// When dates are equal, prioritizes by file type: sqlite.zip > sql.zip > sqlite > sql
// Rationale: SQLite is binary (faster processing), zip files are smaller (better for slow connections).
func selectLatestFile(filenames []string) string {
	if len(filenames) == 0 {
		return ""
	}
	if len(filenames) == 1 {
		return filenames[0]
	}

	// Pattern to extract date: YYYY-MM-DD
	datePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)

	type fileWithMetadata struct {
		filename string
		date     string
		priority int // Higher is better
	}

	var filesWithMetadata []fileWithMetadata
	for _, filename := range filenames {
		matches := datePattern.FindStringSubmatch(filename)
		date := ""
		if len(matches) > 1 {
			date = matches[1]
		}

		// Assign priority based on file extension
		// sqlite.zip (4) > sql.zip (3) > sqlite (2) > sql (1)
		priority := getFileTypePriority(filename)

		filesWithMetadata = append(filesWithMetadata, fileWithMetadata{
			filename: filename,
			date:     date,
			priority: priority,
		})
	}

	// Find the best file (latest date, highest priority on tie)
	best := filesWithMetadata[0]
	for _, f := range filesWithMetadata[1:] {
		// If current file has a date and (best has no date OR current date is later OR same date but higher priority)
		if f.date != "" &&
			(best.date == "" || f.date > best.date || (f.date == best.date && f.priority > best.priority)) {
			best = f
		} else if f.date == "" && best.date == "" && f.priority > best.priority {
			// Both have no date, use priority
			best = f
		}
	}

	return best.filename
}

// getFileTypePriority returns priority for file type selection.
// Higher values are preferred: sqlite.zip (4) > sql.zip (3) > sqlite (2) > sql (1).
func getFileTypePriority(filename string) int {
	if strings.HasSuffix(filename, ".sqlite.zip") {
		return 4
	}
	if strings.HasSuffix(filename, ".sql.zip") {
		return 3
	}
	if strings.HasSuffix(filename, ".sqlite") {
		return 2
	}
	if strings.HasSuffix(filename, ".sql") {
		return 1
	}
	return 0 // Unknown extension
}

// isSFGAFile checks if a filename has a valid SFGA extension.
// Valid extensions: .sql, .sql.zip, .sqlite, .sqlite.zip
func isSFGAFile(filename string) bool {
	return strings.HasSuffix(filename, ".sql") ||
		strings.HasSuffix(filename, ".sql.zip") ||
		strings.HasSuffix(filename, ".sqlite") ||
		strings.HasSuffix(filename, ".sqlite.zip")
}

// generateIDPatterns creates ID patterns with varying zero-padding lengths.
// For ID=1: returns ["0001", "001", "01", "1"]
// For ID=42: returns ["0042", "042", "42"]
// For ID=196: returns ["0196", "196"]
// For ID=1234: returns ["1234"]
// Patterns are ordered by specificity (most zeros first) to prefer standardized naming.
func generateIDPatterns(id int) []string {
	var patterns []string

	// Start with 4-digit format (standard)
	patterns = append(patterns, fmt.Sprintf("%04d", id))

	// Add 3-digit format if ID < 1000
	if id < 1000 {
		patterns = append(patterns, fmt.Sprintf("%03d", id))
	}

	// Add 2-digit format if ID < 100
	if id < 100 {
		patterns = append(patterns, fmt.Sprintf("%02d", id))
	}

	// Add 1-digit format if ID < 10
	if id < 10 {
		patterns = append(patterns, fmt.Sprintf("%d", id))
	}

	return patterns
}

// matchesIDPattern checks if a filename matches any of the given ID patterns.
// A match occurs when the filename starts with: {pattern}- or {pattern}_ or {pattern}.
// This ensures we match "001-file.sql" or "001_file.sql" or "001.sql" but not "1001.sql" for ID=1.
func matchesIDPattern(filename string, patterns []string) bool {
	for _, pattern := range patterns {
		// Check if filename starts with pattern followed by separator or extension
		if strings.HasPrefix(filename, pattern+"-") ||
			strings.HasPrefix(filename, pattern+"_") ||
			strings.HasPrefix(filename, pattern+".") {
			return true
		}
	}
	return false
}

// SFGAMetadata holds metadata extracted from SFGA filename.
type SFGAMetadata struct {
	Filename     string // Full filename
	Version      string // Version string (e.g., "v1.0.0" or "1.0.0")
	RevisionDate string // Revision date in YYYY-MM-DD format
}

// parseSFGAFilename extracts version and revision date from SFGA filename.
// Expected patterns:
//   - {ID}_{name}_{date}_v{version}.{ext}  (e.g., "1000_ruhoff_2023-08-22_v1.0.0.sqlite.zip")
//   - {ID}-{name}-{date}.{ext}              (e.g., "0147-vascan-2025-08-25.sqlite.zip")
//   - {ID}.{ext}                             (e.g., "1000.sql" - no metadata)
//   - {ID}_{name}.{ext}                      (e.g., "1000_ruhoff.sqlite")
//
// Returns SFGAMetadata with filename, version (if found), and revision date (if found).
// Empty strings are returned for missing fields - this is graceful and allows for minimal filenames.
func parseSFGAFilename(filename string) SFGAMetadata {
	metadata := SFGAMetadata{
		Filename: filename,
	}

	// Extract date: YYYY-MM-DD format
	// Only matches valid calendar dates (not IDs like 1000-2000-3000)
	datePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	if dateMatch := datePattern.FindStringSubmatch(filename); len(dateMatch) > 1 {
		metadata.RevisionDate = dateMatch[1]
	}

	// Extract version: v1.0.0 or v1.0 or 1.0.0 format
	// Must be preceded by underscore, dash, or 'v' to avoid matching IDs
	// Examples: _v1.0.0, -v2.3, _1.5.2, v3.0
	versionPattern := regexp.MustCompile(`[_-]v?(\d+\.\d+(?:\.\d+)?)`)
	if versionMatch := versionPattern.FindStringSubmatch(filename); len(versionMatch) > 1 {
		metadata.Version = versionMatch[1]
	}

	return metadata
}

// fetchSFGA fetches and extracts an SFGA file to the cache directory.
// For local files, it resolves by ID pattern and uses sflib Archive.Fetch.
// For URLs, it lists the remote directory to find the matching file by ID.
// Returns (sqlitePath, metadata, warningMessage, error). Warning is non-empty when multiple files found.
// resolveSFGAPath determines the SFGA file path without downloading.
// Returns (sfgaPath, metadata, warning, error).
func resolveSFGAPath(
	source populate.DataSourceConfig,
) (string, SFGAMetadata, string, error) {
	var sfgaPath string
	var warning string
	var err error

	// Determine if parent is URL or local directory
	isURL := populate.IsValidURL(source.Parent)

	if isURL {
		// For URLs, list the directory and find the file matching the ID
		sfgaPath, warning, err = resolveRemoteSFGAFile(source.Parent, source.ID)
		if err != nil {
			return "", SFGAMetadata{}, "", err
		}
	} else {
		// For local directories, resolve the exact filename
		sfgaPath, warning, err = resolveSFGAFile(source.Parent, source.ID)
		if err != nil {
			return "", SFGAMetadata{}, "", err
		}
	}

	// Extract filename from path
	filename := filepath.Base(sfgaPath)

	// Parse metadata from filename
	metadata := parseSFGAFilename(filename)

	return sfgaPath, metadata, warning, nil
}

func fetchSFGA(
	_ context.Context,
	sfgaPath string,
	cacheDir string,
) (string, error) {
	// Create Archive for fetching
	arc := sflib.NewSfga()

	// Fetch and extract to cache directory
	err := arc.Fetch(sfgaPath, cacheDir)
	if err != nil {
		return "", fmt.Errorf("failed to fetch SFGA from %s: %w", sfgaPath, err)
	}

	// Get the path to the extracted SQLite file
	sqlitePath := arc.DbPath()
	if sqlitePath == "" {
		return "", fmt.Errorf(
			"failed to get database path after fetching %s",
			sfgaPath,
		)
	}

	return sqlitePath, nil
}

// openSFGA opens a SQLite database and returns a database handle.
func openSFGA(sqlitePath string) (*sql.DB, error) {
	// Check if file exists
	if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("SFGA file does not exist: %s", sqlitePath)
	}

	// Open SQLite database (using modernc.org/sqlite pure Go driver)
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database %s: %w", sqlitePath, err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to SQLite database %s: %w", sqlitePath, err)
	}

	return db, nil
}
