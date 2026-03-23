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

type StatusHandler struct {
	client *beacon.Client
	repo   storage.Repository
	logger zerolog.Logger
}

func NewStatusHandler(client *beacon.Client, repo storage.Repository, logger zerolog.Logger) *StatusHandler {
	return &StatusHandler{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

// Process fetches and stores validator status.
func (h *StatusHandler) Process(ctx context.Context, job core.Job) (interface{}, error) {
	h.logger.Debug().
		Uint64("validator_index", job.ValidatorIndex).
		Uint64("slot", job.Slot).
		Msg("Fetching validator status from beacon API at specific slot")

	validator, err := h.client.GetValidatorAtSlot(ctx, job.Slot, job.ValidatorIndex)
	if err != nil {
		h.logger.Error().
			Err(err).
			Uint64("validator_index", job.ValidatorIndex).
			Uint64("slot", job.Slot).
			Msg("Failed to fetch validator from beacon API at slot")
		return nil, fmt.Errorf("failed to fetch validator %d: %w", job.ValidatorIndex, err)
	}

	h.logger.Debug().
		Uint64("validator_index", job.ValidatorIndex).
		Str("status", validator.Status).
		Uint64("balance", validator.Balance.Uint64()).
		Uint64("effective_balance", validator.Validator.EffectiveBalance.Uint64()).
		Msg("Successfully fetched validator data from beacon API")

	snapshot := &storage.ValidatorSnapshot{
		ValidatorIndex:   job.ValidatorIndex,
		Slot:             job.Slot,
		Status:           validator.Status,
		Balance:          validator.Balance.Uint64(),
		EffectiveBalance: validator.Validator.EffectiveBalance.Uint64(),
		Timestamp:        time.Now().UTC(),
	}

	h.logger.Info().
		Uint64("validator_index", snapshot.ValidatorIndex).
		Uint64("slot", snapshot.Slot).
		Str("status", snapshot.Status).
		Uint64("balance_gwei", snapshot.Balance).
		Uint64("effective_balance_gwei", snapshot.EffectiveBalance).
		Msg("Prepared snapshot for database")

	h.logger.Debug().
		Uint64("validator_index", snapshot.ValidatorIndex).
		Uint64("slot", snapshot.Slot).
		Msg("Attempting to save validator snapshot to database")

	if err := h.repo.SaveValidatorSnapshot(ctx, snapshot); err != nil {
		h.logger.Error().
			Err(err).
			Uint64("validator_index", job.ValidatorIndex).
			Uint64("slot", snapshot.Slot).
			Uint64("balance", snapshot.Balance).
			Uint64("effective_balance", snapshot.EffectiveBalance).
			Msg("Failed to save validator snapshot")
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	h.logger.Info().
		Uint64("validator_index", snapshot.ValidatorIndex).
		Uint64("slot", snapshot.Slot).
		Uint64("balance", snapshot.Balance).
		Uint64("effective_balance", snapshot.EffectiveBalance).
		Msg("Successfully saved validator snapshot to database")

	return snapshot, nil
}
