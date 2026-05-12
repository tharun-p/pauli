package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/tharun/pauli/internal/storage"
)

// PersistTick writes all non-empty bundle sections in one transaction.
func (r *Repository) PersistTick(ctx context.Context, bundle *storage.PersistBundle) error {
	if bundle == nil {
		return fmt.Errorf("nil persist bundle")
	}
	if !bundle.HasWork() {
		return nil
	}

	tx, err := r.client.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin persist transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := persistSnapshots(ctx, tx, bundle.Snapshots); err != nil {
		return err
	}
	if err := persistDuties(ctx, tx, bundle.Duties); err != nil {
		return err
	}
	if err := persistRewards(ctx, tx, bundle.Rewards); err != nil {
		return err
	}
	if err := persistPenalties(ctx, tx, bundle.Penalties); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit persist transaction: %w", err)
	}
	return nil
}

func persistSnapshots(ctx context.Context, tx pgx.Tx, snapshots []*storage.ValidatorSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	const q = `
		INSERT INTO validator_snapshots (
			validator_index, slot, status, balance, effective_balance, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (validator_index, slot) DO UPDATE SET
			status = EXCLUDED.status,
			balance = EXCLUDED.balance,
			effective_balance = EXCLUDED.effective_balance,
			timestamp = EXCLUDED.timestamp
	`
	now := time.Now().UTC()
	batch := &pgx.Batch{}
	for _, s := range snapshots {
		if s.Timestamp.IsZero() {
			s.Timestamp = now
		}
		batch.Queue(q,
			s.ValidatorIndex,
			s.Slot,
			s.Status,
			s.Balance,
			s.EffectiveBalance,
			s.Timestamp,
		)
	}
	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for range snapshots {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("persist validator_snapshots: %w", err)
		}
	}
	return nil
}

func persistDuties(ctx context.Context, tx pgx.Tx, duties []*storage.AttestationDuty) error {
	if len(duties) == 0 {
		return nil
	}
	const q = `
		INSERT INTO attestation_duties (
			validator_index, epoch, slot, committee_index, committee_position, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (validator_index, epoch, slot) DO UPDATE SET
			committee_index = EXCLUDED.committee_index,
			committee_position = EXCLUDED.committee_position,
			timestamp = EXCLUDED.timestamp
	`
	now := time.Now().UTC()
	for _, d := range duties {
		if d.Timestamp.IsZero() {
			d.Timestamp = now
		}
		if _, err := tx.Exec(ctx, q,
			d.ValidatorIndex,
			d.Epoch,
			d.Slot,
			d.CommitteeIndex,
			d.CommitteePosition,
			d.Timestamp,
		); err != nil {
			return fmt.Errorf("persist attestation_duties: %w", err)
		}
	}
	return nil
}

func persistRewards(ctx context.Context, tx pgx.Tx, rewards []*storage.AttestationReward) error {
	if len(rewards) == 0 {
		return nil
	}
	const q = `
		INSERT INTO attestation_rewards (
			validator_index, epoch, head_reward, source_reward, target_reward, total_reward, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (validator_index, epoch) DO UPDATE SET
			head_reward = EXCLUDED.head_reward,
			source_reward = EXCLUDED.source_reward,
			target_reward = EXCLUDED.target_reward,
			total_reward = EXCLUDED.total_reward,
			timestamp = EXCLUDED.timestamp
	`
	now := time.Now().UTC()
	for _, rw := range rewards {
		if rw.Timestamp.IsZero() {
			rw.Timestamp = now
		}
		if _, err := tx.Exec(ctx, q,
			rw.ValidatorIndex,
			rw.Epoch,
			rw.HeadReward,
			rw.SourceReward,
			rw.TargetReward,
			rw.TotalReward,
			rw.Timestamp,
		); err != nil {
			return fmt.Errorf("persist attestation_rewards: %w", err)
		}
	}
	return nil
}

func persistPenalties(ctx context.Context, tx pgx.Tx, penalties []*storage.ValidatorPenalty) error {
	if len(penalties) == 0 {
		return nil
	}
	const q = `
		INSERT INTO validator_penalties (
			validator_index, epoch, slot, penalty_type, penalty_gwei, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (validator_index, epoch, slot) DO UPDATE SET
			penalty_type = EXCLUDED.penalty_type,
			penalty_gwei = EXCLUDED.penalty_gwei,
			timestamp = EXCLUDED.timestamp
	`
	now := time.Now().UTC()
	for _, p := range penalties {
		if p.Timestamp.IsZero() {
			p.Timestamp = now
		}
		if _, err := tx.Exec(ctx, q,
			p.ValidatorIndex,
			p.Epoch,
			p.Slot,
			p.PenaltyType,
			p.PenaltyGwei,
			p.Timestamp,
		); err != nil {
			return fmt.Errorf("persist validator_penalties: %w", err)
		}
	}
	return nil
}
