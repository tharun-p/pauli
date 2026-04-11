package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/monitor/steps"
)

// Runner performs one queued steps.Job (async step body).
type Runner interface {
	Run(ctx context.Context, job steps.Job) error
}

// Pool runs queued steps concurrently with a fixed number of workers.
type Pool struct {
	size     int
	workChan chan steps.Job
	wg       sync.WaitGroup
	runner   Runner
	logger   zerolog.Logger

	mu      sync.RWMutex
	runCtx  context.Context // context passed to Runner.Run; replaced before drain on Stop
	stopped bool
}

func NewPool(size int, runner Runner, logger zerolog.Logger) *Pool {
	return &Pool{
		size:     size,
		workChan: make(chan steps.Job, size*2),
		runner:   runner,
		logger:   logger,
	}
}

// Start launches workers. runCtx is used for Runner.Run until Stop replaces it with the drain context.
func (p *Pool) Start(runCtx context.Context) {
	p.mu.Lock()
	p.runCtx = runCtx
	p.mu.Unlock()

	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	p.logger.Debug().Int("worker_id", id).Msg("indexing worker started")

	for {
		job, ok := <-p.workChan
		if !ok {
			p.logger.Debug().Int("worker_id", id).Msg("indexing worker work channel closed")
			return
		}
		stepName := "<nil>"
		if job.Step != nil {
			stepName = fmt.Sprintf("%T", job.Step)
		}
		p.mu.RLock()
		rc := p.runCtx
		p.mu.RUnlock()
		if rc == nil {
			rc = context.Background()
		}
		if err := p.runner.Run(rc, job); err != nil {
			p.logger.Error().Err(err).Int("worker_id", id).Str("step", stepName).Msg("async step failed")
		}
	}
}

// ErrPoolStopped is returned from Enqueue after Stop has closed the work channel.
var ErrPoolStopped = errors.New("pool stopped")

func (p *Pool) Enqueue(ctx context.Context, job steps.Job) error {
	p.mu.RLock()
	stopped := p.stopped
	p.mu.RUnlock()
	if stopped {
		return ErrPoolStopped
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.workChan <- job:
		return nil
	}
}

// Stop closes the work channel and waits for workers to drain queued jobs.
// drainCtx is used as the context for Runner.Run while finishing the queue (e.g. shutdown timeout from main).
// Callers should stop producers (e.g. cancel the runner context) before Stop so no new jobs are enqueued.
func (p *Pool) Stop(drainCtx context.Context) {
	if drainCtx == nil {
		drainCtx = context.Background()
	}

	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		p.wg.Wait()
		return
	}
	p.stopped = true
	p.runCtx = drainCtx
	p.mu.Unlock()
	close(p.workChan)
	p.wg.Wait()
}
