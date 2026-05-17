CREATE TABLE IF NOT EXISTS validator_epoch_records
(
    validator_index     UInt64,
    epoch               UInt64,
    epoch_start_slot    UInt64,
    status              LowCardinality(String),
    balance             UInt64,
    effective_balance   UInt64,
    head_reward         Nullable(Int64),
    source_reward       Nullable(Int64),
    target_reward       Nullable(Int64),
    total_reward        Nullable(Int64),
    indexed_at          DateTime64(3) DEFAULT now64(3)
)
ENGINE = ReplacingMergeTree(indexed_at)
ORDER BY (validator_index, epoch)
SETTINGS storage_policy = 'default'
TTL indexed_at + INTERVAL 90 DAY DELETE;
