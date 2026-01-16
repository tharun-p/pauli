package monitor

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
)

// Job represents a unit of work to be processed by a worker.
type Job struct {
	ValidatorIndex uint64
	Slot           uint64
	Epoch          uint64
	Type           JobType
}

// JobType defines the type of monitoring job.
type JobType int

const (
	// JobTypeStatus fetches validator status and balance.
	JobTypeStatus JobType = iota
	// JobTypeDuties fetches attestation duties for an epoch.
	JobTypeDuties
	// JobTypeRewards fetches attestation rewards for an epoch.
	JobTypeRewards
)

// Result represents the outcome of processing a job.
type Result struct {
	Job   Job
	Data  interface{}
	Error error
}

// WorkerPool manages a pool of workers for concurrent processing.
type WorkerPool struct {
	size       int
	jobChan    chan Job
	resultChan chan Result
	wg         sync.WaitGroup
	processor  JobProcessor
	logger     zerolog.Logger
}

// JobProcessor defines the interface for processing jobs.
type JobProcessor interface {
	Process(ctx context.Context, job Job) (interface{}, error)
}

// NewWorkerPool creates a new WorkerPool with the specified size.
func NewWorkerPool(size int, processor JobProcessor, logger zerolog.Logger) *WorkerPool {
	return &WorkerPool{
		size:       size,
		jobChan:    make(chan Job, size*2),
		resultChan: make(chan Result, size*2),
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
			result := Result{
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
func (wp *WorkerPool) Submit(job Job) {
	wp.jobChan <- job
}

// SubmitBatch adds multiple jobs to the queue.
func (wp *WorkerPool) SubmitBatch(jobs []Job) {
	for _, job := range jobs {
		wp.jobChan <- job
	}
}

// Results returns the result channel for reading completed jobs.
func (wp *WorkerPool) Results() <-chan Result {
	return wp.resultChan
}

// Stop gracefully shuts down the worker pool.
func (wp *WorkerPool) Stop() {
	close(wp.jobChan)
	wp.wg.Wait()
	close(wp.resultChan)
	wp.logger.Info().Msg("Worker pool stopped")
}

// Drain reads all remaining results from the result channel.
func (wp *WorkerPool) Drain() []Result {
	var results []Result
	for result := range wp.resultChan {
		results = append(results, result)
	}
	return results
}

// SubmitAndWait submits jobs and waits for all results.
func (wp *WorkerPool) SubmitAndWait(ctx context.Context, jobs []Job) []Result {
	// Submit all jobs
	go func() {
		for _, job := range jobs {
			select {
			case wp.jobChan <- job:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	results := make([]Result, 0, len(jobs))
	for i := 0; i < len(jobs); i++ {
		select {
		case result := <-wp.resultChan:
			results = append(results, result)
		case <-ctx.Done():
			return results
		}
	}

	return results
}
