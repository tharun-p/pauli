package monitor

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
)

const (
	// SlotDuration is the Ethereum slot duration (12 seconds).
	SlotDuration = 12 * time.Second
	// SlotsPerEpoch is the number of slots in an epoch (32).
	SlotsPerEpoch = 32
	// EpochDuration is the duration of an epoch.
	EpochDuration = SlotDuration * SlotsPerEpoch
)

// Scheduler manages slot-based scheduling for monitoring tasks.
type Scheduler struct {
	client         *beacon.Client
	validators     []uint64
	intervalSlots  int
	logger         zerolog.Logger
	genesisTime    time.Time
	lastEpoch      uint64
	lastFinalEpoch uint64
}

// NewScheduler creates a new Scheduler.
func NewScheduler(client *beacon.Client, validators []uint64, intervalSlots int, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		client:        client,
		validators:    validators,
		intervalSlots: intervalSlots,
		logger:        logger,
	}
}

// Initialize fetches genesis time and sets up the scheduler.
func (s *Scheduler) Initialize(ctx context.Context) error {
	genesis, err := s.client.GetGenesis(ctx)
	if err != nil {
		return err
	}

	s.genesisTime = time.Unix(int64(genesis.Data.GenesisTime.Uint64()), 0)
	s.logger.Info().
		Time("genesis_time", s.genesisTime).
		Msg("Scheduler initialized with genesis time")

	return nil
}

// CurrentSlot calculates the current slot based on genesis time.
func (s *Scheduler) CurrentSlot() uint64 {
	elapsed := time.Since(s.genesisTime)
	if elapsed < 0 {
		return 0
	}
	return uint64(elapsed / SlotDuration)
}

// CurrentEpoch returns the current epoch.
func (s *Scheduler) CurrentEpoch() uint64 {
	return s.CurrentSlot() / SlotsPerEpoch
}

// SlotTime returns the start time of a slot.
func (s *Scheduler) SlotTime(slot uint64) time.Time {
	return s.genesisTime.Add(time.Duration(slot) * SlotDuration)
}

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
	targetTime := s.SlotTime(targetSlot)
	waitDuration := time.Until(targetTime)

	if waitDuration > 0 {
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

// ScheduleEvent represents a scheduled monitoring event.
type ScheduleEvent struct {
	Slot       uint64
	Epoch      uint64
	Type       EventType
	Validators []uint64
}

// EventType defines the type of scheduled event.
type EventType int

const (
	// EventTypeSlotPoll is a regular slot polling event.
	EventTypeSlotPoll EventType = iota
	// EventTypeEpochBoundary triggers duty fetching.
	EventTypeEpochBoundary
	// EventTypeEpochFinalized triggers reward fetching.
	EventTypeEpochFinalized
)

// NextEvents returns the events to process for the current slot.
func (s *Scheduler) NextEvents(ctx context.Context, slot uint64) ([]ScheduleEvent, error) {
	var events []ScheduleEvent
	epoch := slot / SlotsPerEpoch

	// Always include a slot poll event
	events = append(events, ScheduleEvent{
		Slot:       slot,
		Epoch:      epoch,
		Type:       EventTypeSlotPoll,
		Validators: s.validators,
	})

	// Check for epoch boundary (first slot of epoch)
	if slot%SlotsPerEpoch == 0 && epoch != s.lastEpoch {
		s.lastEpoch = epoch
		// Fetch duties for the next epoch
		events = append(events, ScheduleEvent{
			Slot:       slot,
			Epoch:      epoch + 1, // Fetch duties for upcoming epoch
			Type:       EventTypeEpochBoundary,
			Validators: s.validators,
		})
	}

	// Check for finalized epoch (typically 2 epochs behind)
	checkpoints, err := s.client.GetFinalityCheckpoints(ctx, "head")
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get finality checkpoints")
	} else {
		finalizedEpoch := checkpoints.Finalized.Epoch.Uint64()
		if finalizedEpoch > s.lastFinalEpoch {
			// Fetch rewards for newly finalized epochs
			for e := s.lastFinalEpoch + 1; e <= finalizedEpoch; e++ {
				events = append(events, ScheduleEvent{
					Slot:       slot,
					Epoch:      e,
					Type:       EventTypeEpochFinalized,
					Validators: s.validators,
				})
			}
			s.lastFinalEpoch = finalizedEpoch
		}
	}

	return events, nil
}

// SlotToEpoch converts a slot to its epoch.
func SlotToEpoch(slot uint64) uint64 {
	return slot / SlotsPerEpoch
}

// EpochStartSlot returns the first slot of an epoch.
func EpochStartSlot(epoch uint64) uint64 {
	return epoch * SlotsPerEpoch
}

// IsEpochBoundary returns true if the slot is the first slot of an epoch.
func IsEpochBoundary(slot uint64) bool {
	return slot%SlotsPerEpoch == 0
}
