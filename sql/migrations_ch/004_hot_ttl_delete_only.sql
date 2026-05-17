-- Hot retention only: drop S3 tiering; delete rows older than 90 days (for DBs created with 001-003 tiered TTL).
ALTER TABLE validator_epoch_records
    MODIFY SETTING storage_policy = 'default';

ALTER TABLE validator_epoch_records
    MODIFY TTL indexed_at + INTERVAL 90 DAY DELETE;

ALTER TABLE blocks
    MODIFY SETTING storage_policy = 'default';

ALTER TABLE blocks
    MODIFY TTL timestamp + INTERVAL 90 DAY DELETE;
