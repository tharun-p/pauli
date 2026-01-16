package beacon

import (
	"context"
	"fmt"
)

// GetAttestationRewards fetches attestation rewards for validators in an epoch.
// The epoch must be finalized for rewards to be available.
func (c *Client) GetAttestationRewards(ctx context.Context, epoch uint64, validatorIndices []uint64) (*AttestationRewardsData, error) {
	path := fmt.Sprintf("/eth/v1/beacon/rewards/attestations/%d", epoch)

	// Convert to string slice for JSON encoding
	indices := make([]string, len(validatorIndices))
	for i, idx := range validatorIndices {
		indices[i] = fmt.Sprintf("%d", idx)
	}

	var resp AttestationRewardsResponse
	if err := c.post(ctx, path, indices, &resp); err != nil {
		return nil, fmt.Errorf("failed to get attestation rewards for epoch %d: %w", epoch, err)
	}

	return &resp.Data, nil
}

// GetAttestationRewardsMap fetches attestation rewards and returns them as a map keyed by validator index.
func (c *Client) GetAttestationRewardsMap(ctx context.Context, epoch uint64, validatorIndices []uint64) (map[uint64]*AttestationReward, error) {
	resp, err := c.GetAttestationRewards(ctx, epoch, validatorIndices)
	if err != nil {
		return nil, err
	}

	rewards := make(map[uint64]*AttestationReward, len(resp.TotalRewards))
	for i := range resp.TotalRewards {
		reward := &resp.TotalRewards[i]
		rewards[reward.ValidatorIndex.Uint64()] = reward
	}

	return rewards, nil
}

// RewardSummary provides a summary of rewards for a validator.
type RewardSummary struct {
	ValidatorIndex uint64
	HeadReward     int64
	SourceReward   int64
	TargetReward   int64
	TotalReward    int64
	IsPenalty      bool // True if total reward is negative
}

// CalculateRewardSummary creates a summary from an AttestationReward.
func CalculateRewardSummary(reward *AttestationReward) *RewardSummary {
	total := reward.Head.Int64() + reward.Source.Int64() + reward.Target.Int64()
	return &RewardSummary{
		ValidatorIndex: reward.ValidatorIndex.Uint64(),
		HeadReward:     reward.Head.Int64(),
		SourceReward:   reward.Source.Int64(),
		TargetReward:   reward.Target.Int64(),
		TotalReward:    total,
		IsPenalty:      total < 0,
	}
}
