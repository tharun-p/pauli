-- Migration: 004_create_validator_penalties
-- Description: Creates the validator_penalties table for tracking slashing and inactivity penalties

CREATE TABLE IF NOT EXISTS validator_penalties (
    validator_index BIGINT,
    epoch           BIGINT,
    slot            BIGINT,
    penalty_type    TEXT,
    penalty_gwei    BIGINT,
    timestamp       TIMESTAMP,
    PRIMARY KEY ((validator_index), epoch, slot)
) WITH CLUSTERING ORDER BY (epoch DESC, slot DESC);
