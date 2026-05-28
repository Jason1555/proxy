package usecases

import (
	"context"
	"net/http"
	"proxy/internal/domain"
	"proxy/internal/pkg/cache"
	"proxy/internal/repository"
	"proxy/internal/testutils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheService_Get_Hit(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		DefaultTTL:  time.Hour,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	entry := &domain.CacheEntry{
		Key:        "test-key",
		Value:      []byte("test-value"),
		StatusCode: 200,
		ExpiresAt:  time.Now().Add(time.Hour),
		Size:       10,
	}

	err := cacheService.Set(context.Background(), entry)
	require.NoError(t, err)

	cached, err := cacheService.Get(context.Background(), "test-key")
	require.NoError(t, err)
	assert.NotNil(t, cached)
	assert.Equal(t, "test-value", string(cached.Value))
}

func TestCacheService_Get_Miss(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		DefaultTTL:  time.Hour,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	cached, err := cacheService.Get(context.Background(), "non-existent-key")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCacheService_Get_Expired(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		DefaultTTL:  time.Hour,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})
	entry := &domain.CacheEntry{
		Key:        "test-key",
		Value:      []byte("test-value"),
		StatusCode: 200,
		ExpiresAt:  time.Now().Add(-time.Minute),
		Size:       10,
	}

	err := cacheService.Set(context.Background(), entry)
	require.NoError(t, err)
	cached, err := cacheService.Get(context.Background(), "test-key")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCacheService_Get_InvalidType(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		DefaultTTL:  time.Hour,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})
	cache.Set("test-key", "invalid-type")
	cached, err := cacheService.Get(context.Background(), "test-key")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCacheService_GetCachePolicy_2xx(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		TTL2xx:      time.Minute * 5,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	policyFunc := cacheService.GetCachePolicy(200, http.Header{}, 100)
	assert.True(t, policyFunc.Cacheable)
	assert.Equal(t, time.Minute*5, policyFunc.TTL)
}

func TestCacheService_GetCachePolicy_3xx(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		TTL3xx:      time.Minute * 5,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	policyFunc := cacheService.GetCachePolicy(304, http.Header{}, 100)
	assert.True(t, policyFunc.Cacheable)
	assert.Equal(t, time.Minute*5, policyFunc.TTL)
}

func TestCacheService_GetCachePolicy_4xx(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		TTL4xx:      time.Minute * 5,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	policyFunc := cacheService.GetCachePolicy(404, http.Header{}, 100)
	assert.True(t, policyFunc.Cacheable)
	assert.Equal(t, time.Minute*5, policyFunc.TTL)
}

func TestCacheService_GetCachePolicy_5xx(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		TTL5xx:      time.Minute * 5,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	policyFunc := cacheService.GetCachePolicy(500, http.Header{}, 100)
	assert.True(t, policyFunc.Cacheable)
	assert.Equal(t, time.Minute*5, policyFunc.TTL)
}

func TestCacheService_GenerateKey(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		DefaultTTL:  time.Hour,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	entry := &domain.CacheEntry{
		Key:       "test-key",
		Value:     []byte("test-value"),
		ExpiresAt: time.Now().Add(time.Hour),
		Size:      10,
	}

	cacheService.Set(context.Background(), entry)

	req := domain.InvalidationRequest{
		Type:  domain.InvalidateByKey,
		Value: "test-key",
	}
	err := cacheService.Invalidate(context.Background(), req)
	require.NoError(t, err)

	cached, err := cacheService.Get(context.Background(), "test-key")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCacheService_Disabled(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		DefaultTTL:  time.Hour,
		Enabled:     false,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	entry := &domain.CacheEntry{
		Key:       "test-key",
		Value:     []byte("test-value"),
		ExpiresAt: time.Now().Add(time.Hour),
		Size:      10,
	}

	err := cacheService.Set(context.Background(), entry)
	require.NoError(t, err)

	cached, err := cacheService.Get(context.Background(), "test-key")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCacheService_SizeLimits(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		DefaultTTL:  time.Hour,
		Enabled:     true,
		MinBodySize: 5,
		MaxBodySize: 15,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	entry := &domain.CacheEntry{
		Key:       "test-key",
		Value:     []byte("test"),
		ExpiresAt: time.Now().Add(time.Hour),
		Size:      4,
	}

	err := cacheService.Set(context.Background(), entry)
	require.NoError(t, err)
	cached, err := cacheService.Get(context.Background(), "test-key")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCacheService_GetStats(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		DefaultTTL:  time.Hour,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	entry := &domain.CacheEntry{
		Key:       "test-key",
		Value:     []byte("test-value"),
		ExpiresAt: time.Now().Add(time.Hour),
		Size:      10,
	}

	err := cacheService.Set(context.Background(), entry)
	require.NoError(t, err)

	_, err = cacheService.Get(context.Background(), "test-key")
	require.NoError(t, err)
	_, err = cacheService.Get(context.Background(), "non-existent-key")
	require.NoError(t, err)
	stats := cacheService.GetStats(context.Background())
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

func TestCacheService_Clear(t *testing.T) {
	storage := repository.NewMapStorage()
	policy := domain.NewLRUPolicy()
	cache := cache.NewCache(storage, policy, 1000)

	config := domain.CacheConfig{
		MaxSize:     1000,
		DefaultTTL:  time.Hour,
		Enabled:     true,
		MinBodySize: 1,
		MaxBodySize: 1000,
	}

	cacheService := NewCacheService(cache, config, &testutils.MockLogger{})

	entry := &domain.CacheEntry{
		Key:       "test-key",
		Value:     []byte("test-value"),
		ExpiresAt: time.Now().Add(time.Hour),
		Size:      10,
	}
	err := cacheService.Set(context.Background(), entry)
	require.NoError(t, err)

	err = cacheService.Clear(context.Background())
	require.NoError(t, err)
	cached, err := cacheService.Get(context.Background(), "test-key")
	require.NoError(t, err)
	assert.Nil(t, cached)
}
