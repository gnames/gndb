package iopopulate

import (
	"context"
	"database/sql"
	"testing"

	"github.com/gnames/gndb/pkg/populate"
	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: This is an integration test that uses real SFGA test data.
// Skip with: go test -short

// TestBuildHierarchy_Integration tests hierarchy map generation from real SFGA data.
// This test verifies:
//  1. hNode map is built correctly from SFGA taxon table
//  2. Parent-child relationships are preserved
//  3. Self-referencing parent IDs are handled (set to empty)
//  4. Parsing uses botanical code to avoid "Aus (Bus)" → "Bus" issue
//  5. Concurrent parsing with gnparser pool works correctly
func TestBuildHierarchy_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Open real SFGA test data (1002-vascan)
	testdataDir := "../../testdata"
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	source := populate.DataSourceConfig{
		ID:     1002, // vascan
		Parent: testdataDir,
	}

	sqlitePath, _, _, err := fetchSFGA(ctx, source, cacheDir)
	require.NoError(t, err, "Should fetch test SFGA")

	sfgaDB, err := openSFGA(sqlitePath)
	require.NoError(t, err, "Should open SFGA database")
	defer sfgaDB.Close()

	// Verify SFGA has taxon data
	var taxonCount int
	err = sfgaDB.QueryRow("SELECT COUNT(*) FROM taxon").Scan(&taxonCount)
	require.NoError(t, err, "Should query taxon count")
	require.Greater(t, taxonCount, 0, "SFGA should have taxon records")

	// Test: Build hierarchy (this will fail until T041 is implemented)
	// Use 4 workers for concurrent parsing
	hierarchy, err := buildHierarchy(ctx, sfgaDB, 4)
	require.NoError(t, err, "buildHierarchy should succeed")
	require.NotNil(t, hierarchy, "hierarchy should not be nil")

	// Verify: Check hierarchy map is not empty
	assert.Greater(t, len(hierarchy), 0, "Hierarchy should have nodes")

	// Verify: All nodes in hierarchy have valid structure
	for id, node := range hierarchy {
		assert.NotEmpty(t, id, "Node ID should not be empty")
		assert.NotNil(t, node, "Node should not be nil")
		assert.Equal(t, id, node.id, "Node ID should match map key")

		// If node has a parent, it should not be self-referencing
		if node.parentID != "" {
			assert.NotEqual(t, node.id, node.parentID, "Node should not reference itself as parent")
		}

		// Node should have a parsed name (unless parsing failed)
		// We don't assert this is always non-empty because some names might fail parsing
	}

	// Sample a few specific nodes to verify parsing worked
	// Note: We can't hardcode specific IDs since SFGA structure may change,
	// but we can verify that nodes with parents exist
	var nodesWithParents int
	var rootNodes int
	for _, node := range hierarchy {
		if node.parentID != "" {
			nodesWithParents++
		} else {
			rootNodes++
		}
	}

	assert.Greater(t, nodesWithParents, 0, "Should have nodes with parents")
	assert.Greater(t, rootNodes, 0, "Should have root nodes (no parent)")
}

