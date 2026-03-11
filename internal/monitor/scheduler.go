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
	slotDuration   time.Duration
	logger         zerolog.Logger
	genesisTime    time.Time
	lastEpoch      uint64
	lastFinalEpoch uint64
}

// NewScheduler creates a new Scheduler.
// slotDuration controls how long a slot is (e.g. 12s on mainnet, 2s on some devnets).
func NewScheduler(client *beacon.Client, validators []uint64, intervalSlots int, slotDuration time.Duration, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		client:        client,
		validators:    validators,
		intervalSlots: intervalSlots,
		slotDuration:  slotDuration,
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

	// Initialize lastFinalEpoch to current finalized epoch to avoid catching up on old epochs
	checkpoints, err := s.client.GetFinalityCheckpoints(ctx, "head")
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get finality checkpoints during initialization")
	} else {
		s.lastFinalEpoch = checkpoints.Finalized.Epoch.Uint64()
		s.logger.Info().
			Uint64("initial_finalized_epoch", s.lastFinalEpoch).
			Msg("Initialized last finalized epoch")
	}

	s.logger.Info().
		Time("genesis_time", s.genesisTime).
		Msg("Scheduler initialized with genesis time")

	return nil
}

// CurrentSlot gets the current slot from the beacon API with caching.
// Uses calculated slot as fallback to avoid blocking on API calls.
func (s *Scheduler) CurrentSlot() uint64 {
	// Use calculated slot as primary (fast, no API call)
	// This is accurate enough for scheduling purposes
	elapsed := time.Since(s.genesisTime)
	if elapsed < 0 {
		return 0
	}
	calculatedSlot := uint64(elapsed / s.slotDuration)

	// Optionally verify with API, but don't block on it
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	slot, err := s.client.GetHeadSlot(ctx)
	if err != nil {
		// Use calculated slot if API fails
		return calculatedSlot
	}

	// Use API slot if available (more accurate)
	return slot
}

// CurrentEpoch returns the current epoch.
func (s *Scheduler) CurrentEpoch() uint64 {
	return s.CurrentSlot() / SlotsPerEpoch
}

// SlotTime returns the start time of a slot.
func (s *Scheduler) SlotTime(slot uint64) time.Time {
	return s.genesisTime.Add(time.Duration(slot) * s.slotDuration)
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
// This uses a simple time-based wait (slot duration * interval) to avoid
// getting stuck if genesis time is in the future or differs from the local clock.
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
	// Also check if we're within 1 slot of an epoch boundary to catch it
	isEpochBoundary := slot%SlotsPerEpoch == 0
	isNearEpochBoundary := (slot+1)%SlotsPerEpoch == 0

	if (isEpochBoundary || isNearEpochBoundary) && epoch != s.lastEpoch {
		if isEpochBoundary {
			s.lastEpoch = epoch
		} else {
			// We're 1 slot before epoch boundary, use next epoch
			s.lastEpoch = epoch + 1
		}

		// Fetch duties for the next epoch (epoch + 1)
		targetEpoch := epoch + 1
		s.logger.Info().
			Uint64("current_slot", slot).
			Uint64("current_epoch", epoch).
			Uint64("target_epoch", targetEpoch).
			Bool("is_epoch_boundary", isEpochBoundary).
			Bool("is_near_boundary", isNearEpochBoundary).
			Msg("Detected epoch boundary, scheduling duties and rewards fetch")

		// Schedule duties fetch for upcoming epoch
		events = append(events, ScheduleEvent{
			Slot:       slot,
			Epoch:      targetEpoch,
			Type:       EventTypeEpochBoundary,
			Validators: s.validators,
		})

		// Schedule rewards fetch for the epoch that just completed (epoch - 1),
		// so we get one rewards record per epoch in order.
		if epoch > 0 {
			rewardEpoch := epoch - 1
			events = append(events, ScheduleEvent{
				Slot:       slot,
				Epoch:      rewardEpoch,
				Type:       EventTypeEpochFinalized,
				Validators: s.validators,
			})
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
