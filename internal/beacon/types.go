package beacon

import (
	"strconv"
)

// Beacon API response wrapper types

// APIResponse is the standard Beacon API response envelope.
type APIResponse[T any] struct {
	Data                T      `json:"data"`
	ExecutionOptimistic bool   `json:"execution_optimistic,omitempty"`
	Finalized           bool   `json:"finalized,omitempty"`
}

// Validator represents a validator's state from the Beacon API.
// Supports MaxEB (EIP-7251) with effective balance up to 2048 ETH.
type Validator struct {
	Index     Uint64Str `json:"index"`
	Balance   Uint64Str `json:"balance"` // Actual balance in Gwei
	Status    string    `json:"status"`
	Validator struct {
		Pubkey                     string    `json:"pubkey"`
		WithdrawalCredentials      string    `json:"withdrawal_credentials"`
		EffectiveBalance           Uint64Str `json:"effective_balance"` // Up to 2048 ETH with MaxEB
		Slashed                    bool      `json:"slashed"`
		ActivationEligibilityEpoch Uint64Str `json:"activation_eligibility_epoch"`
		ActivationEpoch            Uint64Str `json:"activation_epoch"`
		ExitEpoch                  Uint64Str `json:"exit_epoch"`
		WithdrawableEpoch          Uint64Str `json:"withdrawable_epoch"`
	} `json:"validator"`
}

// ValidatorResponse is the response from /eth/v1/beacon/states/{state_id}/validators/{validator_id}.
type ValidatorResponse = APIResponse[Validator]

// ValidatorsResponse is the response from /eth/v1/beacon/states/{state_id}/validators.
type ValidatorsResponse = APIResponse[[]Validator]

// AttesterDuty represents an attestation duty assignment.
type AttesterDuty struct {
	Pubkey                  string    `json:"pubkey"`
	ValidatorIndex          Uint64Str `json:"validator_index"`
	CommitteeIndex          Uint64Str `json:"committee_index"`
	CommitteeLength         Uint64Str `json:"committee_length"`
	CommitteesAtSlot        Uint64Str `json:"committees_at_slot"`
	ValidatorCommitteeIndex Uint64Str `json:"validator_committee_index"`
	Slot                    Uint64Str `json:"slot"`
}

// AttesterDutiesResponse is the response from /eth/v1/validator/duties/attester/{epoch}.
type AttesterDutiesResponse struct {
	DependentRoot           string         `json:"dependent_root"`
	ExecutionOptimistic     bool           `json:"execution_optimistic"`
	Data                    []AttesterDuty `json:"data"`
}

// AttestationReward represents rewards for a single validator's attestation.
type AttestationReward struct {
	ValidatorIndex Uint64Str `json:"validator_index"`
	Head           Int64Str  `json:"head"`   // Can be negative (penalty)
	Target         Int64Str  `json:"target"` // Can be negative (penalty)
	Source         Int64Str  `json:"source"` // Can be negative (penalty)
}

// AttestationRewardsData contains the rewards breakdown.
type AttestationRewardsData struct {
	IdealRewards  []AttestationReward `json:"ideal_rewards"`
	TotalRewards  []AttestationReward `json:"total_rewards"`
}

// AttestationRewardsResponse is the response from /eth/v1/beacon/rewards/attestations/{epoch}.
type AttestationRewardsResponse = APIResponse[AttestationRewardsData]

// BeaconBlockHeader represents a block header from the Beacon API.
type BeaconBlockHeader struct {
	Slot          Uint64Str `json:"slot"`
	ProposerIndex Uint64Str `json:"proposer_index"`
	ParentRoot    string    `json:"parent_root"`
	StateRoot     string    `json:"state_root"`
	BodyRoot      string    `json:"body_root"`
}

// SignedBeaconBlockHeader wraps a block header with its signature.
type SignedBeaconBlockHeader struct {
	Message   BeaconBlockHeader `json:"message"`
	Signature string            `json:"signature"`
}

// BlockHeaderResponse is the response from /eth/v1/beacon/headers/{block_id}.
type BlockHeaderResponse struct {
	Data struct {
		Root      string                  `json:"root"`
		Canonical bool                    `json:"canonical"`
		Header    SignedBeaconBlockHeader `json:"header"`
	} `json:"data"`
	ExecutionOptimistic bool `json:"execution_optimistic"`
	Finalized           bool `json:"finalized"`
}

// FinalityCheckpoints represents the finality checkpoints.
type FinalityCheckpoints struct {
	PreviousJustified Checkpoint `json:"previous_justified"`
	CurrentJustified  Checkpoint `json:"current_justified"`
	Finalized         Checkpoint `json:"finalized"`
}

// Checkpoint represents an epoch checkpoint.
type Checkpoint struct {
	Epoch Uint64Str `json:"epoch"`
	Root  string    `json:"root"`
}

// FinalityCheckpointsResponse is the response from /eth/v1/beacon/states/{state_id}/finality_checkpoints.
type FinalityCheckpointsResponse = APIResponse[FinalityCheckpoints]

// Uint64Str handles JSON numbers that are encoded as strings.
type Uint64Str uint64

func (u *Uint64Str) UnmarshalJSON(data []byte) error {
	// Remove quotes if present
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*u = Uint64Str(v)
	return nil
}

func (u Uint64Str) Uint64() uint64 {
	return uint64(u)
}

// Int64Str handles JSON signed integers that are encoded as strings.
type Int64Str int64

func (i *Int64Str) UnmarshalJSON(data []byte) error {
	// Remove quotes if present
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*i = Int64Str(v)
	return nil
}

func (i Int64Str) Int64() int64 {
	return int64(i)
}

// GenesisResponse is the response from /eth/v1/beacon/genesis.
type GenesisResponse struct {
	Data struct {
		GenesisTime           Uint64Str `json:"genesis_time"`
		GenesisValidatorsRoot string    `json:"genesis_validators_root"`
		GenesisForkVersion    string    `json:"genesis_fork_version"`
	} `json:"data"`
}

// SyncingResponse is the response from /eth/v1/node/syncing.
type SyncingResponse struct {
	Data struct {
		HeadSlot     Uint64Str `json:"head_slot"`
		SyncDistance Uint64Str `json:"sync_distance"`
		IsSyncing    bool      `json:"is_syncing"`
		IsOptimistic bool      `json:"is_optimistic"`
		ELOffline    bool      `json:"el_offline"`
	} `json:"data"`
}
