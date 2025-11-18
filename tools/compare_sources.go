// compare_sources compares a data source between gnames (to-gn) and gndb
// databases. This is a temporary tool for validating gndb implementation.
//
// Usage:
//
//	go run tools/compare_sources.go --source-id 1 --host localhost --port 5432 --user postgres --password secret
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ComparisonResult struct {
	SourceID                 int
	ToGnNameStrings          int
	GndbNameStrings          int
	ToGnNameIndices          int
	GndbNameIndices          int
	ToGnVernStrings          int
	GndbVernStrings          int
	ToGnVernIndices          int
	GndbVernIndices          int
	ToGnVerification         int
	GndbVerification         int
	MetadataMatch            bool
	SampleRecordsMatch       bool
	ClassificationMatch      bool
	VernacularRecordsMatch   bool
	VerificationRecordsMatch bool
	TaxonomicStatusesMatch   bool
	ToGnStatusCounts         map[string]int
	GndbStatusCounts         map[string]int
}

type SampleRecord struct {
	RecordID          string
	NameStringID      string
	Rank              string
	TaxonomicStatus   string
	Classification    sql.NullString
	ClassificationIDs sql.NullString
}

type VernacularRecord struct {
	RecordID           string
	VernacularStringID string
	Language           sql.NullString
	LanguageOrig       sql.NullString
	LangCode           sql.NullString
	Locality           sql.NullString
	CountryCode        sql.NullString
}

type VerificationRecord struct {
	DataSourceID    int
	RecordID        string
	NameStringID    string
	Name            string
	CanonicalID     sql.NullString
	TaxonomicStatus string
	AcceptedNameID  sql.NullString
	AcceptedName    sql.NullString
}

func main() {
	sourceID := flag.Int("source-id", 0, "Data source ID to compare")
	host := flag.String("host", "localhost", "PostgreSQL host")
	port := flag.Int("port", 5432, "PostgreSQL port")
	user := flag.String("user", "postgres", "PostgreSQL user")
	password := flag.String("password", "", "PostgreSQL password")
	sampleSize := flag.Int("sample-size", 100,
		"Number of sample records to compare")

	flag.Parse()

	if *sourceID == 0 {
		fmt.Println("Error: --source-id is required")
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	// Connect to both databases
	tognConn, err := connect(ctx, *host, *port, *user, *password, "gnames")
	if err != nil {
		log.Fatalf("Failed to connect to gnames database: %v", err)
	}
	defer tognConn.Close()

	gndbConn, err := connect(ctx, *host, *port, *user, *password, "gndb")
	if err != nil {
		log.Fatalf("Failed to connect to gndb database: %v", err)
	}
	defer gndbConn.Close()

	fmt.Printf("Comparing data source ID %d\n", *sourceID)
	fmt.Println(string([]rune{'='}[0]) +
		string([]rune{'='}[0]) +
		string([]rune{'='}[0]))
	fmt.Println()

	result := &ComparisonResult{SourceID: *sourceID}

	// 1. Compare record counts
	fmt.Println("1. Record Counts")
	fmt.Println("----------------")
	if err := compareCounts(ctx, tognConn, gndbConn, *sourceID,
		result); err != nil {
		log.Fatalf("Failed to compare counts: %v", err)
	}

	// 2. Compare metadata
	fmt.Println("\n2. Data Source Metadata")
	fmt.Println("-----------------------")
	if err := compareMetadata(ctx, tognConn, gndbConn, *sourceID,
		result); err != nil {
		log.Fatalf("Failed to compare metadata: %v", err)
	}

	// 3. Compare sample records
	fmt.Println("\n3. Sample Name String Indices")
	fmt.Println("-----------------------------")
	if err := compareSampleRecords(ctx, tognConn, gndbConn, *sourceID,
		*sampleSize, result); err != nil {
		log.Fatalf("Failed to compare sample records: %v", err)
	}

	// 4. Compare taxonomic status distribution
	fmt.Println("\n4. Taxonomic Status Distribution")
	fmt.Println("---------------------------------")
	if err := compareTaxonomicStatuses(ctx, tognConn, gndbConn, *sourceID,
		result); err != nil {
		log.Fatalf("Failed to compare taxonomic statuses: %v", err)
	}

	// 5. Compare vernacular records
	fmt.Println("\n5. Sample Vernacular String Indices")
	fmt.Println("-----------------------------------")
	if err := compareVernacularRecords(ctx, tognConn, gndbConn, *sourceID,
		*sampleSize, result); err != nil {
		log.Fatalf("Failed to compare vernacular records: %v", err)
	}

	// 6. Compare verification view
	fmt.Println("\n6. Verification View")
	fmt.Println("--------------------")
	if err := compareVerificationView(ctx, tognConn, gndbConn, *sourceID,
		*sampleSize, result); err != nil {
		log.Fatalf("Failed to compare verification view: %v", err)
	}

	// 7. Summary
	fmt.Println("\n7. Summary")
	fmt.Println("----------")
	printSummary(result)
}

func connect(
	ctx context.Context,
	host string,
	port int,
	user string,
	password string,
	database string,
) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database,
	)

	return pgxpool.New(ctx, connStr)
}

