package auth

import (
	"context"
	"sync"
	"time"

	"github.com/sertdev/pxbin/internal/store"
)

type keyCacheEntry struct {
	key     *store.LLMAPIKey
	expires time.Time
}

// KeyCache provides an in-memory TTL cache for LLM API key lookups,
// eliminating a DB round-trip on every proxied request.
type KeyCache struct {
	mu    sync.RWMutex
	items map[string]*keyCacheEntry // keyed by hash
	ttl   time.Duration
	store *store.Store
}

// NewKeyCache creates a key cache with the given TTL.
func NewKeyCache(s *store.Store, ttl time.Duration) *KeyCache {
	return &KeyCache{
		items: make(map[string]*keyCacheEntry),
		ttl:   ttl,
		store: s,
	}
}

// GetLLMKeyByHash returns a cached key or queries the DB and caches the result.
func (c *KeyCache) GetLLMKeyByHash(ctx context.Context, hash string) (*store.LLMAPIKey, error) {
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.items[hash]
	c.mu.RUnlock()

	if ok && now.Before(entry.expires) {
		return entry.key, nil
	}

	key, err := c.store.GetLLMKeyByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.items[hash] = &keyCacheEntry{key: key, expires: now.Add(c.ttl)}
	c.mu.Unlock()

	return key, nil
}

// Invalidate removes a specific key hash from the cache.
func (c *KeyCache) Invalidate(hash string) {
	c.mu.Lock()
	delete(c.items, hash)
	c.mu.Unlock()
}
