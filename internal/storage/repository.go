package storage

import "context"

// Repository defines the data access methods for validator data.
type Repository interface {
	SaveValidatorSnapshot(ctx context.Context, snapshot *ValidatorSnapshot) error
	SaveValidatorSnapshots(ctx context.Context, snapshots []*ValidatorSnapshot) error
	SaveAttestationDuty(ctx context.Context, duty *AttestationDuty) error
	SaveAttestationDuties(ctx context.Context, duties []*AttestationDuty) error
	SaveAttestationReward(ctx context.Context, reward *AttestationReward) error
	SaveAttestationRewards(ctx context.Context, rewards []*AttestationReward) error
	SaveBlock(ctx context.Context, row *Block) error
	SaveBlocks(ctx context.Context, rows []*Block) error
	GetValidatorSnapshots(ctx context.Context, validatorIndex, fromSlot, toSlot uint64) ([]*ValidatorSnapshot, error)
	ListValidatorSnapshots(ctx context.Context, validatorIndex, fromSlot, toSlot uint64, limit, offset int) ([]*ValidatorSnapshot, error)
	GetAttestationRewards(ctx context.Context, validatorIndex, fromEpoch, toEpoch uint64) ([]*AttestationReward, error)
	// ListAttestationRewards returns attestation rewards in epoch order (newest epoch first). If validatorIndex is nil, all validators are included.
	ListAttestationRewards(ctx context.Context, validatorIndex *uint64, fromEpoch, toEpoch uint64, limit, offset int) ([]*AttestationReward, error)
	ListBlocks(ctx context.Context, validatorIndex *uint64, fromSlot, toSlot uint64, limit, offset int) ([]*Block, error)
	ListSyncCommitteeRewards(ctx context.Context, validatorIndex *uint64, fromSlot, toSlot uint64, limit, offset int) ([]*SyncCommitteeReward, error)
	ListValidators(ctx context.Context, limit, offset int) ([]uint64, error)
	GetLatestSnapshot(ctx context.Context, validatorIndex uint64) (*ValidatorSnapshot, error)
	CountSnapshots(ctx context.Context, validatorIndex uint64) (int, error)

	MarkSlotIndexed(ctx context.Context, slot uint64) error
	MarkEpochIndexed(ctx context.Context, epoch uint64) error
	MaxIndexedSlot(ctx context.Context) (slot uint64, ok bool, err error)
	MaxIndexedEpoch(ctx context.Context) (epoch uint64, ok bool, err error)
	FirstUnindexedSlot(ctx context.Context, from, to uint64) (slot uint64, ok bool, err error)
	FirstUnindexedEpoch(ctx context.Context, from, to uint64) (epoch uint64, ok bool, err error)
	IsSlotIndexed(ctx context.Context, slot uint64) (bool, error)
	IsEpochIndexed(ctx context.Context, epoch uint64) (bool, error)

	Close() error
}

// Store abstracts the database backend (PostgreSQL).
type Store interface {
	RunMigrations() error
	HealthCheck() error
	Repository() Repository
	Close()
}
