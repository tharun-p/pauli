package dispatch

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/monitor/core"
)

// Dispatcher converts monitoring events into worker jobs.
type Dispatcher struct {
	validators []uint64
	submit     func(context.Context, core.Job) error
	logger     zerolog.Logger
}

func New(validators []uint64, submit func(context.Context, core.Job) error, logger zerolog.Logger) *Dispatcher {
	return &Dispatcher{
		validators: validators,
		submit:     submit,
		logger:     logger,
	}
}

func (d *Dispatcher) PollValidatorsForSlotEpoch(ctx context.Context, slot uint64, epoch uint64) {
	d.logger.Info().
		Uint64("slot", slot).
		Uint64("epoch", epoch).
		Int("validators_count", len(d.validators)).
		Msg("Polling validators")

	for _, validatorIndex := range d.validators {
		job := core.Job{
			ValidatorIndex: validatorIndex,
			Slot:           slot,
			Epoch:          epoch,
			Type:           core.JobTypeStatus,
		}

		d.logger.Debug().
			Uint64("validator_index", validatorIndex).
			Uint64("slot", slot).
			Msg("Submitting validator status job")

		select {
		case <-ctx.Done():
			return
		default:
			if err := d.submit(ctx, job); err != nil {
				d.logger.Debug().Err(err).Msg("Skipping status job submit due to context cancellation")
				return
			}
		}
	}
}

func (d *Dispatcher) FetchDutiesForEpoch(ctx context.Context, epoch uint64) {
	slot := epoch * config.SlotsPerEpoch()
	d.logger.Info().
		Uint64("slot", slot).
		Uint64("epoch", epoch).
		Int("validators_count", len(d.validators)).
		Msg("Fetching attestation duties for epoch")

	job := core.Job{
		ValidatorIndex: 0,
		Slot:           slot,
		Epoch:          epoch,
		Type:           core.JobTypeDuties,
	}

	d.logger.Debug().
		Uint64("epoch", epoch).
		Uint64("slot", slot).
		Msg("Submitting attestation duties job")

	select {
	case <-ctx.Done():
		return
	default:
		if err := d.submit(ctx, job); err != nil {
			d.logger.Debug().Err(err).Msg("Skipping duties job submit due to context cancellation")
		}
	}
}

func (d *Dispatcher) FetchRewardsForEpoch(ctx context.Context, epoch uint64) {
	slot := epoch * config.SlotsPerEpoch()
	d.logger.Info().
		Uint64("slot", slot).
		Uint64("epoch", epoch).
		Int("validators_count", len(d.validators)).
		Msg("Fetching attestation rewards for finalized epoch")

	job := core.Job{
		ValidatorIndex: 0,
		Slot:           slot,
		Epoch:          epoch,
		Type:           core.JobTypeRewards,
	}

	select {
	case <-ctx.Done():
		return
	default:
		if err := d.submit(ctx, job); err != nil {
			d.logger.Debug().Err(err).Msg("Skipping rewards job submit due to context cancellation")
		}
	}
}
