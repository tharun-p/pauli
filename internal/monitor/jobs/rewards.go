package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/monitor/core"
	"github.com/tharun/pauli/internal/storage"
)

type RewardsHandler struct {
	client     *beacon.Client
	repo       storage.Repository
	validators []uint64
	logger     zerolog.Logger
}

func NewRewardsHandler(client *beacon.Client, repo storage.Repository, validators []uint64, logger zerolog.Logger) *RewardsHandler {
	return &RewardsHandler{
		client:     client,
		repo:       repo,
		validators: validators,
		logger:     logger,
	}
}

// Process fetches and stores attestation rewards.
func (h *RewardsHandler) Process(ctx context.Context, job core.Job) (interface{}, error) {
	resp, err := h.client.GetAttestationRewards(ctx, job.Epoch, h.validators)
	if err != nil {
		return nil, err
	}

	rewards := make([]*storage.AttestationReward, 0, len(resp.TotalRewards))
	var penalties []*storage.ValidatorPenalty

	for _, r := range resp.TotalRewards {
		totalReward := r.Head.Int64() + r.Source.Int64() + r.Target.Int64()

		reward := &storage.AttestationReward{
			ValidatorIndex: r.ValidatorIndex.Uint64(),
			Epoch:          job.Epoch,
			HeadReward:     r.Head.Int64(),
			SourceReward:   r.Source.Int64(),
			TargetReward:   r.Target.Int64(),
			TotalReward:    totalReward,
			Timestamp:      time.Now().UTC(),
		}
		rewards = append(rewards, reward)

		// Track penalties separately.
		if totalReward < 0 {
			penalty := &storage.ValidatorPenalty{
				ValidatorIndex: r.ValidatorIndex.Uint64(),
				Epoch:          job.Epoch,
				Slot:           job.Slot,
				PenaltyType:    storage.PenaltyTypeAttestationMiss,
				PenaltyGwei:    -totalReward,
				Timestamp:      time.Now().UTC(),
			}
			penalties = append(penalties, penalty)
		}
	}

	if err := h.repo.SaveAttestationRewards(ctx, rewards); err != nil {
		h.logger.Error().
			Err(err).
			Uint64("epoch", job.Epoch).
			Int("rewards_count", len(rewards)).
			Msg("Failed to save attestation rewards")
		return nil, fmt.Errorf("failed to save rewards: %w", err)
	}

	h.logger.Info().
		Uint64("epoch", job.Epoch).
		Int("rewards_count", len(rewards)).
		Msg("Successfully saved attestation rewards to database")

	for _, penalty := range penalties {
		if err := h.repo.SaveValidatorPenalty(ctx, penalty); err != nil {
			h.logger.Error().
				Err(err).
				Uint64("validator_index", penalty.ValidatorIndex).
				Msg("Failed to save validator penalty")
		}
	}

	return rewards, nil
}
