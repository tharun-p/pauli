package runner

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/monitor/steps"
)

// Runner is a pluggable monitor mode: pacing hooks, StepChain, Env, enqueue, and Start (which should call Run(ctx, m)).
type Runner interface {
	Name() string
	Logger() zerolog.Logger
	// Env is the per-iteration step context; Run calls Env().Reset(ctx) before each chain. Non-nil for conforming implementations.
	Env() *steps.Env
	BeforeStep(ctx context.Context) error
	AfterStep(ctx context.Context) error
	// StepChain returns the linear steps for one iteration and whether Run should stop after this pass.
	StepChain(ctx context.Context) ([]steps.Step, bool, error)
	SleepOnSeedError() time.Duration
	Enqueue(ctx context.Context, job steps.Job) error
	Start(ctx context.Context)
}
