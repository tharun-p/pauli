package backfill

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/execution"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/monitor/steps/indexing"
	"github.com/tharun/pauli/internal/storage"
)

// SlotPass indexes up to slots_per_pass unindexed slots behind head-lag.
type SlotPass struct {
	Cfg               config.BackfillConf
	StartSlotOverride *uint64
	EndSlotOverride   *uint64
	Client            *beacon.Client
	Exec              *execution.Client
	Repo              storage.Repository
	GetHead           func(context.Context) (uint64, error)
	Log zerolog.Logger
}

// Run implements steps.Step.
func (s *SlotPass) Run(e *steps.Env) (bool, error) {
	ctx := e.Ctx
	head, err := s.GetHead(ctx)
	if err != nil {
		return false, err
	}

	lag := s.Cfg.LagBehindHead
	var target uint64
	if head > lag {
		target = head - lag
	} else {
		target = 0
	}
	if s.EndSlotOverride != nil && *s.EndSlotOverride < target {
		target = *s.EndSlotOverride
	}

	floor := s.Cfg.StartSlot
	if s.StartSlotOverride != nil && *s.StartSlotOverride > floor {
		floor = *s.StartSlotOverride
	}
	// Do not raise floor from MaxIndexedSlot: realtime may have marked the current head
	// while slots below remain unindexed (e.g. Pauli started at block 97). Gap scan is authoritative.

	if floor > target {
		s.Log.Info().
			Uint64("head_slot", head).
			Uint64("floor", floor).
			Uint64("target_slot", target).
			Msg("backfill: head-lag below start slot; nothing to index yet")
		return false, nil
	}

	first, ok, err := s.Repo.FirstUnindexedSlot(ctx, floor, target)
	if err != nil {
		return false, err
	}
	if !ok {
		s.Log.Info().
			Uint64("head_slot", head).
			Uint64("floor", floor).
			Uint64("target_slot", target).
			Msg("backfill: no unindexed slots in range")
		return false, nil
	}

	idx := &indexing.BlockIndexer{
		Client:    s.Client,
		Execution: s.Exec,
		Repo:      s.Repo,
		Log:       s.Log,
	}

	processed := 0
	for i := 0; i < s.Cfg.SlotsPerPass; i++ {
		slot := first + uint64(i)
		if slot > target {
			break
		}
		done, err := s.Repo.IsSlotIndexed(ctx, slot)
		if err != nil {
			return false, err
		}
		if done {
			continue
		}
		if err := indexing.IndexBlockAtSlot(ctx, idx, slot); err != nil {
			return false, err
		}
		if err := s.Repo.MarkSlotIndexed(ctx, slot); err != nil {
			return false, err
		}
		processed++
	}

	if processed > 0 {
		s.Log.Info().
			Uint64("from_slot", first).
			Int("count", processed).
			Uint64("target_slot", target).
			Msg("backfill: indexed slots")
	}
	return false, nil
}

func (s *SlotPass) Async() bool { return false }

func (s *SlotPass) RunAsync(context.Context, *steps.Env) error { return nil }
