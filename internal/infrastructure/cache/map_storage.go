package cache

type MapStorage struct {
	data map[string]any
}

func NewMapStorage() *MapStorage {
	return &MapStorage{
		data: make(map[string]any),
	}
}

func (s *MapStorage) Set(key string, value any) {
	s.data[key] = value
}

func (s *MapStorage) Get(key string) (any, bool) {
	value, exists := s.data[key]
	return value, exists
}

func (s *MapStorage) Delete(key string) {
	delete(s.data, key)
}

func (s *MapStorage) Len() int {
	return len(s.data)
}

func (s *MapStorage) Clear() {
	s.data = make(map[string]any)
}
