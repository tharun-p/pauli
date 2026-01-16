package beacon

import (
	"context"
	"fmt"
)

// GetValidator fetches a single validator's state.
// stateID can be "head", "genesis", "finalized", "justified", a slot number, or a state root.
// validatorID can be a validator index or a public key.
func (c *Client) GetValidator(ctx context.Context, stateID string, validatorID uint64) (*Validator, error) {
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/validators/%d", stateID, validatorID)

	var resp ValidatorResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get validator %d: %w", validatorID, err)
	}

	return &resp.Data, nil
}

// GetValidatorByPubkey fetches a validator's state by public key.
func (c *Client) GetValidatorByPubkey(ctx context.Context, stateID, pubkey string) (*Validator, error) {
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/validators/%s", stateID, pubkey)

	var resp ValidatorResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get validator %s: %w", pubkey, err)
	}

	return &resp.Data, nil
}

// GetValidators fetches multiple validators' states.
// If validatorIDs is empty, returns all validators.
func (c *Client) GetValidators(ctx context.Context, stateID string, validatorIDs []uint64) ([]Validator, error) {
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/validators", stateID)

	// Add validator IDs as query parameters if specified
	if len(validatorIDs) > 0 {
		path += "?id="
		for i, id := range validatorIDs {
			if i > 0 {
				path += ","
			}
			path += fmt.Sprintf("%d", id)
		}
	}

	var resp ValidatorsResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	return resp.Data, nil
}

// GetValidatorsByStatus fetches validators filtered by status.
// status can be: pending_initialized, pending_queued, active_ongoing, active_exiting,
// active_slashed, exited_unslashed, exited_slashed, withdrawal_possible, withdrawal_done.
func (c *Client) GetValidatorsByStatus(ctx context.Context, stateID string, statuses []string) ([]Validator, error) {
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/validators", stateID)

	if len(statuses) > 0 {
		path += "?status="
		for i, status := range statuses {
			if i > 0 {
				path += ","
			}
			path += status
		}
	}

	var resp ValidatorsResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get validators by status: %w", err)
	}

	return resp.Data, nil
}

// GetFinalityCheckpoints fetches the finality checkpoints for a state.
func (c *Client) GetFinalityCheckpoints(ctx context.Context, stateID string) (*FinalityCheckpoints, error) {
	path := fmt.Sprintf("/eth/v1/beacon/states/%s/finality_checkpoints", stateID)

	var resp FinalityCheckpointsResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get finality checkpoints: %w", err)
	}

	return &resp.Data, nil
}

// GetBlockHeader fetches a block header.
// blockID can be "head", "genesis", "finalized", a slot number, or a block root.
func (c *Client) GetBlockHeader(ctx context.Context, blockID string) (*BlockHeaderResponse, error) {
	path := fmt.Sprintf("/eth/v1/beacon/headers/%s", blockID)

	var resp BlockHeaderResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get block header: %w", err)
	}

	return &resp, nil
}

// GetHeadSlot returns the current head slot.
func (c *Client) GetHeadSlot(ctx context.Context) (uint64, error) {
	resp, err := c.GetBlockHeader(ctx, "head")
	if err != nil {
		return 0, err
	}
	return resp.Data.Header.Message.Slot.Uint64(), nil
}

// GetGenesis fetches genesis information.
func (c *Client) GetGenesis(ctx context.Context) (*GenesisResponse, error) {
	path := "/eth/v1/beacon/genesis"

	var resp GenesisResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get genesis: %w", err)
	}

	return &resp, nil
}

// GetSyncStatus fetches the node's sync status.
func (c *Client) GetSyncStatus(ctx context.Context) (*SyncingResponse, error) {
	path := "/eth/v1/node/syncing"

	var resp SyncingResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	return &resp, nil
}

// IsNodeSynced checks if the beacon node is fully synced.
func (c *Client) IsNodeSynced(ctx context.Context) (bool, error) {
	status, err := c.GetSyncStatus(ctx)
	if err != nil {
		return false, err
	}
	return !status.Data.IsSyncing, nil
}
