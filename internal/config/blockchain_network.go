package config

import (
	"context"
	"time"
)

// BlockchainNetwork holds chain timing used for wall-clock pacing and slot math (genesis is set after beacon init).
type BlockchainNetwork struct {
	slotDuration         time.Duration
	pollingIntervalSlots int
	slotsPerEpoch        uint64
	genesisTime          time.Time
}

// NewBlockchainNetwork builds network timing from application config (genesis time is set later via SetGenesisTime).
func NewBlockchainNetwork(c *Config) *BlockchainNetwork {
	return &BlockchainNetwork{
		slotDuration:         c.SlotDuration(),
		pollingIntervalSlots: c.PollingIntervalSlots,
		slotsPerEpoch:        SlotsPerEpoch(),
	}
}

// SetGenesisTime sets the chain genesis wall time (from beacon genesis API).
func (n *BlockchainNetwork) SetGenesisTime(t time.Time) {
	n.genesisTime = t
}

// GenesisTime returns the configured genesis instant (zero before SetGenesisTime).
func (n *BlockchainNetwork) GenesisTime() time.Time {
	return n.genesisTime
}

// SlotDuration returns wall duration of one consensus slot.
func (n *BlockchainNetwork) SlotDuration() time.Duration {
	return n.slotDuration
}

// SlotsPerEpoch returns configured slots per epoch (e.g. 32).
func (n *BlockchainNetwork) SlotsPerEpoch() uint64 {
	return n.slotsPerEpoch
}

func (n *BlockchainNetwork) pollSlots() int {
	if n.pollingIntervalSlots <= 0 {
		return 1
	}
	return n.pollingIntervalSlots
}

// PollInterval is wall time between realtime poll iterations: slotDuration × polling_interval_slots.
func (n *BlockchainNetwork) PollInterval() time.Duration {
	return n.slotDuration * time.Duration(n.pollSlots())
}

// WaitPollInterval blocks until the next poll window elapses or ctx is cancelled.
func (n *BlockchainNetwork) WaitPollInterval(ctx context.Context) error {
	d := n.PollInterval()
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
