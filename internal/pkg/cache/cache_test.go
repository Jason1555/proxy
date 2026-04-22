package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache_Set_Get(t *testing.T) {
	cache := NewLRUCache(2)
	cache.Set("key1", "value1")
	
	val, ok := cache.Get("key1")

	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestCache_Eviction(t *testing.T) {
	cache := NewLRUCache(2)
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	_, ok := cache.Get("key1")
	assert.False(t, ok)

	val, ok := cache.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, "value2", val)

	val, ok = cache.Get("key3")
	assert.True(t, ok)
	assert.Equal(t, "value3", val)
}

func TestCache_Delete(t *testing.T) {
	cache := NewLRUCache(2)

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	assert.False(t, ok)
}

func TestCache_Clear(t *testing.T) {
	cache := NewLRUCache(10)
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Clear()

	_, ok := cache.Get("key1")
	assert.False(t, ok)
	_, ok = cache.Get("key2")
	assert.False(t, ok)

	assert.Equal(t, 0, cache.Len())
}

func TestCache_Order(t *testing.T) {
	cache := NewLRUCache(2)
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Get("key1") 

	cache.Set("key3", "value3")
	_, ok := cache.Get("key2")
	assert.False(t, ok)

	val, ok := cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}