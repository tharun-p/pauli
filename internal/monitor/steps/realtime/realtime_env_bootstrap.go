// Package realtime implements realtime indexing steps (one file per step). Each type implements Step.
// Epoch-boundary math shared by multiple steps lives in epoch_boundary.go.
package realtime

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/monitor/steps"
)

// RealtimeEnvBootstrap is the first step in the realtime monitor chain. It only
// refreshes shared iteration state from the node: current head slot and the
// configured validator index list. Epoch-boundary work is decided by later steps.
type RealtimeEnvBootstrap struct {
	GetHead    func(context.Context) (uint64, error)
	Validators []uint64
	Log        zerolog.Logger
}

var _ Step = (*RealtimeEnvBootstrap)(nil)

func (RealtimeEnvBootstrap) Async() bool { return false }

func (s RealtimeEnvBootstrap) Run(e *steps.Env) (bool, error) {
	head, err := s.GetHead(e.Ctx)
	if err != nil {
		return false, err
	}
	e.HeadSlot = head
	e.ValidatorIndices = append([]uint64(nil), s.Validators...)

	s.Log.Debug().
		Uint64("head_slot", head).
		Int("validators_count", len(e.ValidatorIndices)).
		Msg("realtime: head and validator indices on env")

	return false, nil
}

func (RealtimeEnvBootstrap) RunAsync(context.Context, *steps.Env) error { return nil }
