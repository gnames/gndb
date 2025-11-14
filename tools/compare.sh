#!/bin/bash
# Convenience wrapper for compare_sources tool
# Reads database credentials from environment variables

set -e

SOURCE_ID="${1:-}"

if [ -z "$SOURCE_ID" ]; then
	echo "Usage: $0 <source-id> [sample-size]"
	echo ""
	echo "Example: $0 1 100"
	echo ""
	echo "Environment variables:"
	echo "  GNDB_DATABASE_HOST     (default: localhost)"
	echo "  GNDB_DATABASE_PORT     (default: 5432)"
	echo "  GNDB_DATABASE_USER     (default: postgres)"
	echo "  GNDB_DATABASE_PASSWORD (required)"
	exit 1
fi

SAMPLE_SIZE="${2:-100}"

# Read from environment or use defaults
DB_HOST="${GNDB_DATABASE_HOST:-localhost}"
DB_PORT="${GNDB_DATABASE_PORT:-5432}"
DB_USER="${GNDB_DATABASE_USER:-postgres}"
DB_PASSWORD="${GNDB_DATABASE_PASSWORD:-}"

if [ -z "$DB_PASSWORD" ]; then
	echo "Error: GNDB_DB_PASSWORD environment variable is required"
	exit 1
fi

# Run the comparison tool
go run "$(dirname "$0")/compare_sources.go" \
	--source-id "$SOURCE_ID" \
	--host "$DB_HOST" \
	--port "$DB_PORT" \
	--user "$DB_USER" \
	--password "$DB_PASSWORD" \
	--sample-size "$SAMPLE_SIZE"
