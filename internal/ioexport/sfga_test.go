package ioexport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/schema"
	"github.com/gnames/gndb/pkg/sources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- slugify ---

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Catalogue of Life", "catalogue-of-life"},
		{"ITIS", "itis"},
		{"CoL", "col"},
		{"My Source 2024", "my-source-2024"},
		{"  leading/trailing  ", "leadingtrailing"},
		{"double--hyphens", "double-hyphens"},
		{"Special @#$ chars", "special-chars"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, slugify(tt.input))
		})
	}
}

// --- buildOutputBase ---

func TestBuildOutputBase_UsesRevisionDate(t *testing.T) {
	ds := schema.DataSource{
		ID:           1,
		TitleShort:   "CoL",
		RevisionDate: "2025-08-25",
	}

	base := buildOutputBase(ds, "/output")

	assert.Equal(t, "/output/0001-col-2025-08-25", base)
}

func TestBuildOutputBase_FallsBackToUpdatedAt(t *testing.T) {
	updated := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	ds := schema.DataSource{
		ID:         132,
		TitleShort: "ITIS",
		UpdatedAt:  updated,
		// RevisionDate is empty
	}

	base := buildOutputBase(ds, "/output")

	assert.Equal(t, "/output/0132-itis-2026-03-23", base)
}

func TestBuildOutputBase_EmptyTitleShort_UsesSourceFallback(t *testing.T) {
	ds := schema.DataSource{
		ID:           1000,
		TitleShort:   "",
		RevisionDate: "2026-01-01",
	}

	base := buildOutputBase(ds, "/out")

	assert.Equal(t, "/out/1000-source-2026-01-01", base)
}

func TestBuildOutputBase_IDZeroPadded(t *testing.T) {
	tests := []struct {
		id   int
		want string
	}{
		{1, "0001"},
		{11, "0011"},
		{132, "0132"},
		{1000, "1000"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("id=%d", tt.id), func(t *testing.T) {
			ds := schema.DataSource{ID: tt.id, TitleShort: "x", RevisionDate: "2025-01-01"}
			base := buildOutputBase(ds, "")
			prefix := strings.Split(filepath.Base(base), "-")[0]
			assert.Equal(t, tt.want, prefix)
		})
	}
}

// --- dataSourceToConfig ---

func TestDataSourceToConfig_MapsFields(t *testing.T) {
	ds := schema.DataSource{
		ID:             42,
		Title:          "My Source",
		TitleShort:     "MS",
		Description:    "A test source",
		WebsiteURL:     "https://example.com",
		DataURL:        "https://example.com/data",
		IsCurated:      true,
		IsAutoCurated:  false,
		HasTaxonData:   true,
		IsOutlinkReady: true,
		OutlinkURL:     "https://example.com/taxon/{}",
	}

	cfg := dataSourceToConfig(ds, "http://serve.example.org/sfga/")

	assert.Equal(t, 42, cfg.ID)
	assert.Equal(t, "http://serve.example.org/sfga/", cfg.Parent)
	assert.Equal(t, "My Source", cfg.Title)
	assert.Equal(t, "MS", cfg.TitleShort)
	assert.Equal(t, "A test source", cfg.Description)
	assert.Equal(t, "https://example.com", cfg.HomeURL)
	assert.Equal(t, "https://example.com/data", cfg.DataURL)
	assert.True(t, cfg.IsCurated)
	assert.False(t, cfg.IsAutoCurated)
	assert.True(t, cfg.HasClassification)
	assert.True(t, cfg.IsOutlinkReady)
	assert.Equal(t, "https://example.com/taxon/{}", cfg.OutlinkURL)
	assert.Equal(t, "name.col__alternative_id", cfg.OutlinkIDColumn)
}

func TestDataSourceToConfig_NoOutlinkURL_OmitsIDColumn(t *testing.T) {
	ds := schema.DataSource{ID: 1, OutlinkURL: ""}

	cfg := dataSourceToConfig(ds, "/output")

	assert.Empty(t, cfg.OutlinkIDColumn)
}

// --- writeCompanionYAML ---

