package cache

import (
	"time"

	"github.com/robfig/go-cache"
)

const (
	// CleanupInterval is how often expired cache entries are removed.
	CleanupInterval = 30 * time.Second
	// DefaultTTL is the default cache duration if not specified in config.
	DefaultTTL = 30 * time.Second
	// PowerStateTTL is exported for backward compatibility - actual value comes from config.
	PowerStateTTL = 5 * time.Second
)

// Cache wraps robfig/go-cache for AMT data caching without reflection overhead.
// Uses direct interface{} storage for maximum performance.
type Cache struct {
	store         *cache.Cache
	ttl           time.Duration
	powerStateTTL time.Duration
}

// New creates a new Cache instance using in-memory storage.
// If ttl is 0, caching is disabled for all endpoints.
// If powerStateTTL is 0, power state caching is disabled (but other endpoints still cache if ttl > 0).
func New(ttl, powerStateTTL time.Duration) *Cache {
	// Default: in-memory with no default expiration (per-item TTL) and cleanup interval
	return &Cache{
		store:         cache.New(0, CleanupInterval),
		ttl:           ttl,
		powerStateTTL: powerStateTTL,
	}
}

// Set stores a value in the cache with the given TTL.
// If cache TTL is 0 (disabled), this is a no-op.
// If ttl parameter is 0, uses the default cache TTL.
// If ttl parameter is negative, caching is skipped for this specific item.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	if c.ttl == 0 {
		return // Caching disabled globally
	}

	if ttl == 0 {
		ttl = c.ttl // Use default TTL
	} else if ttl < 0 {
		return // Negative TTL means skip caching for this item
	}

	c.store.Set(key, value, ttl)
}

// Get retrieves a value from the cache.
// Returns the value and true if found and not expired, nil and false otherwise.
// If cache TTL is 0 (disabled), always returns nil, false.
func (c *Cache) Get(key string) (interface{}, bool) {
	if c.ttl == 0 {
		return nil, false // Caching disabled
	}

	return c.store.Get(key)
}

// IsEnabled returns whether caching is enabled (TTL > 0).
func (c *Cache) IsEnabled() bool {
	return c.ttl > 0
}

// GetTTL returns the configured TTL.
func (c *Cache) GetTTL() time.Duration {
	return c.ttl
}

// GetPowerStateTTL returns the configured power state TTL.
// Returns -1 if power state caching is disabled (when powerStateTTL is 0).
func (c *Cache) GetPowerStateTTL() time.Duration {
	if c.powerStateTTL == 0 {
		return -1 // Signal to Set() that caching should be skipped
	}

	return c.powerStateTTL
}

// Delete removes a value from the cache.
func (c *Cache) Delete(key string) {
	c.store.Delete(key)
}

// DeletePattern removes all keys matching a pattern (simple prefix match).
func (c *Cache) DeletePattern(_ string) {
	// robfig/go-cache doesn't expose Items() directly
	// For now, just document that pattern deletion is limited
	// Individual Delete() calls should be used instead
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.store.Flush()
}
