package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/monitor"
	"github.com/tharun/pauli/internal/store"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	debug := flag.Bool("debug", false, "Verbose debug logging (default: info/warn/error for operations)")
	flag.Parse()

	setupLogger(*debug)

	log.Debug().Str("config", *configPath).Msg("starting validator monitor")

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	log.Debug().
		Str("beacon_url", cfg.BeaconNodeURL).
		Int("validators", len(cfg.Validators)).
		Int("worker_pool_size", cfg.WorkerPoolSize).
		Msg("configuration loaded")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	dbStore, err := store.NewStore(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database store")
	}
	defer dbStore.Close()

	if err := dbStore.RunMigrations(); err != nil {
		log.Fatal().Err(err).Msg("failed to run database migrations")
	}

	if err := dbStore.HealthCheck(); err != nil {
		log.Fatal().Err(err).Msg("database health check failed")
	}
	log.Debug().Str("driver", cfg.DatabaseDriver).Msg("database connection verified")

	repo := dbStore.Repository()

	testCtx, cancelTest := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTest()

	if len(cfg.Validators) > 0 {
		count, err := repo.CountSnapshots(testCtx, cfg.Validators[0])
		if err != nil {
			log.Debug().Err(err).Msg("count snapshots probe failed (ok if tables empty)")
		} else {
			log.Debug().Uint64("validator_index", cfg.Validators[0]).Int("existing_snapshots", count).Msg("snapshot count probe")
		}
	}

	beaconClient := beacon.NewClient(cfg)
	defer beaconClient.Close()

	synced, err := beaconClient.IsNodeSynced(ctx)
	if err != nil {
		log.Error().Err(err).Msg("beacon sync check failed")
	} else if synced {
		log.Debug().Msg("beacon node synced")
	} else {
		log.Warn().Msg("beacon node still syncing")
	}

	if len(cfg.Validators) > 0 {
		testValidator := cfg.Validators[0]
		log.Debug().Uint64("validator_index", testValidator).Msg("test validator API fetch")

		testCtx2, cancelV := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelV()

		validator, err := beaconClient.GetValidator(testCtx2, "head", testValidator)
		if err != nil {
			log.Warn().Err(err).Uint64("validator_index", testValidator).Msg("test validator fetch failed")
		} else {
			log.Debug().
				Uint64("validator_index", testValidator).
				Str("status", validator.Status).
				Uint64("balance_gwei", validator.Balance.Uint64()).
				Uint64("effective_balance_gwei", validator.Validator.EffectiveBalance.Uint64()).
				Msg("test validator fetch ok")
		}
	}

	mon := monitor.NewMonitor(cfg, beaconClient, repo, log.Logger)

	if err := mon.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start monitor")
	}

	log.Info().
		Str("beacon_url", cfg.BeaconNodeURL).
		Int("validators", len(cfg.Validators)).
		Str("database_driver", cfg.DatabaseDriver).
		Msg("pauli running; Ctrl+C to stop (-debug for verbose logs)")

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("shutdown initiated")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		mon.Stop(shutdownCtx)
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("shutdown complete")
	case <-shutdownCtx.Done():
		log.Warn().Msg("shutdown timed out")
	}
}

func setupLogger(debug bool) {
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		// Info: lifecycle (start/stop). Warn: recoverable probes. Error: indexing / runner failures.
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	zerolog.TimeFieldFormat = time.RFC3339

	if isTerminal() {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	} else {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
}

func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
