package proxy

import (
	"context"
	"log"
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
//
// Uses stale-while-revalidate: expired entries are returned immediately while
// a background goroutine refreshes the cache, so the hot path never blocks
// on a DB round-trip.
type ModelCache struct {
	mu          sync.RWMutex
	items       map[string]*modelCacheEntry // keyed by model name
	refreshing  map[string]bool             // in-flight background refreshes
	ttl         time.Duration
	store       *store.Store
}

// NewModelCache creates a model cache with the given TTL.
func NewModelCache(s *store.Store, ttl time.Duration) *ModelCache {
	return &ModelCache{
		items:      make(map[string]*modelCacheEntry),
		refreshing: make(map[string]bool),
		ttl:        ttl,
		store:      s,
	}
}

// GetModelWithUpstream returns a cached result. If the entry is stale, it
// returns the stale value immediately and triggers a background refresh.
// Only truly cold misses (first request for a model) block on the DB.
func (c *ModelCache) GetModelWithUpstream(ctx context.Context, modelName string) (*store.ModelWithUpstream, error) {
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.items[modelName]
	c.mu.RUnlock()

	if ok {
		if now.Before(entry.expires) {
			// Fresh — return immediately.
			return entry.mw, nil
		}
		// Stale — return immediately, refresh in background.
		c.triggerRefresh(modelName)
		return entry.mw, nil
	}

	// Cold miss — must block on DB.
	return c.fetchAndCache(ctx, modelName)
}

// triggerRefresh starts a background goroutine to refresh a stale entry,
// deduplicating concurrent refreshes for the same model.
func (c *ModelCache) triggerRefresh(modelName string) {
	c.mu.Lock()
	if c.refreshing[modelName] {
		c.mu.Unlock()
		return
	}
	c.refreshing[modelName] = true
	c.mu.Unlock()

	go func() {
		defer func() {
			c.mu.Lock()
			delete(c.refreshing, modelName)
			c.mu.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := c.fetchAndCache(ctx, modelName); err != nil {
			log.Printf("model cache: background refresh for %q failed: %v", modelName, err)
		}
	}()
}

func (c *ModelCache) fetchAndCache(ctx context.Context, modelName string) (*store.ModelWithUpstream, error) {
	mw, err := c.store.GetModelWithUpstream(ctx, modelName)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.items[modelName] = &modelCacheEntry{mw: mw, expires: time.Now().Add(c.ttl)}
	c.mu.Unlock()

	return mw, nil
}

// Invalidate removes all cached entries (e.g. after admin changes models/upstreams).
func (c *ModelCache) Invalidate() {
	c.mu.Lock()
	c.items = make(map[string]*modelCacheEntry)
	c.mu.Unlock()
}