func compareCounts(
	ctx context.Context,
	tognConn *pgxpool.Pool,
	gndbConn *pgxpool.Pool,
	sourceID int,
	result *ComparisonResult,
) error {
	// Name string indices count
	var err error
	result.ToGnNameIndices, err = getNameIndicesCount(
		ctx,
		tognConn,
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("to-gn name indices count: %w", err)
	}

	result.GndbNameIndices, err = getNameIndicesCount(
		ctx,
		gndbConn,
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("gndb name indices count: %w", err)
	}

	fmt.Printf("  Name String Indices:\n")
	fmt.Printf("    to-gn: %d\n", result.ToGnNameIndices)
	fmt.Printf("    gndb:  %d\n", result.GndbNameIndices)
	if result.ToGnNameIndices == result.GndbNameIndices {
		fmt.Printf("    ✓ Match\n")
	} else {
		fmt.Printf("    ✗ Mismatch (diff: %d)\n",
			result.GndbNameIndices-result.ToGnNameIndices)
	}

	// Vernacular string indices count
	result.ToGnVernIndices, err = getVernIndicesCount(
		ctx,
		tognConn,
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("to-gn vernacular indices count: %w", err)
	}

	result.GndbVernIndices, err = getVernIndicesCount(
		ctx,
		gndbConn,
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("gndb vernacular indices count: %w", err)
	}

	fmt.Printf("\n  Vernacular String Indices:\n")
	fmt.Printf("    to-gn: %d\n", result.ToGnVernIndices)
	fmt.Printf("    gndb:  %d\n", result.GndbVernIndices)
	if result.ToGnVernIndices == result.GndbVernIndices {
		fmt.Printf("    ✓ Match\n")
	} else {
		fmt.Printf("    ✗ Mismatch (diff: %d)\n",
			result.GndbVernIndices-result.ToGnVernIndices)
	}

	return nil
}

func compareMetadata(
	ctx context.Context,
	tognConn *pgxpool.Pool,
	gndbConn *pgxpool.Pool,
	sourceID int,
	result *ComparisonResult,
) error {
	query := `
		SELECT
			title, title_short, version, record_count, vern_record_count
		FROM data_sources
		WHERE id = $1
	`

	var tognTitle, tognTitleShort, tognVersion string
	var tognRecCount, tognVernRecCount int
	err := tognConn.QueryRow(ctx, query, sourceID).Scan(
		&tognTitle, &tognTitleShort, &tognVersion,
		&tognRecCount, &tognVernRecCount,
	)
	if err != nil {
		return fmt.Errorf("to-gn metadata query: %w", err)
	}

	var gndbTitle, gndbTitleShort, gndbVersion string
	var gndbRecCount, gndbVernRecCount int
	err = gndbConn.QueryRow(ctx, query, sourceID).Scan(
		&gndbTitle, &gndbTitleShort, &gndbVersion,
		&gndbRecCount, &gndbVernRecCount,
	)
	if err != nil {
		return fmt.Errorf("gndb metadata query: %w", err)
	}

	result.MetadataMatch = tognTitle == gndbTitle &&
		tognTitleShort == gndbTitleShort &&
		tognVersion == gndbVersion &&
		tognRecCount == gndbRecCount &&
		tognVernRecCount == gndbVernRecCount

	fmt.Printf("  Title:             %s\n",
		compareStrings(tognTitle, gndbTitle))
	fmt.Printf("  Title Short:       %s\n",
		compareStrings(tognTitleShort, gndbTitleShort))
	fmt.Printf("  Version:           %s\n",
		compareStrings(tognVersion, gndbVersion))
	fmt.Printf("  Record Count:      %s\n",
		compareInts(tognRecCount, gndbRecCount))
	fmt.Printf("  Vern Record Count: %s\n",
		compareInts(tognVernRecCount, gndbVernRecCount))

	return nil
}

