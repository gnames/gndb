package contracts

import (
	"context"
	"database/sql"
)

// DatabaseManager handles core database operations
type DatabaseManager interface {
	// Connect establishes connection to PostgreSQL
	Connect(ctx context.Context, connString string) error

	// Close closes the database connection
	Close() error

	// Ping verifies database connectivity
	Ping(ctx context.Context) error

	// DB returns the underlying *sql.DB for advanced operations
	DB() *sql.DB

	// BeginTx starts a new transaction
	BeginTx(ctx context.Context) (Tx, error)
}

// Tx represents a database transaction
type Tx interface {
	Commit() error
	Rollback() error
	Exec(ctx context.Context, query string, args ...interface{}) error
}

// SchemaCreator creates database schema
type SchemaCreator interface {
	// CreateExtensions creates required PostgreSQL extensions
	CreateExtensions(ctx context.Context) error

	// CreateTables creates all database tables
	CreateTables(ctx context.Context) error

	// CreatePartitions creates hash partitions for name_string_indices
	CreatePartitions(ctx context.Context, count int) error

	// ValidateSchema verifies schema completeness
	ValidateSchema(ctx context.Context) error
}

// Migrator handles schema migrations via Atlas
type Migrator interface {
	// Status returns current migration status
	Status(ctx context.Context) (*MigrationStatus, error)

	// Apply applies pending migrations
	Apply(ctx context.Context) (*MigrationResult, error)

	// Validate validates migration directory integrity
	Validate(ctx context.Context) error
}

// MigrationStatus represents current migration state
type MigrationStatus struct {
	Current string
	Pending []string
	Status  string
}

// MigrationResult represents migration execution result
type MigrationResult struct {
	Applied []string
	Current string
}

// DataPopulator handles bulk data import from SFGA
type DataPopulator interface {
	// ImportDataSource imports a single SFGA data source
	ImportDataSource(ctx context.Context, sfgaPath string) (*ImportStats, error)

	// ValidateSFGA validates SFGA file and version compatibility
	ValidateSFGA(ctx context.Context, sfgaPath string) error

	// GetSFGAVersion returns SFGA schema version
	GetSFGAVersion(sfgaPath string) (string, error)
}

// ImportStats tracks import progress and results
type ImportStats struct {
	DataSourceID      int
	NameStringsCount  int64
	CanonicalsCount   int64
	IndicesCount      int64
	VernacularsCount  int64
	Duration          string
	Errors            []error
}

// Optimizer handles database restructuring and optimization
type Optimizer interface {
	// CreateIndexes creates all secondary indexes
	CreateIndexes(ctx context.Context) error

	// CreateMaterializedViews creates denormalized views
	CreateMaterializedViews(ctx context.Context) error

	// UpdateStatistics runs ANALYZE on all tables
	UpdateStatistics(ctx context.Context) error

	// Vacuum runs VACUUM ANALYZE for table cleanup
	Vacuum(ctx context.Context) error
}
