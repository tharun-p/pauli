-- Migration: 000_create_schema_migrations
-- Description: Creates the schema_migrations table for tracking applied migrations

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     TEXT,
    name        TEXT,
    applied_at  TIMESTAMP,
    checksum    TEXT,
    PRIMARY KEY (version)
);
