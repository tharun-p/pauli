package indexing

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tharun/pauli/internal/beacon"
)

func TestBlockSyncCommitteeRewardsFromBeacon(t *testing.T) {
	t.Parallel()

	t.Run("nil result", func(t *testing.T) {
		require.Nil(t, blockSyncCommitteeRewardsFromBeacon(nil))
	})

	t.Run("empty rows", func(t *testing.T) {
		got := blockSyncCommitteeRewardsFromBeacon(&beacon.SyncCommitteeRewardsResult{
			ExecutionOptimistic: true,
			Finalized:           false,
		})
		require.NotNil(t, got)
		require.True(t, got.ExecutionOptimistic)
		require.False(t, got.Finalized)
		require.Empty(t, got.Rewards)
	})

	t.Run("maps validator indices to reward gwei", func(t *testing.T) {
		got := blockSyncCommitteeRewardsFromBeacon(&beacon.SyncCommitteeRewardsResult{
			ExecutionOptimistic: false,
			Finalized:           true,
			Rows: []beacon.SyncCommitteeRewardRow{
				{ValidatorIndex: beacon.Uint64Str(100), Reward: beacon.Int64Str(1000)},
				{ValidatorIndex: beacon.Uint64Str(200), Reward: beacon.Int64Str(-50)},
			},
		})
		require.Equal(t, map[string]int64{
			"100": 1000,
			"200": -50,
		}, got.Rewards)
		require.False(t, got.ExecutionOptimistic)
		require.True(t, got.Finalized)
	})
}
