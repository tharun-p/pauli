CREATE TABLE IF NOT EXISTS block_proposer_rewards (
    validator_index   BIGINT      NOT NULL,
    validator_pubkey  TEXT        NOT NULL,
    slot_number         BIGINT      NOT NULL,
    block_number        BIGINT,
    rewards             BIGINT      NOT NULL,
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (validator_index, slot_number)
);

CREATE INDEX IF NOT EXISTS idx_block_proposer_rewards_slot
    ON block_proposer_rewards (slot_number DESC);
