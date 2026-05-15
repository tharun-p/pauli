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

const validatorEpochRecordBatchSize = 500

// EpochIndexer fetches network-wide epoch balances and attestation rewards into one table.
type EpochIndexer struct {
	Client *beacon.Client
	Repo   storage.Repository
	Log    zerolog.Logger
}

// IndexEpochAtBoundary snapshots all validators at the epoch start slot, merges attestation
// rewards when available, and marks the epoch indexed only after rewards are persisted.
func IndexEpochAtBoundary(ctx context.Context, idx *EpochIndexer, epoch uint64) error {
	if epoch == 0 {
		indexed, err := idx.Repo.IsEpochIndexed(ctx, 0)
		if err != nil {
			return err
		}
		if indexed {
			return nil
		}
		return idx.Repo.MarkEpochIndexed(ctx, 0)
	}

	indexed, err := idx.Repo.IsEpochIndexed(ctx, epoch)
	if err != nil {
		return err
	}
	if indexed {
		return nil
	}

	slot := epoch * config.SlotsPerEpoch()

	validators, err := idx.Client.GetValidatorsAllAtSlot(ctx, slot)
	if err != nil {
		return fmt.Errorf("get all validators at epoch %d slot %d: %w", epoch, slot, err)
	}

	rewardsByIndex, rewardsOK, err := fetchAttestationRewardsByIndex(ctx, idx.Client, epoch, idx.Log)
	if err != nil {
		return err
	}

	records := mergeValidatorEpochRecords(validators, epoch, slot, rewardsByIndex)
	if err := saveValidatorEpochRecordsBatched(ctx, idx.Repo, records); err != nil {
		return err
	}

	if !rewardsOK {
		idx.Log.Debug().Uint64("epoch", epoch).Msg("epoch balances saved; attestation rewards pending")
		return nil
	}

	if err := idx.Repo.MarkEpochIndexed(ctx, epoch); err != nil {
		return fmt.Errorf("mark epoch %d indexed: %w", epoch, err)
	}

	idx.Log.Debug().Uint64("epoch", epoch).Int("validators", len(records)).Msg("indexed epoch")
	return nil
}

func fetchAttestationRewardsByIndex(ctx context.Context, client *beacon.Client, epoch uint64, log zerolog.Logger) (map[uint64]beacon.AttestationReward, bool, error) {
	resp, err := client.GetAttestationRewards(ctx, epoch, nil)
	if err != nil {
		if rewardsStateNotYetAvailable(err) {
			log.Warn().Err(err).Uint64("epoch", epoch).Msg("attestation rewards not available yet")
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("fetch attestation rewards epoch %d: %w", epoch, err)
	}

	out := make(map[uint64]beacon.AttestationReward, len(resp.TotalRewards))
	for _, r := range resp.TotalRewards {
		out[r.ValidatorIndex.Uint64()] = r
	}
	return out, true, nil
}

func mergeValidatorEpochRecords(validators []beacon.Validator, epoch, slot uint64, rewards map[uint64]beacon.AttestationReward) []*storage.ValidatorEpochRecord {
	now := time.Now().UTC()
	records := make([]*storage.ValidatorEpochRecord, 0, len(validators))
	for i := range validators {
		v := validators[i]
		idx := v.Index.Uint64()
		rec := &storage.ValidatorEpochRecord{
			ValidatorIndex:   idx,
			Epoch:            epoch,
			EpochStartSlot:   slot,
			Status:           v.Status,
			Balance:          v.Balance.Uint64(),
			EffectiveBalance: v.Validator.EffectiveBalance.Uint64(),
			IndexedAt:        now,
		}
		if r, ok := rewards[idx]; ok {
			head := r.Head.Int64()
			source := r.Source.Int64()
			target := r.Target.Int64()
			total := head + source + target
			rec.HeadReward = &head
			rec.SourceReward = &source
			rec.TargetReward = &target
			rec.TotalReward = &total
		}
		records = append(records, rec)
	}
	return records
}

func saveValidatorEpochRecordsBatched(ctx context.Context, repo storage.Repository, records []*storage.ValidatorEpochRecord) error {
	for i := 0; i < len(records); i += validatorEpochRecordBatchSize {
		end := i + validatorEpochRecordBatchSize
		if end > len(records) {
			end = len(records)
		}
		if err := repo.SaveValidatorEpochRecords(ctx, records[i:end]); err != nil {
			return err
		}
	}
	return nil
}
