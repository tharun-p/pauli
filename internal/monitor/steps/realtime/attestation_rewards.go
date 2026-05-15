package realtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/storage"
)

// AttestationRewards (async): on a consensus epoch boundary slot, fetches attestation
// rewards for the beacon node's current finalized checkpoint epoch. Skips when
// HeadSlot matches LastProcessedSlot (chain-end dedup).
type AttestationRewards struct {
	Client            *beacon.Client
	Repo              storage.Repository
	Validators        []uint64
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

	e.RewardsEpoch = new(uint64)
	*e.RewardsEpoch = rewardsEpoch

	s.Log.Debug().
		Uint64("head_slot", e.HeadSlot).
		Uint64("finalized_epoch", finalized).
		Uint64("rewards_epoch", rewardsEpoch).
		Msg("realtime: epoch boundary — scheduling attestation rewards for finalized epoch")

	return true, nil
}

func (s *AttestationRewards) RunAsync(ctx context.Context, e *steps.Env) error {
	epoch := *e.RewardsEpoch
	epochStartSlot := epoch * config.SlotsPerEpoch()
	return fetchAndPersistAttestationRewards(ctx, s.Client, s.Repo, s.Validators, epoch, epochStartSlot, s.Log)
}

func fetchAndPersistAttestationRewards(ctx context.Context, client *beacon.Client, repo storage.Repository, validators []uint64, epoch, epochStartSlot uint64, log zerolog.Logger) error {
	resp, err := client.GetAttestationRewards(ctx, epoch, validators)
	if err != nil {
		if rewardsStateNotYetAvailable(err) {
			log.Warn().Err(err).Uint64("epoch", epoch).
				Msg("attestation rewards not available yet (missing state); will retry next poll")
			return nil
		}
		log.Error().Err(err).Uint64("epoch", epoch).Msg("fetch attestation rewards failed")
		return err
	}

	rewards := make([]*storage.AttestationReward, 0, len(resp.TotalRewards))

	for _, r := range resp.TotalRewards {
		totalReward := r.Head.Int64() + r.Source.Int64() + r.Target.Int64()

		rewards = append(rewards, &storage.AttestationReward{
			ValidatorIndex: r.ValidatorIndex.Uint64(),
			Epoch:          epoch,
			HeadReward:     r.Head.Int64(),
			SourceReward:   r.Source.Int64(),
			TargetReward:   r.Target.Int64(),
			TotalReward:    totalReward,
			Timestamp:      time.Now().UTC(),
		})
	}

	if err := repo.SaveAttestationRewards(ctx, rewards); err != nil {
		log.Error().Err(err).Uint64("epoch", epoch).Int("rewards_count", len(rewards)).Msg("save attestation rewards failed")
		return fmt.Errorf("failed to save rewards: %w", err)
	}

	log.Debug().
		Uint64("epoch", epoch).
		Int("rewards_count", len(rewards)).
		Msg("saved attestation rewards")

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
