package monitor

import (
	"context"
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
	repo       *storage.Repository
	scheduler  *Scheduler
	workerPool *WorkerPool
	logger     zerolog.Logger
	wg         sync.WaitGroup
}

// NewMonitor creates a new Monitor instance.
func NewMonitor(cfg *config.Config, client *beacon.Client, repo *storage.Repository, logger zerolog.Logger) *Monitor {
	scheduler := NewScheduler(client, cfg.Validators, cfg.PollingIntervalSlots, logger)

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

	for {
		// Wait for next slot interval
		slot, err := m.scheduler.WaitForSlotInterval(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			m.logger.Error().Err(err).Msg("Failed to wait for slot")
			continue
		}

		// Get scheduled events
		events, err := m.scheduler.NextEvents(ctx, slot)
		if err != nil {
			m.logger.Error().Err(err).Msg("Failed to get scheduled events")
			continue
		}

		// Process each event
		for _, event := range events {
			m.processEvent(ctx, event)
		}
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
	for _, validatorIndex := range event.Validators {
		job := Job{
			ValidatorIndex: validatorIndex,
			Slot:           event.Slot,
			Epoch:          event.Epoch,
			Type:           JobTypeStatus,
		}

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
	// Fetch duties for all validators at once (more efficient)
	job := Job{
		ValidatorIndex: 0, // Not used for duties
		Slot:           event.Slot,
		Epoch:          event.Epoch,
		Type:           JobTypeDuties,
	}

	select {
	case <-ctx.Done():
		return
	default:
		m.workerPool.Submit(job)
	}
}

// fetchRewards submits jobs to fetch attestation rewards.
func (m *Monitor) fetchRewards(ctx context.Context, event ScheduleEvent) {
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

// processStatusJob fetches and stores validator status.
func (m *Monitor) processStatusJob(ctx context.Context, job Job) (*storage.ValidatorSnapshot, error) {
	validator, err := m.client.GetValidator(ctx, "head", job.ValidatorIndex)
	if err != nil {
		return nil, err
	}

	snapshot := &storage.ValidatorSnapshot{
		ValidatorIndex:   job.ValidatorIndex,
		Slot:             job.Slot,
		Status:           validator.Status,
		Balance:          validator.Balance.Uint64(),
		EffectiveBalance: validator.Validator.EffectiveBalance.Uint64(),
		Timestamp:        time.Now().UTC(),
	}

	// Save to database
	if err := m.repo.SaveValidatorSnapshot(ctx, snapshot); err != nil {
		m.logger.Error().
			Err(err).
			Uint64("validator_index", job.ValidatorIndex).
			Msg("Failed to save validator snapshot")
	}

	return snapshot, nil
}

// processDutiesJob fetches and stores attestation duties.
func (m *Monitor) processDutiesJob(ctx context.Context, job Job) ([]*storage.AttestationDuty, error) {
	resp, err := m.client.GetAttesterDuties(ctx, job.Epoch, m.cfg.Validators)
	if err != nil {
		return nil, err
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

	// Save to database
	if err := m.repo.SaveAttestationDuties(ctx, duties); err != nil {
		m.logger.Error().
			Err(err).
			Uint64("epoch", job.Epoch).
			Msg("Failed to save attestation duties")
	}

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
			Msg("Failed to save attestation rewards")
	}

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
