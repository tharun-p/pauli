CREATE TABLE IF NOT EXISTS attestation_rewards (
    validator_index BIGINT      NOT NULL,
    epoch           BIGINT      NOT NULL,
    head_reward     BIGINT      NOT NULL,
    source_reward   BIGINT      NOT NULL,
    target_reward   BIGINT      NOT NULL,
    total_reward    BIGINT      NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (validator_index, epoch)
);

CREATE INDEX IF NOT EXISTS idx_attestation_rewards_epoch
    ON attestation_rewards (epoch DESC);

