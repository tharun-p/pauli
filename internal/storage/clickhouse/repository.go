package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tharun/pauli/internal/storage"
)

// Repository provides data access methods for validator data backed by ClickHouse.
type Repository struct {
	client *Client
}

// Ensure Repository implements storage.Repository.
var _ storage.Repository = (*Repository)(nil)

// NewRepository creates a new ClickHouse-backed Repository.
func NewRepository(client *Client) (storage.Repository, error) {
	return &Repository{client: client}, nil
}

// Close closes any resources held by the repository.
func (r *Repository) Close() error {
	return nil
}

// SaveValidatorEpochRecords inserts network-wide validator epoch rows in one batch.
func (r *Repository) SaveValidatorEpochRecords(ctx context.Context, records []*storage.ValidatorEpochRecord) error {
	if len(records) == 0 {
		return nil
	}

	batch, err := r.client.Conn.PrepareBatch(ctx, `
		INSERT INTO validator_epoch_records (
			validator_index, epoch, epoch_start_slot, status, balance, effective_balance,
			head_reward, source_reward, target_reward, total_reward, indexed_at
		)
	`)
	if err != nil {
		return fmt.Errorf("prepare validator epoch records batch: %w", err)
	}

	now := time.Now().UTC()
	for _, rec := range records {
		indexedAt := rec.IndexedAt
		if indexedAt.IsZero() {
			indexedAt = now
		}
		if err := batch.Append(
			rec.ValidatorIndex,
			rec.Epoch,
			rec.EpochStartSlot,
			rec.Status,
			rec.Balance,
			rec.EffectiveBalance,
			rec.HeadReward,
			rec.SourceReward,
			rec.TargetReward,
			rec.TotalReward,
			indexedAt,
		); err != nil {
			return fmt.Errorf("append validator epoch record: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to save validator epoch records batch: %w", err)
	}
	return nil
}

// SaveBlock inserts one indexed block row.
func (r *Repository) SaveBlock(ctx context.Context, row *storage.Block) error {
	if row.Timestamp.IsZero() {
		row.Timestamp = time.Now().UTC()
	}

	var syncRewards *string
	if row.SyncCommitteeRewards != nil {
		b, err := json.Marshal(row.SyncCommitteeRewards)
		if err != nil {
			return fmt.Errorf("marshal sync committee rewards: %w", err)
		}
		s := string(b)
		syncRewards = &s
	}

	const query = `
		INSERT INTO blocks (
			validator_index, validator_pubkey, slot_number, block_number, rewards,
			execution_priority_fees_wei, execution_mev_fees_wei, sync_committee_rewards, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	if err := r.client.ExecContext(ctx, query,
		row.ValidatorIndex,
		row.ValidatorPubkey,
		row.SlotNumber,
		row.BlockNumber,
		row.Rewards,
		row.ExecutionPriorityFeesWei,
		row.ExecutionMevFeesWei,
		syncRewards,
		row.Timestamp,
	); err != nil {
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

// GetValidatorSnapshots retrieves epoch balance snapshots for a validator.
func (r *Repository) GetValidatorSnapshots(ctx context.Context, validatorIndex uint64, fromSlot, toSlot uint64) ([]*storage.ValidatorSnapshot, error) {
	const query = `
		SELECT validator_index, epoch_start_slot, status, balance, effective_balance, indexed_at
		FROM validator_epoch_records FINAL
		WHERE validator_index = ? AND epoch_start_slot >= ? AND epoch_start_slot <= ?
		ORDER BY epoch_start_slot DESC
	`
	return r.scanValidatorSnapshots(ctx, query, validatorIndex, fromSlot, toSlot)
}

// ListValidatorSnapshots returns epoch balance snapshots for a validator in a slot range.
func (r *Repository) ListValidatorSnapshots(ctx context.Context, validatorIndex, fromSlot, toSlot uint64, limit, offset int) ([]*storage.ValidatorSnapshot, error) {
	const query = `
		SELECT validator_index, epoch_start_slot, status, balance, effective_balance, indexed_at
		FROM validator_epoch_records FINAL
		WHERE validator_index = ? AND epoch_start_slot >= ? AND epoch_start_slot <= ?
		ORDER BY epoch_start_slot DESC
		LIMIT ? OFFSET ?
	`
	return r.scanValidatorSnapshots(ctx, query, validatorIndex, fromSlot, toSlot, limit, offset)
}

func (r *Repository) scanValidatorSnapshots(ctx context.Context, query string, args ...any) ([]*storage.ValidatorSnapshot, error) {
	rows, err := r.client.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query validator snapshots: %w", err)
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
		SELECT validator_index, epoch, head_reward, source_reward, target_reward, total_reward, indexed_at
		FROM validator_epoch_records FINAL
		WHERE validator_index = ? AND epoch >= ? AND epoch <= ? AND head_reward IS NOT NULL
		ORDER BY epoch DESC
	`
	return r.scanAttestationRewards(ctx, query, validatorIndex, fromEpoch, toEpoch)
}

// ListAttestationRewards returns attestation rewards for an epoch range, optionally filtered to one validator.
func (r *Repository) ListAttestationRewards(ctx context.Context, validatorIndex *uint64, fromEpoch, toEpoch uint64, limit, offset int) ([]*storage.AttestationReward, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT validator_index, epoch, head_reward, source_reward, target_reward, total_reward, indexed_at
		FROM validator_epoch_records FINAL
		WHERE epoch >= ? AND epoch <= ? AND head_reward IS NOT NULL`)
	args := []any{fromEpoch, toEpoch}
	if validatorIndex != nil {
		sb.WriteString(` AND validator_index = ?`)
		args = append(args, *validatorIndex)
	}
	sb.WriteString(` ORDER BY epoch DESC, validator_index ASC LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)

	return r.scanAttestationRewards(ctx, sb.String(), args...)
}

