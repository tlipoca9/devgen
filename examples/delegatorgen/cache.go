package delegatorgen

import (
	"context"
	"sync"
	"time"
)

// InMemoryCache is a simple in-memory cache implementation for demonstration.
type InMemoryCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

type cacheItem struct {
	value     any
	expiresAt time.Time
	isError   bool
}

// NewInMemoryCache creates a new in-memory cache.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		items: make(map[string]*cacheItem),
	}
}

// cachedResult implements UserRepositoryCachedResult.
type cachedResult struct {
	value     any
	expiresAt time.Time
	isError   bool
}

func (r *cachedResult) Value() any           { return r.value }
func (r *cachedResult) ExpiresAt() time.Time { return r.expiresAt }
func (r *cachedResult) IsError() bool        { return r.isError }

// Get retrieves a cached value.
func (c *InMemoryCache) Get(_ context.Context, key string) (UserRepositoryCachedResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	return &cachedResult{
		value:     item.value,
		expiresAt: item.expiresAt,
		isError:   item.isError,
	}, true
}

// Set stores a value in the cache.
func (c *InMemoryCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
		isError:   false,
	}
	return nil
}

// SetError caches an error.
func (c *InMemoryCache) SetError(_ context.Context, key string, err error, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		value:     err,
		expiresAt: time.Now().Add(ttl),
		isError:   true,
	}
	return true, nil
}

// Delete removes keys from the cache.
func (c *InMemoryCache) Delete(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		delete(c.items, key)
	}
	return nil
}
