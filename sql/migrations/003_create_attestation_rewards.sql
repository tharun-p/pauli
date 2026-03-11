-- Migration: 003_create_attestation_rewards
-- Description: Creates the attestation_rewards table for tracking per-epoch rewards breakdown

CREATE TABLE IF NOT EXISTS attestation_rewards (
    validator_index BIGINT,
    epoch           BIGINT,
    head_reward     BIGINT,
    source_reward   BIGINT,
    target_reward   BIGINT,
    total_reward    BIGINT,
    timestamp       TIMESTAMP,
    PRIMARY KEY ((validator_index), epoch)
) WITH CLUSTERING ORDER BY (epoch DESC);
