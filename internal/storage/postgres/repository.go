package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/tharun/pauli/internal/storage"
)

// Repository provides data access methods for validator data backed by PostgreSQL.
type Repository struct {
	client *Client
}

// Ensure Repository implements storage.Repository.
var _ storage.Repository = (*Repository)(nil)

// NewRepository creates a new PostgreSQL-backed Repository.
func NewRepository(client *Client) (storage.Repository, error) {
	return &Repository{client: client}, nil
}

// Close closes any resources held by the repository.
func (r *Repository) Close() error {
	// Nothing to close separately from the client.
	return nil
}

// SaveValidatorSnapshot saves a validator snapshot to the database.
func (r *Repository) SaveValidatorSnapshot(ctx context.Context, snapshot *storage.ValidatorSnapshot) error {
	if snapshot.Timestamp.IsZero() {
		snapshot.Timestamp = time.Now().UTC()
	}

	const query = `
		INSERT INTO validator_snapshots (
			validator_index, slot, status, balance, effective_balance, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (validator_index, slot) DO UPDATE SET
			status = EXCLUDED.status,
			balance = EXCLUDED.balance,
			effective_balance = EXCLUDED.effective_balance,
			timestamp = EXCLUDED.timestamp
	`

	_, err := r.client.Pool.Exec(ctx, query,
		snapshot.ValidatorIndex,
		snapshot.Slot,
		snapshot.Status,
		snapshot.Balance,
		snapshot.EffectiveBalance,
		snapshot.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save validator snapshot (validator=%d, slot=%d): %w",
			snapshot.ValidatorIndex, snapshot.Slot, err)
	}
	return nil
}

