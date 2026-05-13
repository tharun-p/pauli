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
	SaveBlockProposerReward(ctx context.Context, row *BlockProposerReward) error
	SaveBlockProposerRewards(ctx context.Context, rows []*BlockProposerReward) error
	SaveSyncCommitteeReward(ctx context.Context, row *SyncCommitteeReward) error
	SaveSyncCommitteeRewards(ctx context.Context, rows []*SyncCommitteeReward) error
	SaveValidatorPenalty(ctx context.Context, penalty *ValidatorPenalty) error
	GetValidatorSnapshots(ctx context.Context, validatorIndex, fromSlot, toSlot uint64) ([]*ValidatorSnapshot, error)
	GetAttestationRewards(ctx context.Context, validatorIndex, fromEpoch, toEpoch uint64) ([]*AttestationReward, error)
	GetLatestSnapshot(ctx context.Context, validatorIndex uint64) (*ValidatorSnapshot, error)
	CountSnapshots(ctx context.Context, validatorIndex uint64) (int, error)
	Close() error
}

// Store abstracts the database backend (PostgreSQL).
type Store interface {
	RunMigrations() error
	HealthCheck() error
	Repository() Repository
	Close()
}
