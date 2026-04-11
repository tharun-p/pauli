package queue

import (
	"context"
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
}

func NewPool(size int, runner Runner, logger zerolog.Logger) *Pool {
	return &Pool{
		size:     size,
		workChan: make(chan steps.Job, size*2),
		runner:   runner,
		logger:   logger,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	p.logger.Debug().Int("worker_id", id).Msg("indexing worker started")

	for {
		select {
		case <-ctx.Done():
			p.logger.Debug().Int("worker_id", id).Msg("indexing worker shutdown")
			return
		case job, ok := <-p.workChan:
			if !ok {
				p.logger.Debug().Int("worker_id", id).Msg("indexing worker work channel closed")
				return
			}
			stepName := "<nil>"
			if job.Step != nil {
				stepName = fmt.Sprintf("%T", job.Step)
			}
			if err := p.runner.Run(ctx, job); err != nil {
				p.logger.Debug().Err(err).Int("worker_id", id).Str("step", stepName).Msg("async step failed")
			}
		}
	}
}

func (p *Pool) Enqueue(ctx context.Context, job steps.Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.workChan <- job:
		return nil
	}
}

func (p *Pool) Stop() {
	close(p.workChan)
	p.wg.Wait()
}