func (r *Repository) scanAttestationRewards(ctx context.Context, query string, args ...any) ([]*storage.AttestationReward, error) {
	rows, err := r.client.QueryContext(ctx, query, args...)
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

// ListBlocks returns indexed blocks for a slot range, optionally filtered to one proposer.
func (r *Repository) ListBlocks(ctx context.Context, validatorIndex *uint64, fromSlot, toSlot uint64, limit, offset int) ([]*storage.Block, error) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT validator_index, validator_pubkey, slot_number, block_number, rewards,
			execution_priority_fees_wei, execution_mev_fees_wei, timestamp
		FROM blocks FINAL
		WHERE slot_number >= ? AND slot_number <= ?`)
	args := []any{fromSlot, toSlot}
	if validatorIndex != nil {
		sb.WriteString(` AND validator_index = ?`)
		args = append(args, *validatorIndex)
	}
	sb.WriteString(` ORDER BY slot_number DESC, validator_index ASC LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)

	rows, err := r.client.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list blocks: %w", err)
	}
	defer rows.Close()

	var out []*storage.Block
	for rows.Next() {
		var row storage.Block
		if err := rows.Scan(
			&row.ValidatorIndex,
			&row.ValidatorPubkey,
			&row.SlotNumber,
			&row.BlockNumber,
			&row.Rewards,
			&row.ExecutionPriorityFeesWei,
			&row.ExecutionMevFeesWei,
			&row.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}
		cp := row
		out = append(out, &cp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate blocks: %w", err)
	}
	return out, nil
}

// ListSyncCommitteeRewards returns sync committee rewards for a slot range.
func (r *Repository) ListSyncCommitteeRewards(ctx context.Context, validatorIndex *uint64, fromSlot, toSlot uint64, limit, offset int) ([]*storage.SyncCommitteeReward, error) {
	if validatorIndex != nil {
		return r.listSyncCommitteeRewardsScoped(ctx, *validatorIndex, fromSlot, toSlot, limit, offset)
	}
	return r.listSyncCommitteeRewardsGlobal(ctx, fromSlot, toSlot, limit, offset)
}

