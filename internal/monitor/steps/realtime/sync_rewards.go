package realtime

import (
	"strconv"

	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/storage"
)

// blockSyncCommitteeRewardsFromBeacon maps a beacon sync committee rewards response to storage JSONB shape.
func blockSyncCommitteeRewardsFromBeacon(result *beacon.SyncCommitteeRewardsResult) *storage.BlockSyncCommitteeRewards {
	if result == nil {
		return nil
	}
	if len(result.Rows) == 0 {
		return &storage.BlockSyncCommitteeRewards{
			ExecutionOptimistic: result.ExecutionOptimistic,
			Finalized:           result.Finalized,
			Rewards:             map[string]int64{},
		}
	}
	rewards := make(map[string]int64, len(result.Rows))
	for _, row := range result.Rows {
		idx := row.ValidatorIndex.Uint64()
		rewards[strconv.FormatUint(idx, 10)] = row.Reward.Int64()
	}
	return &storage.BlockSyncCommitteeRewards{
		ExecutionOptimistic: result.ExecutionOptimistic,
		Finalized:           result.Finalized,
		Rewards:             rewards,
	}
}
