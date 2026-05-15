package monitor

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/execution"
	"github.com/tharun/pauli/internal/monitor/queue"
	runbackfill "github.com/tharun/pauli/internal/monitor/runner/backfill"
	runrealtime "github.com/tharun/pauli/internal/monitor/runner/realtime"
	"github.com/tharun/pauli/internal/storage"
)

// Monitor wires the network clock, runners, and a concurrent queue (workers run steps.Job via Step.RunAsync).
// Indexing uses runner/realtime.Runner (runner.Runner) only; historical backfill can be added later.
type Monitor struct {
	cfg     *config.Config
	client  *beacon.Client
	repo    storage.Repository
	network *config.BlockchainNetwork
	pool    *queue.Pool
	logger  zerolog.Logger
	wg      sync.WaitGroup
}

// NewMonitor creates a new Monitor instance.
func NewMonitor(cfg *config.Config, client *beacon.Client, repo storage.Repository, logger zerolog.Logger) *Monitor {
	network := config.NewBlockchainNetwork(cfg)
	m := &Monitor{
		cfg:     cfg,
		client:  client,
		repo:    repo,
		network: network,
		logger:  logger,
	}

	m.pool = queue.NewPool(cfg.WorkerPoolSize, queue.StepJobRunner(), logger)

	return m
}

// Start begins the monitoring loop.
func (m *Monitor) Start(ctx context.Context) error {
	if err := InitBeaconNetworkClock(ctx, m.client, m.network, m.logger); err != nil {
		return err
	}

	m.logNodeSyncStatus(ctx)

	enqueue := m.pool.Enqueue
	execClient := execution.NewClient(m.cfg)
	realtimeR := runrealtime.New(m.network, m.client, execClient, m.repo, m.client.GetHeadSlot, m.cfg.Validators, m.logger, enqueue)
	if maxSlot, ok, err := m.repo.MaxIndexedSlot(ctx); err != nil {
		m.logger.Warn().Err(err).Msg("seed realtime cursor: max indexed slot lookup failed")
	} else if ok {
		realtimeR.SetLastProcessedSlot(maxSlot)
		m.logger.Debug().Uint64("last_processed_slot", maxSlot).Msg("seeded realtime cursor from indexer_progress")
	}

	m.pool.Start(ctx)

	m.startBackgroundWorker(ctx, func(runCtx context.Context) { realtimeR.Start(runCtx) })

	if m.cfg.Backfill.Enabled {
		backfillR := runbackfill.New(m.cfg.Backfill, runbackfill.Options{}, m.client, execClient, m.repo, m.client.GetHeadSlot, m.logger.With().Str("runner", "backfill").Logger(), enqueue)
		m.startBackgroundWorker(ctx, func(runCtx context.Context) { backfillR.Start(runCtx) })
		m.logger.Info().Msg("backfill runner started")
	}

	m.logger.Info().
		Int("validators", len(m.cfg.Validators)).
		Int("workers", m.cfg.WorkerPoolSize).
		Bool("backfill", m.cfg.Backfill.Enabled).
		Msg("monitor started")

	return nil
}

func (m *Monitor) startBackgroundWorker(ctx context.Context, run func(context.Context)) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		run(ctx)
	}()
}

// Stop shuts down the monitor: waits for runners to exit (caller should cancel its context first),
// then drains the worker pool using drainCtx for in-flight and queued jobs.
func (m *Monitor) Stop(drainCtx context.Context) {
	if drainCtx == nil {
		drainCtx = context.Background()
	}
	m.logger.Info().Msg("monitor stopping")
	m.wg.Wait()
	m.pool.Stop(drainCtx)
	m.logger.Info().Msg("monitor stopped")
}

// Wait blocks until the monitor is stopped.
func (m *Monitor) Wait() {
	m.wg.Wait()
}
