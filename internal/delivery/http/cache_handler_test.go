package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"proxy/internal/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type CacheServiceMock struct {
	mock.Mock
}

func (m *CacheServiceMock) Get(ctx context.Context, key string) (*domain.CacheEntry, error) {
	args := m.Called(ctx, key)

	entry := args.Get(0)
	if entry == nil {
		return nil, args.Error(1)
	}
	return entry.(*domain.CacheEntry), args.Error(1)
}

func (m *CacheServiceMock) Set(ctx context.Context, entry *domain.CacheEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *CacheServiceMock) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *CacheServiceMock) Invalidate(ctx context.Context, req domain.InvalidationRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *CacheServiceMock) GetStats(ctx context.Context) domain.CacheStats {
	args := m.Called(ctx)
	return args.Get(0).(domain.CacheStats)
}

func (m *CacheServiceMock) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *CacheServiceMock) GetCachePolicy(statusCode int, header http.Header, bodySize int64) domain.ResponseCachePolicy {
	args := m.Called(statusCode, header, bodySize)
	return args.Get(0).(domain.ResponseCachePolicy)
}

func (m *CacheServiceMock) GenerateKey(method, path string, query map[string]string) string {
	args := m.Called(method, path, query)
	return args.String(0)
}

func TestCacheHandler_GetStats_Success(t *testing.T) {
	cacheMock := new(CacheServiceMock)
	expectedStats := domain.CacheStats{
		Size:        100,
		MaxSize:     1000,
		Keys:        5,
		Utilization: 0.1,
		Hits:        10,
		Misses:      2,
	}

	cacheMock.On("GetStats", mock.Anything).Return(expectedStats)

	handler := NewCacheHandler(cacheMock)
	req := httptest.NewRequest(http.MethodGet, "/cache/stats", nil)
	rr := httptest.NewRecorder()

	handler.GetStats(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var actualStats domain.CacheStats
	err := json.Unmarshal(rr.Body.Bytes(), &actualStats)
	assert.NoError(t, err)
	assert.Equal(t, expectedStats, actualStats)
	cacheMock.AssertExpectations(t)
}

func TestCacheHandler_Clear_Success(t *testing.T) {
	cacheMock := new(CacheServiceMock)
	cacheMock.On("Clear", mock.Anything).Return(nil)

	handler := NewCacheHandler(cacheMock)
	req := httptest.NewRequest(http.MethodDelete, "/cache", nil)
	rr := httptest.NewRecorder()

	handler.Clear(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	cacheMock.AssertExpectations(t)
}
