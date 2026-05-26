package domain

import "time"

type MonitoringConfig struct {
	RequestWindow        time.Duration `json:"request_window" yaml:"request_window" toml:"request_window"`
	MaxLatencySamples    int           `json:"max_latency_samples" yaml:"max_latency_samples" toml:"max_latency_samples"`
	MaxPathStats         int           `json:"max_path_stats" yaml:"max_path_stats" toml:"max_path_stats"`
	TrackPathStats       bool          `json:"track_path_stats" yaml:"track_path_stats" toml:"track_path_stats"`
	EnableRequestLogging bool          `json:"enable_request_logging" yaml:"enable_request_logging" toml:"enable_request_logging"`
}

func NewDefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		RequestWindow:     time.Minute,
		MaxLatencySamples: 1024,
		MaxPathStats:      100,
		TrackPathStats:    true,
	}
}

type RequestMetric struct {
	Method     string
	Path       string
	ClientIP   string
	StatusCode int
	BytesIn    int64
	BytesOut   int64
	Latency    time.Duration
	Timestamp  time.Time
}

type IPAccessMetric struct {
	IP        string
	Allowed   bool
	ListType  IPListType
	Reason    string
	Timestamp time.Time
}

type RateLimitMetric struct {
	Key       string
	Reason    string
	Timestamp time.Time
}

type UpstreamMetric struct {
	Name       string
	Healthy    bool
	StatusCode int
	Latency    time.Duration
	Error      string
	Timestamp  time.Time
}

type MonitoringSnapshot struct {
	StartedAt     time.Time                `json:"started_at"`
	UptimeSeconds int64                    `json:"uptime_seconds"`
	Requests      RequestStats             `json:"requests"`
	Traffic       TrafficStats             `json:"traffic"`
	IPAccess      IPAccessMonitoringStats  `json:"ip_access"`
	RateLimit     RateLimitMonitoringStats `json:"rate_limit"`
	Upstreams     map[string]UpstreamStats `json:"upstreams"`
}

type RequestStats struct {
	Total          int64            `json:"total"`
	InFlight       int64            `json:"in_flight"`
	Success        int64            `json:"success"`
	Errors         int64            `json:"errors"`
	RecentRequests int64            `json:"recent_requests"`
	RPS            float64          `json:"rps"`
	ByStatusClass  map[string]int64 `json:"by_status_class"`
	ByMethod       map[string]int64 `json:"by_method"`
	ByPath         map[string]int64 `json:"by_path,omitempty"`
	Latency        LatencyStats     `json:"latency"`
}

type LatencyStats struct {
	MinMS float64 `json:"min_ms"`
	MaxMS float64 `json:"max_ms"`
	AvgMS float64 `json:"avg_ms"`
	P50MS float64 `json:"p50_ms"`
	P95MS float64 `json:"p95_ms"`
	P99MS float64 `json:"p99_ms"`
}

type TrafficStats struct {
	BytesIn  int64 `json:"bytes_in"`
	BytesOut int64 `json:"bytes_out"`
}

type IPAccessMonitoringStats struct {
	Total    int64            `json:"total"`
	Allowed  int64            `json:"allowed"`
	Denied   int64            `json:"denied"`
	Grey     int64            `json:"grey"`
	ByReason map[string]int64 `json:"by_reason"`
}

type RateLimitMonitoringStats struct {
	LimitedRequests int64            `json:"limited_requests"`
	ByKey           map[string]int64 `json:"by_key"`
	ByReason        map[string]int64 `json:"by_reason"`
}

type UpstreamStats struct {
	Name           string    `json:"name"`
	Healthy        bool      `json:"healthy"`
	TotalRequests  int64     `json:"total_requests"`
	Errors         int64     `json:"errors"`
	LastStatusCode int       `json:"last_status_code"`
	LastLatencyMS  float64   `json:"last_latency_ms"`
	AvgLatencyMS   float64   `json:"avg_latency_ms"`
	LastError      string    `json:"last_error,omitempty"`
	LastCheckedAt  time.Time `json:"last_checked_at"`
}

type MonitoringCollector interface {
	BeginRequest()
	CompleteRequest(metric RequestMetric)
	RecordIPAccess(metric IPAccessMetric)
	RecordRateLimitViolation(metric RateLimitMetric)
	RecordUpstream(metric UpstreamMetric)
	Snapshot() MonitoringSnapshot
	Reset()
}
