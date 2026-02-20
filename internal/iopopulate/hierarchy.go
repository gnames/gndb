package iopopulate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gnlib/ent/nomcode"
	"github.com/gnames/gnparser"
	"golang.org/x/sync/errgroup"
)

// hNode represents a node in the taxonomic hierarchy.
// It stores the essential information needed to build classification
// breadcrumbs.
type hNode struct {
	id              string
	parentID        string
	taxonomicStatus string
	name            string // Canonical name parsed by gnparser
	rank            string
}

// badNodes tracks nodes that are referenced but don't exist in the hierarchy.
// This prevents logging the same missing node warning multiple times.
var badNodes = make(map[string]badNodeType)
var badNodesMutex sync.Mutex

type badNodeType int

const (
	missingBadNode badNodeType = iota + 1
	circularBadNode
)

// buildHierarchy implements Phase 3: Classification Hierarchy construction.
// It constructs a map of taxonomy nodes from the SFGA taxon table using
// concurrent workers.
//
// The hierarchy is used in Phase 4 to provide full classification paths
// for taxa and synonyms. Uses gnparser with botanical code to avoid issues
// like "Aus (Bus)" parsing incorrectly.
//
// Concurrent Processing:
//   - Creates its own context for goroutine cancellation
//   - Uses p.cfg.JobsNumber workers to parse names in parallel
//   - Employs errgroup for coordinated error handling
//
// Returns:
//   - map[string]*hNode: Map of taxon IDs to hierarchy nodes with parent
//     pointers
//   - error: Any error encountered during processing
func (p *populator) buildHierarchy() (map[string]*hNode, error) {
	// Create channels for worker communication
	chIn := make(chan nameUsage)
	chOut := make(chan *hNode)

	// Create error group for concurrent processing
	g, ctx := errgroup.WithContext(context.Background())
	var wg sync.WaitGroup

	// Start worker goroutines
	for range p.cfg.JobsNumber {
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			return hierarchyWorker(ctx, chIn, chOut)
		})
	}

	// Hierarchy map to collect results
	hierarchy := make(map[string]*hNode)

	// Start result collector
	g.Go(func() error {
		return createHierarchy(ctx, chOut, hierarchy)
	})

	// Close chOut when all workers are done
	go func() {
		wg.Wait()
		close(chOut)
	}()

	// Load name usage data from SFGA
	err := p.loadNameUsage(ctx, chIn)
	if err != nil {
		return nil, err
	}
	close(chIn)

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return nil, err
	}

	return hierarchy, nil
}

// nameUsage represents a row from the SFGA taxon/name join query.
type nameUsage struct {
	id              string
	parentID        string
	taxonomicStatus string
	scientificName  string
	rank            string
}

// hierarchyWorker processes name usage records concurrently.
// It parses scientific names using the gnparser pool and sends results to chOut.
func hierarchyWorker(
	ctx context.Context,
	chIn <-chan nameUsage,
	chOut chan<- *hNode,
) error {
	pCfg := gnparser.NewConfig(gnparser.OptCode(nomcode.Botanical))
	parser := gnparser.New(pCfg)
	for nu := range chIn {
		row := processHierarchyRow(parser, nu)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case chOut <- row:
		}
	}

	return nil
}

// processHierarchyRow converts a nameUsage record into an hNode.
// It uses gnparser to extract the canonical form of the scientific name.
func processHierarchyRow(parser gnparser.GNparser, nu nameUsage) *hNode {
	// Parse scientific name using botanical code
	parsed := parser.ParseName(nu.scientificName)

	var name string
	if parsed.Parsed {
		name = parsed.Canonical.Simple
	}

	// Handle self-referencing parent IDs
	parentID := nu.parentID
	if parentID == nu.id {
		parentID = ""
	}

	rank := strings.ToLower(nu.rank)

	return &hNode{
		id:              nu.id,
		rank:            rank,
		name:            name,
		parentID:        parentID,
		taxonomicStatus: nu.taxonomicStatus,
	}
}

// createHierarchy collects hNode results from workers into the hierarchy map.
// It also logs progress periodically.
func createHierarchy(ctx context.Context, chOut <-chan *hNode, hierarchy map[string]*hNode) error {
	var count int
	for node := range chOut {
		if node.id == "" {
			continue
		}

		count++
		if count%100_000 == 0 {
			progressReport(count, "hierarchy records")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			hierarchy[node.id] = node
		}
	}
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 80))

	return nil
}

// loadNameUsage reads taxon and name data from SFGA and sends it to chIn.
// It performs a JOIN to get all necessary fields for hierarchy building.
func (p *populator) loadNameUsage(
	ctx context.Context,
	chIn chan<- nameUsage,
) error {
	query := `
		SELECT t.col__id, t.col__parent_id, t.col__status_id,
		       n.col__scientific_name, n.col__rank_id
		FROM taxon t
		JOIN name n ON n.col__id = t.col__name_id
	`

	rows, err := p.sfgaDB.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var nu nameUsage
		err = rows.Scan(
			&nu.id,
			&nu.parentID,
			&nu.taxonomicStatus,
			&nu.scientificName,
			&nu.rank,
		)
		if err != nil {
			return err
		}

		chIn <- nu
	}

	return rows.Err()
}

