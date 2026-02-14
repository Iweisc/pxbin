package proxy

import (
	"sync"

	"github.com/google/uuid"
)

type cachedClient struct {
	client  *UpstreamClient
	baseURL string
	apiKey  string
}

// ClientCache is a thread-safe cache of UpstreamClients keyed by upstream UUID.
// It prevents creating a new HTTP transport per request.
type ClientCache struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]*cachedClient
}

// NewClientCache creates an empty ClientCache.
func NewClientCache() *ClientCache {
	return &ClientCache{
		clients: make(map[uuid.UUID]*cachedClient),
	}
}

// Get returns a cached client for the given upstream ID. If the cached client's
// baseURL or apiKey differ from the provided values, it creates a new client.
func (c *ClientCache) Get(id uuid.UUID, baseURL, apiKey string) *UpstreamClient {
	c.mu.RLock()
	cached, ok := c.clients[id]
	c.mu.RUnlock()

	if ok && cached.baseURL == baseURL && cached.apiKey == apiKey {
		return cached.client
	}

	client := NewUpstreamClient(baseURL, apiKey)

	c.mu.Lock()
	c.clients[id] = &cachedClient{
		client:  client,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
	c.mu.Unlock()

	return client
}
