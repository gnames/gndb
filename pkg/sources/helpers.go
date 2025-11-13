package sources

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// parseFilename extracts metadata from SFGA filename.
// Expected format: {id}_{name}_{date}_{version}.(sql|sqlite)[.zip]
// Examples:
//   - 0001_col_2025-10-03_v2024.1.sqlite.zip  → ID=1, Date=2025-10-03, Version=v2024.1
//   - 0002_gbif_2024-12-15_2024-12-15.sql.zip → ID=2, Date=2024-12-15, Version=2024-12-15
//   - 0003_worms_2025-01-01.sqlite            → ID=3, Date=2025-01-01, Version=""
//   - 1001.sql                                 → ID=1001, Date="", Version=""
//
// Version extraction rules:
//   - If release date is last segment before extension: no version
//   - If underscore + text after date: everything until .sql|.sqlite is version
func parseFilename(path string) FileMetadata {
	var metadata FileMetadata

	// Get filename without directory
	filename := filepath.Base(path)

	// Strip .zip extension if present
	filename = strings.TrimSuffix(filename, ".zip")

	// Extract ID (first 4 digits)
	idPattern := regexp.MustCompile(`^(\d{4})`)
	if matches := idPattern.FindStringSubmatch(filename); len(matches) > 1 {
		if id, err := strconv.Atoi(matches[1]); err == nil {
			metadata.ID = id
		}
	}

	// Extract release date (YYYY-MM-DD pattern)
	datePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	if matches := datePattern.FindStringSubmatch(filename); len(matches) > 1 {
		metadata.ReleaseDate = matches[1]
	}

	// Extract version (everything after last underscore until .sql or .sqlite)
	// Only if there's content after the date
	if metadata.ReleaseDate != "" {
		// Find position of release date in filename
		dateIdx := strings.Index(filename, metadata.ReleaseDate)
		if dateIdx != -1 {
			// Get substring after the date
			afterDate := filename[dateIdx+len(metadata.ReleaseDate):]

			// Check if there's an underscore followed by content before extension
			versionPattern := regexp.MustCompile(`^_(.+?)\.(?:sql|sqlite)$`)
			if matches := versionPattern.FindStringSubmatch(afterDate); len(matches) > 1 {
				metadata.Version = matches[1]
			}
		}
	}

	return metadata
}

// IsValidURL checks if a string is a valid URL.
func IsValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

// filterSources filters data sources based on the filter string.
// Returns filtered sources, warnings (for user display), and error (for fatal issues).
// Supported filters:
//   - "main": Returns sources with ID < 1000 (official sources)
//   - "exclude main": Returns sources with ID >= 1000 (custom sources)
//   - "1,3,5": Returns sources with specified IDs (comma-separated)
//   - "180-208": Returns sources with IDs in range [180, 208] (inclusive)
//   - "-10": Returns sources with IDs from 1 to 10 (inclusive)
//   - "197-": Returns sources with IDs from 197 to end (inclusive)
//   - "1,5,10-20,50-": Mix of individual IDs and ranges
//   - "": Returns all sources (no filtering)
func filterSources(
	sources []DataSourceConfig,
	filter string,
) ([]DataSourceConfig, []string, error) {
	filter = strings.TrimSpace(filter)

	// No filter - return all sources
	if filter == "" {
		return sources, nil, nil
	}

	// Handle "main" filter (ID < 1000)
	if filter == "main" {
		var filtered []DataSourceConfig
		for _, src := range sources {
			if src.ID < 1000 {
				filtered = append(filtered, src)
			}
		}
		return filtered, nil, nil
	}

	// Handle "exclude main" filter (ID >= 1000)
	if filter == "exclude main" {
		var filtered []DataSourceConfig
		for _, src := range sources {
			if src.ID >= 1000 {
				filtered = append(filtered, src)
			}
		}
		return filtered, nil, nil
	}

	// Parse comma-separated items (can be individual IDs or ranges)
	items := strings.Split(filter, ",")
	requestedIDs := make(map[int]bool) // All requested IDs
	explicitIDs := make(map[int]bool)  // Only explicitly specified IDs (not from ranges)
	var warnings []string

	for _, item := range items {
		item = strings.TrimSpace(item)

		// Check if this is a range (contains "-")
		if strings.Contains(item, "-") {
			start, end, err := parseRange(item, sources)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse range '%s': %w", item, err)
			}

			// Add all IDs in range (will silently skip non-existent ones)
			rangeHasMatches := false
			for id := start; id <= end; id++ {
				requestedIDs[id] = true
				// Check if this ID actually exists
				if sourceExists(sources, id) {
					rangeHasMatches = true
				}
			}

			// Warn if range matched no sources
			if !rangeHasMatches {
				warnings = append(warnings, fmt.Sprintf("range '%s' matched no sources", item))
			}
		} else {
			// Single explicit ID
			id, err := strconv.Atoi(item)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid source ID '%s': must be a number or range", item)
			}
			requestedIDs[id] = true
			explicitIDs[id] = true
		}
	}

	// Collect matching sources and check for missing explicit IDs
	var filtered []DataSourceConfig
	foundIDs := make(map[int]bool)

	for _, src := range sources {
		if requestedIDs[src.ID] {
			filtered = append(filtered, src)
			foundIDs[src.ID] = true
		}
	}

	// Warn about explicitly requested IDs that weren't found
	for id := range explicitIDs {
		if !foundIDs[id] {
			warnings = append(warnings, fmt.Sprintf("source ID %d not found in configuration", id))
		}
	}

	// Return error if no sources matched at all
	if len(filtered) == 0 {
		if len(warnings) > 0 {
			return nil, warnings, fmt.Errorf(
				"no sources matched filter '%s': %s",
				filter,
				strings.Join(warnings, "; "),
			)
		}
		return nil, nil, fmt.Errorf("no sources matched filter '%s'", filter)
	}

	// Return filtered sources and warnings for caller to handle
	return filtered, warnings, nil
}

