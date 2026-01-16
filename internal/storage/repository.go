package storage

import (
	"context"
	"fmt"
	"time"
)

// Repository provides data access methods for validator data.
type Repository struct {
	client *Client
}

// NewRepository creates a new Repository with the given client.
func NewRepository(client *Client) *Repository {
	return &Repository{client: client}
}

// SaveValidatorSnapshot saves a validator snapshot to the database.
func (r *Repository) SaveValidatorSnapshot(ctx context.Context, snapshot *ValidatorSnapshot) error {
	query := `
		INSERT INTO validator_snapshots (
			validator_index, slot, status, balance, effective_balance, timestamp
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	if snapshot.Timestamp.IsZero() {
		snapshot.Timestamp = time.Now().UTC()
	}

	err := r.client.Session.Query(query,
		snapshot.ValidatorIndex,
		snapshot.Slot,
		snapshot.Status,
		snapshot.Balance,
		snapshot.EffectiveBalance,
		snapshot.Timestamp,
	).WithContext(ctx).Exec()

	if err != nil {
		return fmt.Errorf("failed to save validator snapshot: %w", err)
	}
	return nil
}

// SaveValidatorSnapshots saves multiple validator snapshots in a batch.
func (r *Repository) SaveValidatorSnapshots(ctx context.Context, snapshots []*ValidatorSnapshot) error {
	batch := r.client.Session.NewBatch(0) // Unlogged batch for performance

	for _, snapshot := range snapshots {
		if snapshot.Timestamp.IsZero() {
			snapshot.Timestamp = time.Now().UTC()
		}
		batch.Query(`
			INSERT INTO validator_snapshots (
				validator_index, slot, status, balance, effective_balance, timestamp
			) VALUES (?, ?, ?, ?, ?, ?)`,
			snapshot.ValidatorIndex,
			snapshot.Slot,
			snapshot.Status,
			snapshot.Balance,
			snapshot.EffectiveBalance,
			snapshot.Timestamp,
		)
	}

	if err := r.client.Session.ExecuteBatch(batch.WithContext(ctx)); err != nil {
		return fmt.Errorf("failed to save validator snapshots batch: %w", err)
	}
	return nil
}

// SaveAttestationDuty saves an attestation duty to the database.
func (r *Repository) SaveAttestationDuty(ctx context.Context, duty *AttestationDuty) error {
	query := `
		INSERT INTO attestation_duties (
			validator_index, epoch, slot, committee_index, committee_position, timestamp
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	if duty.Timestamp.IsZero() {
		duty.Timestamp = time.Now().UTC()
	}

	err := r.client.Session.Query(query,
		duty.ValidatorIndex,
		duty.Epoch,
		duty.Slot,
		duty.CommitteeIndex,
		duty.CommitteePosition,
		duty.Timestamp,
	).WithContext(ctx).Exec()

	if err != nil {
		return fmt.Errorf("failed to save attestation duty: %w", err)
	}
	return nil
}

// SaveAttestationDuties saves multiple attestation duties in a batch.
func (r *Repository) SaveAttestationDuties(ctx context.Context, duties []*AttestationDuty) error {
	batch := r.client.Session.NewBatch(0)

	for _, duty := range duties {
		if duty.Timestamp.IsZero() {
			duty.Timestamp = time.Now().UTC()
		}
		batch.Query(`
			INSERT INTO attestation_duties (
				validator_index, epoch, slot, committee_index, committee_position, timestamp
			) VALUES (?, ?, ?, ?, ?, ?)`,
			duty.ValidatorIndex,
			duty.Epoch,
			duty.Slot,
			duty.CommitteeIndex,
			duty.CommitteePosition,
			duty.Timestamp,
		)
	}

	if err := r.client.Session.ExecuteBatch(batch.WithContext(ctx)); err != nil {
		return fmt.Errorf("failed to save attestation duties batch: %w", err)
	}
	return nil
}

// SaveAttestationReward saves an attestation reward to the database.
func (r *Repository) SaveAttestationReward(ctx context.Context, reward *AttestationReward) error {
	query := `
		INSERT INTO attestation_rewards (
			validator_index, epoch, head_reward, source_reward, target_reward, total_reward, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	if reward.Timestamp.IsZero() {
		reward.Timestamp = time.Now().UTC()
	}

	err := r.client.Session.Query(query,
		reward.ValidatorIndex,
		reward.Epoch,
		reward.HeadReward,
		reward.SourceReward,
		reward.TargetReward,
		reward.TotalReward,
		reward.Timestamp,
	).WithContext(ctx).Exec()

	if err != nil {
		return fmt.Errorf("failed to save attestation reward: %w", err)
	}
	return nil
}

// SaveAttestationRewards saves multiple attestation rewards in a batch.
func (r *Repository) SaveAttestationRewards(ctx context.Context, rewards []*AttestationReward) error {
	batch := r.client.Session.NewBatch(0)

	for _, reward := range rewards {
		if reward.Timestamp.IsZero() {
			reward.Timestamp = time.Now().UTC()
		}
		batch.Query(`
			INSERT INTO attestation_rewards (
				validator_index, epoch, head_reward, source_reward, target_reward, total_reward, timestamp
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			reward.ValidatorIndex,
			reward.Epoch,
			reward.HeadReward,
			reward.SourceReward,
			reward.TargetReward,
			reward.TotalReward,
			reward.Timestamp,
		)
	}

	if err := r.client.Session.ExecuteBatch(batch.WithContext(ctx)); err != nil {
		return fmt.Errorf("failed to save attestation rewards batch: %w", err)
	}
	return nil
}

