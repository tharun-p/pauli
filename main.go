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
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Setup logger
	setupLogger(*debug)

	log.Info().Str("config", *configPath).Msg("Starting Validator Monitor")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().
		Str("beacon_url", cfg.BeaconNodeURL).
		Int("validators", len(cfg.Validators)).
		Int("worker_pool_size", cfg.WorkerPoolSize).
		Msg("Configuration loaded")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize database store based on configuration
	dbStore, err := store.NewStore(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database store")
	}
	defer dbStore.Close()

	// Run migrations
	if err := dbStore.RunMigrations(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	// Test database connection
	if err := dbStore.HealthCheck(); err != nil {
		log.Fatal().Err(err).Msg("Database health check failed")
	}
	log.Info().Str("driver", cfg.DatabaseDriver).Msg("Database connection verified")

	// Create repository
	repo := dbStore.Repository()

	// Test read capability
	testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if len(cfg.Validators) > 0 {
		count, err := repo.CountSnapshots(testCtx, cfg.Validators[0])
		if err != nil {
			log.Warn().Err(err).Msg("Failed to query existing snapshots (this is OK if tables are empty)")
		} else {
			log.Info().Uint64("validator_index", cfg.Validators[0]).Int("existing_snapshots", count).Msg("Found existing snapshots")
		}
	}

	// Create Beacon API client
	beaconClient := beacon.NewClient(cfg)
	defer beaconClient.Close()

	// Verify beacon node connection
	synced, err := beaconClient.IsNodeSynced(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check beacon node sync status")
	} else if synced {
		log.Info().Msg("Beacon node is fully synced")
	} else {
		log.Warn().Msg("Beacon node is still syncing")
	}

	// Test fetching a validator to verify API works
	if len(cfg.Validators) > 0 {
		testValidator := cfg.Validators[0]
		log.Info().Uint64("validator_index", testValidator).Msg("Testing validator API fetch")

		testCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		validator, err := beaconClient.GetValidator(testCtx, "head", testValidator)
		if err != nil {
			log.Error().Err(err).Uint64("validator_index", testValidator).Msg("Failed to fetch test validator - check beacon node URL and validator index")
		} else {
			log.Info().
				Uint64("validator_index", testValidator).
				Str("status", validator.Status).
				Uint64("balance_gwei", validator.Balance.Uint64()).
				Uint64("effective_balance_gwei", validator.Validator.EffectiveBalance.Uint64()).
				Msg("Successfully fetched test validator from beacon API")
		}
	}

	// Create and start monitor
	mon := monitor.NewMonitor(cfg, beaconClient, repo, log.Logger)

	if err := mon.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to start monitor")
	}

	log.Info().Msg("Validator Monitor is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Cancel context to stop all goroutines
	cancel()

	// Give goroutines time to clean up
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop monitor gracefully
	done := make(chan struct{})
	go func() {
		mon.Stop()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("Graceful shutdown completed")
	case <-shutdownCtx.Done():
		log.Warn().Msg("Shutdown timed out, forcing exit")
	}
}

// setupLogger configures the global zerolog logger.
func setupLogger(debug bool) {
	// Set global log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// Configure timestamp format
	zerolog.TimeFieldFormat = time.RFC3339

	// Use console writer for development, JSON for production
	if isTerminal() {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	} else {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
}

// isTerminal checks if stdout is a terminal.
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
