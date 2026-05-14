-- Canonical name for indexed beacon/EL block rows (one row per proposed slot).
ALTER TABLE IF EXISTS block_proposer_rewards RENAME TO blocks;

ALTER INDEX IF EXISTS idx_block_proposer_rewards_slot RENAME TO idx_blocks_slot;
