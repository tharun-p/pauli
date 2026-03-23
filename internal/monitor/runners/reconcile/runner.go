package reconcile

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/config"
)

// Runner handles bounded catch-up for historical slots and epochs.
type Runner struct {
	logger zerolog.Logger

	validators []uint64

	getHead         func(context.Context) (uint64, error)
	getFinalized    func(context.Context) (uint64, error)
	waitForInterval func(context.Context) error

	onPollValidators func(context.Context, uint64, uint64)
	onFetchDuties    func(context.Context, uint64)
	onFetchRewards   func(context.Context, uint64)

	lastSnapshotSlot uint64
	lastDutiesEpoch  uint64
	lastRewardsEpoch uint64
}

func New(
	validators []uint64,
	getHead func(context.Context) (uint64, error),
	getFinalized func(context.Context) (uint64, error),
	waitForInterval func(context.Context) error,
	onPollValidators func(context.Context, uint64, uint64),
	onFetchDuties func(context.Context, uint64),
	onFetchRewards func(context.Context, uint64),
	logger zerolog.Logger,
) *Runner {
	return &Runner{
		logger:           logger,
		validators:       validators,
		getHead:          getHead,
		getFinalized:     getFinalized,
		waitForInterval:  waitForInterval,
		onPollValidators: onPollValidators,
		onFetchDuties:    onFetchDuties,
		onFetchRewards:   onFetchRewards,
	}
}

// InitializeCursors bootstraps in-memory cursors from the current chain state.
func (r *Runner) InitializeCursors(ctx context.Context) {
	headSlot, err := r.getHead(ctx)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to get initial head slot for reconciliation cursors")
		return
	}
	r.lastSnapshotSlot = headSlot
	r.lastDutiesEpoch = headSlot / config.SlotsPerEpoch()

	finalizedEpoch, err := r.getFinalized(ctx)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to get initial finalized epoch for rewards cursor")
		return
	}
	r.lastRewardsEpoch = finalizedEpoch
}

// Run executes reconciliation in a separate loop until caught up.
func (r *Runner) Run(ctx context.Context) {
	r.logger.Info().Msg("Reconciliation loop started")
	defer r.logger.Info().Msg("Reconciliation loop stopped")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		headSlot, err := r.getHead(ctx)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to get head slot during reconciliation")
			time.Sleep(2 * time.Second)
			continue
		}

		currentEpoch := headSlot / config.SlotsPerEpoch()
		r.logger.Info().
			Uint64("slot", headSlot).
			Uint64("epoch", currentEpoch).
			Msg("Reconciling up to head slot")

		pendingSnapshots := r.reconcileSnapshots(ctx, headSlot)
		pendingEpochs := r.reconcileEpochData(ctx, headSlot)
		if !pendingSnapshots && !pendingEpochs {
			r.logger.Info().Msg("Reconciliation caught up; backfill job completed")
			return
		}

		if err := r.waitForInterval(ctx); err != nil && ctx.Err() == nil {
			r.logger.Error().Err(err).Msg("Failed to wait for slot interval in reconciliation loop")
		}
	}
}

func (r *Runner) reconcileSnapshots(ctx context.Context, headSlot uint64) bool {
	const maxSlotsPerLoop = 32
	start := r.lastSnapshotSlot + 1
	if start > headSlot {
		return false
	}

	slotsProcessed := 0
	for slot := start; slot <= headSlot && slotsProcessed < maxSlotsPerLoop; slot++ {
		epoch := slot / config.SlotsPerEpoch()
		r.logger.Debug().
			Uint64("slot", slot).
			Int("validators_count", len(r.validators)).
			Msg("Reconciling validator snapshots for slot")
		r.onPollValidators(ctx, slot, epoch)
		r.lastSnapshotSlot = slot
		slotsProcessed++
	}
	return r.lastSnapshotSlot < headSlot
}

func (r *Runner) reconcileEpochData(ctx context.Context, headSlot uint64) bool {
	currentEpoch := headSlot / config.SlotsPerEpoch()
	finalizedEpoch, err := r.getFinalized(ctx)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to get finality checkpoints during reconciliation")
		return false
	}

	const maxEpochsPerLoop = 8

	dutiesTargetEpoch := currentEpoch + 1
	dutiesProcessed := 0
	for epoch := r.lastDutiesEpoch + 1; epoch <= dutiesTargetEpoch && dutiesProcessed < maxEpochsPerLoop; epoch++ {
		r.logger.Debug().
			Uint64("epoch", epoch).
			Int("validators_count", len(r.validators)).
			Msg("Reconciling attestation duties for epoch")
		r.onFetchDuties(ctx, epoch)
		r.lastDutiesEpoch = epoch
		dutiesProcessed++
	}

	rewardsProcessed := 0
	for epoch := r.lastRewardsEpoch + 1; epoch <= finalizedEpoch && rewardsProcessed < maxEpochsPerLoop; epoch++ {
		r.logger.Debug().
			Uint64("epoch", epoch).
			Int("validators_count", len(r.validators)).
			Msg("Reconciling attestation rewards for epoch")
		r.onFetchRewards(ctx, epoch)
		r.lastRewardsEpoch = epoch
		rewardsProcessed++
	}

	return r.lastDutiesEpoch < dutiesTargetEpoch || r.lastRewardsEpoch < finalizedEpoch
}
