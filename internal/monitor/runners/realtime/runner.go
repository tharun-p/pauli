package realtime

import (
	"context"

	"github.com/rs/zerolog"
)

// Runner executes scheduler-driven monitoring events.
type Runner struct {
	waitForInterval func(context.Context) error
	getHead         func(context.Context) (uint64, error)
	handleForSlot   func(context.Context, uint64) error
	logger          zerolog.Logger
}

func New(
	waitForInterval func(context.Context) error,
	getHead func(context.Context) (uint64, error),
	handleForSlot func(context.Context, uint64) error,
	logger zerolog.Logger,
) *Runner {
	return &Runner{
		waitForInterval: waitForInterval,
		getHead:         getHead,
		handleForSlot:   handleForSlot,
		logger:          logger,
	}
}

func (r *Runner) Run(ctx context.Context) {
	r.logger.Info().Msg("Realtime monitor loop started")

	for {
		if err := r.waitForInterval(ctx); err != nil {
			if ctx.Err() != nil {
				r.logger.Info().Msg("Realtime monitor loop shutting down")
				return
			}
			r.logger.Error().Err(err).Msg("Failed to wait for slot")
			continue
		}

		headSlot, err := r.getHead(ctx)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to get head slot")
			continue
		}

		if err := r.handleForSlot(ctx, headSlot); err != nil {
			r.logger.Error().Err(err).Uint64("slot", headSlot).Msg("Failed to schedule realtime events")
			continue
		}
	}
}
