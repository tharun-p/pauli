package indexing

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/storage"
)

const validatorSnapshotBatchSize = 500

// EpochIndexer fetches all-validator epoch balances and attestation rewards.
type EpochIndexer struct {
	Client *beacon.Client
	Repo   storage.Repository
	Log    zerolog.Logger
}

// IndexEpochAtBoundary snapshots all validators at the epoch start slot and persists attestation rewards.
func IndexEpochAtBoundary(ctx context.Context, idx *EpochIndexer, epoch uint64) error {
	if epoch == 0 {
		return nil
	}
	slot := epoch * config.SlotsPerEpoch()

	validators, err := idx.Client.GetValidatorsAllAtSlot(ctx, slot)
	if err != nil {
		return fmt.Errorf("get all validators at epoch %d slot %d: %w", epoch, slot, err)
	}

	if err := saveValidatorSnapshotsBatched(ctx, idx.Repo, validators, slot, idx.Log); err != nil {
		return err
	}

	if err := fetchAndPersistAllAttestationRewards(ctx, idx.Client, idx.Repo, epoch, idx.Log); err != nil {
		return err
	}

	idx.Log.Debug().Uint64("epoch", epoch).Int("validators", len(validators)).Msg("indexed epoch")
	return nil
}

func saveValidatorSnapshotsBatched(ctx context.Context, repo storage.Repository, validators []beacon.Validator, slot uint64, log zerolog.Logger) error {
	now := time.Now().UTC()
	batch := make([]*storage.ValidatorSnapshot, 0, validatorSnapshotBatchSize)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := repo.SaveValidatorSnapshots(ctx, batch); err != nil {
			return fmt.Errorf("save validator snapshots at slot %d: %w", slot, err)
		}
		batch = batch[:0]
		return nil
	}

	for i := range validators {
		v := validators[i]
		batch = append(batch, &storage.ValidatorSnapshot{
			ValidatorIndex:   v.Index.Uint64(),
			Slot:             slot,
			Status:           v.Status,
			Balance:          v.Balance.Uint64(),
			EffectiveBalance: v.Validator.EffectiveBalance.Uint64(),
			Timestamp:        now,
		})
		if len(batch) >= validatorSnapshotBatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}

func fetchAndPersistAllAttestationRewards(ctx context.Context, client *beacon.Client, repo storage.Repository, epoch uint64, log zerolog.Logger) error {
	resp, err := client.GetAttestationRewards(ctx, epoch, nil)
	if err != nil {
		if rewardsStateNotYetAvailable(err) {
			log.Warn().Err(err).Uint64("epoch", epoch).Msg("attestation rewards not available yet")
			return nil
		}
		return fmt.Errorf("fetch attestation rewards epoch %d: %w", epoch, err)
	}

	rewards := make([]*storage.AttestationReward, 0, len(resp.TotalRewards))

	for _, r := range resp.TotalRewards {
		totalReward := r.Head.Int64() + r.Source.Int64() + r.Target.Int64()
		rewards = append(rewards, &storage.AttestationReward{
			ValidatorIndex: r.ValidatorIndex.Uint64(),
			Epoch:          epoch,
			HeadReward:     r.Head.Int64(),
			SourceReward:   r.Source.Int64(),
			TargetReward:   r.Target.Int64(),
			TotalReward:    totalReward,
			Timestamp:      time.Now().UTC(),
		})
	}

	if err := repo.SaveAttestationRewards(ctx, rewards); err != nil {
		return fmt.Errorf("save attestation rewards epoch %d: %w", epoch, err)
	}
	return nil
}
