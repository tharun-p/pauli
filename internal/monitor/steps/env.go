package steps

import "context"

// Env is shared state for one loop iteration across steps. Runner.Env() returns it; runner.Run resets it each iteration before the step chain.
type Env struct {
	Ctx              context.Context
	HeadSlot         uint64
	ValidatorIndices []uint64
	// RewardsEpoch is set by AttestationRewards in Run when it enqueues work (cloned into steps.Job for RunAsync).
	RewardsEpoch *uint64
	// DeferLastProcessedCommit, when true, tells RecordLastProcessedSlot not to advance
	// lastProcessedSlot this iteration (e.g. rewards epoch not finalized yet — retry same head next poll).
	DeferLastProcessedCommit bool
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
	e.RewardsEpoch = nil
	e.DeferLastProcessedCommit = false
}

// Clone returns a copy of iteration fields safe to use on a worker after the runner resets Env.
func (e *Env) Clone() Env {
	if e == nil {
		return Env{}
	}
	var re *uint64
	if e.RewardsEpoch != nil {
		v := *e.RewardsEpoch
		re = &v
	}
	return Env{
		Ctx:                      e.Ctx,
		HeadSlot:                 e.HeadSlot,
		ValidatorIndices:         append([]uint64(nil), e.ValidatorIndices...),
		RewardsEpoch:             re,
		DeferLastProcessedCommit: e.DeferLastProcessedCommit,
	}
}
