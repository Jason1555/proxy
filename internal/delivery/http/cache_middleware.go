package http

import (
	"bytes"
	"net/http"
	"proxy/internal/domain"
	"proxy/internal/usecases"
	"time"
)

type CacheMiddleware struct {
	cache usecases.CacheService
}

func NewCacheMiddleware(cache usecases.CacheService) *CacheMiddleware {
	return &CacheMiddleware{cache: cache}
}

func (m *CacheMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		query := make(map[string]string)

		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				query[key] = values[0]
			}
		}

		cacheKey := m.cache.GenerateKey(r.Method, r.URL.Path, query)

		entry, err := m.cache.Get(ctx, cacheKey)
		if err != nil {
			http.Error(w, "cache error", http.StatusInternalServerError)
			return
		}

		if entry != nil {
			for k, values := range entry.Headers {
				w.Header().Set(k, values)
			}

			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(entry.StatusCode)

			_, _ = w.Write(entry.Value)
			return
		}

		recorder := newResponseRecorder(w)

		next.ServeHTTP(recorder, r)

		body := recorder.body.Bytes()

		policy := m.cache.GetCachePolicy(
			recorder.statusCode,
			recorder.Header(),
			int64(len(body)),
		)

		if !policy.Cacheable {
			return
		}

		cacheEntry := &domain.CacheEntry{
			Key:        cacheKey,
			Value:      body,
			StatusCode: recorder.statusCode,
			Headers:    headerToMap(recorder.Header()),
			Size:       int64(len(body)),
			ExpiresAt:  time.Now().Add(policy.TTL),
		}

		_ = m.cache.Set(ctx, cacheEntry)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		body:           bytes.NewBuffer(nil),
		statusCode:     http.StatusOK,
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) Header() http.Header {
	return r.ResponseWriter.Header()
}

func headerToMap(header http.Header) map[string]string {
	result := make(map[string]string)

	for key, values := range header {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}

	return result
}
