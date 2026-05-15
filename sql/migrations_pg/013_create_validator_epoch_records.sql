CREATE TABLE IF NOT EXISTS validator_epoch_records (
    validator_index       BIGINT      NOT NULL,
    epoch                 BIGINT      NOT NULL,
    epoch_start_slot      BIGINT      NOT NULL,
    status                TEXT        NOT NULL,
    balance               BIGINT      NOT NULL,
    effective_balance     BIGINT      NOT NULL,
    head_reward           BIGINT,
    source_reward         BIGINT,
    target_reward         BIGINT,
    total_reward          BIGINT,
    indexed_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (validator_index, epoch)
);

CREATE INDEX IF NOT EXISTS idx_validator_epoch_records_epoch
    ON validator_epoch_records (epoch DESC);

CREATE INDEX IF NOT EXISTS idx_validator_epoch_records_validator_epoch
    ON validator_epoch_records (validator_index, epoch DESC);
