package domain

import "time"

type CacheEntry struct {
	Key        string
	Value      []byte
	StatusCode int
	Headers    map[string]string
	ExpiresAt  time.Time
	Tags 	 []string
	Size 	int64
}

type CacheConfig struct {
	MaxSize int64
	TTL     time.Duration
	TTL2xx  time.Duration
	TTL3xx  time.Duration
	TTL4xx  time.Duration
	TTL5xx  time.Duration
	MinBodySize int64
	MaxBodySize int64
}
