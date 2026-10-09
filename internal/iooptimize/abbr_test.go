package iooptimize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbbrRows_Split(t *testing.T) {
	tests := []struct {
		name      string
		canonical string
		want      []abbrRow
	}{
		{
			name:      "binomial",
			canonical: "Caenorhabditis elegans",
			want: []abbrRow{
				{"c-elegans", 2, "Caenorhabditis", "id"},
			},
		},
		{
			name:      "trinomial",
			canonical: "Panthera tigris altaica",
			want: []abbrRow{
				{"p-tigr", 2, "Panthera", "id"},
				{"p-altaic", 3, "Panthera", "id"},
			},
		},
		{
			name:      "repeated epithet keeps both positions",
			canonical: "Bison bison bison",
			want: []abbrRow{
				{"b-bison", 2, "Bison", "id"},
				{"b-bison", 3, "Bison", "id"},
			},
		},
		{
			name:      "epithet with empty stem",
			canonical: "Cavendishia axillaris que",
			want: []abbrRow{
				{"c-axillar", 2, "Cavendishia", "id"},
			},
		},
		{
			name:      "uninomial",
			canonical: "Carex",
			want:      nil,
		},
		{
			name:      "lowercase first word",
			canonical: "aus bus",
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, abbrRows(tt.canonical, "id"))
		})
	}
}
