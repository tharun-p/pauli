package realtime

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/execution"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/storage"
)

// BlockIndexer (async): for each new canonical head slot, fetches proposer metadata, CL block rewards,
// sync committee rewards for all members, and optional EL priority fees, then upserts one row per slot
// (network-wide; not scoped to watched validators).
// Skips when HeadSlot matches LastProcessedSlot (same dedupe contract as other realtime steps).
type BlockIndexer struct {
	Client            *beacon.Client
	Execution         *execution.Client
	Repo              storage.Repository
	Log               zerolog.Logger
	LastProcessedSlot *uint64
}

var _ Step = (*BlockIndexer)(nil)

func (*BlockIndexer) Async() bool { return true }

func (s *BlockIndexer) Run(e *steps.Env) (bool, error) {
	if s.LastProcessedSlot != nil && e.HeadSlot == *s.LastProcessedSlot {
		return false, nil
	}
	return true, nil
}

func (s *BlockIndexer) RunAsync(ctx context.Context, e *steps.Env) error {
	blockID := strconv.FormatUint(e.HeadSlot, 10)

	header, err := s.Client.GetBlockHeader(ctx, blockID)
	if err != nil {
		if beacon.IsNotFound(err) {
			s.Log.Debug().Err(err).Uint64("slot", e.HeadSlot).Msg("realtime: block header not found for block indexer")
			return nil
		}
		return fmt.Errorf("block header for block indexer: %w", err)
	}

	proposerIndex := header.Data.Header.Message.ProposerIndex.Uint64()

	rewardsData, err := s.Client.GetBlockRewards(ctx, blockID)
	if err != nil {
		if rewardsStateNotYetAvailable(err) {
			s.Log.Warn().Err(err).Uint64("slot", e.HeadSlot).
				Msg("block rewards not available yet; will retry next poll if head unchanged")
			return nil
		}
		return fmt.Errorf("get block rewards: %w", err)
	}

	validatorsResp, err := s.Client.GetValidatorsAtSlot(ctx, e.HeadSlot, []uint64{proposerIndex})
	if err != nil {
		return fmt.Errorf("get proposer validator at slot: %w", err)
	}
	if len(validatorsResp) == 0 {
		return fmt.Errorf("no validator state for proposer index %d at slot %d", proposerIndex, e.HeadSlot)
	}
	pubkey := validatorsResp[0].Validator.Pubkey

	var execBlock *uint64
	execBlock, err = s.Client.GetBlockExecutionBlockNumber(ctx, blockID)
	if err != nil {
		if beacon.IsNotFound(err) || rewardsStateNotYetAvailable(err) {
			s.Log.Debug().Err(err).Uint64("slot", e.HeadSlot).Msg("execution block number not available; storing null")
		} else {
			s.Log.Warn().Err(err).Uint64("slot", e.HeadSlot).Msg("get execution block number failed; storing null")
		}
		execBlock = nil
	}

	row := &storage.Block{
		ValidatorIndex:  proposerIndex,
		ValidatorPubkey: pubkey,
		SlotNumber:      e.HeadSlot,
		BlockNumber:     execBlock,
		Rewards:         rewardsData.Total.Uint64(),
		Timestamp:       time.Now().UTC(),
	}

	if s.Execution != nil && execBlock != nil {
		prio, err := s.Execution.PriorityFeesWeiDecimalString(ctx, *execBlock)
		if err != nil {
			s.Log.Warn().Err(err).Uint64("slot", e.HeadSlot).Uint64("block_number", *execBlock).
				Msg("execution priority fees fetch failed; storing null EL columns")
		} else {
			row.ExecutionPriorityFeesWei = &prio
		}
	}

	syncResult, err := s.Client.GetSyncCommitteeRewards(ctx, blockID, nil)
	if err != nil {
		if rewardsStateNotYetAvailable(err) {
			s.Log.Warn().Err(err).Uint64("slot", e.HeadSlot).
				Msg("sync committee rewards not available yet; saving block without sync rewards")
		} else {
			return fmt.Errorf("get sync committee rewards: %w", err)
		}
	} else {
		row.SyncCommitteeRewards = blockSyncCommitteeRewardsFromBeacon(syncResult)
	}

	if err := s.Repo.SaveBlock(ctx, row); err != nil {
		return fmt.Errorf("save block: %w", err)
	}

	syncCount := 0
	if row.SyncCommitteeRewards != nil {
		syncCount = len(row.SyncCommitteeRewards.Rewards)
	}
	s.Log.Debug().
		Uint64("slot", e.HeadSlot).
		Uint64("validator_index", proposerIndex).
		Uint64("rewards_gwei", row.Rewards).
		Int("sync_committee_rewards", syncCount).
		Msg("saved indexed block")

	return nil
}
