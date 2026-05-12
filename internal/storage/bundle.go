package storage

import "sync"

// PersistBundle accumulates rows for one monitor tick; async steps append here, Persist commits in one transaction.
type PersistBundle struct {
	mu sync.Mutex
	// asyncErr is set when any async fetch/build step fails; Persist must not write if non-nil.
	asyncErr error

	Snapshots []*ValidatorSnapshot
	Duties    []*AttestationDuty
	Rewards   []*AttestationReward
	Penalties []*ValidatorPenalty
}

// NewPersistBundle returns an empty bundle for a new Env iteration.
func NewPersistBundle() *PersistBundle {
	return &PersistBundle{}
}

// RecordAsyncError records the first async failure for this tick (subsequent calls are ignored).
func (b *PersistBundle) RecordAsyncError(err error) {
	if b == nil || err == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.asyncErr == nil {
		b.asyncErr = err
	}
}

// AsyncError returns the first async error for this tick, if any.
func (b *PersistBundle) AsyncError() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.asyncErr
}

// HasWork reports whether there is anything to persist.
func (b *PersistBundle) HasWork() bool {
	if b == nil {
		return false
	}
	return len(b.Snapshots) > 0 ||
		len(b.Duties) > 0 ||
		len(b.Rewards) > 0 ||
		len(b.Penalties) > 0
}
