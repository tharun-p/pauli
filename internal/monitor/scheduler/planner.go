package scheduler

import (
	"context"

	"github.com/tharun/pauli/internal/config"
)

// EventType defines the type of scheduled event.
type EventType int

const (
	EventTypeSlotPoll EventType = iota
	EventTypeEpochBoundary
	EventTypeEpochFinalized
)

// Event represents a scheduled monitoring event.
type Event struct {
	Slot  uint64
	Epoch uint64
	Type  EventType
}

// NextEvents returns the events to process for the current slot.
func (s *Scheduler) NextEvents(ctx context.Context, slot uint64) ([]Event, error) {
	var events []Event
	epoch := slot / config.SlotsPerEpoch()

	events = append(events, Event{Slot: slot, Epoch: epoch, Type: EventTypeSlotPoll})

	scheduleEpochEvents, targetEpoch, rewardEpoch, isEpochBoundary, isNearEpochBoundary := s.planEpochBoundary(slot, epoch)
	if !scheduleEpochEvents {
		return events, nil
	}

	s.logger.Info().
		Uint64("current_slot", slot).
		Uint64("current_epoch", epoch).
		Uint64("target_epoch", targetEpoch).
		Bool("is_epoch_boundary", isEpochBoundary).
		Bool("is_near_boundary", isNearEpochBoundary).
		Msg("Detected epoch boundary, scheduling duties and rewards fetch")

	events = append(events, Event{Slot: slot, Epoch: targetEpoch, Type: EventTypeEpochBoundary})
	if rewardEpoch > 0 || epoch > 0 {
		events = append(events, Event{Slot: slot, Epoch: rewardEpoch, Type: EventTypeEpochFinalized})
	}
	return events, nil
}

func (s *Scheduler) planEpochBoundary(slot uint64, epoch uint64) (schedule bool, targetEpoch uint64, rewardEpoch uint64, isEpochBoundary bool, isNearEpochBoundary bool) {
	isEpochBoundary = slot%config.SlotsPerEpoch() == 0
	isNearEpochBoundary = (slot+1)%config.SlotsPerEpoch() == 0

	if !(isEpochBoundary || isNearEpochBoundary) || epoch == s.lastEpoch {
		return false, 0, 0, isEpochBoundary, isNearEpochBoundary
	}

	if isEpochBoundary {
		s.lastEpoch = epoch
	} else {
		s.lastEpoch = epoch + 1
	}

	targetEpoch = epoch + 1
	if epoch > 0 {
		rewardEpoch = epoch - 1
	}
	return true, targetEpoch, rewardEpoch, isEpochBoundary, isNearEpochBoundary
}
