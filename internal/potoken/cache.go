package potoken

import (
	"sync"
	"time"
)

type cacheEntry struct {
	tokens    Tokens
	expiresAt time.Time
}

// tokenCache is a bounded TTL cache keyed by client/video/visitor binding.
type tokenCache struct {
	mu         sync.Mutex
	items      map[string]cacheEntry
	ttl        time.Duration
	maxEntries int
	skew       time.Duration // refresh this long before expiry
}

func newTokenCache(ttl time.Duration, maxEntries int) *tokenCache {
	if ttl <= 0 {
		ttl = 4 * time.Minute
	}
	if maxEntries <= 0 {
		maxEntries = 256
	}
	return &tokenCache{
		items:      make(map[string]cacheEntry),
		ttl:        ttl,
		maxEntries: maxEntries,
		skew:       30 * time.Second,
	}
}

func cacheKey(req Request) string {
	return req.ClientName + "|" + req.VideoID + "|" + req.VisitorData
}

func (c *tokenCache) get(req Request) (Tokens, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := cacheKey(req)
	e, ok := c.items[key]
	if !ok {
		return Tokens{}, false
	}
	now := time.Now()
	if !now.Before(e.expiresAt.Add(-c.skew)) {
		delete(c.items, key)
		return Tokens{}, false
	}
	return e.tokens, true
}

func (c *tokenCache) set(req Request, tok Tokens) {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp := tok.ExpiresAt
	if exp.IsZero() {
		exp = time.Now().Add(c.ttl)
	}
	tok.ExpiresAt = exp
	tok.VideoID = req.VideoID
	tok.Visitor = req.VisitorData
	tok.Client = req.ClientName

	if len(c.items) >= c.maxEntries {
		now := time.Now()
		for k, e := range c.items {
			if now.After(e.expiresAt) {
				delete(c.items, k)
			}
		}
		if len(c.items) >= c.maxEntries {
			for k := range c.items {
				delete(c.items, k)
				break
			}
		}
	}
	c.items[cacheKey(req)] = cacheEntry{tokens: tok, expiresAt: exp}
}

func (c *tokenCache) invalidate(videoID, visitorData string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if videoID == "" && visitorData == "" {
		c.items = make(map[string]cacheEntry)
		return
	}
	for k, e := range c.items {
		if videoID != "" && e.tokens.VideoID != videoID {
			continue
		}
		if visitorData != "" && e.tokens.Visitor != visitorData {
			continue
		}
		delete(c.items, k)
	}
}
