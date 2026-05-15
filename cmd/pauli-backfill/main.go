package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/execution"
	"github.com/tharun/pauli/internal/logsetup"
	"github.com/tharun/pauli/internal/monitor"
	"github.com/tharun/pauli/internal/monitor/runner/backfill"
	"github.com/tharun/pauli/internal/monitor/steps"
	"github.com/tharun/pauli/internal/store"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	fromSlot := flag.Uint64("from-slot", ^uint64(0), "Start slot override (^uint64 max = use config)")
	toSlot := flag.Uint64("to-slot", ^uint64(0), "End slot override (^uint64 max = use head-lag)")
	fromEpoch := flag.Uint64("from-epoch", ^uint64(0), "Start epoch override (^uint64 max = use config)")
	toEpoch := flag.Uint64("to-epoch", ^uint64(0), "End epoch override (^uint64 max = use finalized epoch)")
	debug := flag.Bool("debug", false, "Verbose debug logging")
	flag.Parse()

	logsetup.Setup(*debug)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}
	cfg.Backfill.Enabled = true

	opts := backfill.Options{OneShot: true}
	if *fromSlot != ^uint64(0) {
		opts.StartSlot = fromSlot
	}
	if *toSlot != ^uint64(0) {
		opts.EndSlot = toSlot
	}
	if *fromEpoch != ^uint64(0) {
		opts.StartEpoch = fromEpoch
	}
	if *toEpoch != ^uint64(0) {
		opts.EndEpoch = toEpoch
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	dbStore, err := store.NewStore(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database store")
	}
	defer dbStore.Close()

	if err := dbStore.RunMigrations(); err != nil {
		log.Fatal().Err(err).Msg("failed to run database migrations")
	}

	repo := dbStore.Repository()
	beaconClient := beacon.NewClient(cfg)
	defer beaconClient.Close()

	network := config.NewBlockchainNetwork(cfg)
	if err := monitor.InitBeaconNetworkClock(ctx, beaconClient, network, log.Logger); err != nil {
		log.Fatal().Err(err).Msg("beacon network init failed")
	}

	execClient := execution.NewClient(cfg)
	noopEnqueue := func(context.Context, steps.Job) error { return nil }
	backfillR := backfill.New(cfg.Backfill, opts, beaconClient, execClient, repo, beaconClient.GetHeadSlot, log.Logger, noopEnqueue)

	log.Info().Msg("pauli-backfill running (one-shot); Ctrl+C to cancel")
	backfillR.Start(ctx)
	log.Info().Msg("pauli-backfill finished")
}
