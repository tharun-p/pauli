package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/storage"
)

// Monitor orchestrates the validator monitoring process.
type Monitor struct {
	cfg        *config.Config
	client     *beacon.Client
	repo       storage.Repository
	scheduler  *Scheduler
	workerPool *WorkerPool
	logger     zerolog.Logger
	wg         sync.WaitGroup

	// Cached head slot to avoid redundant API calls
	headSlotCache struct {
		slot      uint64
		timestamp time.Time
		mu        sync.RWMutex
	}

	// Reconciliation cursors (in-memory) to avoid skipping work
	lastSnapshotSlot uint64
	lastDutiesEpoch  uint64
	lastRewardsEpoch uint64
}

// NewMonitor creates a new Monitor instance.
func NewMonitor(cfg *config.Config, client *beacon.Client, repo storage.Repository, logger zerolog.Logger) *Monitor {
	scheduler := NewScheduler(client, cfg.Validators, cfg.PollingIntervalSlots, cfg.SlotDuration(), logger)

	m := &Monitor{
		cfg:       cfg,
		client:    client,
		repo:      repo,
		scheduler: scheduler,
		logger:    logger,
	}

	// Create worker pool with the monitor as the processor
	m.workerPool = NewWorkerPool(cfg.WorkerPoolSize, m, logger)

	return m
}

// Start begins the monitoring loop.
func (m *Monitor) Start(ctx context.Context) error {
	// Initialize scheduler with genesis time
	if err := m.scheduler.Initialize(ctx); err != nil {
		return err
	}

	// Check node sync status
	synced, err := m.client.IsNodeSynced(ctx)
	if err != nil {
		m.logger.Warn().Err(err).Msg("Failed to check node sync status")
	} else if !synced {
		m.logger.Warn().Msg("Beacon node is still syncing, results may be incomplete")
	}

	// Initialize reconciliation cursors from current chain state so we start
	// from "now" and then move forward without skipping.
	if headSlot, err := m.getCachedHeadSlot(ctx); err != nil {
		m.logger.Warn().Err(err).Msg("Failed to get initial head slot for cursors")
	} else {
		m.lastSnapshotSlot = headSlot
		currentEpoch := headSlot / config.SlotsPerEpoch()
		m.lastDutiesEpoch = currentEpoch

		if checkpoints, err := m.client.GetFinalityCheckpoints(ctx, "head"); err != nil {
			m.logger.Warn().Err(err).Msg("Failed to get initial finalized epoch for rewards cursor")
		} else {
			m.lastRewardsEpoch = checkpoints.Finalized.Epoch.Uint64()
		}
	}

	// Start worker pool
	m.workerPool.Start(ctx)

	// Start result processor
	m.wg.Add(1)
	go m.processResults(ctx)

	// Main monitoring loop
	m.wg.Add(1)
	go m.runLoop(ctx)

	m.logger.Info().
		Int("validators", len(m.cfg.Validators)).
		Int("workers", m.cfg.WorkerPoolSize).
		Msg("Monitor started")

	return nil
}

// runLoop is the main monitoring loop.
func (m *Monitor) runLoop(ctx context.Context) {
	defer m.wg.Done()

	m.logger.Info().Msg("Monitor loop started")

	for {
		// Wait for next interval to pace work
		_, err := m.scheduler.WaitForSlotInterval(ctx)
		if err != nil {
			if ctx.Err() != nil {
				m.logger.Info().Msg("Monitor loop shutting down")
				return
			}
			m.logger.Error().Err(err).Msg("Failed to wait for slot")
			continue
		}

		// Get current head slot from beacon (or cache)
		headSlot, err := m.getCachedHeadSlot(ctx)
		if err != nil {
			m.logger.Error().Err(err).Msg("Failed to get head slot")
			continue
		}

		currentEpoch := headSlot / config.SlotsPerEpoch()
		m.logger.Info().
			Uint64("slot", headSlot).
			Uint64("epoch", currentEpoch).
			Msg("Reconciling up to head slot")

		// Reconcile snapshots, duties, and rewards from last processed positions
		m.reconcileSnapshots(ctx, headSlot)
		m.reconcileEpochData(ctx, headSlot)
	}
}

