package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/tharun/pauli/internal/storage"
)

const unindexedChunkSize = 8192

// MarkSlotIndexed records that the slot indexing pipeline completed for slot.
func (r *Repository) MarkSlotIndexed(ctx context.Context, slot uint64) error {
	return r.markProgress(ctx, storage.ProgressKindSlot, slot)
}

// MarkEpochIndexed records that the epoch indexing pipeline completed for epoch.
func (r *Repository) MarkEpochIndexed(ctx context.Context, epoch uint64) error {
	return r.markProgress(ctx, storage.ProgressKindEpoch, epoch)
}

func (r *Repository) markProgress(ctx context.Context, kind string, position uint64) error {
	const q = `INSERT INTO indexer_progress (kind, position, completed_at) VALUES (?, ?, ?)`
	if err := r.client.ExecContext(ctx, q, kind, position, time.Now().UTC()); err != nil {
		return fmt.Errorf("mark indexer progress %s %d: %w", kind, position, err)
	}
	return nil
}

// MaxIndexedSlot returns the highest indexed slot, if any.
func (r *Repository) MaxIndexedSlot(ctx context.Context) (uint64, bool, error) {
	return r.maxProgress(ctx, storage.ProgressKindSlot)
}

// MaxIndexedEpoch returns the highest indexed epoch, if any.
func (r *Repository) MaxIndexedEpoch(ctx context.Context) (uint64, bool, error) {
	return r.maxProgress(ctx, storage.ProgressKindEpoch)
}

func (r *Repository) maxProgress(ctx context.Context, kind string) (uint64, bool, error) {
	const q = `SELECT max(position) FROM indexer_progress FINAL WHERE kind = ?`
	row := r.client.QueryRowContext(ctx, q, kind)
	var max *uint64
	if err := row.Scan(&max); err != nil {
		return 0, false, fmt.Errorf("max indexer progress %s: %w", kind, err)
	}
	if max == nil {
		return 0, false, nil
	}
	return *max, true, nil
}

// FirstUnindexedSlot returns the smallest slot in [from, to] without a slot progress row.
func (r *Repository) FirstUnindexedSlot(ctx context.Context, from, to uint64) (uint64, bool, error) {
	return r.firstUnindexed(ctx, storage.ProgressKindSlot, from, to)
}

// FirstUnindexedEpoch returns the smallest epoch in [from, to] without an epoch progress row.
func (r *Repository) FirstUnindexedEpoch(ctx context.Context, from, to uint64) (uint64, bool, error) {
	return r.firstUnindexed(ctx, storage.ProgressKindEpoch, from, to)
}

func (r *Repository) firstUnindexed(ctx context.Context, kind string, from, to uint64) (uint64, bool, error) {
	if from > to {
		return 0, false, nil
	}

	const q = `
		SELECT min(g.pos) AS pos
		FROM (
			SELECT arrayJoin(range(?, ?)) AS pos
		) AS g
		LEFT JOIN (
			SELECT position FROM indexer_progress FINAL WHERE kind = ?
		) AS p ON p.position = g.pos
		WHERE p.position IS NULL
	`

	for start := from; start <= to; {
		end := start + unindexedChunkSize - 1
		if end > to {
			end = to
		}

		row := r.client.QueryRowContext(ctx, q, start, end+1, kind)
		var pos *uint64
		if err := row.Scan(&pos); err != nil {
			return 0, false, fmt.Errorf("first unindexed %s [%d,%d]: %w", kind, start, end, err)
		}
		if pos != nil {
			return *pos, true, nil
		}
		start = end + 1
	}
	return 0, false, nil
}

// IsSlotIndexed reports whether slot progress exists for slot.
func (r *Repository) IsSlotIndexed(ctx context.Context, slot uint64) (bool, error) {
	return r.isProgress(ctx, storage.ProgressKindSlot, slot)
}

// IsEpochIndexed reports whether epoch progress exists for epoch.
func (r *Repository) IsEpochIndexed(ctx context.Context, epoch uint64) (bool, error) {
	return r.isProgress(ctx, storage.ProgressKindEpoch, epoch)
}

func (r *Repository) isProgress(ctx context.Context, kind string, position uint64) (bool, error) {
	const q = `
		SELECT count() > 0
		FROM indexer_progress FINAL
		WHERE kind = ? AND position = ?
	`
	row := r.client.QueryRowContext(ctx, q, kind, position)
	var ok uint8
	if err := row.Scan(&ok); err != nil {
		return false, fmt.Errorf("is indexer progress %s %d: %w", kind, position, err)
	}
	return ok == 1, nil
}
