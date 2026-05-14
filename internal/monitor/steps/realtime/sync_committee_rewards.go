package realtime

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/storage"
)

// SyncCommitteeRewards (async): for the canonical head block, fetches sync committee rewards
// for configured validators and persists rows per (validator_index, slot).
// Skips when HeadSlot matches LastProcessedSlot (same dedupe contract as BlockIndexer).
type SyncCommitteeRewards struct {
	Client            *beacon.Client
	Repo              storage.Repository
	Validators        []uint64
	Log               zerolog.Logger
	LastProcessedSlot *uint64
}

var _ Step = (*SyncCommitteeRewards)(nil)

func (*SyncCommitteeRewards) Async() bool { return true }

func (s *SyncCommitteeRewards) Run(e *steps.Env) (bool, error) {
	if s.LastProcessedSlot != nil && e.HeadSlot == *s.LastProcessedSlot {
		return false, nil
	}
	return true, nil
}

func (s *SyncCommitteeRewards) RunAsync(ctx context.Context, e *steps.Env) error {
	if len(s.Validators) == 0 {
		return nil
	}

	blockID := strconv.FormatUint(e.HeadSlot, 10)

	result, err := s.Client.GetSyncCommitteeRewards(ctx, blockID, s.Validators)
	if err != nil {
		if rewardsStateNotYetAvailable(err) {
			s.Log.Warn().Err(err).Uint64("slot", e.HeadSlot).
				Msg("sync committee rewards not available yet; will retry next poll if head unchanged")
			return nil
		}
		return fmt.Errorf("get sync committee rewards: %w", err)
	}

	ts := time.Now().UTC()
	rows := make([]*storage.SyncCommitteeReward, 0, len(result.Rows))
	for _, row := range result.Rows {
		idx := row.ValidatorIndex.Uint64()
		if !validatorIndexWatched(s.Validators, idx) {
			continue
		}
		rows = append(rows, &storage.SyncCommitteeReward{
			ValidatorIndex:      idx,
			Slot:                e.HeadSlot,
			RewardGwei:          row.Reward.Int64(),
			ExecutionOptimistic: result.ExecutionOptimistic,
			Finalized:           result.Finalized,
			Timestamp:           ts,
		})
	}

	if len(rows) > 0 {
		if err := s.Repo.SaveSyncCommitteeRewards(ctx, rows); err != nil {
			return fmt.Errorf("save sync committee rewards: %w", err)
		}
	}

	s.Log.Debug().
		Uint64("slot", e.HeadSlot).
		Int("saved_count", len(rows)).
		Msg("sync committee rewards step completed")

	return nil
}
