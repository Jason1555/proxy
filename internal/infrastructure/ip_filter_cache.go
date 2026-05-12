package infrastructure

import (
	"proxy/internal/domain"
	"sync"
	"time"
)

type SimpleIPFilterCache struct {
	mu sync.RWMutex
	cache map[string]*cacheEntry
	ttl time.Duration
}

type cacheEntry struct {
	result *domain.IPCheckResult
	expiresAt time.Time
}

func NewSimpleIPFilterCache(ttl time.Duration) domain.IPFilterCache {
	cache := &SimpleIPFilterCache{
		cache: make(map[string]*cacheEntry), 
		ttl: ttl,
	}

	go cache.cleanupRoutine()

	return cache
}

func (c *SimpleIPFilterCache) Get(ip string) (*domain.IPCheckResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[ip]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		delete(c.cache, ip)
		return nil, false
	}

	return entry.result, true
}

func (c *SimpleIPFilterCache) Set(ip string, result *domain.IPCheckResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[ip] = &cacheEntry{
		result: result,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *SimpleIPFilterCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cacheEntry)
}

func (c *SimpleIPFilterCache) GetStats() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]any{
		"size": len(c.cache),
		"ttl": c.ttl.String(),
	}
}

func (c *SimpleIPFilterCache) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()

		for ip, entry := range c.cache {
			if now.After(entry.expiresAt) {
				delete(c.cache, ip)
			}
		}

		c.mu.Unlock()
	}
}
