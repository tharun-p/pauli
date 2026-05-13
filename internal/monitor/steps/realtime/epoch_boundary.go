package realtime

import "github.com/tharun/pauli/internal/config"

// Shared helpers for consensus epoch boundary detection (e.g. AttestationRewards).

// isConsensusEpochBoundarySlot is true when head is the first or last slot of an epoch
// (Ethereum consensus: 32 slots per epoch in the default configuration).
func isConsensusEpochBoundarySlot(headSlot uint64) bool {
	sp := config.SlotsPerEpoch()
	return headSlot%sp == 0 || (headSlot+1)%sp == 0
}