// SaveValidatorSnapshots saves multiple validator snapshots in one round trip.
func (r *Repository) SaveValidatorSnapshots(ctx context.Context, snapshots []*storage.ValidatorSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	const query = `
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
	for _, snapshot := range snapshots {
		if snapshot.Timestamp.IsZero() {
			snapshot.Timestamp = now
		}
		batch.Queue(query,
			snapshot.ValidatorIndex,
			snapshot.Slot,
			snapshot.Status,
			snapshot.Balance,
			snapshot.EffectiveBalance,
			snapshot.Timestamp,
		)
	}
	br := r.client.Pool.SendBatch(ctx, batch)
	defer br.Close()
	for range snapshots {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save validator snapshots batch: %w", err)
		}
	}
	return nil
}

// SaveAttestationDuty saves an attestation duty to the database.
func (r *Repository) SaveAttestationDuty(ctx context.Context, duty *storage.AttestationDuty) error {
	if duty.Timestamp.IsZero() {
		duty.Timestamp = time.Now().UTC()
	}

	const query = `
		INSERT INTO attestation_duties (
			validator_index, epoch, slot, committee_index, committee_position, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (validator_index, epoch, slot) DO UPDATE SET
			committee_index = EXCLUDED.committee_index,
			committee_position = EXCLUDED.committee_position,
			timestamp = EXCLUDED.timestamp
	`

	_, err := r.client.Pool.Exec(ctx, query,
		duty.ValidatorIndex,
		duty.Epoch,
		duty.Slot,
		duty.CommitteeIndex,
		duty.CommitteePosition,
		duty.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save attestation duty: %w", err)
	}
	return nil
}

// SaveAttestationDuties saves multiple attestation duties.
func (r *Repository) SaveAttestationDuties(ctx context.Context, duties []*storage.AttestationDuty) error {
	for _, duty := range duties {
		if err := r.SaveAttestationDuty(ctx, duty); err != nil {
			return err
		}
	}
	return nil
}

// SaveAttestationReward saves an attestation reward to the database.
func (r *Repository) SaveAttestationReward(ctx context.Context, reward *storage.AttestationReward) error {
	if reward.Timestamp.IsZero() {
		reward.Timestamp = time.Now().UTC()
	}

	const query = `
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

	_, err := r.client.Pool.Exec(ctx, query,
		reward.ValidatorIndex,
		reward.Epoch,
		reward.HeadReward,
		reward.SourceReward,
		reward.TargetReward,
		reward.TotalReward,
		reward.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save attestation reward: %w", err)
	}
	return nil
}

// SaveAttestationRewards saves multiple attestation rewards.
func (r *Repository) SaveAttestationRewards(ctx context.Context, rewards []*storage.AttestationReward) error {
	for _, reward := range rewards {
		if err := r.SaveAttestationReward(ctx, reward); err != nil {
			return err
		}
	}
	return nil
}

// SaveBlockProposerReward upserts a block proposer reward row.
func (r *Repository) SaveBlockProposerReward(ctx context.Context, row *storage.BlockProposerReward) error {
	if row.Timestamp.IsZero() {
		row.Timestamp = time.Now().UTC()
	}

	const query = `
		INSERT INTO block_proposer_rewards (
			validator_index, validator_pubkey, slot_number, block_number, rewards, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (validator_index, slot_number) DO UPDATE SET
			validator_pubkey = EXCLUDED.validator_pubkey,
			block_number = EXCLUDED.block_number,
			rewards = EXCLUDED.rewards,
			timestamp = EXCLUDED.timestamp
	`

	var blockNum interface{}
	if row.BlockNumber != nil {
		blockNum = *row.BlockNumber
	}

	_, err := r.client.Pool.Exec(ctx, query,
		row.ValidatorIndex,
		row.ValidatorPubkey,
		row.SlotNumber,
		blockNum,
		row.Rewards,
		row.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save block proposer reward: %w", err)
	}
	return nil
}

// SaveBlockProposerRewards saves multiple block proposer reward rows.
func (r *Repository) SaveBlockProposerRewards(ctx context.Context, rows []*storage.BlockProposerReward) error {
	for _, row := range rows {
		if err := r.SaveBlockProposerReward(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// SaveValidatorPenalty saves a validator penalty to the database.
func (r *Repository) SaveValidatorPenalty(ctx context.Context, penalty *storage.ValidatorPenalty) error {
	if penalty.Timestamp.IsZero() {
		penalty.Timestamp = time.Now().UTC()
	}

	const query = `
		INSERT INTO validator_penalties (
			validator_index, epoch, slot, penalty_type, penalty_gwei, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (validator_index, epoch, slot) DO UPDATE SET
			penalty_type = EXCLUDED.penalty_type,
			penalty_gwei = EXCLUDED.penalty_gwei,
			timestamp = EXCLUDED.timestamp
	`

	_, err := r.client.Pool.Exec(ctx, query,
		penalty.ValidatorIndex,
		penalty.Epoch,
		penalty.Slot,
		penalty.PenaltyType,
		penalty.PenaltyGwei,
		penalty.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save validator penalty: %w", err)
	}
	return nil
}

// GetValidatorSnapshots retrieves validator snapshots for a given validator within a slot range.
func (r *Repository) GetValidatorSnapshots(ctx context.Context, validatorIndex uint64, fromSlot, toSlot uint64) ([]*storage.ValidatorSnapshot, error) {
	const query = `
		SELECT validator_index, slot, status, balance, effective_balance, timestamp
		FROM validator_snapshots
		WHERE validator_index = $1 AND slot >= $2 AND slot <= $3
		ORDER BY slot DESC
	`

	rows, err := r.client.Pool.Query(ctx, query, validatorIndex, fromSlot, toSlot)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*storage.ValidatorSnapshot
	for rows.Next() {
		var s storage.ValidatorSnapshot
		if err := rows.Scan(
			&s.ValidatorIndex,
			&s.Slot,
			&s.Status,
			&s.Balance,
			&s.EffectiveBalance,
			&s.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan validator snapshot: %w", err)
		}
		snapshot := s
		snapshots = append(snapshots, &snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate validator snapshots: %w", err)
	}
	return snapshots, nil
}

// GetAttestationRewards retrieves attestation rewards for a validator within an epoch range.
func (r *Repository) GetAttestationRewards(ctx context.Context, validatorIndex uint64, fromEpoch, toEpoch uint64) ([]*storage.AttestationReward, error) {
	const query = `
		SELECT validator_index, epoch, head_reward, source_reward, target_reward, total_reward, timestamp
		FROM attestation_rewards
		WHERE validator_index = $1 AND epoch >= $2 AND epoch <= $3
		ORDER BY epoch DESC
	`

	rows, err := r.client.Pool.Query(ctx, query, validatorIndex, fromEpoch, toEpoch)
	if err != nil {
		return nil, fmt.Errorf("failed to get attestation rewards: %w", err)
	}
	defer rows.Close()

	var rewards []*storage.AttestationReward
	for rows.Next() {
		var rwd storage.AttestationReward
		if err := rows.Scan(
			&rwd.ValidatorIndex,
			&rwd.Epoch,
			&rwd.HeadReward,
			&rwd.SourceReward,
			&rwd.TargetReward,
			&rwd.TotalReward,
			&rwd.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attestation reward: %w", err)
		}
		reward := rwd
		rewards = append(rewards, &reward)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate attestation rewards: %w", err)
	}
	return rewards, nil
}

// GetLatestSnapshot retrieves the most recent snapshot for a validator.
func (r *Repository) GetLatestSnapshot(ctx context.Context, validatorIndex uint64) (*storage.ValidatorSnapshot, error) {
	const query = `
		SELECT validator_index, slot, status, balance, effective_balance, timestamp
		FROM validator_snapshots
		WHERE validator_index = $1
		ORDER BY slot DESC
		LIMIT 1
	`

	var snapshot storage.ValidatorSnapshot
	if err := r.client.Pool.QueryRow(ctx, query, validatorIndex).Scan(
		&snapshot.ValidatorIndex,
		&snapshot.Slot,
		&snapshot.Status,
		&snapshot.Balance,
		&snapshot.EffectiveBalance,
		&snapshot.Timestamp,
	); err != nil {
		return nil, fmt.Errorf("failed to get latest snapshot: %w", err)
	}
	return &snapshot, nil
}

// CountSnapshots returns the total number of snapshots for a validator.
func (r *Repository) CountSnapshots(ctx context.Context, validatorIndex uint64) (int, error) {
	const query = `SELECT COUNT(*) FROM validator_snapshots WHERE validator_index = $1`

	var count int
	if err := r.client.Pool.QueryRow(ctx, query, validatorIndex).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count snapshots: %w", err)
	}
	return count, nil
}
