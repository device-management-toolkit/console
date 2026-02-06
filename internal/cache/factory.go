package cache

import (
	"github.com/device-management-toolkit/console/config"
)

// NewFromConfig creates a cache instance based on the application configuration.
// If cfg.Cache.TTL is 0, caching is disabled.
// Currently only supports in-memory backend using robfig/go-cache for optimal performance.
// Redis support can be added later if needed.
func NewFromConfig(cfg *config.Config) *Cache {
	// For now, always use in-memory cache with robfig/go-cache
	// This avoids the reflection overhead of gin-contrib/cache
	return New(cfg.TTL, cfg.PowerStateTTL)
}
