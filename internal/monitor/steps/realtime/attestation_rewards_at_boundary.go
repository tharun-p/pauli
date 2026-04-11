package realtime

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/storage"
)

// AttestationRewardsAtBoundary (async): rewards for RewardsEpoch from Env; body runs on a worker.
type AttestationRewardsAtBoundary struct {
	Client     *beacon.Client
	Repo       storage.Repository
	Validators []uint64
	Log        zerolog.Logger
}

var _ Step = (*AttestationRewardsAtBoundary)(nil)

func (AttestationRewardsAtBoundary) Async() bool { return true }

func (AttestationRewardsAtBoundary) Run(e *steps.Env) (bool, error) {
	if e.RewardsEpoch == nil {
		return false, nil
	}
	return true, nil
}

func (s AttestationRewardsAtBoundary) RunAsync(ctx context.Context, e *steps.Env) error {
	epoch := *e.RewardsEpoch
	epochStartSlot := epoch * config.SlotsPerEpoch()
	return runAttestationRewards(ctx, s.Client, s.Repo, s.Validators, epoch, epochStartSlot, s.Log)
}

func runAttestationRewards(ctx context.Context, client *beacon.Client, repo storage.Repository, validators []uint64, epoch, epochStartSlot uint64, log zerolog.Logger) error {
	resp, err := client.GetAttestationRewards(ctx, epoch, validators)
	if err != nil {
		log.Debug().Err(err).Uint64("epoch", epoch).Msg("fetch attestation rewards failed")
		return err
	}

	rewards := make([]*storage.AttestationReward, 0, len(resp.TotalRewards))
	var penalties []*storage.ValidatorPenalty

	for _, r := range resp.TotalRewards {
		totalReward := r.Head.Int64() + r.Source.Int64() + r.Target.Int64()

		reward := &storage.AttestationReward{
			ValidatorIndex: r.ValidatorIndex.Uint64(),
			Epoch:          epoch,
			HeadReward:     r.Head.Int64(),
			SourceReward:   r.Source.Int64(),
			TargetReward:   r.Target.Int64(),
			TotalReward:    totalReward,
			Timestamp:      time.Now().UTC(),
		}
		rewards = append(rewards, reward)

		if totalReward < 0 {
			penalty := &storage.ValidatorPenalty{
				ValidatorIndex: r.ValidatorIndex.Uint64(),
				Epoch:          epoch,
				Slot:           epochStartSlot,
				PenaltyType:    storage.PenaltyTypeAttestationMiss,
				PenaltyGwei:    -totalReward,
				Timestamp:      time.Now().UTC(),
			}
			penalties = append(penalties, penalty)
		}
	}

	if err := repo.SaveAttestationRewards(ctx, rewards); err != nil {
		log.Debug().Err(err).Uint64("epoch", epoch).Int("rewards_count", len(rewards)).Msg("save attestation rewards failed")
		return fmt.Errorf("failed to save rewards: %w", err)
	}

	log.Debug().
		Uint64("epoch", epoch).
		Int("rewards_count", len(rewards)).
		Msg("saved attestation rewards")

	for _, penalty := range penalties {
		if err := repo.SaveValidatorPenalty(ctx, penalty); err != nil {
			log.Debug().Err(err).Uint64("validator_index", penalty.ValidatorIndex).Msg("save validator penalty failed")
		}
	}

	return nil
}
