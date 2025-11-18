package iopopulate

import (
	"database/sql"
	"testing"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/pkg/errcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite" // Pure Go SQLite driver (no CGo)
)

func TestCheckSfgaVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses SQLite file in short mode")
	}

	tests := []struct {
		name         string
		dbPath       string
		sourceID     int
		wantErr      bool
		wantErrCode  gn.ErrorCode
		wantContains string
	}{
		{
			name:         "version too old",
			dbPath:       "../../testdata/1003-old-data-source.sqlite",
			sourceID:     1003,
			wantErr:      true,
			wantErrCode:  errcode.PopulateSFGAVersionTooOldError,
			wantContains: "too old",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Open the test SQLite database.
			db, err := sql.Open("sqlite", tt.dbPath)
			require.NoError(t, err)
			defer db.Close()

			// Create populator with the test database.
			p := &populator{
				sfgaDB: db,
			}

			// Call the method under test.
			err = p.checkSfgaVersion(tt.sourceID)

			if tt.wantErr {
				require.Error(t, err)

				gnErr, ok := err.(*gn.Error)
				require.True(t, ok, "Error should be of type *gn.Error")

				assert.Equal(t, tt.wantErrCode, gnErr.Code)
				assert.Contains(t, gnErr.Err.Error(), tt.wantContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckSfgaVersionGetError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses SQLite in short mode")
	}

	// Open a database without VERSION table to trigger query error.
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	p := &populator{
		sfgaDB: db,
	}

	err = p.checkSfgaVersion(999)
	require.Error(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateSFGAVersionError, gnErr.Code)
	assert.Contains(t, gnErr.Err.Error(), "failed to get SFGA version")
}

func TestCheckSfgaVersionInvalidFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that uses SQLite in short mode")
	}

	// Create in-memory database with invalid version format.
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE VERSION (ID TEXT)")
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO VERSION (ID) VALUES ('not-a-version')")
	require.NoError(t, err)

	p := &populator{
		sfgaDB: db,
	}

	err = p.checkSfgaVersion(999)
	require.Error(t, err)

	gnErr, ok := err.(*gn.Error)
	require.True(t, ok, "Error should be of type *gn.Error")

	assert.Equal(t, errcode.PopulateSFGAVersionError, gnErr.Code)
	assert.Contains(t, gnErr.Err.Error(), "not a semantic version")
}
