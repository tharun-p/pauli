package backfill

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/monitor/steps/indexing"
	"github.com/tharun/pauli/internal/storage"
)

// EpochPass indexes up to epochs_per_pass unindexed finalized epochs.
type EpochPass struct {
	Cfg                config.BackfillConf
	StartEpochOverride *uint64
	EndEpochOverride   *uint64
	Client             *beacon.Client
	Repo               storage.Repository
	Log zerolog.Logger
}

// Run implements steps.Step.
func (s *EpochPass) Run(e *steps.Env) (bool, error) {
	ctx := e.Ctx

	finalized, err := s.Client.FinalizedEpoch(ctx)
	if err != nil {
		return false, err
	}

	head, err := s.Client.GetHeadSlot(ctx)
	if err != nil {
		return false, err
	}
	headEpoch := head / config.SlotsPerEpoch()
	targetEpoch := finalized
	if headEpoch < targetEpoch {
		targetEpoch = headEpoch
	}
	if s.EndEpochOverride != nil && *s.EndEpochOverride < targetEpoch {
		targetEpoch = *s.EndEpochOverride
	}

	floor := s.Cfg.StartEpoch
	if s.StartEpochOverride != nil && *s.StartEpochOverride > floor {
		floor = *s.StartEpochOverride
	}
	// Do not raise floor from MaxIndexedEpoch; same gap-fill rationale as SlotPass.

	if floor > targetEpoch {
		s.Log.Info().
			Uint64("head_slot", head).
			Uint64("floor_epoch", floor).
			Uint64("target_epoch", targetEpoch).
			Msg("backfill: no epochs to index in range yet")
		return false, nil
	}

	first, ok, err := s.Repo.FirstUnindexedEpoch(ctx, floor, targetEpoch)
	if err != nil {
		return false, err
	}
	if !ok {
		s.Log.Info().
			Uint64("floor_epoch", floor).
			Uint64("target_epoch", targetEpoch).
			Msg("backfill: no unindexed epochs in range")
		return false, nil
	}

	idx := &indexing.EpochIndexer{
		Client: s.Client,
		Repo:   s.Repo,
		Log:    s.Log,
	}

	processed := 0
	for i := 0; i < s.Cfg.EpochsPerPass; i++ {
		epoch := first + uint64(i)
		if epoch > targetEpoch {
			break
		}
		done, err := s.Repo.IsEpochIndexed(ctx, epoch)
		if err != nil {
			return false, err
		}
		if done {
			continue
		}
		if err := indexing.IndexEpochAtBoundary(ctx, idx, epoch); err != nil {
			return false, err
		}
		done, err = s.Repo.IsEpochIndexed(ctx, epoch)
		if err != nil {
			return false, err
		}
		if !done {
			continue
		}
		processed++
	}

	if processed > 0 {
		s.Log.Info().
			Uint64("from_epoch", first).
			Int("count", processed).
			Uint64("target_epoch", targetEpoch).
			Msg("backfill: indexed epochs")
	}
	return false, nil
}

func (s *EpochPass) Async() bool { return false }

func (s *EpochPass) RunAsync(context.Context, *steps.Env) error { return nil }
