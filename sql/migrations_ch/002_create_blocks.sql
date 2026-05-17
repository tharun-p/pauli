CREATE TABLE IF NOT EXISTS blocks
(
    validator_index              UInt64,
    validator_pubkey             String,
    slot_number                  UInt64,
    block_number                 Nullable(UInt64),
    rewards                      UInt64,
    execution_priority_fees_wei  Nullable(String),
    execution_mev_fees_wei       Nullable(String),
    sync_committee_rewards       Nullable(String),
    timestamp                    DateTime64(3) DEFAULT now64(3)
)
ENGINE = ReplacingMergeTree(timestamp)
ORDER BY (validator_index, slot_number)
SETTINGS storage_policy = 'default'
TTL timestamp + INTERVAL 90 DAY DELETE;
