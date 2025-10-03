// Package database implements PostgreSQL database operations using pgxpool.
// This is an impure I/O package that implements contracts defined in pkg/.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/gnames/gndb/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxOperator implements DatabaseOperator interface using pgxpool for connection pooling.
type PgxOperator struct {
	pool *pgxpool.Pool
}

// NewPgxOperator creates a new database operator (without connecting).
func NewPgxOperator() *PgxOperator {
	return &PgxOperator{}
}

// Connect establishes a connection pool to PostgreSQL.
func (p *PgxOperator) Connect(ctx context.Context, cfg *config.DatabaseConfig) error {
	// Build connection string
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)

	// Configure pool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Set pool parameters from config
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MinConns = int32(cfg.MinConnections)
	poolConfig.MaxConnLifetime = time.Duration(cfg.MaxConnLifetime) * time.Minute
	poolConfig.MaxConnIdleTime = time.Duration(cfg.MaxConnIdleTime) * time.Minute

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.pool = pool
	return nil
}

// Close releases all database connections.
func (p *PgxOperator) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}

// CreateSchema creates all base tables from DDL definitions.
func (p *PgxOperator) CreateSchema(ctx context.Context, ddlStatements []string, force bool) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	// If force=true, drop all tables first
	if force {
		if err := p.DropAllTables(ctx); err != nil {
			return fmt.Errorf("failed to drop existing tables: %w", err)
		}
	}

	// Execute all DDL statements in a transaction
	return p.ExecuteDDLBatch(ctx, ddlStatements)
}

// TableExists checks if a table exists in the current database.
func (p *PgxOperator) TableExists(ctx context.Context, tableName string) (bool, error) {
	if p.pool == nil {
		return false, fmt.Errorf("not connected to database")
	}

	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`

	var exists bool
	err := p.pool.QueryRow(ctx, query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}

	return exists, nil
}

// DropAllTables drops all tables in the public schema.
func (p *PgxOperator) DropAllTables(ctx context.Context) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	// Get all table names
	query := `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating table rows: %w", err)
	}

	// Drop each table with CASCADE
	for _, table := range tables {
		dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)
		if err := p.ExecuteDDL(ctx, dropSQL); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

// ExecuteDDL executes a single DDL statement in a transaction.
func (p *PgxOperator) ExecuteDDL(ctx context.Context, ddl string) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("failed to execute DDL: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExecuteDDLBatch executes multiple DDL statements in a single transaction.
func (p *PgxOperator) ExecuteDDLBatch(ctx context.Context, ddlStatements []string) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for i, ddl := range ddlStatements {
		if _, err := tx.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("failed to execute DDL statement %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetSchemaVersion returns the current schema version.
func (p *PgxOperator) GetSchemaVersion(ctx context.Context) (string, error) {
	if p.pool == nil {
		return "", fmt.Errorf("not connected to database")
	}

	// Check if schema_versions table exists
	exists, err := p.TableExists(ctx, "schema_versions")
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}

	query := `SELECT version FROM schema_versions ORDER BY applied_at DESC LIMIT 1`

	var version string
	err = p.pool.QueryRow(ctx, query).Scan(&version)
	if err != nil {
		// No rows is not an error - just means no version set yet
		if err.Error() == "no rows in result set" {
			return "", nil
		}
		return "", fmt.Errorf("failed to get schema version: %w", err)
	}

	return version, nil
}

// SetSchemaVersion inserts or updates the current schema version.
func (p *PgxOperator) SetSchemaVersion(ctx context.Context, version, description string) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	query := `
		INSERT INTO schema_versions (version, description, applied_at)
		VALUES ($1, $2, NOW())
	`

	if _, err := p.pool.Exec(ctx, query, version, description); err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	return nil
}

// EnableExtension enables a PostgreSQL extension.
func (p *PgxOperator) EnableExtension(ctx context.Context, extensionName string) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	ddl := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", extensionName)
	return p.ExecuteDDL(ctx, ddl)
}

// VacuumAnalyze runs VACUUM ANALYZE on specified tables.
func (p *PgxOperator) VacuumAnalyze(ctx context.Context, tableNames []string) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	for _, table := range tableNames {
		sql := fmt.Sprintf("VACUUM ANALYZE %s", table)
		if _, err := p.pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("failed to vacuum analyze %s: %w", table, err)
		}
	}

	return nil
}

// CreateIndexConcurrently creates an index without blocking writes.
func (p *PgxOperator) CreateIndexConcurrently(ctx context.Context, indexDDL string) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	// CONCURRENTLY cannot be run inside a transaction
	if _, err := p.pool.Exec(ctx, indexDDL); err != nil {
		return fmt.Errorf("failed to create index concurrently: %w", err)
	}

	return nil
}

// RefreshMaterializedView refreshes a materialized view.
func (p *PgxOperator) RefreshMaterializedView(ctx context.Context, viewName string, concurrently bool) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	sql := "REFRESH MATERIALIZED VIEW"
	if concurrently {
		sql += " CONCURRENTLY"
	}
	sql += fmt.Sprintf(" %s", viewName)

	if _, err := p.pool.Exec(ctx, sql); err != nil {
		return fmt.Errorf("failed to refresh materialized view %s: %w", viewName, err)
	}

	return nil
}

// SetStatisticsTarget sets the statistics target for a column.
func (p *PgxOperator) SetStatisticsTarget(ctx context.Context, tableName, columnName string, target int) error {
	if p.pool == nil {
		return fmt.Errorf("not connected to database")
	}

	sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET STATISTICS %d", tableName, columnName, target)
	return p.ExecuteDDL(ctx, sql)
}

// GetDatabaseSize returns the total size of the database in bytes.
func (p *PgxOperator) GetDatabaseSize(ctx context.Context) (int64, error) {
	if p.pool == nil {
		return 0, fmt.Errorf("not connected to database")
	}

	query := `SELECT pg_database_size(current_database())`

	var size int64
	err := p.pool.QueryRow(ctx, query).Scan(&size)
	if err != nil {
		return 0, fmt.Errorf("failed to get database size: %w", err)
	}

	return size, nil
}

// GetTableSize returns the total size of a table (including indexes) in bytes.
func (p *PgxOperator) GetTableSize(ctx context.Context, tableName string) (int64, error) {
	if p.pool == nil {
		return 0, fmt.Errorf("not connected to database")
	}

	query := `SELECT pg_total_relation_size($1)`

	var size int64
	err := p.pool.QueryRow(ctx, query, tableName).Scan(&size)
	if err != nil {
		return 0, fmt.Errorf("failed to get table size for %s: %w", tableName, err)
	}

	return size, nil
}