func TestWriteCompanionYAML_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "0001-col-2025-08-25")

	ds := schema.DataSource{
		ID:           1,
		Title:        "Catalogue of Life",
		TitleShort:   "CoL",
		RevisionDate: "2025-08-25",
		WebsiteURL:   "https://catalogueoflife.org",
		IsCurated:    true,
		HasTaxonData: true,
	}

	cfg := config.New()
	cfg.Update([]config.Option{
		config.OptExportOutputDir(dir),
	})

	err := writeCompanionYAML(ds, base, cfg)
	require.NoError(t, err)

	yamlPath := base + ".yaml"
	_, err = os.Stat(yamlPath)
	require.NoError(t, err, "YAML file should exist")
}

func TestWriteCompanionYAML_ContainsRequiredFields(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "0001-col-2025-08-25")

	ds := schema.DataSource{
		ID:             1,
		Title:          "Catalogue of Life",
		TitleShort:     "CoL",
		RevisionDate:   "2025-08-25",
		WebsiteURL:     "https://catalogueoflife.org",
		DataURL:        "https://catalogueoflife.org/data",
		IsCurated:      true,
		HasTaxonData:   true,
		IsOutlinkReady: true,
		OutlinkURL:     "https://catalogueoflife.org/data/taxon/{}",
	}

	cfg := config.New()
	cfg.Update([]config.Option{
		config.OptExportOutputDir(dir),
		config.OptExportParentDir("http://myserver.org/sfga/"),
	})

	err := writeCompanionYAML(ds, base, cfg)
	require.NoError(t, err)

	raw, err := os.ReadFile(base + ".yaml")
	require.NoError(t, err)

	content := string(raw)
	assert.Contains(t, content, "# Generated by gndb export")
	assert.Contains(t, content, "id: 1")
	assert.Contains(t, content, "parent: http://myserver.org/sfga/")
	assert.Contains(t, content, "Catalogue of Life")
	assert.Contains(t, content, "is_curated: true")
	assert.Contains(t, content, "has_classification: true")
	assert.Contains(t, content, "is_outlink_ready: true")
	assert.Contains(t, content, "outlink_url:")
	assert.Contains(t, content, "outlink_id_column: name.col__alternative_id")
}

func TestWriteCompanionYAML_ParentDefaultsToOutputDir(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "0001-col-2025-08-25")

	ds := schema.DataSource{
		ID:           1,
		Title:        "Test",
		TitleShort:   "Test",
		RevisionDate: "2025-01-01",
	}

	cfg := config.New()
	cfg.Update([]config.Option{
		config.OptExportOutputDir(dir),
		// ParentDir not set — should fall back to OutputDir
	})

	err := writeCompanionYAML(ds, base, cfg)
	require.NoError(t, err)

	raw, err := os.ReadFile(base + ".yaml")
	require.NoError(t, err)

	assert.Contains(t, string(raw), fmt.Sprintf("parent: %s", dir))
}

func TestWriteCompanionYAML_NoOutlinkURL_OmitsOutlinkIDColumn(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "0001-col-2025-08-25")

	ds := schema.DataSource{
		ID:           1,
		Title:        "Test",
		TitleShort:   "Test",
		RevisionDate: "2025-01-01",
		// OutlinkURL is empty
	}

	cfg := config.New()
	cfg.Update([]config.Option{config.OptExportOutputDir(dir)})

	err := writeCompanionYAML(ds, base, cfg)
	require.NoError(t, err)

	raw, err := os.ReadFile(base + ".yaml")
	require.NoError(t, err)

	assert.NotContains(t, string(raw), "outlink_id_column")
}

func TestWriteCompanionYAML_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "0001-col-2025-08-25")

	ds := schema.DataSource{
		ID:           1,
		Title:        "Catalogue of Life",
		TitleShort:   "CoL",
		RevisionDate: "2025-08-25",
	}

	cfg := config.New()
	cfg.Update([]config.Option{config.OptExportOutputDir(dir)})

	err := writeCompanionYAML(ds, base, cfg)
	require.NoError(t, err)

	raw, err := os.ReadFile(base + ".yaml")
	require.NoError(t, err)

	// Skip the comment lines and parse the YAML portion.
	lines := strings.Split(string(raw), "\n")
	var yamlLines []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") {
			yamlLines = append(yamlLines, line)
		}
	}

	var parsed map[string]any
	err = yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &parsed)
	assert.NoError(t, err, "YAML file should be valid YAML")
	assert.Equal(t, 1, parsed["id"])
}