// SaveValidatorPenalty saves a validator penalty to the database.
func (r *Repository) SaveValidatorPenalty(ctx context.Context, penalty *ValidatorPenalty) error {
	query := `
		INSERT INTO validator_penalties (
			validator_index, epoch, slot, penalty_type, penalty_gwei, timestamp
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	if penalty.Timestamp.IsZero() {
		penalty.Timestamp = time.Now().UTC()
	}

	err := r.client.Session.Query(query,
		penalty.ValidatorIndex,
		penalty.Epoch,
		penalty.Slot,
		penalty.PenaltyType,
		penalty.PenaltyGwei,
		penalty.Timestamp,
	).WithContext(ctx).Exec()

	if err != nil {
		return fmt.Errorf("failed to save validator penalty: %w", err)
	}
	return nil
}

// GetValidatorSnapshots retrieves validator snapshots for a given validator within a slot range.
func (r *Repository) GetValidatorSnapshots(ctx context.Context, validatorIndex uint64, fromSlot, toSlot uint64) ([]*ValidatorSnapshot, error) {
	query := `
		SELECT validator_index, slot, status, balance, effective_balance, timestamp
		FROM validator_snapshots
		WHERE validator_index = ? AND slot >= ? AND slot <= ?
		ORDER BY slot DESC
	`

	iter := r.client.Session.Query(query, validatorIndex, fromSlot, toSlot).WithContext(ctx).Iter()

	var snapshots []*ValidatorSnapshot
	var snapshot ValidatorSnapshot
	for iter.Scan(
		&snapshot.ValidatorIndex,
		&snapshot.Slot,
		&snapshot.Status,
		&snapshot.Balance,
		&snapshot.EffectiveBalance,
		&snapshot.Timestamp,
	) {
		s := snapshot
		snapshots = append(snapshots, &s)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to get validator snapshots: %w", err)
	}
	return snapshots, nil
}

// GetAttestationRewards retrieves attestation rewards for a validator within an epoch range.
func (r *Repository) GetAttestationRewards(ctx context.Context, validatorIndex uint64, fromEpoch, toEpoch uint64) ([]*AttestationReward, error) {
	query := `
		SELECT validator_index, epoch, head_reward, source_reward, target_reward, total_reward, timestamp
		FROM attestation_rewards
		WHERE validator_index = ? AND epoch >= ? AND epoch <= ?
		ORDER BY epoch DESC
	`

	iter := r.client.Session.Query(query, validatorIndex, fromEpoch, toEpoch).WithContext(ctx).Iter()

	var rewards []*AttestationReward
	var reward AttestationReward
	for iter.Scan(
		&reward.ValidatorIndex,
		&reward.Epoch,
		&reward.HeadReward,
		&reward.SourceReward,
		&reward.TargetReward,
		&reward.TotalReward,
		&reward.Timestamp,
	) {
		r := reward
		rewards = append(rewards, &r)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to get attestation rewards: %w", err)
	}
	return rewards, nil
}

// GetLatestSnapshot retrieves the most recent snapshot for a validator.
func (r *Repository) GetLatestSnapshot(ctx context.Context, validatorIndex uint64) (*ValidatorSnapshot, error) {
	query := `
		SELECT validator_index, slot, status, balance, effective_balance, timestamp
		FROM validator_snapshots
		WHERE validator_index = ?
		ORDER BY slot DESC
		LIMIT 1
	`

	var snapshot ValidatorSnapshot
	err := r.client.Session.Query(query, validatorIndex).WithContext(ctx).Scan(
		&snapshot.ValidatorIndex,
		&snapshot.Slot,
		&snapshot.Status,
		&snapshot.Balance,
		&snapshot.EffectiveBalance,
		&snapshot.Timestamp,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get latest snapshot: %w", err)
	}
	return &snapshot, nil
}
