package iopopulate

import (
	"context"
	"database/sql"
	"strings"

	"github.com/jackc/pgx/v5"
)

type taxonDatum struct {
	colName, gnName, synonymID, taxonID, nameID     string
	nameString, codeID, rankID, statusID            string
	kingdom, kingdomID, phylum, phylumID, subphylum sql.NullString
	subphylumID, class, classID, order, orderID     sql.NullString
	family, familyID, genus, genusID, species       sql.NullString
	speciesID, outlinkIDRaw, localID, globalID      sql.NullString
}

func (p *populator) flatClassification(t taxonDatum) (map[string]string, bool) {
	// Build flat classification map
	flatClsf := make(map[string]string)
	if t.kingdom.Valid {
		flatClsf["kingdom"] = t.kingdom.String
		flatClsf["kingdom_id"] = t.kingdomID.String
	}
	if t.phylum.Valid {
		flatClsf["phylum"] = t.phylum.String
		flatClsf["phylum_id"] = t.phylumID.String
	}
	if t.subphylum.Valid {
		flatClsf["subphylum"] = t.subphylum.String
		flatClsf["subphylum_id"] = t.subphylumID.String
	}
	if t.class.Valid {
		flatClsf["class"] = t.class.String
		flatClsf["class_id"] = t.classID.String
	}
	if t.order.Valid {
		flatClsf["order"] = t.order.String
		flatClsf["order_id"] = t.orderID.String
	}
	if t.family.Valid {
		flatClsf["family"] = t.family.String
		flatClsf["family_id"] = t.familyID.String
	}
	if t.genus.Valid {
		flatClsf["genus"] = t.genus.String
		flatClsf["genus_id"] = t.genusID.String
	}
	if t.species.Valid {
		flatClsf["species"] = t.species.String
		flatClsf["species_id"] = t.speciesID.String
	}

	// Get classification breadcrumbs
	var useFlat bool
	if p.cfg.Populate.WithFlatClassification != nil {
		useFlat = *p.cfg.Populate.WithFlatClassification
	}
	return flatClsf, useFlat
}

// insertNameIndices performs bulk insert using pgx CopyFrom.
func insertNameIndices(p *populator, records [][]any) error {
	// Column names for CopyFrom
	columns := []string{
		"data_source_id", "record_id", "name_string_id",
		"outlink_id", "global_id", "name_id", "local_id",
		"code_id", "rank", "taxonomic_status", "accepted_record_id",
		"classification", "classification_ids", "classification_ranks",
	}

	_, err := p.operator.Pool().CopyFrom(
		context.Background(),
		pgx.Identifier{"name_string_indices"},
		columns,
		pgx.CopyFromRows(records),
	)

	return err
}

// codeIDToInt converts SFGA code ID string to integer.
// 0 - no info, 1 - ICZN (zoological), 2 - ICN (botanical),
// 3 - ICNP (bacterial), 4 - ICTV (viral)
func codeIDToInt(codeID string) int {
	switch strings.ToUpper(codeID) {
	case "ZOOLOGICAL":
		return 1
	case "BOTANICAL":
		return 2
	case "BACTERIAL":
		return 3
	case "VIRUS":
		return 4
	default:
		return 0
	}
}

// buildOutlinkColumn maps table.column format to query alias for SELECT.
// Returns column expression (e.g., "t.col__id", "n.col__name_id") or empty string if table not available.
//
// Parameters:
//   - outlinkColumn: Format "table.column" (e.g., "taxon.col__id", "name.col__alternative_id")
//   - queryType: One of "taxa", "synonyms", "bare_names"
//
// Table availability by query type:
//   - "taxa":       taxon→t, name→n
//   - "synonyms":   taxon→t (accepted), synonym→s, name→n
//   - "bare_names": name→name (no alias)
//
// Returns empty string if:
//   - outlinkColumn is empty
//   - table not available for the query type
func buildOutlinkColumn(outlinkColumn, queryType string) string {
	if outlinkColumn == "" {
		return ""
	}

	// Parse table.column format
	parts := strings.Split(outlinkColumn, ".")
	if len(parts) != 2 {
		return "" // Invalid format
	}

	tableName := parts[0]
	columnName := parts[1]

	// Map table to query alias based on query type
	var alias string
	switch queryType {
	case "taxa":
		switch tableName {
		case "taxon":
			alias = "t"
		case "name":
			alias = "n"
		default:
			return "" // Table not available in taxa query
		}
	case "synonyms":
		switch tableName {
		case "taxon":
			alias = "t" // Accepted taxon
		case "synonym":
			alias = "s"
		case "name":
			alias = "n"
		default:
			return "" // Table not available in synonyms query
		}
	case "bare_names":
		switch tableName {
		case "name":
			alias = "name" // No alias in bare names query
		default:
			return "" // Only name table available in bare names query
		}
	default:
		return "" // Unknown query type
	}

	return alias + "." + columnName
}
