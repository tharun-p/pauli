package scylladb

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tharun/pauli/internal/storage"
)

// Repository provides data access methods for validator data backed by ScyllaDB.
type Repository struct {
	client *Client
}

// NewRepository creates a new Repository with the given client.
func NewRepository(client *Client) *Repository {
	return &Repository{client: client}
}

// Ensure Repository implements storage.Repository.
var _ storage.Repository = (*Repository)(nil)

// Close closes any resources held by the repository.
func (r *Repository) Close() error {
	// Nothing to close separately from the client.
	return nil
}

// SaveValidatorSnapshot saves a validator snapshot to the database.
func (r *Repository) SaveValidatorSnapshot(ctx context.Context, snapshot *storage.ValidatorSnapshot) error {
	query := `
		INSERT INTO validator_snapshots (
			validator_index, slot, status, balance, effective_balance, timestamp
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	if snapshot.Timestamp.IsZero() {
		snapshot.Timestamp = time.Now().UTC()
	}

	log.Debug().
		Uint64("validator_index", snapshot.ValidatorIndex).
		Uint64("slot", snapshot.Slot).
		Str("status", snapshot.Status).
		Uint64("balance", snapshot.Balance).
		Uint64("effective_balance", snapshot.EffectiveBalance).
		Time("timestamp", snapshot.Timestamp).
		Str("query", query).
		Msg("Executing INSERT query")

	err := r.client.Session.Query(query,
		snapshot.ValidatorIndex,
		snapshot.Slot,
		snapshot.Status,
		snapshot.Balance,
		snapshot.EffectiveBalance,
		snapshot.Timestamp,
	).WithContext(ctx).Exec()

	if err != nil {
		log.Debug().
			Err(err).
			Uint64("validator_index", snapshot.ValidatorIndex).
			Uint64("slot", snapshot.Slot).
			Msg("INSERT query failed")
		return fmt.Errorf("failed to save validator snapshot (validator=%d, slot=%d): %w",
			snapshot.ValidatorIndex, snapshot.Slot, err)
	}

	log.Debug().
		Uint64("validator_index", snapshot.ValidatorIndex).
		Uint64("slot", snapshot.Slot).
		Msg("INSERT query executed successfully")

	return nil
}

// SaveValidatorSnapshots saves multiple validator snapshots in a batch.
func (r *Repository) SaveValidatorSnapshots(ctx context.Context, snapshots []*storage.ValidatorSnapshot) error {
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
func (r *Repository) SaveAttestationDuty(ctx context.Context, duty *storage.AttestationDuty) error {
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
func (r *Repository) SaveAttestationDuties(ctx context.Context, duties []*storage.AttestationDuty) error {
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
func (r *Repository) SaveAttestationReward(ctx context.Context, reward *storage.AttestationReward) error {
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
func (r *Repository) SaveAttestationRewards(ctx context.Context, rewards []*storage.AttestationReward) error {
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
func (r *Repository) SaveValidatorPenalty(ctx context.Context, penalty *storage.ValidatorPenalty) error {
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
func (r *Repository) GetValidatorSnapshots(ctx context.Context, validatorIndex uint64, fromSlot, toSlot uint64) ([]*storage.ValidatorSnapshot, error) {
	query := `
		SELECT validator_index, slot, status, balance, effective_balance, timestamp
		FROM validator_snapshots
		WHERE validator_index = ? AND slot >= ? AND slot <= ?
		ORDER BY slot DESC
	`

	iter := r.client.Session.Query(query, validatorIndex, fromSlot, toSlot).WithContext(ctx).Iter()

	var snapshots []*storage.ValidatorSnapshot
	var snapshot storage.ValidatorSnapshot
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
func (r *Repository) GetAttestationRewards(ctx context.Context, validatorIndex uint64, fromEpoch, toEpoch uint64) ([]*storage.AttestationReward, error) {
	query := `
		SELECT validator_index, epoch, head_reward, source_reward, target_reward, total_reward, timestamp
		FROM attestation_rewards
		WHERE validator_index = ? AND epoch >= ? AND epoch <= ?
		ORDER BY epoch DESC
	`

	iter := r.client.Session.Query(query, validatorIndex, fromEpoch, toEpoch).WithContext(ctx).Iter()

	var rewards []*storage.AttestationReward
	var reward storage.AttestationReward
	for iter.Scan(
		&reward.ValidatorIndex,
		&reward.Epoch,
		&reward.HeadReward,
		&reward.SourceReward,
		&reward.TargetReward,
		&reward.TotalReward,
		&reward.Timestamp,
	) {
		rw := reward
		rewards = append(rewards, &rw)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to get attestation rewards: %w", err)
	}
	return rewards, nil
}

// GetLatestSnapshot retrieves the most recent snapshot for a validator.
func (r *Repository) GetLatestSnapshot(ctx context.Context, validatorIndex uint64) (*storage.ValidatorSnapshot, error) {
	query := `
		SELECT validator_index, slot, status, balance, effective_balance, timestamp
		FROM validator_snapshots
		WHERE validator_index = ?
		ORDER BY slot DESC
		LIMIT 1
	`

	var snapshot storage.ValidatorSnapshot
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

// CountSnapshots returns the total number of snapshots for a validator.
func (r *Repository) CountSnapshots(ctx context.Context, validatorIndex uint64) (int, error) {
	query := `SELECT COUNT(*) FROM validator_snapshots WHERE validator_index = ?`

	var count int
	err := r.client.Session.Query(query, validatorIndex).WithContext(ctx).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count snapshots: %w", err)
	}
	return count, nil
}
