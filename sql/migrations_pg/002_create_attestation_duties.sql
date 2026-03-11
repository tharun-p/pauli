CREATE TABLE IF NOT EXISTS attestation_duties (
    validator_index     BIGINT      NOT NULL,
    epoch               BIGINT      NOT NULL,
    slot                BIGINT      NOT NULL,
    committee_index     INTEGER     NOT NULL,
    committee_position  INTEGER     NOT NULL,
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (validator_index, epoch, slot)
);

CREATE INDEX IF NOT EXISTS idx_attestation_duties_epoch
    ON attestation_duties (epoch DESC);