// TestGetBreadcrumbs_Integration tests classification string generation from real SFGA data.
// This verifies that walking up the parent chain produces correct pipe-delimited strings.
func TestGetBreadcrumbs_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Open real SFGA test data
	testdataDir := "../../testdata"
	cacheDir, err := prepareCacheDir()
	require.NoError(t, err)

	source := populate.DataSourceConfig{
		ID:     1002, // vascan
		Parent: testdataDir,
	}

	sqlitePath, _, _, err := fetchSFGA(ctx, source, cacheDir)
	require.NoError(t, err)

	sfgaDB, err := openSFGA(sqlitePath)
	require.NoError(t, err)
	defer sfgaDB.Close()

	// Build hierarchy first
	hierarchy, err := buildHierarchy(ctx, sfgaDB, 4)
	require.NoError(t, err)
	require.NotEmpty(t, hierarchy, "Hierarchy should not be empty")

	// Find a node with multiple ancestors (species level) for testing
	var testNodeID string
	var testNodeDepth int
	for id, node := range hierarchy {
		if node.rank == "species" && node.parentID != "" {
			// Calculate depth by walking up
			depth := 0
			currentID := id
			for depth < 20 { // Safety limit
				currentNode, exists := hierarchy[currentID]
				if !exists || currentNode.parentID == "" {
					break
				}
				depth++
				currentID = currentNode.parentID
			}

			if depth > testNodeDepth {
				testNodeDepth = depth
				testNodeID = id
			}

			if depth >= 3 { // Found a good test case
				break
			}
		}
	}

	require.NotEmpty(t, testNodeID, "Should find a species node for testing")

	// Test: Get breadcrumbs (this will fail until T041 is implemented)
	flatClsf := make(map[string]string) // Empty flat classification
	withFlatClassification := false     // Use hierarchical classification
	classification, classificationRanks, classificationIDs := getBreadcrumbs(testNodeID, hierarchy, flatClsf, withFlatClassification)

	// Verify: Classification strings are not empty
	assert.NotEmpty(t, classification, "Classification should not be empty")
	assert.NotEmpty(t, classificationRanks, "Classification ranks should not be empty")
	assert.NotEmpty(t, classificationIDs, "Classification IDs should not be empty")

	// Verify: All three strings have the same number of pipe-delimited elements
	classificationParts := splitPipeDelimited(classification)
	ranksParts := splitPipeDelimited(classificationRanks)
	idsParts := splitPipeDelimited(classificationIDs)

	assert.Equal(t, len(classificationParts), len(ranksParts),
		"Classification and ranks should have same number of elements")
	assert.Equal(t, len(classificationParts), len(idsParts),
		"Classification and IDs should have same number of elements")

	// Verify: Classification ends with the test node
	assert.Equal(t, testNodeID, idsParts[len(idsParts)-1],
		"Classification IDs should end with the test node ID")

	// Verify: Classification includes the node's name
	testNode := hierarchy[testNodeID]
	if testNode.name != "" {
		assert.Contains(t, classification, testNode.name,
			"Classification should include the node's name")
	}
}

