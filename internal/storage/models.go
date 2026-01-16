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

// ValidatorPenalty represents a penalty applied to a validator.
type ValidatorPenalty struct {
	ValidatorIndex uint64    `json:"validator_index"`
	Epoch          uint64    `json:"epoch"`
	Slot           uint64    `json:"slot"`
	PenaltyType    string    `json:"penalty_type"` // slashing, inactivity_leak, attestation_miss
	PenaltyGwei    int64     `json:"penalty_gwei"` // Penalty amount (positive value)
	Timestamp      time.Time `json:"timestamp"`
}

// PenaltyType constants
const (
	PenaltyTypeSlashing        = "slashing"
	PenaltyTypeInactivityLeak  = "inactivity_leak"
	PenaltyTypeAttestationMiss = "attestation_miss"
)

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
