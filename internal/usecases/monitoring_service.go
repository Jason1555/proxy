package usecases

import (
	"sort"
	"sync"
	"time"

	"proxy/internal/domain"
	"proxy/internal/infrastructure/logger"
)

type MonitoringService interface {
	domain.MonitoringCollector
}

type monitoringService struct {
	mu        sync.RWMutex
	config    domain.MonitoringConfig
	logger    logger.Logger
	startedAt time.Time

	totalRequests int64
	inFlight      int64
	success       int64
	errors        int64
	totalLatency  float64
	minLatency    float64
	maxLatency    float64

	byStatusClass map[string]int64
	byMethod      map[string]int64
	byPath        map[string]int64

	traffic   domain.TrafficStats
	ipAccess  domain.IPAccessMonitoringStats
	rate      domain.RateLimitMonitoringStats
	upstreams map[string]*domain.UpstreamStats

	requestSamples []requestSample
	latencySamples []float64
}

type requestSample struct {
	at        time.Time
	latencyMS float64
}

func NewMonitoringService(config domain.MonitoringConfig, logger logger.Logger) MonitoringService {
	config = normalizeMonitoringConfig(config)

	return &monitoringService{
		config:        config,
		logger:        logger,
		startedAt:     time.Now(),
		byStatusClass: make(map[string]int64),
		byMethod:      make(map[string]int64),
		byPath:        make(map[string]int64),
		ipAccess: domain.IPAccessMonitoringStats{
			ByReason: make(map[string]int64),
		},
		rate: domain.RateLimitMonitoringStats{
			ByKey:    make(map[string]int64),
			ByReason: make(map[string]int64),
		},
		upstreams: make(map[string]*domain.UpstreamStats),
	}
}

func normalizeMonitoringConfig(config domain.MonitoringConfig) domain.MonitoringConfig {
	defaults := domain.NewDefaultMonitoringConfig()
	if config == (domain.MonitoringConfig{}) {
		return defaults
	}

	if config.RequestWindow <= 0 {
		config.RequestWindow = defaults.RequestWindow
	}
	if config.MaxLatencySamples <= 0 {
		config.MaxLatencySamples = defaults.MaxLatencySamples
	}
	if config.MaxPathStats <= 0 {
		config.MaxPathStats = defaults.MaxPathStats
	}

	return config
}

func (s *monitoringService) BeginRequest() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.inFlight++
}

func (s *monitoringService) CompleteRequest(metric domain.RequestMetric) {
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}
	if metric.Method == "" {
		metric.Method = "UNKNOWN"
	}
	if metric.Path == "" {
		metric.Path = "/"
	}
	if metric.StatusCode == 0 {
		metric.StatusCode = 200
	}
	if metric.Latency < 0 {
		metric.Latency = 0
	}

	latencyMS := durationToMS(metric.Latency)
	statusClass := statusClass(metric.StatusCode)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.inFlight > 0 {
		s.inFlight--
	}

	s.totalRequests++
	s.totalLatency += latencyMS
	if s.totalRequests == 1 || latencyMS < s.minLatency {
		s.minLatency = latencyMS
	}
	if latencyMS > s.maxLatency {
		s.maxLatency = latencyMS
	}

	if metric.StatusCode >= 500 {
		s.errors++
	} else {
		s.success++
	}

	s.byStatusClass[statusClass]++
	s.byMethod[metric.Method]++
	if s.config.TrackPathStats && (len(s.byPath) < s.config.MaxPathStats || s.byPath[metric.Path] > 0) {
		s.byPath[metric.Path]++
	}

	s.traffic.BytesIn += metric.BytesIn
	s.traffic.BytesOut += metric.BytesOut

	s.requestSamples = append(s.requestSamples, requestSample{
		at:        metric.Timestamp,
		latencyMS: latencyMS,
	})
	s.latencySamples = append(s.latencySamples, latencyMS)
	if len(s.latencySamples) > s.config.MaxLatencySamples {
		copy(s.latencySamples, s.latencySamples[len(s.latencySamples)-s.config.MaxLatencySamples:])
		s.latencySamples = s.latencySamples[:s.config.MaxLatencySamples]
	}
	s.pruneRequestSamplesLocked(metric.Timestamp)

	if s.config.EnableRequestLogging && s.logger != nil {
		s.logger.Infof(
			"request_metric method=%s path=%s status=%d latency_ms=%.2f bytes_in=%d bytes_out=%d client_ip=%s",
			metric.Method,
			metric.Path,
			metric.StatusCode,
			latencyMS,
			metric.BytesIn,
			metric.BytesOut,
			metric.ClientIP,
		)
	}
}

func (s *monitoringService) RecordIPAccess(metric domain.IPAccessMetric) {
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}
	if metric.Reason == "" {
		metric.Reason = "unknown"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ipAccess.Total++
	if metric.Allowed {
		s.ipAccess.Allowed++
	} else {
		s.ipAccess.Denied++
	}
	if metric.ListType == domain.GreyList {
		s.ipAccess.Grey++
	}
	s.ipAccess.ByReason[metric.Reason]++
}

func (s *monitoringService) RecordRateLimitViolation(metric domain.RateLimitMetric) {
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}
	if metric.Key == "" {
		metric.Key = "unknown"
	}
	if metric.Reason == "" {
		metric.Reason = "unknown"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.rate.LimitedRequests++
	s.rate.ByKey[metric.Key]++
	s.rate.ByReason[metric.Reason]++
}