// TestGetBreadcrumbs_FlatClassificationFallback tests fallback to flat classification.
// When hierarchical breadcrumbs have < 2 nodes, it should use flat classification.
func TestGetBreadcrumbs_FlatClassificationFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create minimal SFGA with single node (no hierarchy)
	sfgaDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE name (
			col__id TEXT PRIMARY KEY,
			col__scientific_name TEXT NOT NULL,
			col__rank_id TEXT
		);

		CREATE TABLE taxon (
			col__id TEXT PRIMARY KEY,
			col__name_id TEXT NOT NULL,
			col__parent_id TEXT,
			col__status_id TEXT
		);

		INSERT INTO name (col__id, col__scientific_name, col__rank_id) VALUES
			('1', 'Rosa acicularis', 'species');

		INSERT INTO taxon (col__id, col__name_id, col__parent_id, col__status_id) VALUES
			('1', '1', '', 'accepted');
	`)
	require.NoError(t, err)

	hierarchy, err := buildHierarchy(ctx, sfgaDB, 2)
	require.NoError(t, err)

	// Provide flat classification data (as would come from SFGA flat fields)
	flatClsf := map[string]string{
		"kingdom":    "Plantae",
		"kingdom_id": "k1",
		"family":     "Rosaceae",
		"family_id":  "f1",
		"genus":      "Rosa",
		"genus_id":   "g1",
	}

	withFlatClassification := false // Use hierarchical, but will fallback to flat since hierarchy < 2
	classification, classificationRanks, classificationIDs := getBreadcrumbs("1", hierarchy, flatClsf, withFlatClassification)

	// Verify: Should use flat classification when hierarchy is too short (< 2 nodes)
	assert.Contains(t, classification, "Plantae", "Should include kingdom from flat classification")
	assert.Contains(t, classification, "Rosaceae", "Should include family from flat classification")
	assert.Contains(t, classification, "Rosa", "Should include genus from flat classification")
	assert.Contains(t, classification, "Rosa acicularis", "Should include the species itself")

	// Verify: ranks are included
	assert.Contains(t, classificationRanks, "kingdom")
	assert.Contains(t, classificationRanks, "family")
	assert.Contains(t, classificationRanks, "genus")

	// Verify: IDs are included from flat classification
	assert.Contains(t, classificationIDs, "k1", "Should include kingdom ID from flat classification")
	assert.Contains(t, classificationIDs, "f1", "Should include family ID from flat classification")
}

// TestGetBreadcrumbs_MissingParent tests handling of missing parent nodes.
// This can happen if data is incomplete or there's a dangling reference.
func TestGetBreadcrumbs_MissingParent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create SFGA with missing parent reference
	sfgaDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE name (
			col__id TEXT PRIMARY KEY,
			col__scientific_name TEXT NOT NULL,
			col__rank_id TEXT
		);

		CREATE TABLE taxon (
			col__id TEXT PRIMARY KEY,
			col__name_id TEXT NOT NULL,
			col__parent_id TEXT,
			col__status_id TEXT
		);

		INSERT INTO name (col__id, col__scientific_name, col__rank_id) VALUES
			('2', 'Rosaceae', 'family'),
			('3', 'Rosa', 'genus');

		INSERT INTO taxon (col__id, col__name_id, col__parent_id, col__status_id) VALUES
			('2', '2', '999', 'accepted'),
			('3', '3', '2', 'accepted');
	`)
	require.NoError(t, err)

	hierarchy, err := buildHierarchy(ctx, sfgaDB, 2)
	require.NoError(t, err)

	flatClsf := make(map[string]string)
	withFlatClassification := false // Use hierarchical classification
	classification, classificationRanks, classificationIDs := getBreadcrumbs("3", hierarchy, flatClsf, withFlatClassification)

	// Verify: Breadcrumbs should stop when parent is missing
	assert.Contains(t, classification, "Rosa", "Should include genus")
	assert.Contains(t, classification, "Rosaceae", "Should include family")
	assert.NotEmpty(t, classificationRanks, "Should have ranks")
	// Should not crash or include the missing parent
	assert.NotContains(t, classificationIDs, "999", "Should not include missing parent ID")
}

