package ioexport

import (
	"github.com/gnames/gndb/pkg/schema"
	"github.com/sfborg/sflib/pkg/coldp"
)

// dataSourceToMeta converts a schema.DataSource to a coldp.Meta,
// making the SFGA archive self-contained.
func dataSourceToMeta(ds schema.DataSource) *coldp.Meta {
	return &coldp.Meta{
		Title:       ds.Title,
		Alias:       ds.TitleShort,
		DOI:         ds.DOI,
		Description: ds.Description,
		URL:         ds.WebsiteURL,
		Citation:    ds.Citation,
		Version:     ds.Version,
		Issued:      ds.RevisionDate,
	}
}