func (r *Repository) listSyncCommitteeRewardsScoped(ctx context.Context, validatorIndex, fromSlot, toSlot uint64, limit, offset int) ([]*storage.SyncCommitteeReward, error) {
	idxKey := strconv.FormatUint(validatorIndex, 10)
	const query = `
		SELECT
			slot_number,
			JSONExtractInt64(sync_committee_rewards, 'rewards', ?) AS reward_gwei,
			JSONExtractBool(sync_committee_rewards, 'execution_optimistic') AS execution_optimistic,
			JSONExtractBool(sync_committee_rewards, 'finalized') AS finalized,
			timestamp
		FROM blocks FINAL
		WHERE slot_number >= ? AND slot_number <= ?
			AND sync_committee_rewards IS NOT NULL
			AND JSONHas(sync_committee_rewards, 'rewards', ?)
		ORDER BY slot_number DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.client.QueryContext(ctx, query, idxKey, fromSlot, toSlot, idxKey, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list sync committee rewards: %w", err)
	}
	defer rows.Close()

	var out []*storage.SyncCommitteeReward
	for rows.Next() {
		var row storage.SyncCommitteeReward
		row.ValidatorIndex = validatorIndex
		if err := rows.Scan(
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

func (r *Repository) listSyncCommitteeRewardsGlobal(ctx context.Context, fromSlot, toSlot uint64, limit, offset int) ([]*storage.SyncCommitteeReward, error) {
	const query = `
		SELECT
			b.slot_number,
			toUInt64(t.1) AS validator_index,
			toInt64(t.2) AS reward_gwei,
			JSONExtractBool(b.sync_committee_rewards, 'execution_optimistic') AS execution_optimistic,
			JSONExtractBool(b.sync_committee_rewards, 'finalized') AS finalized,
			b.timestamp
		FROM blocks AS b FINAL
		ARRAY JOIN JSONExtractKeysAndValuesRaw(JSONExtractRaw(b.sync_committee_rewards, 'rewards'), 'String') AS t
		WHERE b.slot_number >= ? AND b.slot_number <= ?
			AND b.sync_committee_rewards IS NOT NULL
		ORDER BY b.slot_number DESC, validator_index ASC
		LIMIT ? OFFSET ?
	`
	rows, err := r.client.QueryContext(ctx, query, fromSlot, toSlot, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list sync committee rewards: %w", err)
	}
	defer rows.Close()

	var out []*storage.SyncCommitteeReward
	for rows.Next() {
		var row storage.SyncCommitteeReward
		if err := rows.Scan(
			&row.Slot,
			&row.ValidatorIndex,
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

// ListValidators returns distinct validator indices that have epoch records.
func (r *Repository) ListValidators(ctx context.Context, limit, offset int) ([]uint64, error) {
	const query = `
		SELECT DISTINCT validator_index
		FROM validator_epoch_records FINAL
		ORDER BY validator_index ASC
		LIMIT ? OFFSET ?
	`
	rows, err := r.client.QueryContext(ctx, query, limit, offset)
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

// GetLatestSnapshot retrieves the most recent epoch balance snapshot for a validator.
func (r *Repository) GetLatestSnapshot(ctx context.Context, validatorIndex uint64) (*storage.ValidatorSnapshot, error) {
	const query = `
		SELECT validator_index, epoch_start_slot, status, balance, effective_balance, indexed_at
		FROM validator_epoch_records FINAL
		WHERE validator_index = ?
		ORDER BY epoch DESC
		LIMIT 1
	`
	rows, err := r.scanValidatorSnapshots(ctx, query, validatorIndex)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("failed to get latest snapshot: no rows")
	}
	return rows[0], nil
}

// CountSnapshots returns the total number of epoch records for a validator.
func (r *Repository) CountSnapshots(ctx context.Context, validatorIndex uint64) (int, error) {
	const query = `SELECT count() FROM validator_epoch_records FINAL WHERE validator_index = ?`
	row := r.client.QueryRowContext(ctx, query, validatorIndex)
	var count uint64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count snapshots: %w", err)
	}
	return int(count), nil
}