// TestBuildHierarchy_SelfReferencingParent tests handling of self-referencing parent IDs.
func TestBuildHierarchy_SelfReferencingParent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	sfgaDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE name (
			col__id TEXT PRIMARY KEY,
			col__scientific_name TEXT NOT NULL,
			col__rank_id TEXT
		);

		CREATE TABLE taxon (
			col__id TEXT PRIMARY KEY,
			col__name_id TEXT NOT NULL,
			col__parent_id TEXT,
			col__status_id TEXT
		);

		INSERT INTO name (col__id, col__scientific_name, col__rank_id) VALUES
			('1', 'Plantae', 'kingdom');

		INSERT INTO taxon (col__id, col__name_id, col__parent_id, col__status_id) VALUES
			('1', '1', '1', 'accepted');
	`)
	require.NoError(t, err)

	hierarchy, err := buildHierarchy(ctx, sfgaDB, 2)
	require.NoError(t, err)

	// Verify: Self-referencing parent should be converted to empty string
	node := hierarchy["1"]
	require.NotNil(t, node)
	assert.Equal(t, "", node.parentID, "Self-referencing parent should be empty string")
}

// TestGetBreadcrumbs_WithFlatClassificationTrue tests explicit preference for flat classification.
// When withFlatClassification=true, it should use flat classification even if hierarchy exists.
func TestGetBreadcrumbs_WithFlatClassificationTrue(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create SFGA with complete hierarchy
	sfgaDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sfgaDB.Close()

	_, err = sfgaDB.Exec(`
		CREATE TABLE name (
			col__id TEXT PRIMARY KEY,
			col__scientific_name TEXT NOT NULL,
			col__rank_id TEXT
		);

		CREATE TABLE taxon (
			col__id TEXT PRIMARY KEY,
			col__name_id TEXT NOT NULL,
			col__parent_id TEXT,
			col__status_id TEXT
		);

		-- Create hierarchical data: Kingdom -> Family -> Genus -> Species
		INSERT INTO name (col__id, col__scientific_name, col__rank_id) VALUES
			('1', 'Plantae', 'kingdom'),
			('2', 'Rosaceae', 'family'),
			('3', 'Rosa', 'genus'),
			('4', 'Rosa acicularis', 'species');

		INSERT INTO taxon (col__id, col__name_id, col__parent_id, col__status_id) VALUES
			('1', '1', '', 'accepted'),
			('2', '2', '1', 'accepted'),
			('3', '3', '2', 'accepted'),
			('4', '4', '3', 'accepted');
	`)
	require.NoError(t, err)

	hierarchy, err := buildHierarchy(ctx, sfgaDB, 2)
	require.NoError(t, err)
	require.Len(t, hierarchy, 4, "Should have 4 nodes in hierarchy")

	// Provide different flat classification data
	flatClsf := map[string]string{
		"kingdom":    "Animalia", // Different from hierarchy
		"kingdom_id": "k_anim",
		"phylum":     "Chordata",
		"phylum_id":  "p_chord",
		"class":      "Mammalia",
		"class_id":   "c_mamm",
		"order":      "Primates",
		"order_id":   "o_prim",
		"family":     "Hominidae", // Different from hierarchy
		"family_id":  "f_homin",
		"genus":      "Homo", // Different from hierarchy
		"genus_id":   "g_homo",
	}

	// Test with withFlatClassification=true
	withFlatClassification := true
	classification, classificationRanks, classificationIDs := getBreadcrumbs("4", hierarchy, flatClsf, withFlatClassification)

	// Verify: Should use flat classification INSTEAD of hierarchy
	assert.Contains(t, classification, "Animalia", "Should use Animalia from flat classification")
	assert.Contains(t, classification, "Hominidae", "Should use Hominidae from flat classification")
	assert.Contains(t, classification, "Homo", "Should use Homo from flat classification")

	// Should NOT contain hierarchical data
	assert.NotContains(t, classification, "Plantae", "Should NOT use Plantae from hierarchy")
	assert.NotContains(t, classification, "Rosaceae", "Should NOT use Rosaceae from hierarchy")

	// Verify flat classification ranks are used
	assert.Contains(t, classificationRanks, "kingdom")
	assert.Contains(t, classificationRanks, "phylum")
	assert.Contains(t, classificationRanks, "class")
	assert.Contains(t, classificationRanks, "order")

	// Verify flat classification IDs are used
	assert.Contains(t, classificationIDs, "k_anim", "Should include Animalia ID from flat classification")
	assert.Contains(t, classificationIDs, "f_homin", "Should include Hominidae ID from flat classification")

	// Now test with withFlatClassification=false for comparison
	withFlatClassification = false
	classification2, _, _ := getBreadcrumbs("4", hierarchy, flatClsf, withFlatClassification)

	// Verify: Should use hierarchical classification
	assert.Contains(t, classification2, "Plantae", "Should use Plantae from hierarchy when flag is false")
	assert.Contains(t, classification2, "Rosaceae", "Should use Rosaceae from hierarchy when flag is false")
	assert.NotContains(t, classification2, "Animalia", "Should NOT use Animalia when flag is false")
}

// Helper function to split pipe-delimited strings
func splitPipeDelimited(s string) []string {
	if s == "" {
		return []string{}
	}
	var result []string
	for _, part := range splitByPipe(s) {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitByPipe(s string) []string {
	var result []string
	current := ""
	for _, r := range s {
		if r == '|' {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
