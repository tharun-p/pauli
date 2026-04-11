package queue

import (
	"context"
	"fmt"

	"github.com/tharun/pauli/internal/monitor/steps"
)

// StepJobRunner returns a Runner that executes job.Step.RunAsync.
func StepJobRunner() Runner {
	return stepJobRunner{}
}

type stepJobRunner struct{}

func (stepJobRunner) Run(ctx context.Context, job steps.Job) error {
	if job.Step == nil {
		return fmt.Errorf("nil step in job")
	}
	return job.Step.RunAsync(ctx, &job.Env)
}
