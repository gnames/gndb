package ioexport

import (
	_ "embed"
	"log/slog"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data/rankdict.yaml
var rankdictYAML []byte

// rankInfo stores rank and nomenclatural code for a higher taxon.
type rankInfo struct {
	Rank string `yaml:"rank"`
	Code string `yaml:"code"`
}

// rankDict maps canonical taxon name → rank + code.
// Loaded once at package init from the embedded rankdict.yaml.
var rankDict map[string]rankInfo

func init() {
	rankDict = make(map[string]rankInfo)
	if err := yaml.Unmarshal(rankdictYAML, rankDict); err != nil {
		slog.Warn("Failed to load rank dictionary", "error", err)
	}
}

// taxonRanks is the set of ranks that have corresponding flat fields in
// coldp.Taxon. Only these ranks are meaningful during export; all others
// are silently skipped.
var taxonRanks = map[string]bool{
	"kingdom":     true,
	"phylum":      true,
	"subphylum":   true,
	"class":       true,
	"subclass":    true,
	"order":       true,
	"suborder":    true,
	"superfamily": true,
	"family":      true,
	"subfamily":   true,
	"tribe":       true,
	"subtribe":    true,
	"genus":       true,
	"subgenus":    true,
	"species":     true,
}

// isTaxonRank returns true when r corresponds to a coldp.Taxon flat field.
func isTaxonRank(r string) bool {
	return taxonRanks[strings.ToLower(r)]
}

// inferRank is a last-resort fallback used only when classification_ranks
// is empty (Case 3). It attempts to determine the rank of a single
// breadcrumb name. Returns an empty string when the rank cannot be
// determined or does not correspond to a coldp.Taxon flat field.
//
// Priority:
//  1. Dictionary lookup (order and above — covers Animalia/Plantae)
//  2. Suffix rules (family and below, code-aware via codeID)
func inferRank(name string, codeID int) string {
	// 1. Dictionary lookup
	if info, ok := rankDict[name]; ok && isTaxonRank(info.Rank) {
		return info.Rank
	}

	// 2. Suffix rules
	rank := inferRankBySuffix(name, codeID)
	if isTaxonRank(rank) {
		return rank
	}
	return ""
}

// inferRankBySuffix returns a rank string based on nomenclatural suffixes.
// codeID: 1 = ICZN (zoology), 2 = ICN (botany); 0 = unknown.
// Returns empty string when no suffix matches.
func inferRankBySuffix(name string, codeID int) string {
	lower := strings.ToLower(name)

	// Unambiguous botanical suffixes
	switch {
	case strings.HasSuffix(lower, "aceae"):
		return "family"
	case strings.HasSuffix(lower, "oideae"):
		return "subfamily"
	case strings.HasSuffix(lower, "eae"):
		return "tribe"
	case strings.HasSuffix(lower, "ales"):
		return "order"
	case strings.HasSuffix(lower, "ineae"):
		return "suborder"
	}

	// Unambiguous zoological suffixes
	switch {
	case strings.HasSuffix(lower, "idae"):
		return "family"
	case strings.HasSuffix(lower, "ini"):
		return "tribe"
	case strings.HasSuffix(lower, "oidea"):
		return "superfamily"
	}

	// Botanical-only class/phylum suffixes
	switch {
	case strings.HasSuffix(lower, "opsida"),
		strings.HasSuffix(lower, "mycetes"),
		strings.HasSuffix(lower, "phyceae"):
		return "class"
	case strings.HasSuffix(lower, "phyta"),
		strings.HasSuffix(lower, "mycota"):
		return "phylum"
	}

	// Ambiguous: -inae means subfamily (ICZN) or subtribe (ICN)
	if strings.HasSuffix(lower, "inae") {
		if codeID == 2 { // ICN (botany)
			return "subtribe"
		}
		return "subfamily" // ICZN or unknown — more common globally
	}

	// -ina means subtribe in ICZN
	if strings.HasSuffix(lower, "ina") && codeID == 1 {
		return "subtribe"
	}

	return ""
}

// inferCodeFromBreadcrumb performs a majority-vote on nomenclatural code
// across all names in a pipe-delimited classification string.
// Returns 1 (ICZN), 2 (ICN), or 0 (unknown).
func inferCodeFromBreadcrumb(classification string) int {
	names := strings.Split(classification, "|")
	iczn, icn := 0, 0

	for _, name := range names {
		if info, ok := rankDict[name]; ok {
			switch info.Code {
			case "iczn":
				iczn++
			case "icn":
				icn++
			}
			continue
		}
		lower := strings.ToLower(name)
		switch {
		case strings.HasSuffix(lower, "idae"),
			strings.HasSuffix(lower, "ini"):
			iczn++
		case strings.HasSuffix(lower, "aceae"),
			strings.HasSuffix(lower, "oideae"),
			strings.HasSuffix(lower, "eae"):
			icn++
		}
	}

	switch {
	case iczn > icn:
		return 1
	case icn > iczn:
		return 2
	default:
		return 0
	}
}