// --- writeConsolidatedYAML ---

func TestWriteConsolidatedYAML_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfg := config.New()
	cfg.Update([]config.Option{config.OptExportOutputDir(dir)})

	exported := []schema.DataSource{
		{ID: 1, TitleShort: "CoL", RevisionDate: "2025-08-25"},
		{ID: 132, TitleShort: "ITIS", RevisionDate: "2024-12-01"},
	}

	err := writeConsolidatedYAML(exported, cfg)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "sources-export.yaml"))
	assert.NoError(t, err, "sources-export.yaml should exist")
}

func TestWriteConsolidatedYAML_ContainsAllSources(t *testing.T) {
	dir := t.TempDir()
	cfg := config.New()
	cfg.Update([]config.Option{
		config.OptExportOutputDir(dir),
		config.OptExportParentDir("http://myserver.org/sfga/"),
	})

	exported := []schema.DataSource{
		{ID: 1, Title: "Catalogue of Life", TitleShort: "CoL",
			RevisionDate: "2025-08-25", IsCurated: true},
		{ID: 132, Title: "ITIS", TitleShort: "ITIS",
			RevisionDate: "2024-12-01"},
		{ID: 11, Title: "GBIF Backbone", TitleShort: "GBIF",
			RevisionDate: "2025-01-15"},
	}

	err := writeConsolidatedYAML(exported, cfg)
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(dir, "sources-export.yaml"))
	require.NoError(t, err)

	content := string(raw)
	assert.Contains(t, content, "# Generated by gndb export")
	assert.Contains(t, content, "Contains 3 data source(s)")
	assert.Contains(t, content, "data_sources:")
	assert.Contains(t, content, "id: 1")
	assert.Contains(t, content, "id: 132")
	assert.Contains(t, content, "id: 11")
	assert.Contains(t, content, "parent: http://myserver.org/sfga/")
	assert.Contains(t, content, "Catalogue of Life")
	assert.Contains(t, content, "is_curated: true")
}

func TestWriteConsolidatedYAML_ValidSourcesYAML(t *testing.T) {
	dir := t.TempDir()
	cfg := config.New()
	cfg.Update([]config.Option{config.OptExportOutputDir(dir)})

	exported := []schema.DataSource{
		{ID: 1, TitleShort: "CoL", RevisionDate: "2025-08-25"},
		{ID: 132, TitleShort: "ITIS", RevisionDate: "2024-12-01"},
	}

	err := writeConsolidatedYAML(exported, cfg)
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(dir, "sources-export.yaml"))
	require.NoError(t, err)

	// Strip comment lines and parse as SourcesConfig.
	lines := strings.Split(string(raw), "\n")
	var yamlLines []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") {
			yamlLines = append(yamlLines, line)
		}
	}

	var sc sources.SourcesConfig
	err = yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &sc)
	require.NoError(t, err, "should parse as SourcesConfig")
	assert.Len(t, sc.DataSources, 2)
	assert.Equal(t, 1, sc.DataSources[0].ID)
	assert.Equal(t, 132, sc.DataSources[1].ID)
}

func TestWriteConsolidatedYAML_EmptyList_NoFile(t *testing.T) {
	dir := t.TempDir()
	cfg := config.New()
	cfg.Update([]config.Option{config.OptExportOutputDir(dir)})

	err := writeConsolidatedYAML(nil, cfg)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "sources-export.yaml"))
	assert.True(t, os.IsNotExist(err), "no file should be created for empty list")
}

func TestWriteConsolidatedYAML_ParentDefaultsToOutputDir(t *testing.T) {
	dir := t.TempDir()
	cfg := config.New()
	cfg.Update([]config.Option{config.OptExportOutputDir(dir)})

	exported := []schema.DataSource{
		{ID: 1, TitleShort: "CoL", RevisionDate: "2025-08-25"},
	}

	err := writeConsolidatedYAML(exported, cfg)
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(dir, "sources-export.yaml"))
	require.NoError(t, err)

	assert.Contains(t, string(raw), fmt.Sprintf("parent: %s", dir))
}
