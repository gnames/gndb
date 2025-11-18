// Package iodb implements database operations using pgxpool.
// This is an impure I/O package that implements contracts
// defined in pkg/.
package iodb

import (
	"context"
	"fmt"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxOperator implements db.Operator interface using
// pgxpool for connection pooling.
type pgxOperator struct {
	pool *pgxpool.Pool
}

// NewPgxOperator creates a new database operator
// (without connecting).
func NewPgxOperator() db.Operator {
	return &pgxOperator{}
}

// Connect establishes a connection pool to PostgreSQL.
// Uses sensible hardcoded pool settings that work well for
// most use cases.
func (p *pgxOperator) Connect(
	ctx context.Context,
	cfg *config.DatabaseConfig,
) error {
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

	// Configure pool with sensible defaults
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return ConnectionError(cfg.Host, cfg.Port,
			cfg.Database, cfg.User, err)
	}

	// Hardcoded pool settings (can be made configurable
	// later if needed)
	poolConfig.MaxConns = 10       // Max connections
	poolConfig.MinConns = 2        // Keep 2 connections warm
	poolConfig.MaxConnLifetime = 0 // No lifetime limit
	poolConfig.MaxConnIdleTime = 0 // No idle timeout

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return ConnectionError(cfg.Host, cfg.Port,
			cfg.Database, cfg.User, err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return ConnectionError(cfg.Host, cfg.Port,
			cfg.Database, cfg.User, err)
	}

	p.pool = pool
	return nil
}

// Close releases all database connections.
func (p *pgxOperator) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}

// Pool returns the underlying pgxpool.Pool for advanced
// operations.
func (p *pgxOperator) Pool() *pgxpool.Pool {
	return p.pool
}

// TableExists checks if a table exists in the current
// database.
func (p *pgxOperator) TableExists(
	ctx context.Context,
	tableName string,
) (bool, error) {
	if p.pool == nil {
		return false, NotConnectedError()
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
		return false, TableExistsCheckError(tableName, err)
	}

	return exists, nil
}

// HasTables checks if the database has any tables in the
// public schema.
func (p *pgxOperator) HasTables(
	ctx context.Context,
) (bool, error) {
	if p.pool == nil {
		return false, NotConnectedError()
	}

	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
		)
	`

	var hasTables bool
	err := p.pool.QueryRow(ctx, query).Scan(&hasTables)
	if err != nil {
		return false, TableCheckError(err)
	}

	return hasTables, nil
}

// DropAllTables drops all tables in the public schema.
func (p *pgxOperator) DropAllTables(ctx context.Context) error {
	if p.pool == nil {
		return NotConnectedError()
	}

	// Get all table names
	query := `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return QueryTablesError(err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return ScanTableError(err)
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return ScanTableError(err)
	}

	// Drop each table with CASCADE
	for _, table := range tables {
		dropSQL := fmt.Sprintf(
			"DROP TABLE IF EXISTS %s CASCADE", table)
		if _, err := p.pool.Exec(ctx, dropSQL); err != nil {
			return DropTableError(table, err)
		}
	}

	return nil
}

// DropMaterializedViews drops all materialized views in the
// public schema.
func (p *pgxOperator) DropMaterializedViews(
	ctx context.Context,
) error {
	if p.pool == nil {
		return NotConnectedError()
	}

	// Get all materialized view names
	query := `
		SELECT matviewname
		FROM pg_matviews
		WHERE schemaname = 'public'
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return QueryViewsError(err)
	}
	defer rows.Close()

	var views []string
	for rows.Next() {
		var viewName string
		if err := rows.Scan(&viewName); err != nil {
			return ScanViewError(err)
		}
		views = append(views, viewName)
	}

	if err := rows.Err(); err != nil {
		return ScanViewError(err)
	}

	// Drop each materialized view
	for _, view := range views {
		dropSQL := fmt.Sprintf(
			"DROP MATERIALIZED VIEW IF EXISTS %s CASCADE", view)
		if _, err := p.pool.Exec(ctx, dropSQL); err != nil {
			return DropViewError(view, err)
		}
	}

	return nil
}

// CreateMaterializedViews creates all materialized views for
// the database. Currently creates the verification view used
// for fast name lookups.
func (p *pgxOperator) CreateMaterializedViews(
	ctx context.Context,
) error {
	if p.pool == nil {
		return NotConnectedError()
	}

	// Create verification materialized view
	viewSQL := `CREATE MATERIALIZED VIEW verification AS
WITH taxon_names AS (
	SELECT nsi.data_source_id, nsi.record_id,
		nsi.name_string_id, ns.name
	FROM name_string_indices nsi
	JOIN name_strings ns
		ON nsi.name_string_id = ns.id
)
SELECT nsi.data_source_id, nsi.record_id, nsi.name_string_id,
	ns.name, nsi.name_id, nsi.code_id, ns.year, ns.cardinality,
	ns.canonical_id, ns.virus, ns.bacteria, ns.parse_quality,
	nsi.local_id, nsi.outlink_id, nsi.taxonomic_status,
	nsi.accepted_record_id, tn.name_string_id as accepted_name_id,
	tn.name as accepted_name, nsi.classification,
	nsi.classification_ranks, nsi.classification_ids
FROM name_string_indices nsi
JOIN name_strings ns ON ns.id = nsi.name_string_id
LEFT JOIN taxon_names tn
	ON nsi.data_source_id = tn.data_source_id AND
	   nsi.accepted_record_id = tn.record_id
WHERE
	(
		ns.canonical_id is not NULL AND
		surrogate != TRUE AND
		(bacteria != TRUE OR parse_quality < 3)
	) OR ns.virus = TRUE`

	if _, err := p.pool.Exec(ctx, viewSQL); err != nil {
		return CreateViewError("verification", err)
	}

	// Create indexes on verification view
	indexes := []string{
		"CREATE INDEX ON verification (canonical_id)",
		"CREATE INDEX ON verification (name_string_id)",
		"CREATE INDEX ON verification (year)",
	}

	for _, idx := range indexes {
		if _, err := p.pool.Exec(ctx, idx); err != nil {
			return CreateViewIndexError("verification", err)
		}
	}

	return nil
}
