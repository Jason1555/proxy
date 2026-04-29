package domain

import "time"

type CacheEntry struct {
	Key        string
	Value      []byte
	StatusCode int
	Headers    map[string]string
	ExpiresAt  time.Time
	Tags       []string
	Size	   int64
}

type CacheConfig struct {
	MaxSize int64
	DefaultTTL     time.Duration
	TTL2xx  time.Duration
	TTL3xx  time.Duration
	TTL4xx  time.Duration
	TTL5xx  time.Duration
	MinBodySize int64
	MaxBodySize int64
	Enabled bool
}

type CacheStats struct {
	Size 	 int64
	MaxSize int64
	Keys	int
	Utilization float64
	Hits    int64
	Misses  int64
}

type InvalidationRequest struct {
	Type InvalidationType `json:"type"`
	Value string `json:"value"`
}

type InvalidationType string

const (
	InvalidateByKey InvalidationType = "key"
	InvalidateByPrefix InvalidationType = "prefix"
	InvalidateByTag InvalidationType = "tag"
	InvalidateAll InvalidationType = "all"
)

type CacheHitMiss struct {
	Key string
	Hit bool
	Size int64
	Latency time.Duration
}

type ResponseCachePolicy struct {
	Cacheable bool
	TTL time.Duration
	Tags []string
	Reason string
}

func (p *ResponseCachePolicy) IsCacheable(statusCode int) bool {
	switch statusCode{
	case 200, 201, 204:
		return true
	case 301, 302, 304:
		return true
	default:
		return false
	}
}
