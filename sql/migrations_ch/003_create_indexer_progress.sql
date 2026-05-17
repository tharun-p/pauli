CREATE TABLE IF NOT EXISTS indexer_progress
(
    kind         LowCardinality(String),
    position     UInt64,
    completed_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = ReplacingMergeTree(completed_at)
ORDER BY (kind, position)
SETTINGS storage_policy = 'default';
