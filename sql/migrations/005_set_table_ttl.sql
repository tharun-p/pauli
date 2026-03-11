-- Migration: 005_set_table_ttl
-- Description: Sets default TTL on all tables (placeholder - TTL is set dynamically)
-- Note: This migration is a marker. Actual TTL is applied via ALTER TABLE with config value.

-- TTL will be applied programmatically based on config.scylladb.ttl_days
-- This file exists for migration tracking purposes.
-- Using a valid no-op CQL statement
SELECT now() FROM system.local;
