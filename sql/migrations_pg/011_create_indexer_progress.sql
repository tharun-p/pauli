-- Tracks completed slot and epoch indexing passes (including empty slots with no block).
CREATE TABLE IF NOT EXISTS indexer_progress (
    kind         TEXT        NOT NULL,
    position     BIGINT      NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (kind, position)
);

CREATE INDEX IF NOT EXISTS idx_indexer_progress_slot
    ON indexer_progress (position)
    WHERE kind = 'slot';

CREATE INDEX IF NOT EXISTS idx_indexer_progress_epoch
    ON indexer_progress (position DESC)
    WHERE kind = 'epoch';
