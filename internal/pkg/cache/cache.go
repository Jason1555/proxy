package cache

import "sync"

type Storage interface {
	Set(key string, value any)
	Get(key string) (any, bool)
	Delete(key string)
	Len() int
	Clear()
}

type EvictionPolicy interface {
	OnGet(key string)
	OnSet(key string, value any)
	OnDelete(key string)
	GetVictim() string
}

type Cache struct {
	mu sync.RWMutex
	Storage
	policy  EvictionPolicy
	maxSize int
}

func NewCache(storage Storage, policy EvictionPolicy, maxSize int) *Cache {
	return &Cache{
		Storage: storage,
		policy:  policy,
		maxSize: maxSize,
	}
}

func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.Storage.Get(key)

	if !exists && c.Len() >= c.maxSize {
		if victim := c.policy.GetVictim(); victim != "" {
			c.Storage.Delete(victim)
			c.policy.OnDelete(victim)
		}
	}

	c.Storage.Set(key, value)
	c.policy.OnSet(key, value)
}

func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.Storage.Get(key)
	if exists {
		c.policy.OnGet(key)
	}
	return value, exists
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Storage.Delete(key)
	c.policy.OnDelete(key)
}
