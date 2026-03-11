CREATE TABLE IF NOT EXISTS validator_penalties (
    validator_index BIGINT      NOT NULL,
    epoch           BIGINT      NOT NULL,
    slot            BIGINT      NOT NULL,
    penalty_type    TEXT        NOT NULL,
    penalty_gwei    BIGINT      NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (validator_index, epoch, slot)
);

CREATE INDEX IF NOT EXISTS idx_validator_penalties_epoch
    ON validator_penalties (epoch DESC);

