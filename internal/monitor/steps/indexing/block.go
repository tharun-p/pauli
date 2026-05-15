package indexing

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/execution"
	"github.com/tharun/pauli/internal/storage"
)

// BlockIndexer indexes one canonical block at slot (network-wide).
type BlockIndexer struct {
	Client    *beacon.Client
	Execution *execution.Client
	Repo      storage.Repository
	Log       zerolog.Logger
}

// IndexBlockAtSlot fetches and persists block metadata, CL rewards, and sync committee rewards.
// Missing block header (404) is not an error — empty slots are valid.
func IndexBlockAtSlot(ctx context.Context, idx *BlockIndexer, slot uint64) error {
	blockID := strconv.FormatUint(slot, 10)

	header, err := idx.Client.GetBlockHeader(ctx, blockID)
	if err != nil {
		if beacon.IsNotFound(err) {
			idx.Log.Debug().Err(err).Uint64("slot", slot).Msg("block header not found; empty slot")
			return nil
		}
		return fmt.Errorf("block header slot %d: %w", slot, err)
	}

	proposerIndex := header.Data.Header.Message.ProposerIndex.Uint64()

	rewardsData, err := idx.Client.GetBlockRewards(ctx, blockID)
	if err != nil {
		if rewardsStateNotYetAvailable(err) {
			idx.Log.Warn().Err(err).Uint64("slot", slot).Msg("block rewards not available yet")
			return nil
		}
		return fmt.Errorf("get block rewards slot %d: %w", slot, err)
	}

	validatorsResp, err := idx.Client.GetValidatorsAtSlot(ctx, slot, []uint64{proposerIndex})
	if err != nil {
		return fmt.Errorf("get proposer validator at slot %d: %w", slot, err)
	}
	if len(validatorsResp) == 0 {
		return fmt.Errorf("no validator state for proposer %d at slot %d", proposerIndex, slot)
	}
	pubkey := validatorsResp[0].Validator.Pubkey

	var execBlock *uint64
	execBlock, err = idx.Client.GetBlockExecutionBlockNumber(ctx, blockID)
	if err != nil {
		if beacon.IsNotFound(err) || rewardsStateNotYetAvailable(err) {
			idx.Log.Debug().Err(err).Uint64("slot", slot).Msg("execution block number not available")
		} else {
			idx.Log.Warn().Err(err).Uint64("slot", slot).Msg("get execution block number failed")
		}
		execBlock = nil
	}

	row := &storage.Block{
		ValidatorIndex:  proposerIndex,
		ValidatorPubkey: pubkey,
		SlotNumber:      slot,
		BlockNumber:     execBlock,
		Rewards:         rewardsData.Total.Uint64(),
		Timestamp:       time.Now().UTC(),
	}

	if idx.Execution != nil && execBlock != nil {
		prio, err := idx.Execution.PriorityFeesWeiDecimalString(ctx, *execBlock)
		if err != nil {
			idx.Log.Warn().Err(err).Uint64("slot", slot).Uint64("block_number", *execBlock).
				Msg("execution priority fees fetch failed")
		} else {
			row.ExecutionPriorityFeesWei = &prio
		}
	}

	syncResult, err := idx.Client.GetSyncCommitteeRewards(ctx, blockID, nil)
	if err != nil {
		if rewardsStateNotYetAvailable(err) {
			idx.Log.Warn().Err(err).Uint64("slot", slot).Msg("sync committee rewards not available yet")
		} else {
			return fmt.Errorf("get sync committee rewards slot %d: %w", slot, err)
		}
	} else {
		row.SyncCommitteeRewards = blockSyncCommitteeRewardsFromBeacon(syncResult)
	}

	if err := idx.Repo.SaveBlock(ctx, row); err != nil {
		return fmt.Errorf("save block slot %d: %w", slot, err)
	}

	syncCount := 0
	if row.SyncCommitteeRewards != nil {
		syncCount = len(row.SyncCommitteeRewards.Rewards)
	}
	idx.Log.Debug().
		Uint64("slot", slot).
		Uint64("validator_index", proposerIndex).
		Uint64("rewards_gwei", row.Rewards).
		Int("sync_committee_rewards", syncCount).
		Msg("saved indexed block")

	return nil
}

func rewardsStateNotYetAvailable(err error) bool {
	if err == nil {
		return false
	}
	if beacon.IsNotFound(err) {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "404") || strings.Contains(s, "NOT_FOUND") || strings.Contains(s, "missing state")
}
