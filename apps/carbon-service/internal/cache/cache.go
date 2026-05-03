package cache

import (
	"context"
	"sync"
	"time"
)

type entry struct {
	value     float64
	expiresAt time.Time
}

// GridCache is a thread-safe in-memory store for grid carbon intensity values.
// Grid data changes at most hourly, so a 1-hour TTL is appropriate.
type GridCache struct {
	mu    sync.RWMutex
	store map[string]entry
	ttl   time.Duration
}

func New(ttl time.Duration) *GridCache {
	return &GridCache{
		store: make(map[string]entry),
		ttl:   ttl,
	}
}

// Get returns the cached intensity for region and whether it was a hit.
func (c *GridCache) Get(region string) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.store[region]
	if !ok || time.Now().After(e.expiresAt) {
		return 0, false
	}
	return e.value, true
}

// Set stores the intensity for region with the configured TTL.
func (c *GridCache) Set(region string, intensity float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[region] = entry{value: intensity, expiresAt: time.Now().Add(c.ttl)}
}

// StartEviction runs a background goroutine that prunes expired entries.
// It stops when ctx is cancelled.
func (c *GridCache) StartEviction(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.ttl)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.evict()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *GridCache) evict() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, e := range c.store {
		if now.After(e.expiresAt) {
			delete(c.store, k)
		}
	}
}