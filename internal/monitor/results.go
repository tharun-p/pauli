package monitor

import (
	"context"
	"fmt"

	"github.com/tharun/pauli/internal/monitor/core"
	"github.com/tharun/pauli/internal/storage"
)

// processResults processes results from the worker pool and logs them.
func (m *Monitor) processResults(ctx context.Context) {
	defer m.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-m.workerPool.Results():
			if !ok {
				return
			}

			m.handleResult(result)
		}
	}
}

func (m *Monitor) handleResult(result core.Result) {
	if result.Error != nil {
		m.logger.Error().
			Err(result.Error).
			Uint64("validator_index", result.Job.ValidatorIndex).
			Uint64("slot", result.Job.Slot).
			Int("job_type", int(result.Job.Type)).
			Msg("Job failed")
		return
	}

	if err := m.logResult(result); err != nil {
		m.logger.Warn().
			Err(err).
			Int("job_type", int(result.Job.Type)).
			Msg("Skipping result log due to unexpected result payload")
	}
}

// logResult logs successful results in JSON format.
func (m *Monitor) logResult(result core.Result) error {
	switch result.Job.Type {
	case core.JobTypeStatus:
		snapshot, ok := result.Data.(*storage.ValidatorSnapshot)
		if !ok {
			return fmt.Errorf("expected *storage.ValidatorSnapshot, got %T", result.Data)
		}
		m.logStatus(snapshot)
		return nil

	case core.JobTypeDuties:
		duties, ok := result.Data.([]*storage.AttestationDuty)
		if !ok {
			return fmt.Errorf("expected []*storage.AttestationDuty, got %T", result.Data)
		}
		m.logDuties(duties)
		return nil

	case core.JobTypeRewards:
		rewards, ok := result.Data.([]*storage.AttestationReward)
		if !ok {
			return fmt.Errorf("expected []*storage.AttestationReward, got %T", result.Data)
		}
		m.logRewards(rewards)
		return nil
	}

	return fmt.Errorf("unknown job type: %d", result.Job.Type)
}

func (m *Monitor) logStatus(snapshot *storage.ValidatorSnapshot) {
	m.logger.Info().
		Uint64("slot", snapshot.Slot).
		Uint64("validator_index", snapshot.ValidatorIndex).
		Str("status", snapshot.Status).
		Uint64("effective_balance_gwei", snapshot.EffectiveBalance).
		Uint64("balance_gwei", snapshot.Balance).
		Msg("validator_status")
}

func (m *Monitor) logDuties(duties []*storage.AttestationDuty) {
	for _, duty := range duties {
		m.logger.Info().
			Uint64("slot", duty.Slot).
			Uint64("epoch", duty.Epoch).
			Uint64("validator_index", duty.ValidatorIndex).
			Int("committee_index", duty.CommitteeIndex).
			Int("committee_position", duty.CommitteePosition).
			Msg("attestation_duty")
	}
}

func (m *Monitor) logRewards(rewards []*storage.AttestationReward) {
	for _, reward := range rewards {
		m.logger.Info().
			Uint64("epoch", reward.Epoch).
			Uint64("validator_index", reward.ValidatorIndex).
			Int64("head_reward", reward.HeadReward).
			Int64("source_reward", reward.SourceReward).
			Int64("target_reward", reward.TargetReward).
			Int64("total_reward_gwei", reward.TotalReward).
			Bool("duty_success", reward.TotalReward >= 0).
			Msg("attestation_reward")
	}
}
