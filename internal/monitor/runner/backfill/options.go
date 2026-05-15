package backfill

// Options overrides backfill bounds for one-shot CLI runs.
type Options struct {
	StartSlot  *uint64
	EndSlot    *uint64
	StartEpoch *uint64
	EndEpoch   *uint64
	OneShot    bool
}
