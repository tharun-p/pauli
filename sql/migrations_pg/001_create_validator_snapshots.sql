CREATE TABLE IF NOT EXISTS validator_snapshots (
    validator_index     BIGINT      NOT NULL,
    slot                BIGINT      NOT NULL,
    status              TEXT        NOT NULL,
    balance             BIGINT      NOT NULL,
    effective_balance   BIGINT      NOT NULL,
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (validator_index, slot)
);

CREATE INDEX IF NOT EXISTS idx_validator_snapshots_slot
    ON validator_snapshots (slot DESC);