func compareSampleRecords(
	ctx context.Context,
	tognConn *pgxpool.Pool,
	gndbConn *pgxpool.Pool,
	sourceID int,
	sampleSize int,
	result *ComparisonResult,
) error {
	query := `
		SELECT
			record_id, name_string_id, rank, taxonomic_status,
			classification, classification_ids
		FROM name_string_indices
		WHERE data_source_id = $1
		ORDER BY record_id
		LIMIT $2
	`

	tognRecords, err := getSampleRecords(
		ctx,
		tognConn,
		query,
		sourceID,
		sampleSize,
	)
	if err != nil {
		return fmt.Errorf("to-gn sample: %w", err)
	}

	gndbRecords, err := getSampleRecords(
		ctx,
		gndbConn,
		query,
		sourceID,
		sampleSize,
	)
	if err != nil {
		return fmt.Errorf("gndb sample: %w", err)
	}

	if len(tognRecords) != len(gndbRecords) {
		fmt.Printf("  Sample size mismatch: to-gn=%d, gndb=%d\n",
			len(tognRecords), len(gndbRecords))
		result.SampleRecordsMatch = false
		return nil
	}

	mismatches := 0
	classificationMismatches := 0
	for i := range len(tognRecords) {
		togn := tognRecords[i]
		gndb := gndbRecords[i]

		if togn.RecordID != gndb.RecordID ||
			togn.NameStringID != gndb.NameStringID ||
			togn.Rank != gndb.Rank ||
			togn.TaxonomicStatus != gndb.TaxonomicStatus {
			mismatches++
			if mismatches <= 5 {
				fmt.Printf("  Mismatch at record %s:\n", togn.RecordID)
				fmt.Printf("    to-gn: %+v\n", togn)
				fmt.Printf("    gndb:  %+v\n", gndb)
			}
		}

		// Check classification separately
		if !compareNullableStrings(
			togn.Classification,
			gndb.Classification,
		) {
			classificationMismatches++
		}
	}

	result.SampleRecordsMatch = mismatches == 0
	result.ClassificationMatch = classificationMismatches == 0

	fmt.Printf("  Sampled %d records\n", len(tognRecords))
	if result.SampleRecordsMatch {
		fmt.Printf("  ✓ All sample records match\n")
	} else {
		fmt.Printf("  ✗ %d record mismatches found\n", mismatches)
	}

	if result.ClassificationMatch {
		fmt.Printf("  ✓ All classifications match\n")
	} else {
		fmt.Printf("  ✗ %d classification mismatches found\n",
			classificationMismatches)
	}

	return nil
}

