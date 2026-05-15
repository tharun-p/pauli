package realtime

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/monitor/steps/indexing"
	"github.com/tharun/pauli/internal/storage"
)

// AttestationRewards (async): on a consensus epoch boundary slot, indexes network-wide
// validator epoch records (balances + attestation rewards) for the finalized epoch.
type AttestationRewards struct {
	Client            *beacon.Client
	Repo              storage.Repository
	Log               zerolog.Logger
	LastProcessedSlot *uint64
}

var _ Step = (*AttestationRewards)(nil)

func (AttestationRewards) Async() bool { return true }

func (s *AttestationRewards) Run(e *steps.Env) (bool, error) {
	if s.LastProcessedSlot != nil && e.HeadSlot == *s.LastProcessedSlot {
		e.RewardsEpoch = nil
		return false, nil
	}

	headEpoch := e.HeadSlot / config.SlotsPerEpoch()
	if !isConsensusEpochBoundarySlot(e.HeadSlot) || headEpoch == 0 {
		e.RewardsEpoch = nil
		return false, nil
	}

	finalized, err := s.Client.FinalizedEpoch(e.Ctx)
	if err != nil {
		return false, err
	}

	rewardsEpoch := finalized
	indexed, err := s.Repo.IsEpochIndexed(e.Ctx, rewardsEpoch)
	if err != nil {
		return false, err
	}
	if indexed {
		e.RewardsEpoch = nil
		return false, nil
	}

	e.RewardsEpoch = new(uint64)
	*e.RewardsEpoch = rewardsEpoch

	s.Log.Debug().
		Uint64("head_slot", e.HeadSlot).
		Uint64("finalized_epoch", finalized).
		Uint64("rewards_epoch", rewardsEpoch).
		Msg("realtime: epoch boundary — scheduling network-wide epoch index")

	return true, nil
}

func (s *AttestationRewards) RunAsync(ctx context.Context, e *steps.Env) error {
	epoch := *e.RewardsEpoch
	return indexing.IndexEpochAtBoundary(ctx, &indexing.EpochIndexer{
		Client: s.Client,
		Repo:   s.Repo,
		Log:    s.Log,
	}, epoch)
}
