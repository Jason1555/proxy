package cache

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockMapStorage

type mockStorage struct {
	mu   sync.RWMutex
	data map[string]any
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: make(map[string]any),
	}
}

func (s *mockStorage) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *mockStorage) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.data[key]
	return value, exists
}

func (s *mockStorage) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *mockStorage) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

func (s *mockStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]any)
}

// MockPolicy

type mockPolicy struct {
	victim string

	onGetCalled    []string
	onSetCalled    []string
	onDeleteCalled []string
}

func (m *mockPolicy) OnGet(key string) {
	m.onGetCalled = append(m.onGetCalled, key)
}

func (m *mockPolicy) OnSet(key string, value any) {
	m.onSetCalled = append(m.onSetCalled, key)
}

func (m *mockPolicy) OnDelete(key string) {
	m.onDeleteCalled = append(m.onDeleteCalled, key)
}

func (m *mockPolicy) GetVictim() string {
	return m.victim
}

// Tests

func TestCache_Set_Get(t *testing.T) {
	storage := newMockStorage()
	policy := &mockPolicy{}
	cache := NewCache(storage, policy, 2)

	cache.Set("key1", "value1")

	v, ok := cache.Get("key1")

	assert.True(t, ok)
	assert.Equal(t, "value1", v)
	assert.Contains(t, policy.onGetCalled, "key1")
	assert.Contains(t, policy.onSetCalled, "key1")
}

func TestCache_Delete(t *testing.T) {
	storage := newMockStorage()
	policy := &mockPolicy{}
	cache := NewCache(storage, policy, 2)

	cache.Set("key1", "value1")
	cache.Delete("key1")

	v, ok := cache.Get("key1")

	assert.False(t, ok)
	assert.Nil(t, v)
	assert.Contains(t, policy.onDeleteCalled, "key1")
}

func TestCache_Eviction(t *testing.T) {
	storage := newMockStorage()
	policy := &mockPolicy{victim: "key1"}
	cache := NewCache(storage, policy, 1)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	_, ok1 := cache.Get("key1")
	v2, ok2 := cache.Get("key2")

	assert.False(t, ok1)
	assert.True(t, ok2)
	assert.Equal(t, "value2", v2)

	assert.Contains(t, policy.onDeleteCalled, "key1")
	assert.Contains(t, policy.onSetCalled, "key2")
}

func TestCache_Get_NotFound(t *testing.T) {
	storage := newMockStorage()
	policy := &mockPolicy{}
	cache := NewCache(storage, policy, 2)

	v, ok := cache.Get("nonexistent")

	assert.False(t, ok)
	assert.Nil(t, v)
	assert.Empty(t, policy.onGetCalled)
}
