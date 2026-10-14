package iooptimize

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// abbrGeneraSourceIDs are data sources preferred for classifications
// of genera, the best first: Catalogue of Life XR covers more genera
// than Catalogue of Life.
var abbrGeneraSourceIDs = []int{13, 1}

// abbrIRMNGID is the data source of the Interim Register of Marine and
// Nonmarine Genera. It lists genera and higher taxa only, and is used to
// tell real genera from misspelled ones.
const abbrIRMNGID = 181

// abbrGeneraSQL fills abbr_genera from abbr_lookups and the
// classifications of a data source ($1).
//
// The size of a genus is the number of its distinct keys in
// abbr_lookups. All keys of a genus start with the same letter, so it
// is the number of distinct epithet stems.
//
// The classification of a genus is cut from classifications of its
// species and lower names: the part above the genus rank, and the genus
// itself. Synonyms have the classification of their current name, so
// a synonym's own genus replaces the genus of the path. Paths of
// accepted names are preferred, and then the most common path.
//
// Homonyms are separated by kingdom, not by nomenclatural code. Codes
// of records are sometimes wrong (beetles marked as botanical), and
// groups such as Chromista have no fixed code.
//
// A genus is known if it is a genus of IRMNG ($2), accepted or a
// synonym, or if it has a classification. Misspelled genera
// (Caenorbditis) are usually neither. A classification rescues new
// genera that IRMNG does not have yet. All rows of a homonym have the
// same value, because every row of a classified genus has a
// classification.
const abbrGeneraSQL = `
INSERT INTO abbr_genera
  (genus, kingdom, epithets_num, names_num,
   classification, classification_ranks, data_source_id, known)
WITH irmng AS (
  SELECT DISTINCT c.name AS genus
  FROM name_string_indices nsi
  JOIN name_strings ns ON ns.id = nsi.name_string_id
  JOIN canonicals c ON c.id = ns.canonical_id
  WHERE nsi.data_source_id = $2
    AND nsi.rank = 'genus'
), sizes AS (
  SELECT genus, count(DISTINCT abbr_key)::int AS epithets_num
  FROM abbr_lookups
  GROUP BY genus
), recs AS (
  SELECT split_part(c.name, ' ', 1) AS genus,
         nsi.taxonomic_status IN ('accepted', 'provisionally_accepted')
           AS accepted,
         string_to_array(nsi.classification, '|') AS path,
         string_to_array(nsi.classification_ranks, '|') AS ranks
  FROM name_string_indices nsi
  JOIN name_strings ns ON ns.id = nsi.name_string_id
  JOIN canonicals c ON c.id = ns.canonical_id
  WHERE nsi.data_source_id = $1
    AND ns.cardinality > 1
    AND ns.virus IS NOT TRUE
), cut AS (
  SELECT genus, accepted, path, ranks,
         array_position(ranks, 'genus') AS genus_pos,
         coalesce(path[array_position(ranks, 'kingdom')], '') AS kingdom
  FROM recs
), paths AS (
  SELECT genus, kingdom,
         array_to_string(path[1:genus_pos - 1] || genus, '|')
           AS classification,
         array_to_string(ranks[1:genus_pos - 1] || 'genus'::text, '|')
           AS classification_ranks,
         count(*) AS num,
         count(*) FILTER (WHERE accepted) AS accepted_num
  FROM cut
  WHERE genus_pos IS NOT NULL
  GROUP BY 1, 2, 3, 4
), kingdoms AS (
  SELECT DISTINCT ON (genus, kingdom)
         genus, kingdom, classification, classification_ranks,
         sum(num) OVER (PARTITION BY genus, kingdom)::int AS names_num
  FROM paths
  ORDER BY genus, kingdom, accepted_num DESC, num DESC, classification
)
SELECT s.genus,
       coalesce(k.kingdom, ''),
       s.epithets_num,
       coalesce(k.names_num, 0),
       coalesce(k.classification, ''),
       coalesce(k.classification_ranks, ''),
       CASE WHEN k.genus IS NULL THEN 0 ELSE $1 END,
       k.genus IS NOT NULL OR i.genus IS NOT NULL
FROM sizes s
LEFT JOIN kingdoms k ON k.genus = s.genus
LEFT JOIN irmng i ON i.genus = s.genus
ORDER BY s.genus, k.names_num DESC`

// buildAbbrGenera rebuilds abbr_genera, the sizes and classifications
// of genera from abbr_lookups. It must run after abbr_lookups is
// built. It returns the number of genera.
func buildAbbrGenera(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	if _, err := pool.Exec(ctx, "TRUNCATE abbr_genera"); err != nil {
		return 0, abbrError("truncate genera", err)
	}

	sourceID, err := abbrGeneraSource(ctx, pool)
	if err != nil {
		return 0, err
	}
	slog.Info("Classifying abbreviation genera", "data_source_id", sourceID)

	_, err = pool.Exec(ctx, abbrGeneraSQL, sourceID, abbrIRMNGID)
	if err != nil {
		return 0, abbrError("insert genera", err)
	}

	var generaNum, classifiedNum, knownNum, homonymsNum int64
	q := `
SELECT count(DISTINCT genus),
       count(DISTINCT genus) FILTER (WHERE data_source_id <> 0),
       count(DISTINCT genus) FILTER (WHERE known),
       (SELECT count(*) FROM (
          SELECT genus FROM abbr_genera GROUP BY genus HAVING count(*) > 1
        ) h)
FROM abbr_genera`
	err = pool.QueryRow(ctx, q).Scan(
		&generaNum, &classifiedNum, &knownNum, &homonymsNum,
	)
	if err != nil {
		return 0, abbrError("count genera", err)
	}
	slog.Info("Built abbreviation genera",
		"genera", generaNum,
		"classified", classifiedNum,
		"known", knownNum,
		"homonyms", homonymsNum,
	)
	return generaNum, nil
}

// abbrGeneraSource returns the data source for classifications of
// genera: the first preferred source that has genus ranks, otherwise
// the largest such source. It returns 0 if no source has them, and
// then genera get no classifications.
func abbrGeneraSource(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	rows, err := pool.Query(ctx,
		"SELECT id FROM data_sources ORDER BY record_count DESC, id")
	if err != nil {
		return 0, abbrError("data sources", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return 0, abbrError("data sources", err)
	}

	candidates := append([]int{}, abbrGeneraSourceIDs...)
	candidates = append(candidates, ids...)

	q := `
SELECT EXISTS (
  SELECT FROM name_string_indices
  WHERE data_source_id = $1
    AND classification_ranks LIKE '%genus%'
)`
	for _, id := range candidates {
		var ok bool
		if err := pool.QueryRow(ctx, q, id).Scan(&ok); err != nil {
			return 0, abbrError("classification source", err)
		}
		if ok {
			return id, nil
		}
	}
	return 0, nil
}
