-- Legacy tables superseded by validator_epoch_records (013). Safe on fresh installs (IF EXISTS).
DROP TABLE IF EXISTS attestation_duties;
DROP TABLE IF EXISTS validator_snapshots;
DROP TABLE IF EXISTS attestation_rewards;
