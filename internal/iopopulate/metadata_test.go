package iopopulate

import (
	"testing"
	"time"

	"github.com/gnames/gndb/pkg/sources"
	"github.com/stretchr/testify/assert"
)

func TestBuildDataSourceRecord(t *testing.T) {
	tests := []struct {
		name            string
		source          sources.DataSourceConfig
		sfgaMeta        *sfgaMetadata
		sfgaFileMeta    SFGAMetadata
		recordCount     int
		vernRecordCount int
		wantTitle       string
		wantTitleShort  string
		wantVersion     string
		wantDescription string
	}{
		{
			name: "use sources.yaml title when provided",
			source: sources.DataSourceConfig{
				ID:         1,
				Title:      "Config Title",
				TitleShort: "CT",
			},
			sfgaMeta: &sfgaMetadata{
				Title:       "SFGA Title",
				Description: "SFGA Desc",
				DOI:         "10.1234/test",
			},
			sfgaFileMeta: SFGAMetadata{
				Version:      "1.0.0",
				RevisionDate: "2024-01-01",
			},
			recordCount:     100,
			vernRecordCount: 50,
			wantTitle:       "Config Title",
			wantTitleShort:  "CT",
			wantVersion:     "1.0.0",
			wantDescription: "SFGA Desc",
		},
		{
			name: "fallback to SFGA title",
			source: sources.DataSourceConfig{
				ID:         2,
				TitleShort: "Test",
			},
			sfgaMeta: &sfgaMetadata{
				Title:       "SFGA Title",
				Description: "SFGA Description",
			},
			sfgaFileMeta:    SFGAMetadata{},
			recordCount:     200,
			vernRecordCount: 0,
			wantTitle:       "SFGA Title",
			wantTitleShort:  "Test",
			wantVersion:     "",
			wantDescription: "SFGA Description",
		},
		{
			name: "use sources.yaml description when provided",
			source: sources.DataSourceConfig{
				ID:          3,
				Title:       "Test Source",
				Description: "Config Description",
			},
			sfgaMeta: &sfgaMetadata{
				Description: "SFGA Description",
			},
			sfgaFileMeta:    SFGAMetadata{},
			recordCount:     0,
			vernRecordCount: 0,
			wantTitle:       "Test Source",
			wantDescription: "Config Description",
		},
		{
			name: "curation flags passed through",
			source: sources.DataSourceConfig{
				ID:                1,
				TitleShort:        "Test",
				IsCurated:         true,
				IsAutoCurated:     true,
				HasClassification: true,
				IsOutlinkReady:    true,
			},
			sfgaMeta:        &sfgaMetadata{},
			sfgaFileMeta:    SFGAMetadata{},
			recordCount:     0,
			vernRecordCount: 0,
			wantTitle:       "",
			wantTitleShort:  "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDataSourceRecord(
				tt.source, tt.sfgaMeta, tt.sfgaFileMeta,
				tt.recordCount, tt.vernRecordCount,
			)

			assert.Equal(t, tt.source.ID, result.ID)
			assert.Equal(t, tt.wantTitle, result.Title)
			assert.Equal(t, tt.wantTitleShort, result.TitleShort)
			assert.Equal(t, tt.wantVersion, result.Version)
			assert.Equal(t, tt.wantDescription, result.Description)
			assert.Equal(t, tt.recordCount, result.RecordCount)
			assert.Equal(t, tt.vernRecordCount, result.VernRecordCount)
			assert.Equal(t, tt.source.IsCurated, result.IsCurated)
			assert.Equal(t, tt.source.IsAutoCurated, result.IsAutoCurated)
			assert.Equal(t, tt.source.HasClassification, result.HasTaxonData)
			assert.Equal(t, tt.source.IsOutlinkReady, result.IsOutlinkReady)

			// UpdatedAt should be recent
			assert.WithinDuration(t, time.Now(), result.UpdatedAt, time.Second)
		})
	}
}

func TestBuildDataSourceRecordURLs(t *testing.T) {
	source := sources.DataSourceConfig{
		ID:         1,
		TitleShort: "Test",
		HomeURL:    "https://example.com",
		DataURL:    "https://example.com/data",
		OutlinkURL: "https://example.com/outlink/%s",
	}

	result := buildDataSourceRecord(
		source, &sfgaMetadata{}, SFGAMetadata{}, 0, 0,
	)

	assert.Equal(t, "https://example.com", result.WebsiteURL)
	assert.Equal(t, "https://example.com/data", result.DataURL)
	assert.Equal(t, "https://example.com/outlink/%s", result.OutlinkURL)
}

func TestBuildDataSourceRecordDOI(t *testing.T) {
	source := sources.DataSourceConfig{
		ID:         1,
		TitleShort: "Test",
	}

	sfgaMeta := &sfgaMetadata{
		DOI: "10.1234/example",
	}

	result := buildDataSourceRecord(
		source, sfgaMeta, SFGAMetadata{}, 0, 0,
	)

	assert.Equal(t, "10.1234/example", result.DOI)
}
