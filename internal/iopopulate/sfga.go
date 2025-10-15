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
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/sfborg/sflib"
)

// resolveSFGAFile finds the SFGA file in the parent directory matching the given ID.
// Matches pattern: {4-digit-ID}* with SFGA extensions (.sql, .sql.zip, .sqlite, .sqlite.zip)
// If multiple files match, selects the one with the latest date.
// Returns (filePath, warningMessage, error). Warning is non-empty when multiple files found.
func resolveSFGAFile(parentDir string, id int) (string, string, error) {
	// Format ID with leading zeros (4 digits)
	idPattern := fmt.Sprintf("%04d", id)

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
		// Match files starting with the ID pattern with SFGA extensions
		if strings.HasPrefix(filename, idPattern) && isSFGAFile(filename) {
			matches = append(matches, filename)
		}
	}

	// Handle no matches
	if len(matches) == 0 {
		return "", "", fmt.Errorf("no files found matching ID %d (pattern %s*) in %s", id, idPattern, parentDir)
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
// Matches pattern: {4-digit-ID}* with SFGA extensions
// If multiple files match, selects the one with the latest date.
// Returns (fullURL, warningMessage, error). Warning is non-empty when multiple files found.
func resolveRemoteSFGAFile(baseURL string, id int) (string, string, error) {
	// Format ID with leading zeros (4 digits)
	idPattern := fmt.Sprintf("%04d", id)

	// Fetch directory listing
	resp, err := http.Get(baseURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch directory listing from %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to fetch directory listing from %s: status %d", baseURL, resp.StatusCode)
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

		// Match files starting with the ID pattern with SFGA extensions
		if strings.HasPrefix(filename, idPattern) && isSFGAFile(filename) {
			matches = append(matches, filename)
		}
	}

	// Handle no matches
	if len(matches) == 0 {
		return "", "", fmt.Errorf("no files found matching ID %d (pattern %s*) at %s", id, idPattern, baseURL)
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
		if f.date != "" && (best.date == "" || f.date > best.date || (f.date == best.date && f.priority > best.priority)) {
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

// fetchSFGA fetches and extracts an SFGA file to the cache directory.
// For local files, it resolves by ID pattern and uses sflib Archive.Fetch.
// For URLs, it lists the remote directory to find the matching file by ID.
// Returns (sqlitePath, warningMessage, error). Warning is non-empty when multiple files found.
func fetchSFGA(ctx context.Context, source populate.DataSourceConfig, cacheDir string) (string, string, error) {
	var sfgaPath string
	var warning string
	var err error

	// Determine if parent is URL or local directory
	isURL := populate.IsValidURL(source.Parent)

	if isURL {
		// For URLs, list the directory and find the file matching the ID
		sfgaPath, warning, err = resolveRemoteSFGAFile(source.Parent, source.ID)
		if err != nil {
			return "", "", err
		}
	} else {
		// For local directories, resolve the exact filename
		sfgaPath, warning, err = resolveSFGAFile(source.Parent, source.ID)
		if err != nil {
			return "", "", err
		}
	}

	// Create Archive for fetching
	arc := sflib.NewSfga()

	// Fetch and extract to cache directory
	err = arc.Fetch(sfgaPath, cacheDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch SFGA from %s: %w", sfgaPath, err)
	}

	// Get the path to the extracted SQLite file
	sqlitePath := arc.DbPath()
	if sqlitePath == "" {
		return "", "", fmt.Errorf("failed to get database path after fetching %s", sfgaPath)
	}

	return sqlitePath, warning, nil
}

// openSFGA opens a SQLite database and returns a database handle.
func openSFGA(sqlitePath string) (*sql.DB, error) {
	// Check if file exists
	if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("SFGA file does not exist: %s", sqlitePath)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", sqlitePath)
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