func (s *monitoringService) RecordUpstream(metric domain.UpstreamMetric) {
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}
	if metric.Name == "" {
		metric.Name = "default"
	}
	if metric.Latency < 0 {
		metric.Latency = 0
	}

	latencyMS := durationToMS(metric.Latency)

	s.mu.Lock()
	defer s.mu.Unlock()

	stats, exists := s.upstreams[metric.Name]
	if !exists {
		stats = &domain.UpstreamStats{Name: metric.Name}
		s.upstreams[metric.Name] = stats
	}

	previousTotal := stats.TotalRequests
	stats.TotalRequests++
	stats.Healthy = metric.Healthy
	stats.LastStatusCode = metric.StatusCode
	stats.LastLatencyMS = latencyMS
	stats.AvgLatencyMS = ((stats.AvgLatencyMS * float64(previousTotal)) + latencyMS) / float64(stats.TotalRequests)
	stats.LastCheckedAt = metric.Timestamp

	if metric.Error != "" || !metric.Healthy || metric.StatusCode >= 500 {
		stats.Errors++
		stats.LastError = metric.Error
	}
	if metric.Error == "" && metric.Healthy && metric.StatusCode < 500 {
		stats.LastError = ""
	}
}

func (s *monitoringService) Snapshot() domain.MonitoringSnapshot {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneRequestSamplesLocked(now)

	requests := domain.RequestStats{
		Total:          s.totalRequests,
		InFlight:       s.inFlight,
		Success:        s.success,
		Errors:         s.errors,
		RecentRequests: int64(len(s.requestSamples)),
		RPS:            float64(len(s.requestSamples)) / s.config.RequestWindow.Seconds(),
		ByStatusClass:  cloneIntMap(s.byStatusClass),
		ByMethod:       cloneIntMap(s.byMethod),
		ByPath:         cloneIntMap(s.byPath),
		Latency:        s.buildLatencyStatsLocked(),
	}

	return domain.MonitoringSnapshot{
		StartedAt:     s.startedAt,
		UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()),
		Requests:      requests,
		Traffic:       s.traffic,
		IPAccess: domain.IPAccessMonitoringStats{
			Total:    s.ipAccess.Total,
			Allowed:  s.ipAccess.Allowed,
			Denied:   s.ipAccess.Denied,
			Grey:     s.ipAccess.Grey,
			ByReason: cloneIntMap(s.ipAccess.ByReason),
		},
		RateLimit: domain.RateLimitMonitoringStats{
			LimitedRequests: s.rate.LimitedRequests,
			ByKey:           cloneIntMap(s.rate.ByKey),
			ByReason:        cloneIntMap(s.rate.ByReason),
		},
		Upstreams: cloneUpstreams(s.upstreams),
	}
}

func (s *monitoringService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startedAt = time.Now()
	s.totalRequests = 0
	s.inFlight = 0
	s.success = 0
	s.errors = 0
	s.totalLatency = 0
	s.minLatency = 0
	s.maxLatency = 0
	s.byStatusClass = make(map[string]int64)
	s.byMethod = make(map[string]int64)
	s.byPath = make(map[string]int64)
	s.traffic = domain.TrafficStats{}
	s.ipAccess = domain.IPAccessMonitoringStats{ByReason: make(map[string]int64)}
	s.rate = domain.RateLimitMonitoringStats{
		ByKey:    make(map[string]int64),
		ByReason: make(map[string]int64),
	}
	s.upstreams = make(map[string]*domain.UpstreamStats)
	s.requestSamples = nil
	s.latencySamples = nil
}

func (s *monitoringService) pruneRequestSamplesLocked(now time.Time) {
	cutoff := now.Add(-s.config.RequestWindow)
	keepFrom := 0

	for keepFrom < len(s.requestSamples) && s.requestSamples[keepFrom].at.Before(cutoff) {
		keepFrom++
	}

	if keepFrom > 0 {
		copy(s.requestSamples, s.requestSamples[keepFrom:])
		s.requestSamples = s.requestSamples[:len(s.requestSamples)-keepFrom]
	}
}

func (s *monitoringService) buildLatencyStatsLocked() domain.LatencyStats {
	if s.totalRequests == 0 {
		return domain.LatencyStats{}
	}

	samples := append([]float64(nil), s.latencySamples...)
	sort.Float64s(samples)

	return domain.LatencyStats{
		MinMS: s.minLatency,
		MaxMS: s.maxLatency,
		AvgMS: s.totalLatency / float64(s.totalRequests),
		P50MS: percentile(samples, 0.50),
		P95MS: percentile(samples, 0.95),
		P99MS: percentile(samples, 0.99),
	}
}

func percentile(sortedSamples []float64, p float64) float64 {
	if len(sortedSamples) == 0 {
		return 0
	}
	if len(sortedSamples) == 1 {
		return sortedSamples[0]
	}

	index := int(float64(len(sortedSamples)-1) * p)
	return sortedSamples[index]
}

func cloneIntMap(src map[string]int64) map[string]int64 {
	dst := make(map[string]int64, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneUpstreams(src map[string]*domain.UpstreamStats) map[string]domain.UpstreamStats {
	dst := make(map[string]domain.UpstreamStats, len(src))
	for key, value := range src {
		if value != nil {
			dst[key] = *value
		}
	}
	return dst
}

func statusClass(statusCode int) string {
	return string(rune('0'+statusCode/100)) + "xx"
}

func durationToMS(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}
