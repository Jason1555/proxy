package usecases

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"proxy/internal/domain"
	"proxy/internal/infrastructure/logger"
	"proxy/internal/pkg/cache"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

type CacheService interface {
	Get(ctx context.Context, key string) (*domain.CacheEntry, error)
	Set(ctx context.Context, entry *domain.CacheEntry) error
	Delete(ctx context.Context, key string) error
	Invalidate(ctx context.Context, req domain.InvalidationRequest) error
	GetStats(ctx context.Context) domain.CacheStats
	Clear(ctx context.Context) error
	GetCachePolicy(statusCode int, header http.Header, bodySize int64) domain.ResponseCachePolicy
	GenerateKey(method, url string, query map[string]string) string
}

type cacheService struct {
	cache  *cache.Cache
	config domain.CacheConfig
	logger logger.Logger
	stats  *cacheStats
}

type cacheStats struct {
	hits   int64
	misses int64
}

func NewCacheService(cache *cache.Cache, config domain.CacheConfig, logger logger.Logger) CacheService {
	return &cacheService{
		cache:  cache,
		config: config,
		logger: logger,
		stats:  &cacheStats{},
	}
}

func (s *cacheService) Get(ctx context.Context, key string) (*domain.CacheEntry, error) {
	if !s.config.Enabled {
		atomic.AddInt64(&s.stats.misses, 1)
		return nil, nil
	}

	startTime := time.Now()

	value, exists := s.cache.Get(key)
	if !exists {
		atomic.AddInt64(&s.stats.misses, 1)
		s.logger.Debugf("Cache miss for key: %s", key)
		return nil, nil
	}

	entry, ok := value.(domain.CacheEntry)
	if !ok {
		atomic.AddInt64(&s.stats.misses, 1)
		s.logger.Warnf("Cache entry for key %s has invalid type", key)
		return nil, nil
	}

	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		s.cache.Delete(key)
		atomic.AddInt64(&s.stats.misses, 1)
		s.logger.Debugf("Cache entry for key %s has expired", key)
		return nil, nil
	}

	atomic.AddInt64(&s.stats.hits, 1)
	latency := time.Since(startTime)
	s.logger.Debugf("Cache hit for key %s, size: %d bytes, latency: %dms", key, entry.Size, latency.Milliseconds())
	return &entry, nil
}

func (s *cacheService) Set(ctx context.Context, entry *domain.CacheEntry) error {
	if !s.config.Enabled {
		return nil
	}

	if entry.Size < s.config.MinBodySize || entry.Size > s.config.MaxBodySize {
		s.logger.Debugf("Entry size %d is outside of %d and %d bounds", entry.Size, s.config.MinBodySize, s.config.MaxBodySize)
		return nil
	}

	if entry.ExpiresAt.IsZero() {
		entry.ExpiresAt = time.Now().Add(s.config.DefaultTTL)
	}
	s.cache.Set(entry.Key, *entry)
	s.logger.Debugf("Cache entry set for key %s, size: %d bytes, expires at: %s", entry.Key, entry.Size, time.Until(entry.ExpiresAt))
	return nil
}

func (s *cacheService) Delete(ctx context.Context, key string) error {
	s.cache.Delete(key)
	s.logger.Debugf("Cache entry deleted for key: %s", key)
	return nil
}

func (s *cacheService) Invalidate(ctx context.Context, req domain.InvalidationRequest) error {
	if !s.config.Enabled {
		return nil
	}

	switch req.Type {
	case domain.InvalidateByKey:
		s.cache.Delete(req.Value)
		s.logger.Infof("Cache entry invalidated by key: %s", req.Value)

	case domain.InvalidateAll:
		s.cache.Clear()
		s.logger.Infof("All cache entries invalidated")

	default:
		return fmt.Errorf("invalid invalidation type: %s", req.Type)
	}

	return nil
}

func (s *cacheService) GetStats(ctx context.Context) domain.CacheStats {
	hits := atomic.LoadInt64(&s.stats.hits)
	misses := atomic.LoadInt64(&s.stats.misses)

	return domain.CacheStats{
		Size:        s.config.MaxSize,
		MaxSize:     s.config.MaxSize,
		Keys:        s.cache.Len(),
		Utilization: float64(s.cache.Len()) / float64(s.config.MaxSize),
		Hits:        hits,
		Misses:      misses,
	}
}

func (s *cacheService) Clear(ctx context.Context) error {
	s.cache.Clear()
	s.logger.Infof("Cache cleared")
	return nil
}

func (s *cacheService) GetCachePolicy(statusCode int, header http.Header, bodySize int64) domain.ResponseCachePolicy {
	policy := domain.ResponseCachePolicy{
		Cacheable: false,
		Reason:    "Not cacheable by default",
	}

	cacheControl := header.Get("Cache-Control")
	if cacheControl != "" {
		if contains(cacheControl, "no-store") || contains(cacheControl, "no-cache") {
			policy.Reason = "Response has no-store or no-cache directive"
			return policy
		}
	}

	switch statusCode {
	case 200:
		policy.Cacheable = true
		policy.TTL = s.config.TTL2xx
		policy.Reason = "2xx responses are cacheable by default"
		policy.Tags = []string{"2xx", "success"}

	case 301, 302, 304:
		policy.Cacheable = true
		policy.TTL = s.config.TTL3xx
		policy.Reason = "3xx responses are cacheable by default"
		policy.Tags = []string{"3xx", "redirect"}

	case 400, 401, 403, 404:
		policy.Cacheable = true
		policy.TTL = s.config.TTL4xx
		policy.Reason = "4xx responses are cacheable by default"
		policy.Tags = []string{"4xx", "client-error"}

	case 500, 502, 503, 504:
		policy.Cacheable = true
		policy.TTL = s.config.TTL5xx
		policy.Reason = "5xx responses are cacheable by default"
		policy.Tags = []string{"5xx", "server-error"}

	default:
		policy.Reason = fmt.Sprintf("Status code %d is not cacheable by default", statusCode)
	}

	if bodySize < s.config.MinBodySize || bodySize > s.config.MaxBodySize {
		policy.Cacheable = false
		policy.Reason = fmt.Sprintf("Response body size %d is outside of %d and %d bounds", bodySize, s.config.MinBodySize, s.config.MaxBodySize)
	}

	return policy
}

func (s *cacheService) GenerateKey(method, url string, query map[string]string) string {
	key := fmt.Sprintf("%s:%s", method, url)

	keys := make([]string, 0, len(query))

	for k := range query {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		key += fmt.Sprintf(":%s=%s", k, query[k])
	}

	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("cache:%s:%x", method, hash)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