// reconcileSnapshots ensures we process every slot from lastSnapshotSlot+1 up to headSlot,
// bounded per loop to respect rate limits.
func (m *Monitor) reconcileSnapshots(ctx context.Context, headSlot uint64) {
	const maxSlotsPerLoop = 32

	start := m.lastSnapshotSlot + 1
	if start > headSlot {
		return
	}

	slotsProcessed := 0
	for slot := start; slot <= headSlot && slotsProcessed < maxSlotsPerLoop; slot++ {
		event := ScheduleEvent{
			Slot:       slot,
			Epoch:      slot / config.SlotsPerEpoch(),
			Type:       EventTypeSlotPoll,
			Validators: m.cfg.Validators,
		}
		m.logger.Debug().
			Uint64("slot", slot).
			Int("validators_count", len(event.Validators)).
			Msg("Reconciling validator snapshots for slot")

		m.pollValidators(ctx, event)
		m.lastSnapshotSlot = slot
		slotsProcessed++
	}
}

// reconcileEpochData ensures we process every epoch for duties and rewards from the last
// seen cursors up to the current chain state, bounded per loop.
func (m *Monitor) reconcileEpochData(ctx context.Context, headSlot uint64) {
	currentEpoch := headSlot / config.SlotsPerEpoch()

	checkpoints, err := m.client.GetFinalityCheckpoints(ctx, "head")
	if err != nil {
		m.logger.Warn().Err(err).Msg("Failed to get finality checkpoints during reconciliation")
		return
	}
	finalizedEpoch := checkpoints.Finalized.Epoch.Uint64()

	const maxEpochsPerLoop = 8

	// Reconcile duties for upcoming epochs (current and next)
	dutiesTargetEpoch := currentEpoch + 1
	dutiesProcessed := 0
	for epoch := m.lastDutiesEpoch + 1; epoch <= dutiesTargetEpoch && dutiesProcessed < maxEpochsPerLoop; epoch++ {
		event := ScheduleEvent{
			Slot:       EpochStartSlot(epoch),
			Epoch:      epoch,
			Type:       EventTypeEpochBoundary,
			Validators: m.cfg.Validators,
		}
		m.logger.Debug().
			Uint64("epoch", epoch).
			Int("validators_count", len(event.Validators)).
			Msg("Reconciling attestation duties for epoch")

		m.fetchDuties(ctx, event)
		m.lastDutiesEpoch = epoch
		dutiesProcessed++
	}

	// Reconcile rewards for finalized epochs
	rewardsProcessed := 0
	for epoch := m.lastRewardsEpoch + 1; epoch <= finalizedEpoch && rewardsProcessed < maxEpochsPerLoop; epoch++ {
		event := ScheduleEvent{
			Slot:       EpochStartSlot(epoch),
			Epoch:      epoch,
			Type:       EventTypeEpochFinalized,
			Validators: m.cfg.Validators,
		}
		m.logger.Debug().
			Uint64("epoch", epoch).
			Int("validators_count", len(event.Validators)).
			Msg("Reconciling attestation rewards for epoch")

		m.fetchRewards(ctx, event)
		m.lastRewardsEpoch = epoch
		rewardsProcessed++
	}
}

// processEvent handles a scheduled event.
func (m *Monitor) processEvent(ctx context.Context, event ScheduleEvent) {
	switch event.Type {
	case EventTypeSlotPoll:
		m.pollValidators(ctx, event)
	case EventTypeEpochBoundary:
		m.fetchDuties(ctx, event)
	case EventTypeEpochFinalized:
		m.fetchRewards(ctx, event)
	}
}

// pollValidators submits jobs to poll validator status.
func (m *Monitor) pollValidators(ctx context.Context, event ScheduleEvent) {
	m.logger.Info().
		Uint64("slot", event.Slot).
		Uint64("epoch", event.Epoch).
		Int("validators_count", len(event.Validators)).
		Msg("Polling validators")

	for _, validatorIndex := range event.Validators {
		job := Job{
			ValidatorIndex: validatorIndex,
			Slot:           event.Slot,
			Epoch:          event.Epoch,
			Type:           JobTypeStatus,
		}

		m.logger.Debug().
			Uint64("validator_index", validatorIndex).
			Uint64("slot", event.Slot).
			Msg("Submitting validator status job")

		select {
		case <-ctx.Done():
			return
		default:
			m.workerPool.Submit(job)
		}
	}
}

