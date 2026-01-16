package beacon

import (
	"context"
	"fmt"
)

// GetAttesterDuties fetches attestation duties for validators in an epoch.
// The request body contains the list of validator indices.
func (c *Client) GetAttesterDuties(ctx context.Context, epoch uint64, validatorIndices []uint64) (*AttesterDutiesResponse, error) {
	path := fmt.Sprintf("/eth/v1/validator/duties/attester/%d", epoch)

	// Convert to string slice for JSON encoding
	indices := make([]string, len(validatorIndices))
	for i, idx := range validatorIndices {
		indices[i] = fmt.Sprintf("%d", idx)
	}

	var resp AttesterDutiesResponse
	if err := c.post(ctx, path, indices, &resp); err != nil {
		return nil, fmt.Errorf("failed to get attester duties for epoch %d: %w", epoch, err)
	}

	return &resp, nil
}

// GetAttesterDutiesMap fetches attestation duties and returns them as a map keyed by validator index.
func (c *Client) GetAttesterDutiesMap(ctx context.Context, epoch uint64, validatorIndices []uint64) (map[uint64]*AttesterDuty, error) {
	resp, err := c.GetAttesterDuties(ctx, epoch, validatorIndices)
	if err != nil {
		return nil, err
	}

	duties := make(map[uint64]*AttesterDuty, len(resp.Data))
	for i := range resp.Data {
		duty := &resp.Data[i]
		duties[duty.ValidatorIndex.Uint64()] = duty
	}

	return duties, nil
}

// SlotToEpoch converts a slot number to an epoch number.
// There are 32 slots per epoch.
func SlotToEpoch(slot uint64) uint64 {
	return slot / 32
}

// EpochStartSlot returns the first slot of an epoch.
func EpochStartSlot(epoch uint64) uint64 {
	return epoch * 32
}

// EpochEndSlot returns the last slot of an epoch.
func EpochEndSlot(epoch uint64) uint64 {
	return (epoch+1)*32 - 1
}

// IsEpochBoundary returns true if the slot is the first slot of an epoch.
func IsEpochBoundary(slot uint64) bool {
	return slot%32 == 0
}
