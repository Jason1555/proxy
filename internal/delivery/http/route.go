package http

import (
	stdhttp "net/http"
)

type Router struct {
	cache      *CacheHandler
	ipFilter   *IPFilterHandler
	rateLimit  *RateLimitHandler
	monitoring *MonitoringHandler
}

func NewRouter(
	cache *CacheHandler,
	ipFilter *IPFilterHandler,
	rateLimit *RateLimitHandler,
	monitoring *MonitoringHandler,
) *Router {
	return &Router{
		cache:      cache,
		ipFilter:   ipFilter,
		rateLimit:  rateLimit,
		monitoring: monitoring,
	}
}

func (r *Router) Register(mux *stdhttp.ServeMux) {
	mux.HandleFunc("GET /api/cache/stats", r.cache.GetStats)
	mux.HandleFunc("DELETE /api/cache", r.cache.Clear)
	mux.HandleFunc("POST /api/cache/invalidate", r.cache.Invalidate)

	mux.HandleFunc("GET /api/ip/check", r.ipFilter.CheckIP)

	mux.HandleFunc("GET /api/ip/policy", r.ipFilter.GetPolicy)
	mux.HandleFunc("POST /api/ip/policy/reload", r.ipFilter.ReloadPolicy)

	mux.HandleFunc("GET /api/ip/allowlist", r.ipFilter.GetAllowList)
	mux.HandleFunc("GET /api/ip/denylist", r.ipFilter.GetDenyList)
	mux.HandleFunc("GET /api/ip/greylist", r.ipFilter.GetGreyList)

	mux.HandleFunc("POST /api/ip/allowlist", r.ipFilter.AddToAllowList)
	mux.HandleFunc("POST /api/ip/denylist", r.ipFilter.AddToDenyList)
	mux.HandleFunc("POST /api/ip/greylist", r.ipFilter.AddToGreyList)

	mux.HandleFunc("DELETE /api/ip/{id}", r.ipFilter.RemoveEntry)

	mux.HandleFunc("GET /api/ip/stats", r.ipFilter.GetStats)

	mux.HandleFunc("GET /api/rate/status", r.rateLimit.GetStatus)
	mux.HandleFunc("GET /api/rate/metrics", r.rateLimit.GetMetrics)
	mux.HandleFunc("POST /api/rate/reset", r.rateLimit.ResetIP)

	mux.HandleFunc("GET /api/metrics", r.monitoring.GetMetrics)
	mux.HandleFunc("GET /api/health", r.monitoring.GetHealth)
	mux.HandleFunc("GET /api/upstreams", r.monitoring.GetUpstreams)
}
