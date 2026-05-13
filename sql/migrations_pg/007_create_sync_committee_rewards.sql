CREATE TABLE IF NOT EXISTS sync_committee_rewards (
    validator_index        BIGINT      NOT NULL,
    slot                   BIGINT      NOT NULL,
    reward_gwei            BIGINT      NOT NULL,
    execution_optimistic   BOOLEAN     NOT NULL DEFAULT FALSE,
    finalized              BOOLEAN     NOT NULL DEFAULT FALSE,
    timestamp              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (validator_index, slot)
);

CREATE INDEX IF NOT EXISTS idx_sync_committee_rewards_slot
    ON sync_committee_rewards (slot DESC);
