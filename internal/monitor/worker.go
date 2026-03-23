package monitor

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/monitor/core"
)

// WorkerPool manages a pool of workers for concurrent processing.
type WorkerPool struct {
	size       int
	jobChan    chan core.Job
	resultChan chan core.Result
	wg         sync.WaitGroup
	processor  core.JobProcessor
	logger     zerolog.Logger
}

// NewWorkerPool creates a new WorkerPool with the specified size.
func NewWorkerPool(size int, processor core.JobProcessor, logger zerolog.Logger) *WorkerPool {
	return &WorkerPool{
		size:       size,
		jobChan:    make(chan core.Job, size*2),
		resultChan: make(chan core.Result, size*2),
		processor:  processor,
		logger:     logger,
	}
}

// Start launches the worker goroutines.
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.size; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}

	wp.logger.Info().Int("workers", wp.size).Msg("Worker pool started")
}

// worker is the main worker loop.
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	wp.logger.Debug().Int("worker_id", id).Msg("Worker started")

	for {
		select {
		case <-ctx.Done():
			wp.logger.Debug().Int("worker_id", id).Msg("Worker shutting down")
			return
		case job, ok := <-wp.jobChan:
			if !ok {
				wp.logger.Debug().Int("worker_id", id).Msg("Job channel closed")
				return
			}

			data, err := wp.processor.Process(ctx, job)
			result := core.Result{
				Job:   job,
				Data:  data,
				Error: err,
			}

			select {
			case wp.resultChan <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Submit adds a job to the queue.
func (wp *WorkerPool) Submit(ctx context.Context, job core.Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case wp.jobChan <- job:
		return nil
	}
}

// Results returns the result channel for reading completed jobs.
func (wp *WorkerPool) Results() <-chan core.Result {
	return wp.resultChan
}

// Stop gracefully shuts down the worker pool.
func (wp *WorkerPool) Stop() {
	close(wp.jobChan)
	wp.wg.Wait()
	close(wp.resultChan)
	wp.logger.Info().Msg("Worker pool stopped")
}
