package monitor

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/monitor/queue"
	runrealtime "github.com/tharun/pauli/internal/monitor/runner/realtime"
	"github.com/tharun/pauli/internal/storage"
)

// Monitor wires the network clock, runners, and a concurrent queue (workers run steps.Job via Step.RunAsync).
// Indexing uses runner/realtime.Runner (runner.Runner) only; historical backfill can be added later.
type Monitor struct {
	cfg               *config.Config
	client            *beacon.Client
	repo              storage.Repository
	network *config.BlockchainNetwork
	pool    *queue.Pool
	logger            zerolog.Logger
	wg                sync.WaitGroup
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
	if err := initBeaconNetworkClock(ctx, m.client, m.network, m.logger); err != nil {
		return err
	}

	m.logNodeSyncStatus(ctx)

	enqueue := m.pool.Enqueue
	realtimeR := runrealtime.New(m.network, m.client, m.repo, m.client.GetHeadSlot, m.cfg.Validators, m.logger, enqueue)

	m.pool.Start(ctx)

	m.startBackgroundWorker(ctx, func(runCtx context.Context) { realtimeR.Start(runCtx) })

	m.logger.Debug().
		Int("validators", len(m.cfg.Validators)).
		Int("workers", m.cfg.WorkerPoolSize).
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

// Stop gracefully shuts down the monitor.
func (m *Monitor) Stop() {
	m.logger.Debug().Msg("monitor stopping")
	m.pool.Stop()
	m.wg.Wait()
	m.logger.Debug().Msg("monitor stopped")
}

// Wait blocks until the monitor is stopped.
func (m *Monitor) Wait() {
	m.wg.Wait()
}