// parseRange parses a range string like "180-208", "-10", or "197-"
// Returns (start, end, error)
func parseRange(rangeStr string, sources []DataSourceConfig) (int, int, error) {
	parts := strings.Split(rangeStr, "-")

	// Handle different range formats
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format: expected 'X-Y', '-Y', or 'X-'")
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	var start, end int
	var err error

	// Handle "-10" (from 1 to 10)
	if startStr == "" {
		start = 1
		end, err = strconv.Atoi(endStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end value: %w", err)
		}
	} else if endStr == "" {
		// Handle "197-" (from 197 to max)
		start, err = strconv.Atoi(startStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start value: %w", err)
		}
		// Find maximum ID in sources
		end = findMaxSourceID(sources)
		if end == 0 {
			return 0, 0, fmt.Errorf("no sources available to determine end of range")
		}
	} else {
		// Handle "180-208" (from 180 to 208)
		start, err = strconv.Atoi(startStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start value: %w", err)
		}
		end, err = strconv.Atoi(endStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end value: %w", err)
		}
	}

	// Validate range
	if start > end {
		return 0, 0, fmt.Errorf("start (%d) must be <= end (%d)", start, end)
	}

	return start, end, nil
}

// findMaxSourceID returns the maximum ID among all sources
func findMaxSourceID(sources []DataSourceConfig) int {
	maxID := 0
	for _, src := range sources {
		if src.ID > maxID {
			maxID = src.ID
		}
	}
	return maxID
}

// sourceExists checks if a source with the given ID exists
func sourceExists(sources []DataSourceConfig, id int) bool {
	for _, src := range sources {
		if src.ID == id {
			return true
		}
	}
	return false
}

// ExtractOutlinkID extracts the outlink ID from a column value.
// If the column name ends with "col__alternative_id", it extracts the value
// after "gnoutlink:" from a comma-separated list of namespace:value pairs.
// Otherwise, it returns the value as-is.
//
// Examples:
//   - ExtractOutlinkID("taxon.col__id", "12345") → "12345"
//   - ExtractOutlinkID("taxon.col__alternative_id", "wikidata:Q123,gnoutlink:Homo_sapiens") → "Homo_sapiens"
//   - ExtractOutlinkID("name.col__alternative_id", "gnoutlink:url-encoded-name") → "url-encoded-name"
//   - ExtractOutlinkID("taxon.col__alternative_id", "wikidata:Q123") → "" (no gnoutlink namespace)
func ExtractOutlinkID(columnName, value string) string {
	// If column is not col__alternative_id, return value as-is
	if !strings.HasSuffix(columnName, "col__alternative_id") {
		return value
	}

	// Extract gnoutlink: namespace from comma-separated list
	for part := range strings.SplitSeq(value, ",") {
		var ok bool
		part = strings.TrimSpace(part)
		part, ok = strings.CutPrefix(part, "gnoutlink:")
		if ok {
			return part
		}
	}

	// gnoutlink namespace not found
	return ""
}