// fetchDuties submits jobs to fetch attestation duties.
func (m *Monitor) fetchDuties(ctx context.Context, event ScheduleEvent) {
	m.logger.Info().
		Uint64("slot", event.Slot).
		Uint64("epoch", event.Epoch).
		Int("validators_count", len(event.Validators)).
		Msg("Fetching attestation duties for epoch")

	// Fetch duties for all validators at once (more efficient)
	job := Job{
		ValidatorIndex: 0, // Not used for duties
		Slot:           event.Slot,
		Epoch:          event.Epoch,
		Type:           JobTypeDuties,
	}

	m.logger.Debug().
		Uint64("epoch", event.Epoch).
		Uint64("slot", event.Slot).
		Msg("Submitting attestation duties job")

	select {
	case <-ctx.Done():
		return
	default:
		m.workerPool.Submit(job)
	}
}

// fetchRewards submits jobs to fetch attestation rewards.
func (m *Monitor) fetchRewards(ctx context.Context, event ScheduleEvent) {
	m.logger.Info().
		Uint64("slot", event.Slot).
		Uint64("epoch", event.Epoch).
		Int("validators_count", len(event.Validators)).
		Msg("Fetching attestation rewards for finalized epoch")

	// Fetch rewards for all validators at once
	job := Job{
		ValidatorIndex: 0, // Not used for rewards
		Slot:           event.Slot,
		Epoch:          event.Epoch,
		Type:           JobTypeRewards,
	}

	select {
	case <-ctx.Done():
		return
	default:
		m.workerPool.Submit(job)
	}
}

// Process implements JobProcessor interface.
func (m *Monitor) Process(ctx context.Context, job Job) (interface{}, error) {
	switch job.Type {
	case JobTypeStatus:
		return m.processStatusJob(ctx, job)
	case JobTypeDuties:
		return m.processDutiesJob(ctx, job)
	case JobTypeRewards:
		return m.processRewardsJob(ctx, job)
	default:
		return nil, nil
	}
}

// getCachedHeadSlot gets the head slot, using cache if recent (< 6 seconds old).
func (m *Monitor) getCachedHeadSlot(ctx context.Context) (uint64, error) {
	m.headSlotCache.mu.RLock()
	cached := m.headSlotCache.slot
	cacheTime := m.headSlotCache.timestamp
	m.headSlotCache.mu.RUnlock()

	// Use cache if less than 6 seconds old (half a slot)
	if cached > 0 && time.Since(cacheTime) < 6*time.Second {
		return cached, nil
	}

	// Fetch fresh slot
	slot, err := m.client.GetHeadSlot(ctx)
	if err != nil {
		// Return cached value if available, even if stale
		if cached > 0 {
			m.logger.Warn().Err(err).Uint64("cached_slot", cached).Msg("Using cached slot due to API error")
			return cached, nil
		}
		return 0, err
	}

	// Update cache
	m.headSlotCache.mu.Lock()
	m.headSlotCache.slot = slot
	m.headSlotCache.timestamp = time.Now()
	m.headSlotCache.mu.Unlock()

	return slot, nil
}

