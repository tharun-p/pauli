package scheduler

import (
	"context"
	"time"
)

// WaitForNextSlot waits until the next slot begins.
func (s *Scheduler) WaitForNextSlot(ctx context.Context) (uint64, error) {
	currentSlot := s.CurrentSlot()
	nextSlot := currentSlot + 1
	nextSlotTime := s.SlotTime(nextSlot)
	waitDuration := time.Until(nextSlotTime)

	if waitDuration > 0 {
		s.logger.Debug().
			Uint64("current_slot", currentSlot).
			Uint64("next_slot", nextSlot).
			Dur("wait_duration", waitDuration).
			Msg("Waiting for next slot")

		timer := time.NewTimer(waitDuration)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timer.C:
		}
	}

	return nextSlot, nil
}

// WaitForSlotInterval waits for the configured number of slots.
func (s *Scheduler) WaitForSlotInterval(ctx context.Context) (uint64, error) {
	currentSlot := s.CurrentSlot()
	targetSlot := currentSlot + uint64(s.intervalSlots)
	waitDuration := s.slotDuration * time.Duration(s.intervalSlots)

	if waitDuration > 0 {
		s.logger.Debug().
			Uint64("current_slot", currentSlot).
			Uint64("target_slot", targetSlot).
			Dur("wait_duration", waitDuration).
			Msg("Waiting for slot interval")

		timer := time.NewTimer(waitDuration)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timer.C:
		}
	}

	return targetSlot, nil
}
