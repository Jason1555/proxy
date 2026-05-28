package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{
			"method",
			"path",
			"status",
		},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "proxy_http_request_duration_seconds",
			Help:    "Request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{
			"path",
		},
	)

	ACLAllowedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "proxy_acl_allowed_total",
			Help: "Allowed requests",
		},
	)

	ACLDeniedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "proxy_acl_denied_total",
			Help: "Denied requests",
		},
	)
)

func Init() {

	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		ACLAllowedTotal,
		ACLDeniedTotal,
	)
}
