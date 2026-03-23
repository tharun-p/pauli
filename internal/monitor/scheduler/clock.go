package scheduler

import (
	"context"
	"time"

	"github.com/tharun/pauli/internal/config"
)

// CurrentSlot gets the current slot from the beacon API with caching.
// Uses calculated slot as fallback to avoid blocking on API calls.
func (s *Scheduler) CurrentSlot() uint64 {
	elapsed := time.Since(s.genesisTime)
	if elapsed < 0 {
		return 0
	}
	calculatedSlot := uint64(elapsed / s.slotDuration)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	slot, err := s.client.GetHeadSlot(ctx)
	if err != nil {
		return calculatedSlot
	}

	return slot
}

// CurrentEpoch returns the current epoch.
func (s *Scheduler) CurrentEpoch() uint64 {
	return s.CurrentSlot() / config.SlotsPerEpoch()
}

// SlotTime returns the start time of a slot.
func (s *Scheduler) SlotTime(slot uint64) time.Time {
	return s.genesisTime.Add(time.Duration(slot) * s.slotDuration)
}

func SlotToEpoch(slot uint64) uint64 { return slot / config.SlotsPerEpoch() }
func EpochStartSlot(epoch uint64) uint64 { return epoch * config.SlotsPerEpoch() }
func IsEpochBoundary(slot uint64) bool { return slot%config.SlotsPerEpoch() == 0 }
