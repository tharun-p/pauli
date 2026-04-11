package runner

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/monitor/steps"
)

// Run executes the standard monitor loop for runner until ctx is done or StepChain requests stop.
func Run(ctx context.Context, runner Runner) {
	engine := engine{runner: runner}
	engine.run(ctx)
}

type engine struct {
	runner Runner
}

func (engine *engine) run(ctx context.Context) {
	name := engine.runner.Name()
	if name == "" {
		name = "runner"
	}
	log := engine.runner.Logger().With().Str("runner", name).Logger()
	log.Debug().Msg("started")
	defer log.Debug().Msg("stopped")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := engine.runner.BeforeStep(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error().Err(err).Msg("before-step failed")
			continue
		}

		chain, stop, err := engine.runner.StepChain(ctx)
		if err != nil {
			log.Error().Err(err).Msg("step chain failed")
			if pauseOrExit(ctx, engine.runner.SleepOnSeedError()) {
				return
			}
			continue
		}

		env := engine.runner.Env()
		if env == nil {
			env = steps.NewEnv()
		}
		env.Reset(ctx)

		if engine.runStepChain(ctx, log, env, chain, engine.runner.SleepOnSeedError()) {
			return
		}

		if stop {
			return
		}

		if err := engine.runner.AfterStep(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error().Err(err).Msg("after-step failed")
		}
	}
}

func (engine *engine) runStepChain(ctx context.Context, log zerolog.Logger, env *steps.Env, chain []steps.Step, errDelay time.Duration) (exitRun bool) {
	for _, step := range chain {
		enqueue, err := step.Run(env)
		if err != nil {
			log.Error().Err(err).Msg("step failed")
			if errDelay > 0 && pauseOrExit(ctx, errDelay) {
				return true
			}
			return false
		}
		if !step.Async() || !enqueue {
			continue
		}
		job := steps.Job{Step: step, Env: env.Clone()}
		if err := engine.runner.Enqueue(ctx, job); err != nil {
			if ctx.Err() != nil {
				return true
			}
			log.Error().Err(err).Msg("enqueue failed")
			if errDelay > 0 && pauseOrExit(ctx, errDelay) {
				return true
			}
			return false
		}
	}
	return false
}

// pauseOrExit sleeps d (if d > 0) or returns immediately; returns true if ctx is cancelled (caller should exit Run).
func pauseOrExit(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() != nil
	}
	select {
	case <-ctx.Done():
		return true
	case <-time.After(d):
		return ctx.Err() != nil
	}
}
