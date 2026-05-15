package storage

import "time"

// ValidatorSnapshot represents a point-in-time snapshot of a validator's state.
type ValidatorSnapshot struct {
	ValidatorIndex   uint64    `json:"validator_index"`
	Slot             uint64    `json:"slot"`
	Status           string    `json:"status"`
	Balance          uint64    `json:"balance"`           // Actual balance in Gwei
	EffectiveBalance uint64    `json:"effective_balance"` // Effective balance in Gwei (MaxEB aware, up to 2048 ETH)
	Timestamp        time.Time `json:"timestamp"`
}

// AttestationDuty represents a validator's attestation duty assignment.
type AttestationDuty struct {
	ValidatorIndex    uint64    `json:"validator_index"`
	Epoch             uint64    `json:"epoch"`
	Slot              uint64    `json:"slot"`
	CommitteeIndex    int       `json:"committee_index"`
	CommitteePosition int       `json:"committee_position"`
	Timestamp         time.Time `json:"timestamp"`
}

// AttestationReward represents a validator's attestation rewards for an epoch.
type AttestationReward struct {
	ValidatorIndex uint64    `json:"validator_index"`
	Epoch          uint64    `json:"epoch"`
	HeadReward     int64     `json:"head_reward"`   // Can be negative (penalty)
	SourceReward   int64     `json:"source_reward"` // Can be negative (penalty)
	TargetReward   int64     `json:"target_reward"` // Can be negative (penalty)
	TotalReward    int64     `json:"total_reward"`  // Sum of head + source + target
	Timestamp      time.Time `json:"timestamp"`
}

// BlockSyncCommitteeRewards holds all sync committee member rewards for one beacon block slot.
type BlockSyncCommitteeRewards struct {
	ExecutionOptimistic bool             `json:"execution_optimistic"`
	Finalized           bool             `json:"finalized"`
	Rewards             map[string]int64 `json:"rewards"` // validator index (decimal string) -> reward_gwei
}

// Block is one indexed canonical beacon block at slot_number (proposer CL rewards and optional EL fee fields).
type Block struct {
	ValidatorIndex           uint64                    `json:"validator_index"`
	ValidatorPubkey          string                    `json:"validator_pubkey"`
	SlotNumber               uint64                    `json:"slot_number"`
	BlockNumber              *uint64                   `json:"block_number,omitempty"`                // Execution layer block number when available
	Rewards                  uint64                    `json:"rewards"`                               // Proposer reward total (gwei)
	ExecutionPriorityFeesWei *string                   `json:"execution_priority_fees_wei,omitempty"` // Sum of priority tips (wei), decimal string
	ExecutionMevFeesWei      *string                   `json:"execution_mev_fees_wei,omitempty"`      // Reserved; NULL in v1
	SyncCommitteeRewards     *BlockSyncCommitteeRewards `json:"sync_committee_rewards,omitempty"`
	Timestamp                time.Time                 `json:"timestamp"`
}

// SyncCommitteeReward is one row of sync committee reward for a validator at a beacon block slot.
type SyncCommitteeReward struct {
	ValidatorIndex      uint64    `json:"validator_index"`
	Slot                uint64    `json:"slot"`
	RewardGwei          int64     `json:"reward_gwei"`
	ExecutionOptimistic bool      `json:"execution_optimistic"`
	Finalized           bool      `json:"finalized"`
	Timestamp           time.Time `json:"timestamp"`
}

// ValidatorStatus constants from Beacon API
const (
	StatusPendingInitialized = "pending_initialized"
	StatusPendingQueued      = "pending_queued"
	StatusActiveOngoing      = "active_ongoing"
	StatusActiveExiting      = "active_exiting"
	StatusActiveSlashed      = "active_slashed"
	StatusExitedUnslashed    = "exited_unslashed"
	StatusExitedSlashed      = "exited_slashed"
	StatusWithdrawalPossible = "withdrawal_possible"
	StatusWithdrawalDone     = "withdrawal_done"
)

// IsActiveStatus returns true if the status indicates an active validator.
func IsActiveStatus(status string) bool {
	return status == StatusActiveOngoing ||
		status == StatusActiveExiting ||
		status == StatusActiveSlashed
}

// IsSlashedStatus returns true if the status indicates a slashed validator.
func IsSlashedStatus(status string) bool {
	return status == StatusActiveSlashed || status == StatusExitedSlashed
}
