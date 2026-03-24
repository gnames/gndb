// gen_rankdict queries the gnames PostgreSQL database and writes a
// rank dictionary YAML file for use during gndb export.
//
// The dictionary maps higher taxon names (order and above) for Animalia
// and Plantae to their rank and nomenclatural code (iczn/icn). It is
// used at export time to infer ranks for breadcrumb nodes that lack
// rank information in the database.
//
// The output file should be committed to:
//
//	internal/ioexport/data/rankdict.yaml
//
// Connection settings are read from GNDB_DATABASE_* environment variables
// (set via .envrc / direnv). The --output flag overrides the output path.
//
// Usage:
//
//	go run tools/gen_rankdict/main.go [--output path/to/rankdict.yaml]
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type rankEntry struct {
	Name string
	Rank string
	Code string // "iczn" or "icn"
}

// rankOrder defines the display order for ranks in the output file.
var rankOrder = []string{
	"kingdom",
	"subkingdom",
	"phylum",
	"subphylum",
	"class",
	"subclass",
	"order",
	"suborder",
}

const query = `
SELECT DISTINCT ON (c.name)
    c.name          AS taxon_name,
    lower(nsi.rank) AS rank,
    CASE
        WHEN nsi.classification LIKE 'Plantae%'
          OR c.name = 'Plantae'  THEN 'icn'
        WHEN nsi.classification LIKE 'Animalia%'
          OR c.name = 'Animalia' THEN 'iczn'
        ELSE 'unknown'
    END             AS code
FROM name_string_indices nsi
JOIN name_strings ns ON ns.id = nsi.name_string_id
JOIN canonicals    c  ON c.id  = ns.canonical_id
WHERE nsi.data_source_id IN (1, 3, 9)
  AND lower(nsi.rank) IN (
      'kingdom', 'subkingdom',
      'phylum',  'subphylum',
      'class',   'subclass',
      'order',   'suborder'
  )
  AND (
      nsi.classification LIKE 'Animalia%'
   OR nsi.classification LIKE 'Plantae%'
   OR c.name IN ('Animalia', 'Plantae')
  )
  AND nsi.taxonomic_status NOT IN ('synonym', 'ambiguous synonym', 'misapplied')
  AND c.name != ''
  AND c.name NOT LIKE '% %'
ORDER BY c.name,
    CASE nsi.data_source_id
        WHEN 1 THEN 1
        WHEN 3 THEN 2
        WHEN 9 THEN 3
        ELSE        4
    END
`

func main() {
	output := flag.String("output",
		"internal/ioexport/data/rankdict.yaml",
		"Output file path")
	flag.Parse()

	host := envOr("GNDB_DATABASE_HOST", "localhost")
	port := envIntOr("GNDB_DATABASE_PORT", 5432)
	user := envOr("GNDB_DATABASE_USER", "postgres")
	password := envOr("GNDB_DATABASE_PASSWORD", "postgres")
	database := envOr("GNDB_DATABASE_DATABASE", "gnames")

	ctx := context.Background()

	pool, err := connect(ctx, host, port, user, password, database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	fmt.Printf("Connected to %s@%s:%d/%s\n", user, host, port, database)

	entries, err := queryEntries(ctx, pool)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Found %d entries\n", len(entries))

	if err := writeYAML(*output, entries); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	fmt.Printf("Written to %s\n", *output)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func connect(
	ctx context.Context,
	host string,
	port int,
	user string,
	password string,
	database string,
) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database,
	)
	return pgxpool.New(ctx, dsn)
}

func queryEntries(
	ctx context.Context,
	pool *pgxpool.Pool,
) ([]rankEntry, error) {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var entries []rankEntry
	for rows.Next() {
		var e rankEntry
		if err := rows.Scan(&e.Name, &e.Rank, &e.Code); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if e.Code == "unknown" {
			continue
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

func writeYAML(outputPath string, entries []rankEntry) error {
	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	// Group entries by rank
	byRank := make(map[string][]rankEntry)
	for _, e := range entries {
		byRank[e.Rank] = append(byRank[e.Rank], e)
	}

	// Sort each group by name
	for rank := range byRank {
		sort.Slice(byRank[rank], func(i, j int) bool {
			return byRank[rank][i].Name < byRank[rank][j].Name
		})
	}

	// Calculate max name width per rank group for alignment
	maxWidth := func(group []rankEntry) int {
		w := 0
		for _, e := range group {
			if len(e.Name) > w {
				w = len(e.Name)
			}
		}
		return w
	}

	// Write header
	fmt.Fprintf(w, "# Generated by tools/gen_rankdict.go on %s\n",
		time.Now().Format("2006-01-02"))
	fmt.Fprintf(w, "# Sources: CoL (1), ITIS (3), WoRMS (9)\n")
	fmt.Fprintf(w, "# Scope: Animalia (iczn) + Plantae (icn) only\n")
	fmt.Fprintf(w, "# Total entries: %d\n", len(entries))
	fmt.Fprintf(w, "---\n")

	// Write entries grouped by rank in canonical order
	for _, rank := range rankOrder {
		group, ok := byRank[rank]
		if !ok || len(group) == 0 {
			continue
		}

		fmt.Fprintf(w, "\n# %ss (%d)\n", rank, len(group))

		width := maxWidth(group)
		for _, e := range group {
			padding := strings.Repeat(" ",
				width-len(e.Name)+1)
			fmt.Fprintf(w, "%s:%s{rank: %s, code: %s}\n",
				e.Name, padding, e.Rank, e.Code)
		}
	}

	return w.Flush()
}