// getBreadcrumbs generates classification strings by walking up the parent chain.
// It returns three pipe-delimited strings: names, ranks, and IDs.
//
// If withFlatClassification is true, it uses flat classification exclusively.
// Otherwise, it uses hierarchical classification and falls back to flat
// classification only when the hierarchy has fewer than 2 nodes.
//
// Parameters:
//   - id: The taxon ID to generate breadcrumbs for
//   - hierarchy: The complete hierarchy map
//   - flatClsf: Flat classification data from SFGA (optional)
//   - withFlatClassification: If true, prefer flat classification over hierarchy
//
// Returns:
//   - classification: Pipe-delimited canonical names (e.g., "Plantae|Rosaceae|Rosa")
//   - classificationRanks: Pipe-delimited ranks (e.g., "kingdom|family|genus")
//   - classificationIDs: Pipe-delimited taxon IDs (e.g., "1|5|6")
func getBreadcrumbs(
	id string,
	hierarchy map[string]*hNode,
	flatClsf map[string]string,
	withFlatClassification bool,
) (classification, classificationRanks, classificationIDs string) {
	var nodes []*hNode

	// If flat classification is NOT preferred, build hierarchy breadcrumbs
	if !withFlatClassification {
		nodes = breadcrumbsNodes(id, hierarchy)
	}

	// Fall back to flat classification if:
	// 1. Flat classification is preferred (withFlatClassification=true), OR
	// 2. Hierarchy is too short (< 2 nodes)
	if len(nodes) < 2 {
		nodes = getFlatClsf(flatClsf, nodes)
	}

	// Build pipe-delimited strings
	names := make([]string, len(nodes))
	ranks := make([]string, len(nodes))
	ids := make([]string, len(nodes))

	for i := range nodes {
		names[i] = nodes[i].name
		ranks[i] = nodes[i].rank
		ids[i] = nodes[i].id
	}

	return strings.Join(names, "|"),
		strings.Join(ranks, "|"),
		strings.Join(ids, "|")
}

// breadcrumbsNodes walks up the parent chain from the given ID to the root.
// It returns the path from root to the specified node.
func breadcrumbsNodes(id string, hierarchy map[string]*hNode) []*hNode {
	id = strings.TrimSpace(id)
	var result []*hNode

	currID := id
	visited := make(map[string]bool) // Prevent infinite loops

	for {
		// Check for circular references
		if visited[currID] {
			badNodesMutex.Lock()
			if _, ok := badNodes[currID]; !ok {
				badNodes[currID] = circularBadNode
			}
			badNodesMutex.Unlock()
			return result
		}
		visited[currID] = true

		// Get the node
		node, ok := hierarchy[currID]
		if !ok {
			// Node doesn't exist
			badNodesMutex.Lock()
			if _, ok := badNodes[currID]; !ok {
				badNodes[currID] = missingBadNode
			}
			badNodesMutex.Unlock()
			return result
		}

		// Prepend node to result (we're walking up, but want root-to-leaf order)
		result = append([]*hNode{node}, result...)

		// Stop if we've reached the root
		if node.parentID == "" {
			return result
		}

		currID = node.parentID
	}
}

// getFlatClsf combines flat classification data with existing nodes.
// Flat classification provides predefined ranks when hierarchical data is incomplete.
//
// The ranks are processed in order: kingdom, phylum, subphylum, class, order, etc.
// This ensures a consistent classification structure.
func getFlatClsf(flatClsf map[string]string, nodes []*hNode) []*hNode {
	var result []*hNode

	// Predefined rank order for flat classification
	ranks := []string{
		"kingdom",
		"phylum",
		"subphylum",
		"class",
		"order",
		"suborder",
		"superfamily",
		"family",
		"subfamily",
		"tribe",
		"subtribe",
		"genus",
		"subgenus",
		"section",
		"species",
	}

	// Add nodes from flat classification
	for _, rank := range ranks {
		name := flatClsf[rank]
		id := flatClsf[rank+"_id"]
		if name != "" {
			result = append(result, &hNode{
				name: name,
				id:   id,
				rank: rank,
			})
		}
	}

	// Append any existing hierarchical nodes
	result = append(result, nodes...)

	return result
}

// progressReport logs progress to stderr with humanized numbers.
// It clears the line before writing to avoid leftover characters.
func progressReport(recNum int, entity string) {
	str := fmt.Sprintf("Processed %s %s", humanize.Comma(int64(recNum)), entity)
	fmt.Fprintf(os.Stderr, "\r%s", strings.Repeat(" ", 80))
	fmt.Fprintf(os.Stderr, "\r%s", str)
}
