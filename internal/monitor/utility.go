package monitor

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/tharun/pauli/internal/beacon"
	"github.com/tharun/pauli/internal/config"
)

// initBeaconNetworkClock loads genesis into network (wall-time anchor) and logs initial finality (debug).
func initBeaconNetworkClock(ctx context.Context, client *beacon.Client, network *config.BlockchainNetwork, log zerolog.Logger) error {
	genesis, err := client.GetGenesis(ctx)
	if err != nil {
		return err
	}

	network.SetGenesisTime(time.Unix(int64(genesis.Data.GenesisTime.Uint64()), 0))

	checkpoints, err := client.GetFinalityCheckpoints(ctx, "head")
	if err != nil {
		log.Warn().Err(err).Msg("beacon init: finality checkpoints unavailable")
	} else {
		log.Debug().
			Uint64("initial_finalized_epoch", checkpoints.Finalized.Epoch.Uint64()).
			Msg("beacon init: observed finalized epoch")
	}

	log.Debug().
		Time("genesis_time", network.GenesisTime()).
		Msg("beacon clock initialized")

	return nil
}

func (m *Monitor) logNodeSyncStatus(ctx context.Context) {
	// Check node sync status.
	synced, err := m.client.IsNodeSynced(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("node sync status check failed")
		return
	}
	if !synced {
		m.logger.Warn().Msg("beacon node still syncing")
	}
}