func compareTaxonomicStatuses(
	ctx context.Context,
	tognConn *pgxpool.Pool,
	gndbConn *pgxpool.Pool,
	sourceID int,
	result *ComparisonResult,
) error {
	query := `
		SELECT taxonomic_status, COUNT(*)
		FROM name_string_indices
		WHERE data_source_id = $1
		GROUP BY taxonomic_status
		ORDER BY taxonomic_status
	`

	tognCounts, err := getTaxonomicStatusCounts(ctx, tognConn, query,
		sourceID)
	if err != nil {
		return fmt.Errorf("to-gn status counts: %w", err)
	}

	gndbCounts, err := getTaxonomicStatusCounts(ctx, gndbConn, query,
		sourceID)
	if err != nil {
		return fmt.Errorf("gndb status counts: %w", err)
	}

	result.ToGnStatusCounts = tognCounts
	result.GndbStatusCounts = gndbCounts

	// Compare the counts
	allMatch := true
	allStatuses := make(map[string]bool)
	for status := range tognCounts {
		allStatuses[status] = true
	}
	for status := range gndbCounts {
		allStatuses[status] = true
	}

	for status := range allStatuses {
		tognCount := tognCounts[status]
		gndbCount := gndbCounts[status]

		if tognCount == gndbCount {
			fmt.Printf("  %s: ✓ %d\n", status, tognCount)
		} else {
			fmt.Printf("  %s: ✗ to-gn=%d gndb=%d (diff: %d)\n",
				status, tognCount, gndbCount, gndbCount-tognCount)
			allMatch = false
		}
	}

	result.TaxonomicStatusesMatch = allMatch

	if allMatch {
		fmt.Printf("\n  ✓ All taxonomic status counts match\n")
	} else {
		fmt.Printf("\n  ✗ Taxonomic status counts differ\n")
	}

	return nil
}

func compareVernacularRecords(
	ctx context.Context,
	tognConn *pgxpool.Pool,
	gndbConn *pgxpool.Pool,
	sourceID int,
	sampleSize int,
	result *ComparisonResult,
) error {
	// If there are no vernacular records, skip comparison
	if result.ToGnVernIndices == 0 && result.GndbVernIndices == 0 {
		fmt.Printf("  No vernacular records in either database\n")
		result.VernacularRecordsMatch = true
		return nil
	}

	query := `
		SELECT
			record_id, vernacular_string_id, language, language_orig,
			lang_code, locality, country_code
		FROM vernacular_string_indices
		WHERE data_source_id = $1
		ORDER BY record_id, vernacular_string_id
		LIMIT $2
	`

	tognRecords, err := getVernacularSampleRecords(
		ctx,
		tognConn,
		query,
		sourceID,
		sampleSize,
	)
	if err != nil {
		return fmt.Errorf("to-gn vernacular sample: %w", err)
	}

	gndbRecords, err := getVernacularSampleRecords(
		ctx,
		gndbConn,
		query,
		sourceID,
		sampleSize,
	)
	if err != nil {
		return fmt.Errorf("gndb vernacular sample: %w", err)
	}

	if len(tognRecords) != len(gndbRecords) {
		fmt.Printf("  Sample size mismatch: to-gn=%d, gndb=%d\n",
			len(tognRecords), len(gndbRecords))
		result.VernacularRecordsMatch = false
		return nil
	}

	mismatches := 0
	for i := range len(tognRecords) {
		togn := tognRecords[i]
		gndb := gndbRecords[i]

		if togn.RecordID != gndb.RecordID ||
			togn.VernacularStringID != gndb.VernacularStringID ||
			!compareNullableStrings(togn.Language, gndb.Language) ||
			!compareNullableStrings(togn.LanguageOrig, gndb.LanguageOrig) ||
			!compareNullableStrings(togn.LangCode, gndb.LangCode) ||
			!compareNullableStrings(togn.Locality, gndb.Locality) ||
			!compareNullableStrings(togn.CountryCode, gndb.CountryCode) {
			mismatches++
			if mismatches <= 5 {
				fmt.Printf("  Mismatch at record %s:\n", togn.RecordID)
				fmt.Printf("    to-gn: %+v\n", togn)
				fmt.Printf("    gndb:  %+v\n", gndb)
			}
		}
	}

	result.VernacularRecordsMatch = mismatches == 0

	fmt.Printf("  Sampled %d vernacular records\n", len(tognRecords))
	if result.VernacularRecordsMatch {
		fmt.Printf("  ✓ All vernacular records match\n")
	} else {
		fmt.Printf("  ✗ %d vernacular record mismatches found\n",
			mismatches)
	}

	return nil
}

