package realtime

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/storage"
)

// Persist (sync): waits for async indexing jobs for this tick, then commits Env.Bundle in one DB transaction.
type Persist struct {
	AwaitAsync func()
	Repo       storage.Repository
	Log        zerolog.Logger
}

var _ Step = (*Persist)(nil)

func (Persist) Async() bool { return false }

func (Persist) RunAsync(context.Context, *steps.Env) error { return nil }

func (s Persist) Run(e *steps.Env) (bool, error) {
	if s.AwaitAsync != nil {
		s.AwaitAsync()
	}
	if e == nil || e.Bundle == nil {
		s.Log.Error().Msg("persist: nil env or bundle")
		return false, nil
	}
	b := e.Bundle
	if err := b.AsyncError(); err != nil {
		s.Log.Warn().Err(err).Msg("persist skipped: async step failed; aborting tick")
		return false, err
	}
	if !b.HasWork() {
		s.Log.Debug().Msg("persist skipped: empty bundle")
		return false, nil
	}
	ctx := e.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.Repo.PersistTick(ctx, b); err != nil {
		s.Log.Error().Err(err).Msg("persist transaction failed")
		return false, err
	}
	s.Log.Debug().
		Int("snapshots", len(b.Snapshots)).
		Int("duties", len(b.Duties)).
		Int("rewards", len(b.Rewards)).
		Int("penalties", len(b.Penalties)).
		Msg("persist committed")
	return false, nil
}
