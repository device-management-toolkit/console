package cache

import (
	"sync"
	"time"
)

const (
	// CleanupInterval is how often expired cache entries are removed.
	CleanupInterval = 30 * time.Second
	// PowerStateTTL is the cache duration for power state (changes frequently).
	PowerStateTTL = 5 * time.Second
	// FeaturesTTL is the cache duration for features (rarely changes).
	FeaturesTTL = 30 * time.Second
	// KVMTTL is the cache duration for KVM display settings (rarely changes).
	KVMTTL = 30 * time.Second
)

// Entry represents a cached value with expiration.
type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// Cache is a simple in-memory cache with TTL support.
type Cache struct {
	mu    sync.RWMutex
	items map[string]Entry
}

// New creates a new Cache instance.
func New() *Cache {
	c := &Cache{
		items: make(map[string]Entry),
	}
	// Start cleanup goroutine
	go c.cleanupExpired()

	return c
}

// Set stores a value in the cache with the given TTL.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves a value from the cache.
// Returns the value and true if found and not expired, nil and false otherwise.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.items[key]
	if !found {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Value, true
}

// Delete removes a value from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// DeletePattern removes all keys matching a pattern (simple prefix match).
func (c *Cache) DeletePattern(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]Entry)
}

// cleanupExpired runs periodically to remove expired entries.
func (c *Cache) cleanupExpired() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()

		now := time.Now()

		for key, entry := range c.items {
			if now.After(entry.ExpiresAt) {
				delete(c.items, key)
			}
		}

		c.mu.Unlock()
	}
}