func compareVerificationView(
	ctx context.Context,
	tognConn *pgxpool.Pool,
	gndbConn *pgxpool.Pool,
	sourceID int,
	sampleSize int,
	result *ComparisonResult,
) error {
	// Count verification records for this source
	var err error
	result.ToGnVerification, err = getVerificationCount(
		ctx,
		tognConn,
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("to-gn verification count: %w", err)
	}

	result.GndbVerification, err = getVerificationCount(
		ctx,
		gndbConn,
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("gndb verification count: %w", err)
	}

	fmt.Printf("  Verification Records:\\n")
	fmt.Printf("    to-gn: %d\\n", result.ToGnVerification)
	fmt.Printf("    gndb:  %d\\n", result.GndbVerification)
	if result.ToGnVerification == result.GndbVerification {
		fmt.Printf("    ✓ Match\\n")
	} else {
		fmt.Printf("    ✗ Mismatch (diff: %d)\\n",
			result.GndbVerification-result.ToGnVerification)
	}

	// If counts don't match, no point in comparing samples
	if result.ToGnVerification != result.GndbVerification {
		result.VerificationRecordsMatch = false
		return nil
	}

	// Compare sample verification records
	query := `
		SELECT data_source_id, record_id, name_string_id, name,
			canonical_id, taxonomic_status, accepted_name_id, accepted_name
		FROM verification
		WHERE data_source_id = $1
		ORDER BY record_id
		LIMIT $2
	`

	tognRecords, err := getVerificationSampleRecords(
		ctx,
		tognConn,
		query,
		sourceID,
		sampleSize,
	)
	if err != nil {
		return fmt.Errorf("to-gn verification sample: %w", err)
	}

	gndbRecords, err := getVerificationSampleRecords(
		ctx,
		gndbConn,
		query,
		sourceID,
		sampleSize,
	)
	if err != nil {
		return fmt.Errorf("gndb verification sample: %w", err)
	}

	if len(tognRecords) != len(gndbRecords) {
		fmt.Printf("  Sample size mismatch: to-gn=%d, gndb=%d\\n",
			len(tognRecords), len(gndbRecords))
		result.VerificationRecordsMatch = false
		return nil
	}

	mismatches := 0
	for i := range len(tognRecords) {
		togn := tognRecords[i]
		gndb := gndbRecords[i]

		if togn.RecordID != gndb.RecordID ||
			togn.NameStringID != gndb.NameStringID ||
			!compareNullableStrings(togn.CanonicalID, gndb.CanonicalID) ||
			togn.TaxonomicStatus != gndb.TaxonomicStatus ||
			!compareNullableStrings(togn.AcceptedNameID, gndb.AcceptedNameID) ||
			!compareNullableStrings(togn.AcceptedName, gndb.AcceptedName) {
			mismatches++
			if mismatches <= 5 {
				fmt.Printf("  Mismatch at record %s:\\n", togn.RecordID)
				fmt.Printf("    to-gn: %+v\\n", togn)
				fmt.Printf("    gndb:  %+v\\n", gndb)
			}
		}
	}

	result.VerificationRecordsMatch = mismatches == 0

	fmt.Printf("\\n  Sampled %d verification records\\n", len(tognRecords))
	if result.VerificationRecordsMatch {
		fmt.Printf("  ✓ All verification records match\\n")
	} else {
		fmt.Printf("  ✗ %d verification record mismatches found\\n",
			mismatches)
	}

	return nil
}

func getNameIndicesCount(
	ctx context.Context,
	conn *pgxpool.Pool,
	sourceID int,
) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM name_string_indices
	          WHERE data_source_id = $1`
	err := conn.QueryRow(ctx, query, sourceID).Scan(&count)
	return count, err
}

func getVernIndicesCount(
	ctx context.Context,
	conn *pgxpool.Pool,
	sourceID int,
) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM vernacular_string_indices
	          WHERE data_source_id = $1`
	err := conn.QueryRow(ctx, query, sourceID).Scan(&count)
	return count, err
}

