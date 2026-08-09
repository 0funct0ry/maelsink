package lineedit

import (
	"context"
	"sync"
	"time"
)

// IDCache memoizes the result of a slow/flaky fetch (e.g. "list recent
// message IDs over the API") for a short TTL, so that repeated Tab presses
// while a user is typing don't hammer the API. It is used by completer.go
// to implement the ~2s TTL called for by SPEC.md §7.5.10.
type IDCache struct {
	ttl time.Duration

	mu   sync.Mutex
	last time.Time
	ids  []string
}

// NewIDCache returns a cache that considers a previous fetch fresh for ttl.
func NewIDCache(ttl time.Duration) *IDCache {
	return &IDCache{ttl: ttl}
}

// Get returns the cached ids if the last fetch happened within ttl of now;
// otherwise it calls fetch(ctx), caches the result (even if nil/empty — an
// offline no-op result is cached too, so a flaky server doesn't get hit on
// every keypress), and returns it. Safe for concurrent use.
func (c *IDCache) Get(ctx context.Context, fetch func(ctx context.Context) []string) []string {
	c.mu.Lock()
	if time.Since(c.last) < c.ttl {
		ids := c.ids
		c.mu.Unlock()
		return ids
	}
	c.mu.Unlock()

	ids := fetch(ctx)

	c.mu.Lock()
	c.ids = ids
	c.last = time.Now()
	c.mu.Unlock()

	return ids
}
