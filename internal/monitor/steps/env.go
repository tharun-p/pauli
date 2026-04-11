package steps

import "context"

// Env is shared state for one loop iteration across steps. Runner.Env() returns it; runner.Run resets it each iteration before the step chain.
type Env struct {
	Ctx              context.Context
	HeadSlot         uint64
	ValidatorIndices []uint64
	DutiesEpoch      *uint64
	RewardsEpoch     *uint64
}

// NewEnv allocates an Env (e.g. for a Runner field).
func NewEnv() *Env {
	return &Env{}
}

// Reset clears iteration fields and sets Ctx for one pass over the step chain.
func (e *Env) Reset(ctx context.Context) {
	e.Ctx = ctx
	e.HeadSlot = 0
	e.ValidatorIndices = e.ValidatorIndices[:0]
	e.DutiesEpoch = nil
	e.RewardsEpoch = nil
}

// Clone returns a copy of iteration fields safe to use on a worker after the runner resets Env.
func (e *Env) Clone() Env {
	if e == nil {
		return Env{}
	}
	var de, re *uint64
	if e.DutiesEpoch != nil {
		v := *e.DutiesEpoch
		de = &v
	}
	if e.RewardsEpoch != nil {
		v := *e.RewardsEpoch
		re = &v
	}
	return Env{
		Ctx:              e.Ctx,
		HeadSlot:         e.HeadSlot,
		ValidatorIndices: append([]uint64(nil), e.ValidatorIndices...),
		DutiesEpoch:      de,
		RewardsEpoch:     re,
	}
}
