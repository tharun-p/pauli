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

type DutiesHandler struct {
	client     *beacon.Client
	repo       storage.Repository
	validators []uint64
	logger     zerolog.Logger
}

func NewDutiesHandler(client *beacon.Client, repo storage.Repository, validators []uint64, logger zerolog.Logger) *DutiesHandler {
	return &DutiesHandler{
		client:     client,
		repo:       repo,
		validators: validators,
		logger:     logger,
	}
}

// Process fetches and stores attestation duties.
func (h *DutiesHandler) Process(ctx context.Context, job core.Job) (interface{}, error) {
	h.logger.Debug().
		Uint64("epoch", job.Epoch).
		Int("validators_count", len(h.validators)).
		Msg("Fetching attestation duties from beacon API")

	resp, err := h.client.GetAttesterDuties(ctx, job.Epoch, h.validators)
	if err != nil {
		h.logger.Error().
			Err(err).
			Uint64("epoch", job.Epoch).
			Msg("Failed to fetch attestation duties from beacon API")
		return nil, fmt.Errorf("failed to fetch duties for epoch %d: %w", job.Epoch, err)
	}

	h.logger.Debug().
		Uint64("epoch", job.Epoch).
		Int("duties_count", len(resp.Data)).
		Msg("Successfully fetched attestation duties from beacon API")

	if len(resp.Data) == 0 {
		h.logger.Warn().
			Uint64("epoch", job.Epoch).
			Msg("No duties returned for epoch")
		return []*storage.AttestationDuty{}, nil
	}

	duties := make([]*storage.AttestationDuty, 0, len(resp.Data))
	for _, d := range resp.Data {
		duty := &storage.AttestationDuty{
			ValidatorIndex:    d.ValidatorIndex.Uint64(),
			Epoch:             job.Epoch,
			Slot:              d.Slot.Uint64(),
			CommitteeIndex:    int(d.CommitteeIndex.Uint64()),
			CommitteePosition: int(d.ValidatorCommitteeIndex.Uint64()),
			Timestamp:         time.Now().UTC(),
		}
		duties = append(duties, duty)
	}

	h.logger.Info().
		Uint64("epoch", job.Epoch).
		Int("duties_count", len(duties)).
		Msg("Prepared duties for database")

	h.logger.Debug().
		Uint64("epoch", job.Epoch).
		Int("duties_count", len(duties)).
		Msg("Attempting to save attestation duties to database")

	if err := h.repo.SaveAttestationDuties(ctx, duties); err != nil {
		h.logger.Error().
			Err(err).
			Uint64("epoch", job.Epoch).
			Int("duties_count", len(duties)).
			Msg("Failed to save attestation duties")
		return nil, fmt.Errorf("failed to save duties: %w", err)
	}

	h.logger.Info().
		Uint64("epoch", job.Epoch).
		Int("duties_count", len(duties)).
		Msg("Successfully saved attestation duties to database")

	return duties, nil
}
