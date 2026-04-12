// Package realtime implements realtime indexing steps (one file per step). Each type implements Step.
package realtime

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/monitor/steps"
)

// GetValidatorDetails (sync): head, validator copy, epoch-boundary duties/rewards plan; updates LastEpoch.
type GetValidatorDetails struct {
	GetHead    func(context.Context) (uint64, error)
	Validators []uint64
	Log        zerolog.Logger
	LastEpoch  *uint64
}

var _ Step = (*GetValidatorDetails)(nil)

func (GetValidatorDetails) Async() bool { return false }

func (s GetValidatorDetails) Run(e *steps.Env) (bool, error) {
	head, err := s.GetHead(e.Ctx)
	if err != nil {
		return false, err
	}
	e.HeadSlot = head
	e.ValidatorIndices = append([]uint64(nil), s.Validators...)

	headEpoch := head / config.SlotsPerEpoch()
	duties, rewards, newLast, ok := computeBoundaryWork(head, headEpoch, *s.LastEpoch)
	if !ok {
		e.DutiesEpoch = nil
		e.RewardsEpoch = nil
		return false, nil
	}

	*s.LastEpoch = newLast
	e.DutiesEpoch = new(uint64)
	*e.DutiesEpoch = *duties
	if rewards != nil {
		e.RewardsEpoch = new(uint64)
		*e.RewardsEpoch = *rewards
	} else {
		e.RewardsEpoch = nil
	}

	s.Log.Debug().
		Uint64("head_slot", head).
		Uint64("head_epoch", headEpoch).
		Interface("duties_epoch", e.DutiesEpoch).
		Interface("rewards_epoch", e.RewardsEpoch).
		Msg("realtime: epoch boundary planned")

	return false, nil
}

func (GetValidatorDetails) RunAsync(context.Context, *steps.Env) error { return nil }

func computeBoundaryWork(headSlot, headEpoch, lastEpoch uint64) (duties *uint64, rewards *uint64, newLast uint64, ok bool) {
	sp := config.SlotsPerEpoch()
	first := headSlot%sp == 0
	last := (headSlot+1)%sp == 0
	if !first && !last {
		return nil, nil, 0, false
	}
	if headEpoch == lastEpoch {
		return nil, nil, 0, false
	}

	nextEpoch := headEpoch + 1
	duties = new(uint64)
	*duties = nextEpoch

	if headEpoch > 0 {
		r := headEpoch - 1
		rewards = new(uint64)
		*rewards = r
	}

	if first {
		newLast = headEpoch
	} else {
		newLast = headEpoch + 1
	}

	return duties, rewards, newLast, true
}
