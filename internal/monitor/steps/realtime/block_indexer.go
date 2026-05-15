package realtime

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/execution"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/monitor/steps/indexing"
	"github.com/tharun/pauli/internal/storage"
)

// BlockIndexer (async): for each new canonical head slot, fetches proposer metadata, CL block rewards,
// sync committee rewards for all members, and optional EL priority fees, then upserts one row per slot
// (network-wide; not scoped to watched validators).
// Skips when HeadSlot matches LastProcessedSlot (same dedupe contract as other realtime steps).
type BlockIndexer struct {
	Client            *beacon.Client
	Execution         *execution.Client
	Repo              storage.Repository
	Log               zerolog.Logger
	LastProcessedSlot *uint64
}

var _ Step = (*BlockIndexer)(nil)

func (*BlockIndexer) Async() bool { return true }

func (s *BlockIndexer) Run(e *steps.Env) (bool, error) {
	if s.LastProcessedSlot != nil && e.HeadSlot == *s.LastProcessedSlot {
		return false, nil
	}
	return true, nil
}

func (s *BlockIndexer) RunAsync(ctx context.Context, e *steps.Env) error {
	idx := &indexing.BlockIndexer{
		Client:    s.Client,
		Execution: s.Execution,
		Repo:      s.Repo,
		Log:       s.Log,
	}
	if err := indexing.IndexBlockAtSlot(ctx, idx, e.HeadSlot); err != nil {
		return err
	}
	if err := s.Repo.MarkSlotIndexed(ctx, e.HeadSlot); err != nil {
		return err
	}
	return nil
}
