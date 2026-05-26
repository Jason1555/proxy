package testutils

import (
	"context"
	"proxy/internal/domain"
	"sync"
)

type MockRateLimitStore struct {
	mu    sync.Mutex
	store map[string]*domain.RateLimitState
}

func NewRateLimitStore() *MockRateLimitStore {
	return &MockRateLimitStore{
		store: make(map[string]*domain.RateLimitState),
	}
}

func (m *MockRateLimitStore) Update(ctx context.Context, key string, fn func(*domain.RateLimitState) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.store[key]
	if !exists {
		state = &domain.RateLimitState{}
		m.store[key] = state
	}
	return fn(state)
}

func (m *MockRateLimitStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.store, key)
	return nil
}

func (m *MockRateLimitStore) Keys(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := make([]string, 0, len(m.store))
	for k := range m.store {
		keys = append(keys, k)
	}

	return keys, nil
}
