-- Migration: 002_create_attestation_duties
-- Description: Creates the attestation_duties table for tracking per-epoch duty assignments

CREATE TABLE IF NOT EXISTS attestation_duties (
    validator_index    BIGINT,
    epoch              BIGINT,
    slot               BIGINT,
    committee_index    INT,
    committee_position INT,
    timestamp          TIMESTAMP,
    PRIMARY KEY ((validator_index), epoch, slot)
) WITH CLUSTERING ORDER BY (epoch DESC, slot DESC);