// processStatusJob fetches and stores validator status.
func (m *Monitor) processStatusJob(ctx context.Context, job Job) (*storage.ValidatorSnapshot, error) {
	m.logger.Debug().
		Uint64("validator_index", job.ValidatorIndex).
		Uint64("slot", job.Slot).
		Msg("Fetching validator status from beacon API at specific slot")

	validator, err := m.client.GetValidatorAtSlot(ctx, job.Slot, job.ValidatorIndex)
	if err != nil {
		m.logger.Error().
			Err(err).
			Uint64("validator_index", job.ValidatorIndex).
			Uint64("slot", job.Slot).
			Msg("Failed to fetch validator from beacon API at slot")
		return nil, fmt.Errorf("failed to fetch validator %d: %w", job.ValidatorIndex, err)
	}

	m.logger.Debug().
		Uint64("validator_index", job.ValidatorIndex).
		Str("status", validator.Status).
		Uint64("balance", validator.Balance.Uint64()).
		Uint64("effective_balance", validator.Validator.EffectiveBalance.Uint64()).
		Msg("Successfully fetched validator data from beacon API")

	snapshot := &storage.ValidatorSnapshot{
		ValidatorIndex:   job.ValidatorIndex,
		Slot:             job.Slot,
		Status:           validator.Status,
		Balance:          validator.Balance.Uint64(),
		EffectiveBalance: validator.Validator.EffectiveBalance.Uint64(),
		Timestamp:        time.Now().UTC(),
	}

	m.logger.Info().
		Uint64("validator_index", snapshot.ValidatorIndex).
		Uint64("slot", snapshot.Slot).
		Str("status", snapshot.Status).
		Uint64("balance_gwei", snapshot.Balance).
		Uint64("effective_balance_gwei", snapshot.EffectiveBalance).
		Msg("Prepared snapshot for database")

	// Save to database
	m.logger.Debug().
		Uint64("validator_index", snapshot.ValidatorIndex).
		Uint64("slot", snapshot.Slot).
		Msg("Attempting to save validator snapshot to database")

	if err := m.repo.SaveValidatorSnapshot(ctx, snapshot); err != nil {
		m.logger.Error().
			Err(err).
			Uint64("validator_index", job.ValidatorIndex).
			Uint64("slot", snapshot.Slot).
			Uint64("balance", snapshot.Balance).
			Uint64("effective_balance", snapshot.EffectiveBalance).
			Msg("Failed to save validator snapshot")
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	m.logger.Info().
		Uint64("validator_index", snapshot.ValidatorIndex).
		Uint64("slot", snapshot.Slot).
		Uint64("balance", snapshot.Balance).
		Uint64("effective_balance", snapshot.EffectiveBalance).
		Msg("Successfully saved validator snapshot to database")

	return snapshot, nil
}

// processDutiesJob fetches and stores attestation duties.
func (m *Monitor) processDutiesJob(ctx context.Context, job Job) ([]*storage.AttestationDuty, error) {
	m.logger.Debug().
		Uint64("epoch", job.Epoch).
		Int("validators_count", len(m.cfg.Validators)).
		Msg("Fetching attestation duties from beacon API")

	resp, err := m.client.GetAttesterDuties(ctx, job.Epoch, m.cfg.Validators)
	if err != nil {
		m.logger.Error().
			Err(err).
			Uint64("epoch", job.Epoch).
			Msg("Failed to fetch attestation duties from beacon API")
		return nil, fmt.Errorf("failed to fetch duties for epoch %d: %w", job.Epoch, err)
	}

	m.logger.Debug().
		Uint64("epoch", job.Epoch).
		Int("duties_count", len(resp.Data)).
		Msg("Successfully fetched attestation duties from beacon API")

	if len(resp.Data) == 0 {
		m.logger.Warn().
			Uint64("epoch", job.Epoch).
			Msg("No duties returned for epoch")
		return []*storage.AttestationDuty{}, nil
	}

	duties := make([]*storage.AttestationDuty, 0, len(resp.Data))
	for _, d := range resp.Data {
		duty := &storage.AttestationDuty{
			ValidatorIndex:    d.ValidatorIndex.Uint64(),
			Epoch:             job.Epoch,
			Slot:              d.Slot.Uint64(),
			CommitteeIndex:    int(d.CommitteeIndex.Uint64()),
			CommitteePosition: int(d.ValidatorCommitteeIndex.Uint64()),
			Timestamp:         time.Now().UTC(),
		}
		duties = append(duties, duty)
	}

	m.logger.Info().
		Uint64("epoch", job.Epoch).
		Int("duties_count", len(duties)).
		Msg("Prepared duties for database")

	// Save to database
	m.logger.Debug().
		Uint64("epoch", job.Epoch).
		Int("duties_count", len(duties)).
		Msg("Attempting to save attestation duties to database")

	if err := m.repo.SaveAttestationDuties(ctx, duties); err != nil {
		m.logger.Error().
			Err(err).
			Uint64("epoch", job.Epoch).
			Int("duties_count", len(duties)).
			Msg("Failed to save attestation duties")
		return nil, fmt.Errorf("failed to save duties: %w", err)
	}

	m.logger.Info().
		Uint64("epoch", job.Epoch).
		Int("duties_count", len(duties)).
		Msg("Successfully saved attestation duties to database")

	return duties, nil
}

// processRewardsJob fetches and stores attestation rewards.
func (m *Monitor) processRewardsJob(ctx context.Context, job Job) ([]*storage.AttestationReward, error) {
	resp, err := m.client.GetAttestationRewards(ctx, job.Epoch, m.cfg.Validators)
	if err != nil {
		return nil, err
	}

	rewards := make([]*storage.AttestationReward, 0, len(resp.TotalRewards))
	var penalties []*storage.ValidatorPenalty

	for _, r := range resp.TotalRewards {
		totalReward := r.Head.Int64() + r.Source.Int64() + r.Target.Int64()

		reward := &storage.AttestationReward{
			ValidatorIndex: r.ValidatorIndex.Uint64(),
			Epoch:          job.Epoch,
			HeadReward:     r.Head.Int64(),
			SourceReward:   r.Source.Int64(),
			TargetReward:   r.Target.Int64(),
			TotalReward:    totalReward,
			Timestamp:      time.Now().UTC(),
		}
		rewards = append(rewards, reward)

		// Track penalties separately
		if totalReward < 0 {
			penalty := &storage.ValidatorPenalty{
				ValidatorIndex: r.ValidatorIndex.Uint64(),
				Epoch:          job.Epoch,
				Slot:           job.Slot,
				PenaltyType:    storage.PenaltyTypeAttestationMiss,
				PenaltyGwei:    -totalReward, // Convert to positive
				Timestamp:      time.Now().UTC(),
			}
			penalties = append(penalties, penalty)
		}
	}

	// Save rewards to database
	if err := m.repo.SaveAttestationRewards(ctx, rewards); err != nil {
		m.logger.Error().
			Err(err).
			Uint64("epoch", job.Epoch).
			Int("rewards_count", len(rewards)).
			Msg("Failed to save attestation rewards")
		return nil, fmt.Errorf("failed to save rewards: %w", err)
	}

	m.logger.Info().
		Uint64("epoch", job.Epoch).
		Int("rewards_count", len(rewards)).
		Msg("Successfully saved attestation rewards to database")

	// Save penalties to database
	for _, penalty := range penalties {
		if err := m.repo.SaveValidatorPenalty(ctx, penalty); err != nil {
			m.logger.Error().
				Err(err).
				Uint64("validator_index", penalty.ValidatorIndex).
				Msg("Failed to save validator penalty")
		}
	}

	return rewards, nil
}

// processResults processes results from the worker pool and logs them.
func (m *Monitor) processResults(ctx context.Context) {
	defer m.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-m.workerPool.Results():
			if !ok {
				return
			}

			if result.Error != nil {
				m.logger.Error().
					Err(result.Error).
					Uint64("validator_index", result.Job.ValidatorIndex).
					Uint64("slot", result.Job.Slot).
					Int("job_type", int(result.Job.Type)).
					Msg("Job failed")
				continue
			}

			// Log successful results
			m.logResult(result)
		}
	}
}

