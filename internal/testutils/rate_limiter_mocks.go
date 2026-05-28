package testutils

import (
	"context"
	"sync"

	"proxy/internal/domain"
)

type MockRateLimitStore struct {
	Mu    sync.Mutex
	Store map[string]*domain.RateLimitState

	UpdateErr error
	GetErr    error
	DeleteErr error
	KeysErr   error
}

func NewMockRateLimitStore() *MockRateLimitStore {
	return &MockRateLimitStore{
		Store: make(map[string]*domain.RateLimitState),
	}
}

// backward compatibility for tests
func NewRateLimitStore() *MockRateLimitStore {
	return NewMockRateLimitStore()
}

func (m *MockRateLimitStore) Update(
	ctx context.Context,
	key string,
	fn func(*domain.RateLimitState) error,
) error {

	if m.UpdateErr != nil {
		return m.UpdateErr
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()

	state, exists := m.Store[key]
	if !exists {
		state = &domain.RateLimitState{}
		m.Store[key] = state
	}

	return fn(state)
}

func (m *MockRateLimitStore) Get(
	ctx context.Context,
	key string,
) (*domain.RateLimitState, error) {

	if m.GetErr != nil {
		return nil, m.GetErr
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()

	state, exists := m.Store[key]
	if !exists {
		return nil, nil
	}

	return state, nil
}

func (m *MockRateLimitStore) Delete(
	ctx context.Context,
	key string,
) error {

	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()

	delete(m.Store, key)
	return nil
}

func (m *MockRateLimitStore) Keys(
	ctx context.Context,
) ([]string, error) {

	if m.KeysErr != nil {
		return nil, m.KeysErr
	}

	m.Mu.Lock()
	defer m.Mu.Unlock()

	keys := make([]string, 0, len(m.Store))

	for k := range m.Store {
		keys = append(keys, k)
	}

	return keys, nil
}
