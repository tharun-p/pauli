CREATE TABLE IF NOT EXISTS schema_migrations
(
    version    String,
    name       String,
    applied_at DateTime64(3),
    checksum   String
)
ENGINE = ReplacingMergeTree(applied_at)
ORDER BY version
SETTINGS storage_policy = 'default';
