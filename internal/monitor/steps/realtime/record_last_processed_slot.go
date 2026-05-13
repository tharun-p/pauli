package realtime

import (
	"context"

	"github.com/tharun/pauli/internal/monitor/steps"
)

// RecordLastProcessedSlot (sync) runs last in the realtime chain. After all prior
// steps have run and enqueued without error, it stores Env.HeadSlot so the next
// poll can skip re-processing the same head — unless Env.DeferLastProcessedCommit is set
// (e.g. attestation rewards waiting for finalization), in which case the slot cursor is unchanged.
type RecordLastProcessedSlot struct {
	LastProcessedSlot *uint64
}

var _ Step = (*RecordLastProcessedSlot)(nil)

func (*RecordLastProcessedSlot) Async() bool { return false }

func (s *RecordLastProcessedSlot) Run(e *steps.Env) (bool, error) {
	if e.DeferLastProcessedCommit {
		return false, nil
	}
	*s.LastProcessedSlot = e.HeadSlot
	return false, nil
}

func (*RecordLastProcessedSlot) RunAsync(context.Context, *steps.Env) error { return nil }
