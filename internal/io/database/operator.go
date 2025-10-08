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

// HasTables checks if the database has any tables in the public schema.
func (p *PgxOperator) HasTables(ctx context.Context) (bool, error) {
	if p.pool == nil {
		return false, fmt.Errorf("not connected to database")
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
		return false, fmt.Errorf("failed to check for tables: %w", err)
	}

	return hasTables, nil
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
		if _, err := p.pool.Exec(ctx, dropSQL); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

// Pool returns the underlying pgxpool.Pool for advanced operations.
func (p *PgxOperator) Pool() *pgxpool.Pool {
	return p.pool
}
