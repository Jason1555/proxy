package repository

import (
	"context"
	"proxy/internal/domain"
	"sync"
)

// InMemoryRateLimitStore реализует интерфейс RateLimitStore для хранения в памяти
type InMemoryRateLimitStore struct {
	mu    sync.RWMutex
	store map[string]*domain.RateLimitState
}

// NewInMemoryRateLimitStore создает новый in-memory репозиторий для rate limit
func NewInMemoryRateLimitStore() *InMemoryRateLimitStore {
	return &InMemoryRateLimitStore{
		store: make(map[string]*domain.RateLimitState),
	}
}

func (r *InMemoryRateLimitStore) Update(ctx context.Context, key string, fn func(*domain.RateLimitState) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.store[key]
	if !exists {
		state = &domain.RateLimitState{}
		r.store[key] = state
	}

	// Выполняем функцию, которая обновляет состояние токенов
	if err := fn(state); err != nil {
		return err
	}

	return nil
}

func (r *InMemoryRateLimitStore) Get(ctx context.Context, key string) (*domain.RateLimitState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, exists := r.store[key]
	if !exists {
		return nil, nil
	}

	// Возвращаем копию состояния, чтобы избежать data race при параллельном чтении метрик
	stateCopy := *state
	return &stateCopy, nil
}

func (r *InMemoryRateLimitStore) Delete(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.store, key)
	return nil
}

func (r *InMemoryRateLimitStore) Keys(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.store))
	for k := range r.store {
		keys = append(keys, k)
	}

	return keys, nil
}
