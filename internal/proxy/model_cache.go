package proxy

import (
	"context"
	"sync"
	"time"

	"github.com/sertdev/pxbin/internal/store"
)

type modelCacheEntry struct {
	mw      *store.ModelWithUpstream // nil = not found (negative cache)
	expires time.Time
}

// ModelCache provides an in-memory TTL cache for model→upstream resolution,
// eliminating a DB JOIN query on every proxied request.
type ModelCache struct {
	mu    sync.RWMutex
	items map[string]*modelCacheEntry // keyed by model name
	ttl   time.Duration
	store *store.Store
}

// NewModelCache creates a model cache with the given TTL.
func NewModelCache(s *store.Store, ttl time.Duration) *ModelCache {
	return &ModelCache{
		items: make(map[string]*modelCacheEntry),
		ttl:   ttl,
		store: s,
	}
}

// GetModelWithUpstream returns a cached result or queries the DB and caches it.
func (c *ModelCache) GetModelWithUpstream(ctx context.Context, modelName string) (*store.ModelWithUpstream, error) {
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.items[modelName]
	c.mu.RUnlock()

	if ok && now.Before(entry.expires) {
		return entry.mw, nil
	}

	mw, err := c.store.GetModelWithUpstream(ctx, modelName)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.items[modelName] = &modelCacheEntry{mw: mw, expires: now.Add(c.ttl)}
	c.mu.Unlock()

	return mw, nil
}

// Invalidate removes all cached entries (e.g. after admin changes models/upstreams).
func (c *ModelCache) Invalidate() {
	c.mu.Lock()
	c.items = make(map[string]*modelCacheEntry)
	c.mu.Unlock()
}
