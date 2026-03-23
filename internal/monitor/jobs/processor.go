package jobs

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/monitor/core"
	"github.com/tharun/pauli/internal/storage"
)

// Processor executes monitoring jobs against beacon API and storage.
type Processor struct {
	handlers map[core.JobType]Handler
}

// Handler is a common interface implemented by each job type processor.
type Handler interface {
	Process(ctx context.Context, job core.Job) (interface{}, error)
}

func NewProcessor(
	client *beacon.Client,
	repo storage.Repository,
	validators []uint64,
	logger zerolog.Logger,
) *Processor {
	statusHandler := NewStatusHandler(client, repo, logger)
	dutiesHandler := NewDutiesHandler(client, repo, validators, logger)
	rewardsHandler := NewRewardsHandler(client, repo, validators, logger)

	return &Processor{
		handlers: map[core.JobType]Handler{
			core.JobTypeStatus:  statusHandler,
			core.JobTypeDuties:  dutiesHandler,
			core.JobTypeRewards: rewardsHandler,
		},
	}
}

// Process implements core.JobProcessor.
func (p *Processor) Process(ctx context.Context, job core.Job) (interface{}, error) {
	handler, ok := p.handlers[job.Type]
	if !ok {
		return nil, nil
	}
	return handler.Process(ctx, job)
}
