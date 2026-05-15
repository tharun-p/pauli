package backfill

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/execution"
	"github.com/tharun/pauli/internal/monitor/runner"
	"github.com/tharun/pauli/internal/monitor/steps"
	stepbf "github.com/tharun/pauli/internal/monitor/steps/backfill"
	"github.com/tharun/pauli/internal/storage"
)

// Runner implements runner.Runner for dual-track slot and epoch backfill.
type Runner struct {
	cfg     config.BackfillConf
	opts    Options
	client  *beacon.Client
	exec    *execution.Client
	repo    storage.Repository
	getHead func(context.Context) (uint64, error)
	log     zerolog.Logger
	enqueue func(context.Context, steps.Job) error
	idle    bool
	env     *steps.Env
}

var _ runner.Runner = (*Runner)(nil)

// New constructs a backfill runner.
func New(
	cfg config.BackfillConf,
	opts Options,
	client *beacon.Client,
	exec *execution.Client,
	repo storage.Repository,
	getHead func(context.Context) (uint64, error),
	log zerolog.Logger,
	enqueue func(context.Context, steps.Job) error,
) *Runner {
	return &Runner{
		cfg:     cfg,
		opts:    opts,
		client:  client,
		exec:    exec,
		repo:    repo,
		getHead: getHead,
		log:     log,
		enqueue: enqueue,
		env:     steps.NewEnv(),
	}
}

func (r *Runner) Name() string { return "backfill" }

func (r *Runner) Logger() zerolog.Logger { return r.log }

func (r *Runner) Env() *steps.Env { return r.env }

func (r *Runner) Enqueue(ctx context.Context, job steps.Job) error {
	return r.enqueue(ctx, job)
}

func (r *Runner) BeforeStep(ctx context.Context) error {
	delay := r.cfg.PollDelay()
	if r.idle {
		r.log.Debug().Dur("poll_delay", delay).Msg("backfill idle; waiting")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

func (r *Runner) AfterStep(ctx context.Context) error {
	caught, err := r.caughtUp(ctx)
	if err != nil {
		return err
	}
	r.idle = caught
	return nil
}

func (r *Runner) caughtUp(ctx context.Context) (bool, error) {
	endSlot, err := r.slotTarget(ctx)
	if err != nil {
		return false, err
	}
	floorSlot := r.cfg.StartSlot
	if r.opts.StartSlot != nil {
		floorSlot = *r.opts.StartSlot
	}
	if endSlot >= floorSlot {
		_, slotOK, err := r.repo.FirstUnindexedSlot(ctx, floorSlot, endSlot)
		if err != nil {
			return false, err
		}
		if slotOK {
			return false, nil
		}
	}

	endEpoch, err := r.epochTarget(ctx)
	if err != nil {
		return false, err
	}
	floorEpoch := r.cfg.StartEpoch
	if r.opts.StartEpoch != nil {
		floorEpoch = *r.opts.StartEpoch
	}
	if endEpoch >= floorEpoch {
		_, epochOK, err := r.repo.FirstUnindexedEpoch(ctx, floorEpoch, endEpoch)
		if err != nil {
			return false, err
		}
		if epochOK {
			return false, nil
		}
	}
	return true, nil
}

func (r *Runner) slotTarget(ctx context.Context) (uint64, error) {
	if r.opts.EndSlot != nil {
		return *r.opts.EndSlot, nil
	}
	head, err := r.getHead(ctx)
	if err != nil {
		return 0, err
	}
	lag := r.cfg.LagBehindHead
	if head > lag {
		return head - lag, nil
	}
	return 0, nil
}

func (r *Runner) epochTarget(ctx context.Context) (uint64, error) {
	if r.opts.EndEpoch != nil {
		return *r.opts.EndEpoch, nil
	}
	finalized, err := r.client.FinalizedEpoch(ctx)
	if err != nil {
		return 0, err
	}
	head, err := r.client.GetHeadSlot(ctx)
	if err != nil {
		return 0, err
	}
	headEpoch := head / config.SlotsPerEpoch()
	if headEpoch < finalized {
		return headEpoch, nil
	}
	return finalized, nil
}

func (r *Runner) StepChain(ctx context.Context) ([]steps.Step, bool, error) {
	if r.opts.OneShot {
		done, err := r.oneShotComplete(ctx)
		if err != nil {
			return nil, false, err
		}
		if done {
			return nil, true, nil
		}
	}
	return r.stepChain(), false, nil
}

func (r *Runner) oneShotComplete(ctx context.Context) (bool, error) {
	endSlot := r.oneShotEndSlot(ctx)
	floorSlot := r.cfg.StartSlot
	if r.opts.StartSlot != nil {
		floorSlot = *r.opts.StartSlot
	}
	if endSlot >= floorSlot {
		_, slotOK, err := r.repo.FirstUnindexedSlot(ctx, floorSlot, endSlot)
		if err != nil {
			return false, err
		}
		if slotOK {
			return false, nil
		}
	}

	endEpoch, err := r.oneShotEndEpoch(ctx)
	if err != nil {
		return false, err
	}
	floorEpoch := r.cfg.StartEpoch
	if r.opts.StartEpoch != nil {
		floorEpoch = *r.opts.StartEpoch
	}
	if endEpoch >= floorEpoch {
		_, epochOK, err := r.repo.FirstUnindexedEpoch(ctx, floorEpoch, endEpoch)
		if err != nil {
			return false, err
		}
		if epochOK {
			return false, nil
		}
	}
	return true, nil
}

func (r *Runner) oneShotEndSlot(ctx context.Context) uint64 {
	if r.opts.EndSlot != nil {
		return *r.opts.EndSlot
	}
	head, err := r.getHead(ctx)
	if err != nil {
		return 0
	}
	lag := r.cfg.LagBehindHead
	if head > lag {
		return head - lag
	}
	return 0
}

func (r *Runner) oneShotEndEpoch(ctx context.Context) (uint64, error) {
	if r.opts.EndEpoch != nil {
		return *r.opts.EndEpoch, nil
	}
	return r.client.FinalizedEpoch(ctx)
}

func (r *Runner) SleepOnSeedError() time.Duration { return 0 }

func (r *Runner) Start(ctx context.Context) {
	runner.Run(ctx, r)
}

func (r *Runner) stepChain() []steps.Step {
	return []steps.Step{
		&stepbf.SlotPass{
			Cfg:               r.cfg,
			StartSlotOverride: r.opts.StartSlot,
			EndSlotOverride:   r.opts.EndSlot,
			Client:            r.client,
			Exec:              r.exec,
			Repo:              r.repo,
			GetHead:           r.getHead,
			Log:               r.log,
		},
		&stepbf.EpochPass{
			Cfg:                r.cfg,
			StartEpochOverride: r.opts.StartEpoch,
			EndEpochOverride:   r.opts.EndEpoch,
			Client:             r.client,
			Repo:               r.repo,
			Log:                r.log,
		},
	}
}
