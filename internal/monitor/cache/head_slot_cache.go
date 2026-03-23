package cache

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// HeadSlotCache caches head slot responses for short intervals.
type HeadSlotCache struct {
	mu        sync.RWMutex
	slot      uint64
	timestamp time.Time

	ttl    time.Duration
	fetch  func(context.Context) (uint64, error)
	logger zerolog.Logger
}

func NewHeadSlotCache(
	ttl time.Duration,
	fetch func(context.Context) (uint64, error),
	logger zerolog.Logger,
) *HeadSlotCache {
	return &HeadSlotCache{
		ttl:    ttl,
		fetch:  fetch,
		logger: logger,
	}
}

func (c *HeadSlotCache) read() (uint64, time.Time) {
	c.mu.RLock()
	slot := c.slot
	timestamp := c.timestamp
	c.mu.RUnlock()
	return slot, timestamp
}

func (c *HeadSlotCache) write(slot uint64, timestamp time.Time) {
	c.mu.Lock()
	c.slot = slot
	c.timestamp = timestamp
	c.mu.Unlock()
}

// Get returns a fresh head slot or a cached value if still valid.
func (c *HeadSlotCache) Get(ctx context.Context) (uint64, error) {
	cached, cacheTime := c.read()

	if cached > 0 && time.Since(cacheTime) < c.ttl {
		return cached, nil
	}

	slot, err := c.fetch(ctx)
	if err != nil {
		// Return cached value if available, even if stale.
		if cached > 0 {
			c.logger.Warn().Err(err).Uint64("cached_slot", cached).Msg("Using cached slot due to API error")
			return cached, nil
		}
		return 0, err
	}

	c.write(slot, time.Now())
	return slot, nil
}