// logResult logs the result in the required JSON format.
func (m *Monitor) logResult(result Result) {
	switch result.Job.Type {
	case JobTypeStatus:
		if snapshot, ok := result.Data.(*storage.ValidatorSnapshot); ok {
			m.logger.Info().
				Uint64("slot", snapshot.Slot).
				Uint64("validator_index", snapshot.ValidatorIndex).
				Str("status", snapshot.Status).
				Uint64("effective_balance_gwei", snapshot.EffectiveBalance).
				Uint64("balance_gwei", snapshot.Balance).
				Msg("validator_status")
		}

	case JobTypeDuties:
		if duties, ok := result.Data.([]*storage.AttestationDuty); ok {
			for _, duty := range duties {
				m.logger.Info().
					Uint64("slot", duty.Slot).
					Uint64("epoch", duty.Epoch).
					Uint64("validator_index", duty.ValidatorIndex).
					Int("committee_index", duty.CommitteeIndex).
					Int("committee_position", duty.CommitteePosition).
					Msg("attestation_duty")
			}
		}

	case JobTypeRewards:
		if rewards, ok := result.Data.([]*storage.AttestationReward); ok {
			for _, reward := range rewards {
				m.logger.Info().
					Uint64("epoch", reward.Epoch).
					Uint64("validator_index", reward.ValidatorIndex).
					Int64("head_reward", reward.HeadReward).
					Int64("source_reward", reward.SourceReward).
					Int64("target_reward", reward.TargetReward).
					Int64("total_reward_gwei", reward.TotalReward).
					Bool("duty_success", reward.TotalReward >= 0).
					Msg("attestation_reward")
			}
		}
	}
}

// Stop gracefully shuts down the monitor.
func (m *Monitor) Stop() {
	m.logger.Info().Msg("Stopping monitor...")
	m.workerPool.Stop()
	m.wg.Wait()
	m.logger.Info().Msg("Monitor stopped")
}

// Wait blocks until the monitor is stopped.
func (m *Monitor) Wait() {
	m.wg.Wait()
}
