ALTER TABLE blocks
    ADD COLUMN IF NOT EXISTS sync_committee_rewards JSONB;

UPDATE blocks b
SET sync_committee_rewards = agg.payload
FROM (
    SELECT slot,
        jsonb_build_object(
            'execution_optimistic', bool_or(execution_optimistic),
            'finalized', bool_or(finalized),
            'rewards', jsonb_object_agg(validator_index::text, reward_gwei)
        ) AS payload
    FROM sync_committee_rewards
    GROUP BY slot
) agg
WHERE b.slot_number = agg.slot;

DROP INDEX IF EXISTS idx_sync_committee_rewards_slot;
DROP TABLE IF EXISTS sync_committee_rewards;

CREATE INDEX IF NOT EXISTS idx_blocks_sync_committee_rewards_gin
    ON blocks USING gin ((sync_committee_rewards->'rewards') jsonb_path_ops);
