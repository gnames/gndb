package iooptimize

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/gnames/gndb/pkg/schema"
)

func TestReliableNamesMatch(t *testing.T) {
	reliable, err := newReliableNames(map[string][]string{
		"Acacia may":           {"Acacia may"},
		"Eriopygodes imbecill": {"Eriopygodes imbecilla"},
		"Hipposideros ater":    {"Hipposideros ater"},
		"Tribolium castane":    {"Tribolium castaneum"},
	})
	require.NoError(t, err)

	tests := []struct {
		msg, canonical, stem string
		match                string
	}{
		{
			msg:       "ocr error",
			canonical: "Eriopvgodes imbecilla",
			stem:      "Eriopvgodes imbecill",
			match:     "Eriopygodes imbecilla",
		},
		{
			msg:       "stem is shared with a reliable name",
			canonical: "Tribolium castanea",
			stem:      "Tribolium castane",
		},
		{
			msg:       "nothing reliable nearby",
			canonical: "Pardosa moesta",
			stem:      "Pardosa moest",
		},
		{
			msg:       "too many changes for a short word",
			canonical: "Acacia mal",
			stem:      "Acacia mal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			name := ionName{canonical: tt.canonical, stem: tt.stem}
			match, ok := reliable.match(name)
			assert.Equal(t, tt.match, match)
			assert.Equal(t, tt.match != "", ok)
		})
	}
}

// TestFlagMisspellings_Database runs the SQL of the step on a real schema.
// It uses a temporary schema, so tables of the test database stay intact.
func TestFlagMisspellings_Database(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	database := os.Getenv("GNDB_TEST_DB_DATABASE")
	if database == "" {
		t.Skip("Database not configured")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, testDSN(database, "public"))
	require.NoError(t, err)
	defer admin.Close()

	const testSchema = "misspellings_test"
	dropSchema := "DROP SCHEMA IF EXISTS " + testSchema + " CASCADE"
	_, err = admin.Exec(ctx, dropSchema)
	require.NoError(t, err)
	_, err = admin.Exec(ctx, "CREATE SCHEMA "+testSchema)
	require.NoError(t, err)
	defer func() { _, _ = admin.Exec(ctx, dropSchema) }()

	dsn := testDSN(database, testSchema)
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, schema.Migrate(gormDB))

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	insertMisspellingData(t, pool)

	names, err := findMisspellings(ctx, pool)
	require.NoError(t, err)
	require.Len(t, names, 1)
	assert.Equal(t, "Eriopvgodes imbecilla", names[0].canonical)
	assert.Equal(t, "Eriopygodes imbecilla", names[0].reliable)

	require.NoError(t, saveMisspellingFlags(ctx, pool, names))
	assert.Equal(t, []string{"Eriopvgodes imbecilla"}, flagged(t, pool))

	// A second run replaces flags of the first one.
	require.NoError(t, saveMisspellingFlags(ctx, pool, nil))
	assert.Empty(t, flagged(t, pool))
}

func testDSN(database, searchPath string) string {
	host := os.Getenv("GNDB_TEST_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	user := os.Getenv("GNDB_TEST_DB_USER")
	if user == "" {
		user = "postgres"
	}
	dsn := fmt.Sprintf(
		"host=%s user=%s dbname=%s search_path=%s sslmode=disable",
		host, user, database, searchPath,
	)
	if password := os.Getenv("GNDB_TEST_DB_PASSWORD"); password != "" {
		dsn += " password=" + password
	}
	return dsn
}

func flagged(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	ctx := context.Background()
	rows, err := pool.Query(
		ctx, "SELECT name FROM canonicals WHERE maybe_misspelling",
	)
	require.NoError(t, err)
	defer rows.Close()

	var res []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		res = append(res, name)
	}
	require.NoError(t, rows.Err())
	return res
}

// insertMisspellingData creates one name for every branch of the rule.
func insertMisspellingData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	const curated, ion, unreliable = 1, ionDataSourceID, 500

	names := []struct {
		canonical, stem string
		cardinality     int
		sources         []int
	}{
		{"Eriopygodes imbecilla", "Eriopygodes imbecill", 2, []int{curated}},
		{"Hipposideros ater", "Hipposideros ater", 2, []int{curated}},
		{"Pomatomus", "Pomatomus", 1, []int{curated}},
		// The only name to flag.
		{"Eriopvgodes imbecilla", "Eriopvgodes imbecill", 2, []int{ion}},
		// Another source has it, even though it is not reliable.
		{"Hipposideros ator", "Hipposideros ator", 2, []int{ion, unreliable}},
		// Uninomials block nothing.
		{"Pomatomvs", "Pomatomvs", 1, []int{ion}},
		// Nothing reliable nearby.
		{"Pardosa moesta", "Pardosa moest", 2, []int{ion}},
	}

	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := pool.Exec(ctx, query, args...)
		require.NoError(t, err)
	}

	for _, id := range []int{curated, ion, unreliable} {
		exec(
			`INSERT INTO data_sources (id, title, is_curated)
			 VALUES ($1, $2, $3)`,
			id, fmt.Sprintf("source %d", id), id == curated,
		)
	}

	for i, name := range names {
		canonicalID := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		stemID := fmt.Sprintf("00000000-0000-0000-0001-%012d", i)
		stringID := fmt.Sprintf("00000000-0000-0000-0002-%012d", i)

		exec(
			"INSERT INTO canonicals (id, name) VALUES ($1, $2)",
			canonicalID, name.canonical,
		)
		exec(
			"INSERT INTO canonical_stems (id, name) VALUES ($1, $2)",
			stemID, name.stem,
		)
		exec(
			`INSERT INTO name_strings
			   (id, name, cardinality, canonical_id, canonical_stem_id)
			 VALUES ($1, $2, $3, $4, $5)`,
			stringID, name.canonical, name.cardinality, canonicalID, stemID,
		)
		for _, source := range name.sources {
			exec(
				`INSERT INTO name_string_indices
				   (data_source_id, record_id, accepted_record_id,
				    name_string_id)
				 VALUES ($1, $2, $2, $3)`,
				source, fmt.Sprintf("r%d", i), stringID,
			)
		}
	}
}
