package backfill

import (
	"context"
	"testing"

	"github.com/tharun/pauli/internal/config"
)

func TestOneShotEndSlot_override(t *testing.T) {
	end := uint64(500)
	r := &Runner{
		cfg: config.BackfillConf{LagBehindHead: 4},
		opts: Options{
			EndSlot: &end,
		},
		getHead: func(context.Context) (uint64, error) { return 1000, nil },
	}
	got := r.oneShotEndSlot(context.Background())
	if got != 500 {
		t.Fatalf("oneShotEndSlot = %d, want 500", got)
	}
}

func TestOneShotEndSlot_headLag(t *testing.T) {
	r := &Runner{
		cfg:     config.BackfillConf{LagBehindHead: 4},
		getHead: func(context.Context) (uint64, error) { return 1000, nil },
	}
	got := r.oneShotEndSlot(context.Background())
	if got != 996 {
		t.Fatalf("oneShotEndSlot = %d, want 996", got)
	}
}
