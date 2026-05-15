package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/tharun/pauli/internal/storage"
)

const unindexedSlotChunkSize = 8192

// MarkSlotIndexed records that the slot indexing pipeline completed for slot.
func (r *Repository) MarkSlotIndexed(ctx context.Context, slot uint64) error {
	return r.markProgress(ctx, storage.ProgressKindSlot, slot)
}

// MarkEpochIndexed records that the epoch indexing pipeline completed for epoch.
func (r *Repository) MarkEpochIndexed(ctx context.Context, epoch uint64) error {
	return r.markProgress(ctx, storage.ProgressKindEpoch, epoch)
}

func (r *Repository) markProgress(ctx context.Context, kind string, position uint64) error {
	const q = `
		INSERT INTO indexer_progress (kind, position, completed_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (kind, position) DO UPDATE SET completed_at = EXCLUDED.completed_at`
	if _, err := r.client.Pool.Exec(ctx, q, kind, position); err != nil {
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
	const q = `SELECT MAX(position) FROM indexer_progress WHERE kind = $1`
	var max *int64
	if err := r.client.Pool.QueryRow(ctx, q, kind).Scan(&max); err != nil {
		return 0, false, fmt.Errorf("max indexer progress %s: %w", kind, err)
	}
	if max == nil {
		return 0, false, nil
	}
	return uint64(*max), true, nil
}

// FirstUnindexedSlot returns the smallest slot in [from, to] without a slot progress row.
func (r *Repository) FirstUnindexedSlot(ctx context.Context, from, to uint64) (uint64, bool, error) {
	if from > to {
		return 0, false, nil
	}
	const q = `
		SELECT MIN(g.slot)::bigint
		FROM generate_series($1::bigint, $2::bigint) AS g(slot)
		LEFT JOIN indexer_progress p
			ON p.kind = $3 AND p.position = g.slot
		WHERE p.position IS NULL`

	for start := from; start <= to; {
		end := start + unindexedSlotChunkSize - 1
		if end > to {
			end = to
		}
		var slot *int64
		if err := r.client.Pool.QueryRow(ctx, q, start, end, storage.ProgressKindSlot).Scan(&slot); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				start = end + 1
				continue
			}
			return 0, false, fmt.Errorf("first unindexed slot [%d,%d]: %w", start, end, err)
		}
		if slot != nil {
			return uint64(*slot), true, nil
		}
		start = end + 1
	}
	return 0, false, nil
}

// FirstUnindexedEpoch returns the smallest epoch in [from, to] without an epoch progress row.
func (r *Repository) FirstUnindexedEpoch(ctx context.Context, from, to uint64) (uint64, bool, error) {
	if from > to {
		return 0, false, nil
	}
	const q = `
		SELECT MIN(g.epoch)::bigint
		FROM generate_series($1::bigint, $2::bigint) AS g(epoch)
		LEFT JOIN indexer_progress p
			ON p.kind = $3 AND p.position = g.epoch
		WHERE p.position IS NULL`

	for start := from; start <= to; {
		end := start + unindexedSlotChunkSize - 1
		if end > to {
			end = to
		}
		var epoch *int64
		if err := r.client.Pool.QueryRow(ctx, q, start, end, storage.ProgressKindEpoch).Scan(&epoch); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				start = end + 1
				continue
			}
			return 0, false, fmt.Errorf("first unindexed epoch [%d,%d]: %w", start, end, err)
		}
		if epoch != nil {
			return uint64(*epoch), true, nil
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
	const q = `SELECT 1 FROM indexer_progress WHERE kind = $1 AND position = $2 LIMIT 1`
	var one int
	err := r.client.Pool.QueryRow(ctx, q, kind, position).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("is indexer progress %s %d: %w", kind, position, err)
	}
	return true, nil
}
