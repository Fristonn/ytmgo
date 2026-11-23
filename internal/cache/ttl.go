package cache

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type entry struct {
	value      any
	expiresAt  time.Time
	lastAccess time.Time
}

// TTL is a bounded in-memory TTL cache with singleflight deduplication.
type TTL struct {
	mu         sync.RWMutex
	items      map[string]entry
	ttl        time.Duration
	maxEntries int
	sf         singleflight.Group
}

// New creates a TTL cache. maxEntries bounds memory; oldest entries are evicted.
func New(ttl time.Duration, maxEntries int) *TTL {
	if maxEntries <= 0 {
		maxEntries = 1024
	}
	return &TTL{
		items:      make(map[string]entry, 64),
		ttl:        ttl,
		maxEntries: maxEntries,
	}
}

// Get returns a cached value if present and not expired.
func (c *TTL) Get(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}
		return nil, false
	}
	c.mu.Lock()
	if cur, ok := c.items[key]; ok {
		cur.lastAccess = time.Now()
		c.items[key] = cur
	}
	c.mu.Unlock()
	return e.value, true
}

// Set stores a value with the configured TTL.
func (c *TTL) Set(key string, value any) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value with a custom TTL.
func (c *TTL) SetWithTTL(key string, value any, ttl time.Duration) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= c.maxEntries {
		c.evictLocked(now)
	}
	c.items[key] = entry{
		value:      value,
		expiresAt:  now.Add(ttl),
		lastAccess: now,
	}
}

// Do deduplicates concurrent work for the same key and caches the result.
func (c *TTL) Do(key string, fn func() (any, error)) (any, error, bool) {
	if v, ok := c.Get(key); ok {
		return v, nil, true
	}
	v, err, _ := c.sf.Do(key, func() (any, error) {
		if v, ok := c.Get(key); ok {
			return v, nil
		}
		val, err := fn()
		if err != nil {
			return nil, err
		}
		c.Set(key, val)
		return val, nil
	})
	return v, err, false
}

// Len returns the number of entries currently stored.
func (c *TTL) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *TTL) evictLocked(now time.Time) {
	// First drop expired.
	for k, e := range c.items {
		if now.After(e.expiresAt) {
			delete(c.items, k)
		}
	}
	if len(c.items) < c.maxEntries {
		return
	}
	// Evict least recently accessed.
	var oldestKey string
	var oldestTime time.Time
	first := true
	for k, e := range c.items {
		if first || e.lastAccess.Before(oldestTime) {
			oldestKey = k
			oldestTime = e.lastAccess
			first = false
		}
	}
	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}
