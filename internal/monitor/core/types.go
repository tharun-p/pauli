package core

import "context"

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

// JobProcessor defines the interface for processing jobs.
type JobProcessor interface {
	Process(ctx context.Context, job Job) (interface{}, error)
}
