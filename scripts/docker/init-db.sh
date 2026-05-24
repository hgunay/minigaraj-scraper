#!/bin/bash
# ============================================================================
# MiniGaraj Scraper - PostgreSQL initialization script
# Runs automatically on first container start via docker-entrypoint-initdb.d
#
# Creates the application user with appropriate permissions.
# The database itself is created by POSTGRES_DB env var.
# The scraper schema + tables are created by golang-migrate migrations.
#
# Author: Hakan Gunay
# Date: 2026-04-04
# ============================================================================

set -e

APP_USER="${APP_USER:-scraper}"
APP_PASSWORD="${APP_PASSWORD:-scraper}"
DB_NAME="${POSTGRES_DB:-minigaraj_scraper}"

echo "=== MiniGaraj Scraper DB Init ==="
echo "Database: $DB_NAME"
echo "App User: $APP_USER"

# Create application user if not exists
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$DB_NAME" <<-EOSQL
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${APP_USER}') THEN
            CREATE USER "${APP_USER}" WITH PASSWORD '${APP_PASSWORD}';
            RAISE NOTICE 'User ${APP_USER} created';
        ELSE
            RAISE NOTICE 'User ${APP_USER} already exists';
        END IF;
    END
    \$\$;

    -- Grant connect
    GRANT CONNECT ON DATABASE "${DB_NAME}" TO "${APP_USER}";

    -- Grant schema usage (scraper schema created by migrations)
    GRANT USAGE ON SCHEMA public TO "${APP_USER}";
    GRANT CREATE ON SCHEMA public TO "${APP_USER}";

    -- Grant CRUD on all current and future tables
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO "${APP_USER}";
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO "${APP_USER}";

    ALTER DEFAULT PRIVILEGES FOR USER "${POSTGRES_USER}" IN SCHEMA public
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "${APP_USER}";
    ALTER DEFAULT PRIVILEGES FOR USER "${POSTGRES_USER}" IN SCHEMA public
        GRANT USAGE, SELECT ON SEQUENCES TO "${APP_USER}";
EOSQL

echo "=== Init complete ==="
