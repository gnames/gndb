package ioschema

import "fmt"

// formatCollationSQL formats the collation SQL statement.
func formatCollationSQL(
	template string,
	table string,
	column string,
	varchar int,
) string {
	return fmt.Sprintf(template, table, column, varchar)
}
