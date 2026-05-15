package config

import "testing"

func TestBackfillConf_setDefaults(t *testing.T) {
	b := BackfillConf{Enabled: true}
	b.setDefaults()
	if b.LagBehindHead != 4 {
		t.Fatalf("LagBehindHead = %d, want 4", b.LagBehindHead)
	}
	if b.SlotsPerPass != 8 {
		t.Fatalf("SlotsPerPass = %d, want 8", b.SlotsPerPass)
	}
	if b.EpochsPerPass != 2 {
		t.Fatalf("EpochsPerPass = %d, want 2", b.EpochsPerPass)
	}
	if b.PollDelayMs != 100 {
		t.Fatalf("PollDelayMs = %d, want 100", b.PollDelayMs)
	}
}

func TestBackfillConf_PollDelay(t *testing.T) {
	b := BackfillConf{PollDelayMs: 250}
	if d := b.PollDelay(); d.Milliseconds() != 250 {
		t.Fatalf("PollDelay = %v, want 250ms", d)
	}
}
