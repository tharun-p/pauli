package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	monitorcache "github.com/tharun/pauli/internal/monitor/cache"
	"github.com/tharun/pauli/internal/monitor/dispatch"
	"github.com/tharun/pauli/internal/monitor/jobs"
	"github.com/tharun/pauli/internal/monitor/runners/realtime"
	"github.com/tharun/pauli/internal/monitor/runners/reconcile"
	scheduler "github.com/tharun/pauli/internal/monitor/scheduler"
	"github.com/tharun/pauli/internal/storage"
)

const headSlotCacheTTL = 6 * time.Second

// Monitor orchestrates the validator monitoring process.
type Monitor struct {
	cfg        *config.Config
	client     *beacon.Client
	repo       storage.Repository
	scheduler  *scheduler.Scheduler
	realtime   *realtime.Runner
	reconciler *reconcile.Runner
	dispatcher *dispatch.Dispatcher
	headSlot   *monitorcache.HeadSlotCache
	workerPool *WorkerPool
	logger     zerolog.Logger
	wg         sync.WaitGroup
}

// NewMonitor creates a new Monitor instance.
func NewMonitor(cfg *config.Config, client *beacon.Client, repo storage.Repository, logger zerolog.Logger) *Monitor {
	m := &Monitor{
		cfg:       cfg,
		client:    client,
		repo:      repo,
		scheduler: scheduler.New(client, cfg.Validators, cfg.PollingIntervalSlots, cfg.SlotDuration(), logger),
		logger:    logger,
	}

	m.initWorkers()
	m.initCache()
	m.initRunners()
	return m
}

func (m *Monitor) initWorkers() {
	jobProcessor := jobs.NewProcessor(m.client, m.repo, m.cfg.Validators, m.logger)
	m.workerPool = NewWorkerPool(m.cfg.WorkerPoolSize, jobProcessor, m.logger)
	m.dispatcher = dispatch.New(m.cfg.Validators, m.workerPool.Submit, m.logger)
}

func (m *Monitor) initCache() {
	m.headSlot = monitorcache.NewHeadSlotCache(
		headSlotCacheTTL,
		func(ctx context.Context) (uint64, error) {
			return m.client.GetHeadSlot(ctx)
		},
		m.logger,
	)
}

func (m *Monitor) initRunners() {
	m.reconciler = reconcile.New(
		m.cfg.Validators,
		m.headSlot.Get,
		m.getFinalizedEpoch,
		m.waitForSlotInterval,
		m.dispatcher.PollValidatorsForSlotEpoch,
		m.dispatcher.FetchDutiesForEpoch,
		m.dispatcher.FetchRewardsForEpoch,
		m.logger,
	)
	m.realtime = realtime.New(
		m.waitForSlotInterval,
		m.headSlot.Get,
		m.handleRealtimeSlot,
		m.logger,
	)
}

// Start begins the monitoring loop.
func (m *Monitor) Start(ctx context.Context) error {
	// Initialize scheduler with genesis time
	if err := m.scheduler.Initialize(ctx); err != nil {
		return err
	}

	m.logNodeSyncStatus(ctx)

	// Initialize reconciliation cursors, then start backfill in parallel.
	m.reconciler.InitializeCursors(ctx)

	// Start worker pool
	m.workerPool.Start(ctx)

	m.startBackgroundWorker(ctx, func(runCtx context.Context) { m.processResults(runCtx) })
	m.startBackgroundWorker(ctx, func(runCtx context.Context) { m.realtime.Run(runCtx) })
	m.startBackgroundWorker(ctx, func(runCtx context.Context) { m.reconciler.Run(runCtx) })

	m.logger.Info().
		Int("validators", len(m.cfg.Validators)).
		Int("workers", m.cfg.WorkerPoolSize).
		Msg("Monitor started")

	return nil
}

func (m *Monitor) startBackgroundWorker(ctx context.Context, run func(context.Context)) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		run(ctx)
	}()
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
