package realtime

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/storage"
)

// ValidatorsBalanceAtSlot (async): full validator snapshot at Env.HeadSlot; body runs on a worker.
// Skips when HeadSlot matches LastProcessedSlot (updated by RecordLastProcessedSlot at chain end).
type ValidatorsBalanceAtSlot struct {
	Client            *beacon.Client
	Repo              storage.Repository
	Validators        []uint64
	Log               zerolog.Logger
	LastProcessedSlot *uint64
}

var _ Step = (*ValidatorsBalanceAtSlot)(nil)

func (ValidatorsBalanceAtSlot) Async() bool { return true }

func (s ValidatorsBalanceAtSlot) Run(e *steps.Env) (bool, error) {
	if s.LastProcessedSlot != nil && e.HeadSlot == *s.LastProcessedSlot {
		return false, nil
	}
	return true, nil
}

func (s ValidatorsBalanceAtSlot) RunAsync(ctx context.Context, e *steps.Env) error {
	return runValidatorSnapshots(ctx, s.Client, s.Repo, s.Validators, e.HeadSlot, s.Log)
}

func runValidatorSnapshots(ctx context.Context, client *beacon.Client, repo storage.Repository, validators []uint64, slot uint64, log zerolog.Logger) error {
	if len(validators) == 0 {
		return nil
	}

	log.Debug().
		Uint64("slot", slot).
		Int("validators_count", len(validators)).
		Msg("fetching validator states at slot (batched)")

	validatorsResp, err := client.GetValidatorsAtSlot(ctx, slot, validators)
	if err != nil {
		log.Error().Err(err).Uint64("slot", slot).Msg("fetch validators at slot failed")
		return fmt.Errorf("failed to fetch validators at slot %d: %w", slot, err)
	}

	byIndex := make(map[uint64]beacon.Validator, len(validatorsResp))
	for i := range validatorsResp {
		v := validatorsResp[i]
		byIndex[v.Index.Uint64()] = v
	}

	now := time.Now().UTC()
	snapshots := make([]*storage.ValidatorSnapshot, 0, len(validators))
	for _, idx := range validators {
		v, ok := byIndex[idx]
		if !ok {
			log.Debug().
				Uint64("validator_index", idx).
				Uint64("slot", slot).
				Msg("validator missing from batched beacon response")
			continue
		}
		snapshots = append(snapshots, &storage.ValidatorSnapshot{
			ValidatorIndex:   idx,
			Slot:             slot,
			Status:           v.Status,
			Balance:          v.Balance.Uint64(),
			EffectiveBalance: v.Validator.EffectiveBalance.Uint64(),
			Timestamp:        now,
		})
	}

	if len(snapshots) == 0 {
		err := fmt.Errorf("no snapshots built for slot %d (requested %d validators)", slot, len(validators))
		log.Error().Err(err).Uint64("slot", slot).Int("validators_requested", len(validators)).Msg("validator snapshot batch empty")
		return err
	}

	log.Debug().
		Uint64("slot", slot).
		Int("snapshots_count", len(snapshots)).
		Msg("saving validator snapshots batch")

	if err := repo.SaveValidatorSnapshots(ctx, snapshots); err != nil {
		log.Error().Err(err).Uint64("slot", slot).Int("count", len(snapshots)).Msg("save validator snapshots failed")
		return fmt.Errorf("failed to save snapshots: %w", err)
	}

	log.Debug().
		Uint64("slot", slot).
		Int("snapshots_count", len(snapshots)).
		Msg("saved validator snapshots batch")

	return nil
}
