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

// ValidatorDuties (async): attester duties for DutiesEpoch from Env; body runs on a worker.
type ValidatorDuties struct {
	Client     *beacon.Client
	Repo       storage.Repository
	Validators []uint64
	Log        zerolog.Logger
}

var _ Step = (*ValidatorDuties)(nil)

func (ValidatorDuties) Async() bool { return true }

func (ValidatorDuties) Run(e *steps.Env) (bool, error) {
	if e.DutiesEpoch == nil {
		return false, nil
	}
	return true, nil
}

func (s ValidatorDuties) RunAsync(ctx context.Context, e *steps.Env) error {
	epoch := *e.DutiesEpoch
	return runAttesterDuties(ctx, s.Client, s.Repo, s.Validators, epoch, s.Log)
}

func runAttesterDuties(ctx context.Context, client *beacon.Client, repo storage.Repository, validators []uint64, epoch uint64, log zerolog.Logger) error {
	log.Debug().
		Uint64("epoch", epoch).
		Int("validators_count", len(validators)).
		Msg("fetching attestation duties")

	resp, err := client.GetAttesterDuties(ctx, epoch, validators)
	if err != nil {
		log.Error().Err(err).Uint64("epoch", epoch).Msg("fetch attestation duties failed")
		return fmt.Errorf("failed to fetch duties for epoch %d: %w", epoch, err)
	}

	log.Debug().
		Uint64("epoch", epoch).
		Int("duties_count", len(resp.Data)).
		Msg("fetched attestation duties")

	if len(resp.Data) == 0 {
		log.Debug().Uint64("epoch", epoch).Msg("no duties returned for epoch")
		return nil
	}

	duties := make([]*storage.AttestationDuty, 0, len(resp.Data))
	for _, d := range resp.Data {
		duty := &storage.AttestationDuty{
			ValidatorIndex:    d.ValidatorIndex.Uint64(),
			Epoch:             epoch,
			Slot:              d.Slot.Uint64(),
			CommitteeIndex:    int(d.CommitteeIndex.Uint64()),
			CommitteePosition: int(d.ValidatorCommitteeIndex.Uint64()),
			Timestamp:         time.Now().UTC(),
		}
		duties = append(duties, duty)
	}

	log.Debug().
		Uint64("epoch", epoch).
		Int("duties_count", len(duties)).
		Msg("saving attestation duties")

	if err := repo.SaveAttestationDuties(ctx, duties); err != nil {
		log.Error().Err(err).Uint64("epoch", epoch).Int("duties_count", len(duties)).Msg("save attestation duties failed")
		return fmt.Errorf("failed to save duties: %w", err)
	}

	log.Debug().
		Uint64("epoch", epoch).
		Int("duties_count", len(duties)).
		Msg("saved attestation duties")

	return nil
}
