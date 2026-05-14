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

// BlockProposerRewards (async): when the canonical head block was proposed by one
// of the configured validators, fetches block rewards and persists a row.
// Skips when HeadSlot matches LastProcessedSlot (same dedupe contract as other realtime steps).
type BlockProposerRewards struct {
	Client            *beacon.Client
	Execution         *execution.Client
	Repo              storage.Repository
	Validators        []uint64
	Log               zerolog.Logger
	LastProcessedSlot *uint64
}

var _ Step = (*BlockProposerRewards)(nil)

func (*BlockProposerRewards) Async() bool { return true }

func (s *BlockProposerRewards) Run(e *steps.Env) (bool, error) {
	if s.LastProcessedSlot != nil && e.HeadSlot == *s.LastProcessedSlot {
		return false, nil
	}
	return true, nil
}

func (s *BlockProposerRewards) RunAsync(ctx context.Context, e *steps.Env) error {
	if len(s.Validators) == 0 {
		return nil
	}

	blockID := strconv.FormatUint(e.HeadSlot, 10)

	header, err := s.Client.GetBlockHeader(ctx, blockID)
	if err != nil {
		if beacon.IsNotFound(err) {
			s.Log.Debug().Err(err).Uint64("slot", e.HeadSlot).Msg("realtime: block header not found for proposer check")
			return nil
		}
		return fmt.Errorf("block header for proposer rewards: %w", err)
	}

	proposerIndex := header.Data.Header.Message.ProposerIndex.Uint64()
	if !validatorIndexWatched(s.Validators, proposerIndex) {
		return nil
	}

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

	row := &storage.BlockProposerReward{
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

	if err := s.Repo.SaveBlockProposerReward(ctx, row); err != nil {
		return fmt.Errorf("save block proposer reward: %w", err)
	}

	s.Log.Debug().
		Uint64("slot", e.HeadSlot).
		Uint64("validator_index", proposerIndex).
		Uint64("rewards_gwei", row.Rewards).
		Msg("saved block proposer reward")

	return nil
}

func validatorIndexWatched(validators []uint64, index uint64) bool {
	for _, v := range validators {
		if v == index {
			return true
		}
	}
	return false
}
