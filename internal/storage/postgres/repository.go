package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

// SaveBlock upserts one indexed block row (canonical proposer at slot).
func (r *Repository) SaveBlock(ctx context.Context, row *storage.Block) error {
	if row.Timestamp.IsZero() {
		row.Timestamp = time.Now().UTC()
	}

	const query = `
		INSERT INTO blocks (
			validator_index, validator_pubkey, slot_number, block_number, rewards,
			execution_priority_fees_wei, execution_mev_fees_wei, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (validator_index, slot_number) DO UPDATE SET
			validator_pubkey = EXCLUDED.validator_pubkey,
			block_number = EXCLUDED.block_number,
			rewards = EXCLUDED.rewards,
			execution_priority_fees_wei = EXCLUDED.execution_priority_fees_wei,
			execution_mev_fees_wei = EXCLUDED.execution_mev_fees_wei,
			timestamp = EXCLUDED.timestamp
	`

	var blockNum interface{}
	if row.BlockNumber != nil {
		blockNum = *row.BlockNumber
	}
	var priWei interface{}
	if row.ExecutionPriorityFeesWei != nil {
		priWei = *row.ExecutionPriorityFeesWei
	}
	var mevWei interface{}
	if row.ExecutionMevFeesWei != nil {
		mevWei = *row.ExecutionMevFeesWei
	}

	_, err := r.client.Pool.Exec(ctx, query,
		row.ValidatorIndex,
		row.ValidatorPubkey,
		row.SlotNumber,
		blockNum,
		row.Rewards,
		priWei,
		mevWei,
		row.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}
	return nil
}

