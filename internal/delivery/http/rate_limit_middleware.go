package http

import (
	"net/http"
	"proxy/internal/usecases"
)

type RateLimitMiddleware struct {
	service usecases.RateLimitService
}

func NewRateLimitMiddleware(service usecases.RateLimitService) *RateLimitMiddleware {
	return &RateLimitMiddleware{service: service}
}

func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		bodySize := max(r.ContentLength, 0)

		decision, err := m.service.EvaluateRequest(r.Context(), ip, bodySize)
		if err != nil {
			http.Error(w, "rate limiter error", http.StatusInternalServerError)
			return
		}

		if !decision.Allowed {
			http.Error(w, decision.Reason, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	ip := r.RemoteAddr
	for i := 0; i < len(ip); i++ {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}
