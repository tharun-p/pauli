-- Execution-layer fee columns for block proposer rewards (slot-level; v1 fills priority fees only).
ALTER TABLE block_proposer_rewards
    ADD COLUMN IF NOT EXISTS execution_priority_fees_wei TEXT,
    ADD COLUMN IF NOT EXISTS execution_mev_fees_wei TEXT;

COMMENT ON COLUMN block_proposer_rewards.execution_priority_fees_wei IS 'Sum of effective priority (tip) fees in wei, decimal string.';
COMMENT ON COLUMN block_proposer_rewards.execution_mev_fees_wei IS 'Reserved for future MEV attribution; NULL in v1.';
