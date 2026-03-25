package gndb

// Exporter defines the interface for exporting gnames PostgreSQL data
// to SFGA SQLite format files.
type Exporter interface {
	// Export reads data from PostgreSQL for the configured source IDs
	// and writes one SFGA .sqlite file per source into the output directory.
	Export() error
}
