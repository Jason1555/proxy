package usecases

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"proxy/internal/domain"
	"proxy/internal/infrastructure/logger"
)

type RateLimitStore interface {
	Update(ctx context.Context, key string, fn func(*domain.RateLimitState) error) error
	Get(ctx context.Context, key string) (*domain.RateLimitState, error)
	Delete(ctx context.Context, key string) error
	Keys(ctx context.Context) ([]string, error)
}

type RateLimitService interface {
	EvaluateRequest(ctx context.Context, ip string, bodySize int64) (Decision, error)

	AddConnection(ctx context.Context, ip string) (bool, error)
	RemoveConnection(ctx context.Context, ip string) error

	GetStatus(ctx context.Context, ip string) (*domain.RateLimitStatus, error)
	GetMetrics(ctx context.Context) domain.RateLimitMetrics

	ResetIP(ctx context.Context, ip string) error
	GetTopViolators(ctx context.Context, limit int) []domain.RateLimitViolator
}

type rateLimitService struct {
	store      RateLimitStore
	config     domain.RateLimitConfig
	logger     logger.Logger
	monitoring domain.MonitoringCollector

	mu         sync.Mutex
	violations []domain.RateLimitViolation
}

func NewRateLimitService(
	store RateLimitStore,
	config domain.RateLimitConfig,
	logger logger.Logger,
	monitoring domain.MonitoringCollector,
) RateLimitService {
	s := &rateLimitService{
		store:      store,
		config:     config,
		logger:     logger,
		monitoring: monitoring,
		violations: make([]domain.RateLimitViolation, 0),
	}

	go s.cleanupViolations()
	return s
}

func (s *rateLimitService) EvaluateRequest(
	ctx context.Context,
	ip string,
	bodySize int64,
) (Decision, error) {

	if !s.config.Enabled {
		return Decision{Allowed: true}, nil
	}

	key := s.getKey(ip)

	var decision Decision

	err := s.store.Update(ctx, key, func(state *domain.RateLimitState) error {
		if state.Key == "" {
			initState(state, key, s.config)
		}

		limiter := NewRateLimiter(state, s.config)
		decision = limiter.Evaluate(bodySize)

		if !decision.Allowed {
			s.recordViolation(key, decision.FailedOn, decision.Reason)
		}

		return nil
	})

	if err != nil {
		return Decision{}, err
	}

	if !decision.Allowed {
		s.logger.Warnf("rate limit exceeded for %s: %s", key, decision.Reason)
	}

	return decision, nil
}

func (s *rateLimitService) AddConnection(ctx context.Context, ip string) (bool, error) {
	key := s.getKey(ip)

	var allowed bool

	err := s.store.Update(ctx, key, func(state *domain.RateLimitState) error {
		if state.Key == "" {
			initState(state, key, s.config)
		}

		limiter := NewRateLimiter(state, s.config)
		allowed = limiter.AllowConnection()

		if !allowed {
			s.recordViolation(key, "CONNECTION", "connection limit exceeded")
			return fmt.Errorf("connection limit exceeded")
		}

		return nil
	})

	return allowed, err
}

func (s *rateLimitService) RemoveConnection(ctx context.Context, ip string) error {
	key := s.getKey(ip)

	return s.store.Update(ctx, key, func(state *domain.RateLimitState) error {
		limiter := NewRateLimiter(state, s.config)
		limiter.CloseConnection()
		return nil
	})
}

func (s *rateLimitService) GetStatus(ctx context.Context, ip string) (*domain.RateLimitStatus, error) {
	key := s.getKey(ip)

	state, err := s.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if state == nil {
		return nil, fmt.Errorf("rate limit state not found")
	}

	limiter := NewRateLimiter(state, s.config)
	status := limiter.GetStatus()

	return &status, nil
}

func (s *rateLimitService) GetMetrics(ctx context.Context) domain.RateLimitMetrics {
	keys, _ := s.store.Keys(ctx)

	metrics := domain.RateLimitMetrics{
		UniqueClients: int64(len(keys)),
	}

	for _, key := range keys {
		state, _ := s.store.Get(ctx, key)
		if state == nil {
			continue
		}

		metrics.TotalRequests += state.TotalRequests
		metrics.LimitedRequests += state.LimitedRequests
		metrics.ConnectionLimited += state.ActiveConnections
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	counts := map[string]int64{}
	last := map[string]time.Time{}

	for _, v := range s.violations {
		counts[v.Key]++
		if v.Timestamp.After(last[v.Key]) {
			last[v.Key] = v.Timestamp
		}
	}

	for k, c := range counts {
		metrics.TopViolators = append(metrics.TopViolators, domain.RateLimitViolator{
			Key:           k,
			Violations:    c,
			LastViolation: last[k],
		})
	}

	sort.Slice(metrics.TopViolators, func(i, j int) bool {
		return metrics.TopViolators[i].Violations > metrics.TopViolators[j].Violations
	})

	if len(keys) > 0 {
		metrics.AverageRequestsPerClient =
			float64(metrics.TotalRequests) / float64(len(keys))
	}

	return metrics
}

func (s *rateLimitService) ResetIP(ctx context.Context, ip string) error {
	return s.store.Delete(ctx, s.getKey(ip))
}

func (s *rateLimitService) GetTopViolators(ctx context.Context, limit int) []domain.RateLimitViolator {
	s.mu.Lock()
	defer s.mu.Unlock()

	counts := map[string]int64{}
	last := map[string]time.Time{}

	for _, v := range s.violations {
		counts[v.Key]++
		if v.Timestamp.After(last[v.Key]) {
			last[v.Key] = v.Timestamp
		}
	}

	result := make([]domain.RateLimitViolator, 0, len(counts))

	for k, c := range counts {
		result = append(result, domain.RateLimitViolator{
			Key:           k,
			Violations:    c,
			LastViolation: last[k],
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Violations > result[j].Violations
	})

	if limit > len(result) {
		limit = len(result)
	}

	return result[:limit]
}

func (s *rateLimitService) getKey(ip string) string {
	if s.config.SubnetMask == "" {
		return ip
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}

	return ip
}

func initState(state *domain.RateLimitState, key string, config domain.RateLimitConfig) {
	now := time.Now()

	state.Key = key

	state.RPSTokens = float64(config.RPS)
	state.RPMTokens = float64(config.RPM)
	state.RPHTokens = float64(config.RPH)
	state.RPDTokens = float64(config.RPD)

	state.DownloadTokens = float64(config.DownloadBytesPerSecond)
	state.UploadTokens = float64(config.UploadBytesPerSecond)
	state.TotalBytesTokens = float64(config.TotalBytesPerDay)

	state.NewConnectionsTokens = float64(config.NewConnectionsPerSecond)

	state.CreatedAt = now
	state.LastRefillAt = now
	state.LastSeenAt = now
}

func (s *rateLimitService) recordViolation(key, limitType, reason string) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.violations = append(s.violations, domain.RateLimitViolation{
		Key:       key,
		LimitType: limitType,
		Reason:    reason,
		Timestamp: now,
	})

	if s.monitoring != nil {
		s.monitoring.RecordRateLimitViolation(domain.RateLimitMetric{
			Key:       key,
			Reason:    reason,
			Timestamp: now,
		})
	}
}

func (s *rateLimitService) cleanupViolations() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-time.Hour)

		s.mu.Lock()

		newSlice := make([]domain.RateLimitViolation, 0, len(s.violations))
		for _, v := range s.violations {
			if v.Timestamp.After(cutoff) {
				newSlice = append(newSlice, v)
			}
		}

		s.violations = newSlice

		s.mu.Unlock()
	}
}
