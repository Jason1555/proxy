package http

import (
	"net"
	"net/http"
	"proxy/internal/usecases"
	"strings"
)

type IPFilterMiddleware struct {
	service usecases.IPFilterService
}

func NewIPFilterMiddleware(service usecases.IPFilterService) *IPFilterMiddleware {
	return &IPFilterMiddleware{service: service}
}

func (m *IPFilterMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := extractClientIP(r)

		result, err := m.service.CheckIP(r.Context(), clientIP)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-IP-Filter", string(result.ListType))

		w.Header().Set("X-IP-Filter-Reason", result.Reason)

		if !result.IsAllowed {
			http.Error(w, result.Reason, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
