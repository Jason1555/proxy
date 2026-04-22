package cache

import (
	"container/list"
	"sync"
)

type Cache interface {
	Get(key string) (any, bool)
	Set(key string, value any)
	Delete(key string)
	Clear()
	Len() int
}

type lruCache struct {
	maxSize int
	mu      sync.RWMutex
	cache   map[string]*list.Element
	order   *list.List
}

type cacheEntry struct {
	key   string
	value any
}

func NewLRUCache(maxSize int) Cache {
	return &lruCache{
		maxSize: maxSize,
		cache:   make(map[string]*list.Element),
		order:   list.New(),
	}
}

func (l *lruCache) Get(key string) (any, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	elem, exists := l.cache[key]
	if !exists {
		return nil, false
	}

	l.order.MoveToFront(elem)
	return elem.Value.(*cacheEntry).value, true
}

func (l *lruCache) Set(key string, value any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if elem, exists := l.cache[key]; exists {
		elem.Value = value
		l.order.MoveToFront(elem)
		return
	}

	elem := l.order.PushFront(&cacheEntry{key, value})
	l.cache[key] = elem

	if l.order.Len() > l.maxSize {
		lastElem := l.order.Back()
		if lastElem != nil {
			l.order.Remove(lastElem)
			delete(l.cache, lastElem.Value.(*cacheEntry).key)
		}
	}
}

func (l *lruCache) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if elem, exists := l.cache[key]; exists {
		l.order.Remove(elem)
		delete(l.cache, key)
	}
}

func (l *lruCache) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string]*list.Element)
	l.order = list.New()
}

func (l *lruCache) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.cache)
}
