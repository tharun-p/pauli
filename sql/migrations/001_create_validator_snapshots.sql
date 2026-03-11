-- Migration: 001_create_validator_snapshots
-- Description: Creates the validator_snapshots table for tracking per-slot balance and status

CREATE TABLE IF NOT EXISTS validator_snapshots (
    validator_index BIGINT,
    slot            BIGINT,
    status          TEXT,
    balance         BIGINT,
    effective_balance BIGINT,
    timestamp       TIMESTAMP,
    PRIMARY KEY ((validator_index), slot)
) WITH CLUSTERING ORDER BY (slot DESC);
