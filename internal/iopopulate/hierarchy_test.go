package iopopulate

import (
	"testing"

	"github.com/gnames/gnparser"
	"github.com/stretchr/testify/assert"
)

func TestBreadcrumbsNodes(t *testing.T) {
	// Build test hierarchy:
	// Plantae (1) -> Rosaceae (2) -> Rosa (3)
	hierarchy := map[string]*hNode{
		"1": {id: "1", parentID: "", name: "Plantae", rank: "kingdom"},
		"2": {id: "2", parentID: "1", name: "Rosaceae", rank: "family"},
		"3": {id: "3", parentID: "2", name: "Rosa", rank: "genus"},
	}

	tests := []struct {
		name       string
		id         string
		wantLen    int
		wantFirst  string
		wantLast   string
	}{
		{
			name:       "full path from leaf",
			id:         "3",
			wantLen:    3,
			wantFirst:  "Plantae",
			wantLast:   "Rosa",
		},
		{
			name:       "path from middle",
			id:         "2",
			wantLen:    2,
			wantFirst:  "Plantae",
			wantLast:   "Rosaceae",
		},
		{
			name:       "root only",
			id:         "1",
			wantLen:    1,
			wantFirst:  "Plantae",
			wantLast:   "Plantae",
		},
		{
			name:       "non-existent node",
			id:         "999",
			wantLen:    0,
			wantFirst:  "",
			wantLast:   "",
		},
		{
			name:       "whitespace in ID",
			id:         " 3 ",
			wantLen:    3,
			wantFirst:  "Plantae",
			wantLast:   "Rosa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := breadcrumbsNodes(tt.id, hierarchy)
			assert.Len(t, result, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantFirst, result[0].name)
				assert.Equal(t, tt.wantLast, result[len(result)-1].name)
			}
		})
	}
}

func TestBreadcrumbsNodesCircularReference(t *testing.T) {
	// Build hierarchy with circular reference: 1 -> 2 -> 3 -> 1
	hierarchy := map[string]*hNode{
		"1": {id: "1", parentID: "3", name: "A", rank: "genus"},
		"2": {id: "2", parentID: "1", name: "B", rank: "family"},
		"3": {id: "3", parentID: "2", name: "C", rank: "order"},
	}

	// Should not infinite loop and return partial result
	result := breadcrumbsNodes("1", hierarchy)
	assert.NotNil(t, result)
	// Should stop when circular reference detected
	assert.LessOrEqual(t, len(result), 3)
}

func TestGetFlatClsf(t *testing.T) {
	tests := []struct {
		name        string
		flatClsf    map[string]string
		nodes       []*hNode
		wantLen     int
		wantFirst   string
		wantLast    string
	}{
		{
			name: "basic flat classification",
			flatClsf: map[string]string{
				"kingdom":    "Animalia",
				"kingdom_id": "1",
				"phylum":     "Chordata",
				"phylum_id":  "2",
			},
			nodes:     []*hNode{},
			wantLen:   2,
			wantFirst: "Animalia",
			wantLast:  "Chordata",
		},
		{
			name: "flat classification with existing nodes",
			flatClsf: map[string]string{
				"kingdom":    "Animalia",
				"kingdom_id": "1",
			},
			nodes: []*hNode{
				{id: "5", name: "Species", rank: "species"},
			},
			wantLen:   2,
			wantFirst: "Animalia",
			wantLast:  "Species",
		},
		{
			name:        "empty flat classification",
			flatClsf:    map[string]string{},
			nodes:       []*hNode{},
			wantLen:     0,
			wantFirst:   "",
			wantLast:    "",
		},
		{
			name: "all ranks",
			flatClsf: map[string]string{
				"kingdom":    "Plantae",
				"kingdom_id": "1",
				"family":     "Rosaceae",
				"family_id":  "2",
				"genus":      "Rosa",
				"genus_id":   "3",
			},
			nodes:     []*hNode{},
			wantLen:   3,
			wantFirst: "Plantae",
			wantLast:  "Rosa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFlatClsf(tt.flatClsf, tt.nodes)
			assert.Len(t, result, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantFirst, result[0].name)
				assert.Equal(t, tt.wantLast, result[len(result)-1].name)
			}
		})
	}
}

func TestGetBreadcrumbs(t *testing.T) {
	hierarchy := map[string]*hNode{
		"1": {id: "1", parentID: "", name: "Plantae", rank: "kingdom"},
		"2": {id: "2", parentID: "1", name: "Rosaceae", rank: "family"},
		"3": {id: "3", parentID: "2", name: "Rosa", rank: "genus"},
	}

	flatClsf := map[string]string{
		"kingdom":    "Plantae",
		"kingdom_id": "1",
		"family":     "Rosaceae",
		"family_id":  "2",
	}

	tests := []struct {
		name                    string
		id                      string
		withFlatClassification  bool
		wantClassification      string
		wantClassificationRanks string
		wantClassificationIDs   string
	}{
		{
			name:                    "use hierarchy",
			id:                      "3",
			withFlatClassification:  false,
			wantClassification:      "Plantae|Rosaceae|Rosa",
			wantClassificationRanks: "kingdom|family|genus",
			wantClassificationIDs:   "1|2|3",
		},
		{
			name:                    "use flat classification",
			id:                      "3",
			withFlatClassification:  true,
			wantClassification:      "Plantae|Rosaceae",
			wantClassificationRanks: "kingdom|family",
			wantClassificationIDs:   "1|2",
		},
		{
			name:                    "short hierarchy falls back to flat",
			id:                      "1",
			withFlatClassification:  false,
			wantClassification:      "Plantae|Rosaceae|Plantae",
			wantClassificationRanks: "kingdom|family|kingdom",
			wantClassificationIDs:   "1|2|1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classification, classificationRanks, classificationIDs := getBreadcrumbs(
				tt.id, hierarchy, flatClsf, tt.withFlatClassification,
			)
			assert.Equal(t, tt.wantClassification, classification)
			assert.Equal(t, tt.wantClassificationRanks, classificationRanks)
			assert.Equal(t, tt.wantClassificationIDs, classificationIDs)
		})
	}
}

func TestProcessHierarchyRow(t *testing.T) {
	parser := gnparser.New(gnparser.NewConfig())

	tests := []struct {
		name     string
		nu       nameUsage
		wantName string
		wantRank string
	}{
		{
			name: "simple name",
			nu: nameUsage{
				id:              "1",
				parentID:        "",
				taxonomicStatus: "accepted",
				scientificName:  "Rosa alba",
				rank:            "SPECIES",
			},
			wantName: "Rosa alba",
			wantRank: "species",
		},
		{
			name: "self-referencing parent",
			nu: nameUsage{
				id:              "1",
				parentID:        "1",
				taxonomicStatus: "accepted",
				scientificName:  "Plantae",
				rank:            "Kingdom",
			},
			wantName: "Plantae",
			wantRank: "kingdom",
		},
		{
			name: "unparseable name",
			nu: nameUsage{
				id:              "1",
				parentID:        "",
				taxonomicStatus: "accepted",
				scientificName:  "???",
				rank:            "genus",
			},
			wantName: "",
			wantRank: "genus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processHierarchyRow(parser, tt.nu)
			assert.Equal(t, tt.nu.id, result.id)
			assert.Equal(t, tt.wantName, result.name)
			assert.Equal(t, tt.wantRank, result.rank)
			if tt.nu.parentID == tt.nu.id {
				assert.Empty(t, result.parentID)
			}
		})
	}
}
