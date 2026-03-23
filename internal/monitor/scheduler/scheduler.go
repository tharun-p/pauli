package scheduler

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
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

// New creates a new Scheduler.
// slotDuration controls how long a slot is (e.g. 12s on mainnet, 2s on some devnets).
func New(client *beacon.Client, validators []uint64, intervalSlots int, slotDuration time.Duration, logger zerolog.Logger) *Scheduler {
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
