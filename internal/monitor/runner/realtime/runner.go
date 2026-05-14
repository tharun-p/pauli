package realtime

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/execution"
	"github.com/tharun/pauli/internal/monitor/runner"
	"github.com/tharun/pauli/internal/monitor/steps"
	steprt "github.com/tharun/pauli/internal/monitor/steps/realtime"
	"github.com/tharun/pauli/internal/storage"
)

// Runner implements runner.Runner: network pacing and a fixed linear chain of indexing steps.
type Runner struct {
	network    *config.BlockchainNetwork
	client     *beacon.Client
	exec       *execution.Client
	repo       storage.Repository
	getHead    func(context.Context) (uint64, error)
	validators []uint64
	log        zerolog.Logger
	enqueue    func(context.Context, steps.Job) error
	// Updated only by RecordLastProcessedSlot after a full successful chain pass; other
	// steps skip when Env.HeadSlot equals this (dedup across polls for the same head).
	lastProcessedSlot uint64
	env               *steps.Env
}

var _ runner.Runner = (*Runner)(nil)

// New constructs a realtime runner.
func New(
	network *config.BlockchainNetwork,
	client *beacon.Client,
	exec *execution.Client,
	repo storage.Repository,
	getHead func(context.Context) (uint64, error),
	validators []uint64,
	log zerolog.Logger,
	enqueue func(context.Context, steps.Job) error,
) *Runner {
	return &Runner{
		network:    network,
		client:     client,
		exec:       exec,
		repo:       repo,
		getHead:    getHead,
		validators: validators,
		log:        log,
		enqueue:    enqueue,
		// Sentinel: no successful chain yet, so first HeadSlot always runs all steps.
		lastProcessedSlot: ^uint64(0),
		env:               steps.NewEnv(),
	}
}

func (r *Runner) Name() string { return "realtime" }

func (r *Runner) Logger() zerolog.Logger { return r.log }

func (r *Runner) Env() *steps.Env { return r.env }

func (r *Runner) Enqueue(ctx context.Context, job steps.Job) error {
	return r.enqueue(ctx, job)
}

func (r *Runner) BeforeStep(ctx context.Context) error {
	r.log.Debug().
		Dur("poll_interval", r.network.PollInterval()).
		Msg("realtime runner pacing wait")
	return r.network.WaitPollInterval(ctx)
}

func (r *Runner) AfterStep(context.Context) error { return nil }

func (r *Runner) StepChain(ctx context.Context) ([]steps.Step, bool, error) {
	return r.stepChain(), false, nil
}

func (r *Runner) SleepOnSeedError() time.Duration { return 0 }

func (r *Runner) Start(ctx context.Context) {
	runner.Run(ctx, r)
}

func (r *Runner) stepChain() []steps.Step {
	return []steps.Step{
		steprt.RealtimeEnvBootstrap{
			GetHead:    r.getHead,
			Validators: r.validators,
			Log:        r.log,
		},
		steprt.ValidatorsBalanceAtSlot{
			Client:            r.client,
			Repo:              r.repo,
			Validators:        r.validators,
			Log:               r.log,
			LastProcessedSlot: &r.lastProcessedSlot,
		},
		&steprt.AttestationRewards{
			Client:            r.client,
			Repo:              r.repo,
			Validators:        r.validators,
			Log:               r.log,
			LastProcessedSlot: &r.lastProcessedSlot,
		},
		&steprt.BlockProposerRewards{
			Client:            r.client,
			Execution:         r.exec,
			Repo:              r.repo,
			Validators:        r.validators,
			Log:               r.log,
			LastProcessedSlot: &r.lastProcessedSlot,
		},
		&steprt.SyncCommitteeRewards{
			Client:            r.client,
			Repo:              r.repo,
			Validators:        r.validators,
			Log:               r.log,
			LastProcessedSlot: &r.lastProcessedSlot,
		},
		&steprt.RecordLastProcessedSlot{
			LastProcessedSlot: &r.lastProcessedSlot,
		},
	}
}