func getVerificationCount(
	ctx context.Context,
	conn *pgxpool.Pool,
	sourceID int,
) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM verification
	          WHERE data_source_id = $1`
	err := conn.QueryRow(ctx, query, sourceID).Scan(&count)
	return count, err
}

func getSampleRecords(
	ctx context.Context,
	conn *pgxpool.Pool,
	query string,
	sourceID int,
	limit int,
) ([]SampleRecord, error) {
	rows, err := conn.Query(ctx, query, sourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []SampleRecord
	for rows.Next() {
		var rec SampleRecord
		err := rows.Scan(
			&rec.RecordID,
			&rec.NameStringID,
			&rec.Rank,
			&rec.TaxonomicStatus,
			&rec.Classification,
			&rec.ClassificationIDs,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	return records, rows.Err()
}

func getTaxonomicStatusCounts(
	ctx context.Context,
	conn *pgxpool.Pool,
	query string,
	sourceID int,
) (map[string]int, error) {
	rows, err := conn.Query(ctx, query, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, err
		}
		counts[status] = count
	}

	return counts, rows.Err()
}

func getVernacularSampleRecords(
	ctx context.Context,
	conn *pgxpool.Pool,
	query string,
	sourceID int,
	limit int,
) ([]VernacularRecord, error) {
	rows, err := conn.Query(ctx, query, sourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []VernacularRecord
	for rows.Next() {
		var rec VernacularRecord
		err := rows.Scan(
			&rec.RecordID,
			&rec.VernacularStringID,
			&rec.Language,
			&rec.LanguageOrig,
			&rec.LangCode,
			&rec.Locality,
			&rec.CountryCode,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	return records, rows.Err()
}

func getVerificationSampleRecords(
	ctx context.Context,
	conn *pgxpool.Pool,
	query string,
	sourceID int,
	limit int,
) ([]VerificationRecord, error) {
	rows, err := conn.Query(ctx, query, sourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []VerificationRecord
	for rows.Next() {
		var rec VerificationRecord
		err := rows.Scan(
			&rec.DataSourceID,
			&rec.RecordID,
			&rec.NameStringID,
			&rec.Name,
			&rec.CanonicalID,
			&rec.TaxonomicStatus,
			&rec.AcceptedNameID,
			&rec.AcceptedName,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	return records, rows.Err()
}

func compareStrings(a, b string) string {
	if a == b {
		return fmt.Sprintf("✓ %s", a)
	}
	return fmt.Sprintf("✗ to-gn='%s' gndb='%s'", a, b)
}

func compareInts(a, b int) string {
	if a == b {
		return fmt.Sprintf("✓ %d", a)
	}
	return fmt.Sprintf("✗ to-gn=%d gndb=%d (diff: %d)", a, b, b-a)
}

func compareNullableStrings(a, b sql.NullString) bool {
	if !a.Valid && !b.Valid {
		return true
	}
	if !a.Valid || !b.Valid {
		return false
	}
	return a.String == b.String
}

func printSummary(result *ComparisonResult) {
	allMatch := result.ToGnNameIndices == result.GndbNameIndices &&
		result.ToGnVernIndices == result.GndbVernIndices &&
		result.ToGnVerification == result.GndbVerification &&
		result.MetadataMatch &&
		result.SampleRecordsMatch &&
		result.ClassificationMatch &&
		result.TaxonomicStatusesMatch &&
		result.VernacularRecordsMatch &&
		result.VerificationRecordsMatch

	if allMatch {
		fmt.Println("  ✓ All comparisons match!")
		fmt.Println("  The imports are identical.")
	} else {
		fmt.Println("  ✗ Differences found:")
		if result.ToGnNameIndices != result.GndbNameIndices {
			fmt.Printf("    - Name indices count differs\n")
		}
		if result.ToGnVernIndices != result.GndbVernIndices {
			fmt.Printf("    - Vernacular indices count differs\n")
		}
		if result.ToGnVerification != result.GndbVerification {
			fmt.Printf("    - Verification view count differs\n")
		}
		if !result.MetadataMatch {
			fmt.Printf("    - Metadata differs\n")
		}
		if !result.SampleRecordsMatch {
			fmt.Printf("    - Sample records differ\n")
		}
		if !result.ClassificationMatch {
			fmt.Printf("    - Classifications differ\n")
		}
		if !result.TaxonomicStatusesMatch {
			fmt.Printf("    - Taxonomic status counts differ\n")
		}
		if !result.VernacularRecordsMatch {
			fmt.Printf("    - Vernacular records differ\n")
		}
		if !result.VerificationRecordsMatch {
			fmt.Printf("    - Verification view records differ\n")
		}
	}
}