// SaveBlocks saves multiple indexed block rows.
func (r *Repository) SaveBlocks(ctx context.Context, rows []*storage.Block) error {
	for _, row := range rows {
		if err := r.SaveBlock(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

// SaveSyncCommitteeReward upserts a sync committee reward row.
func (r *Repository) SaveSyncCommitteeReward(ctx context.Context, row *storage.SyncCommitteeReward) error {
	if row.Timestamp.IsZero() {
		row.Timestamp = time.Now().UTC()
	}

	const query = `
		INSERT INTO sync_committee_rewards (
			validator_index, slot, reward_gwei, execution_optimistic, finalized, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (validator_index, slot) DO UPDATE SET
			reward_gwei = EXCLUDED.reward_gwei,
			execution_optimistic = EXCLUDED.execution_optimistic,
			finalized = EXCLUDED.finalized,
			timestamp = EXCLUDED.timestamp
	`

	_, err := r.client.Pool.Exec(ctx, query,
		row.ValidatorIndex,
		row.Slot,
		row.RewardGwei,
		row.ExecutionOptimistic,
		row.Finalized,
		row.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save sync committee reward: %w", err)
	}
	return nil
}

// SaveSyncCommitteeRewards saves multiple sync committee reward rows.
func (r *Repository) SaveSyncCommitteeRewards(ctx context.Context, rows []*storage.SyncCommitteeReward) error {
	for _, row := range rows {
		if err := r.SaveSyncCommitteeReward(ctx, row); err != nil {
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

// ListValidatorSnapshots returns snapshots for a validator in a slot range with pagination (newest slots first).
func (r *Repository) ListValidatorSnapshots(ctx context.Context, validatorIndex, fromSlot, toSlot uint64, limit, offset int) ([]*storage.ValidatorSnapshot, error) {
	const query = `
		SELECT validator_index, slot, status, balance, effective_balance, timestamp
		FROM validator_snapshots
		WHERE validator_index = $1 AND slot >= $2 AND slot <= $3
		ORDER BY slot DESC
		LIMIT $4 OFFSET $5
	`

	rows, err := r.client.Pool.Query(ctx, query, validatorIndex, fromSlot, toSlot, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list validator snapshots: %w", err)
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

// ListAttestationRewards returns attestation rewards for an epoch range, optionally filtered to one validator.
func (r *Repository) ListAttestationRewards(ctx context.Context, validatorIndex *uint64, fromEpoch, toEpoch uint64, limit, offset int) ([]*storage.AttestationReward, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT validator_index, epoch, head_reward, source_reward, target_reward, total_reward, timestamp
		FROM attestation_rewards
		WHERE epoch >= $1 AND epoch <= $2`)
	args := []any{fromEpoch, toEpoch}
	argPos := 3
	if validatorIndex != nil {
		fmt.Fprintf(&sb, " AND validator_index = $%d", argPos)
		args = append(args, *validatorIndex)
		argPos++
	}
	fmt.Fprintf(&sb, " ORDER BY epoch DESC, validator_index ASC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.client.Pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list attestation rewards: %w", err)
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

// ListBlocks returns indexed blocks for a slot range, optionally filtered to one proposer validator_index.
func (r *Repository) ListBlocks(ctx context.Context, validatorIndex *uint64, fromSlot, toSlot uint64, limit, offset int) ([]*storage.Block, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT validator_index, validator_pubkey, slot_number, block_number, rewards,
			execution_priority_fees_wei, execution_mev_fees_wei, timestamp
		FROM blocks
		WHERE slot_number >= $1 AND slot_number <= $2`)
	args := []any{fromSlot, toSlot}
	argPos := 3
	if validatorIndex != nil {
		fmt.Fprintf(&sb, " AND validator_index = $%d", argPos)
		args = append(args, *validatorIndex)
		argPos++
	}
	fmt.Fprintf(&sb, " ORDER BY slot_number DESC, validator_index ASC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.client.Pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list blocks: %w", err)
	}
	defer rows.Close()

	var out []*storage.Block
	for rows.Next() {
		var row storage.Block
		var blockNum sql.NullInt64
		var priWei, mevWei sql.NullString
		if err := rows.Scan(
			&row.ValidatorIndex,
			&row.ValidatorPubkey,
			&row.SlotNumber,
			&blockNum,
			&row.Rewards,
			&priWei,
			&mevWei,
			&row.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}
		if blockNum.Valid {
			bn := uint64(blockNum.Int64)
			row.BlockNumber = &bn
		}
		if priWei.Valid {
			s := priWei.String
			row.ExecutionPriorityFeesWei = &s
		}
		if mevWei.Valid {
			s := mevWei.String
			row.ExecutionMevFeesWei = &s
		}
		cp := row
		out = append(out, &cp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate blocks: %w", err)
	}
	return out, nil
}

// ListSyncCommitteeRewards returns sync committee rewards for a slot range, optionally filtered to one validator.
func (r *Repository) ListSyncCommitteeRewards(ctx context.Context, validatorIndex *uint64, fromSlot, toSlot uint64, limit, offset int) ([]*storage.SyncCommitteeReward, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT validator_index, slot, reward_gwei, execution_optimistic, finalized, timestamp
		FROM sync_committee_rewards
		WHERE slot >= $1 AND slot <= $2`)
	args := []any{fromSlot, toSlot}
	argPos := 3
	if validatorIndex != nil {
		fmt.Fprintf(&sb, " AND validator_index = $%d", argPos)
		args = append(args, *validatorIndex)
		argPos++
	}
	fmt.Fprintf(&sb, " ORDER BY slot DESC, validator_index ASC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.client.Pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sync committee rewards: %w", err)
	}
	defer rows.Close()

	var out []*storage.SyncCommitteeReward
	for rows.Next() {
		var row storage.SyncCommitteeReward
		if err := rows.Scan(
			&row.ValidatorIndex,
			&row.Slot,
			&row.RewardGwei,
			&row.ExecutionOptimistic,
			&row.Finalized,
			&row.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sync committee reward: %w", err)
		}
		cp := row
		out = append(out, &cp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate sync committee rewards: %w", err)
	}
	return out, nil
}

// GetValidatorPenalties returns penalties for a validator in an epoch range with pagination.
func (r *Repository) GetValidatorPenalties(ctx context.Context, validatorIndex, fromEpoch, toEpoch uint64, limit, offset int) ([]*storage.ValidatorPenalty, error) {
	const query = `
		SELECT validator_index, epoch, slot, penalty_type, penalty_gwei, timestamp
		FROM validator_penalties
		WHERE validator_index = $1 AND epoch >= $2 AND epoch <= $3
		ORDER BY epoch DESC, slot DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.client.Pool.Query(ctx, query, validatorIndex, fromEpoch, toEpoch, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator penalties: %w", err)
	}
	defer rows.Close()

	var out []*storage.ValidatorPenalty
	for rows.Next() {
		var row storage.ValidatorPenalty
		if err := rows.Scan(
			&row.ValidatorIndex,
			&row.Epoch,
			&row.Slot,
			&row.PenaltyType,
			&row.PenaltyGwei,
			&row.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan validator penalty: %w", err)
		}
		cp := row
		out = append(out, &cp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate validator penalties: %w", err)
	}
	return out, nil
}

// ListValidators returns distinct validator indices that have snapshots, ordered ascending.
func (r *Repository) ListValidators(ctx context.Context, limit, offset int) ([]uint64, error) {
	const query = `
		SELECT DISTINCT validator_index
		FROM validator_snapshots
		ORDER BY validator_index ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.client.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list validators: %w", err)
	}
	defer rows.Close()

	var indices []uint64
	for rows.Next() {
		var idx uint64
		if err := rows.Scan(&idx); err != nil {
			return nil, fmt.Errorf("failed to scan validator index: %w", err)
		}
		indices = append(indices, idx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate validators: %w", err)
	}
	return indices, nil
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
