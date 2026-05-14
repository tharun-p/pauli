package realtime

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tharun/pauli/internal/monitor/steps"
)

func TestBlockIndexer_Run_enqueueOnlyOnNewHead(t *testing.T) {
	last := uint64(10)
	s := &BlockIndexer{LastProcessedSlot: &last}

	ok, err := s.Run(&steps.Env{HeadSlot: 10})
	require.NoError(t, err)
	require.False(t, ok, "same head as lastProcessed should not enqueue")

	ok, err = s.Run(&steps.Env{HeadSlot: 11})
	require.NoError(t, err)
	require.True(t, ok, "new head should enqueue async work")
}

func TestBlockIndexer_Async(t *testing.T) {
	s := &BlockIndexer{}
	require.True(t, s.Async())
}
