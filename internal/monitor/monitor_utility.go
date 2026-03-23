package monitor

import (
	"context"

	scheduler "github.com/tharun/pauli/internal/monitor/scheduler"
)

func (m *Monitor) logNodeSyncStatus(ctx context.Context) {
	// Check node sync status.
	synced, err := m.client.IsNodeSynced(ctx)
	if err != nil {
		m.logger.Warn().Err(err).Msg("Failed to check node sync status")
		return
	}
	if !synced {
		m.logger.Warn().Msg("Beacon node is still syncing, results may be incomplete")
	}
}

func (m *Monitor) waitForSlotInterval(ctx context.Context) error {
	_, err := m.scheduler.WaitForSlotInterval(ctx)
	return err
}

func (m *Monitor) handleRealtimeSlot(ctx context.Context, slot uint64) error {
	events, err := m.scheduler.NextEvents(ctx, slot)
	if err != nil {
		return err
	}

	for _, event := range events {
		switch event.Type {
		case scheduler.EventTypeSlotPoll:
			m.dispatcher.PollValidatorsForSlotEpoch(ctx, event.Slot, event.Epoch)
		case scheduler.EventTypeEpochBoundary:
			m.dispatcher.FetchDutiesForEpoch(ctx, event.Epoch)
		case scheduler.EventTypeEpochFinalized:
			m.dispatcher.FetchRewardsForEpoch(ctx, event.Epoch)
		}
	}
	return nil
}

func (m *Monitor) getFinalizedEpoch(ctx context.Context) (uint64, error) {
	checkpoints, err := m.client.GetFinalityCheckpoints(ctx, "head")
	if err != nil {
		return 0, err
	}
	return checkpoints.Finalized.Epoch.Uint64(), nil
}
