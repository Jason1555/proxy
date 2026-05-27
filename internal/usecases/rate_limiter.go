package usecases

import (
	"proxy/internal/domain"
	"sync"
	"time"
)

type RateLimiter struct {
	mu     sync.Mutex
	state  *domain.RateLimitState
	config domain.RateLimitConfig
}

func NewRateLimiter(state *domain.RateLimitState, config domain.RateLimitConfig) *RateLimiter {
	now := time.Now()

	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}

	if state.LastRefillAt.IsZero() {
		state.LastRefillAt = now
	}

	return &RateLimiter{
		state:  state,
		config: config,
	}
}

type Decision struct {
	Allowed    bool
	Reason     string
	FailedOn   string
	RetryAfter time.Duration
}

func (rl *RateLimiter) Evaluate(bodySize int64) Decision {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.config.Enabled {
		return Decision{Allowed: true}
	}

	now := time.Now()
	rl.refillAll(now)

	rl.state.LastSeenAt = now
	rl.state.TotalRequests++

	// 1. RPS
	if !consume(&rl.state.RPSTokens, 1) {
		return rl.reject("RPS limit exceeded", "RPS")
	}

	// 2. RPM
	if !consume(&rl.state.RPMTokens, 1) {
		return rl.reject("RPM limit exceeded", "RPM")
	}

	// 3. RPH
	if !consume(&rl.state.RPHTokens, 1) {
		return rl.reject("RPH limit exceeded", "RPH")
	}

	// 4. RPD
	if !consume(&rl.state.RPDTokens, 1) {
		return rl.reject("RPD limit exceeded", "RPD")
	}

	// 5. upload bandwidth
	if !consume(&rl.state.UploadTokens, float64(bodySize)) {
		return rl.reject("upload bandwidth exceeded", "UPLOAD")
	}

	// 6. total bandwidth
	if !consume(&rl.state.TotalBytesTokens, float64(bodySize)) {
		return rl.reject("total bandwidth exceeded", "TOTAL")
	}

	return Decision{Allowed: true}
}

func (rl *RateLimiter) AllowConnection() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.refillAll(now)

	if !consume(&rl.state.NewConnectionsTokens, 1) {
		rl.state.LimitedRequests++
		return false
	}

	if rl.config.MaxConcurrentConnections > 0 &&
		rl.state.ActiveConnections >= rl.config.MaxConcurrentConnections {
		rl.state.LimitedRequests++
		return false
	}

	rl.state.ActiveConnections++
	return true
}

func (rl *RateLimiter) CloseConnection() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.state.ActiveConnections > 0 {
		rl.state.ActiveConnections--
	}
}

func (rl *RateLimiter) GetStatus() domain.RateLimitStatus {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return domain.RateLimitStatus{
		Key: rl.state.Key,

		RPSRemaining: int64(rl.state.RPSTokens),
		RPMRemaining: int64(rl.state.RPMTokens),
		RPHRemaining: int64(rl.state.RPHTokens),
		RPDRemaining: int64(rl.state.RPDTokens),

		DownloadRemaining:   int64(rl.state.DownloadTokens),
		UploadRemaining:     int64(rl.state.UploadTokens),
		TotalBytesRemaining: int64(rl.state.TotalBytesTokens),

		NewConnectionsRemaining: int64(rl.state.NewConnectionsTokens),

		ActiveConnections: rl.state.ActiveConnections,
	}
}

func (rl *RateLimiter) reject(reason, failedOn string) Decision {
	rl.state.LimitedRequests++

	return Decision{
		Allowed:  false,
		Reason:   reason,
		FailedOn: failedOn,
	}
}

func (rl *RateLimiter) refillAll(now time.Time) {
	elapsed := now.Sub(rl.state.LastRefillAt).Seconds()
	if elapsed <= 0 {
		return
	}

	refill(&rl.state.RPSTokens, float64(rl.config.RPS), float64(rl.config.RPS), elapsed)
	refill(&rl.state.RPMTokens, float64(rl.config.RPM), float64(rl.config.RPM)/60, elapsed)
	refill(&rl.state.RPHTokens, float64(rl.config.RPH), float64(rl.config.RPH)/3600, elapsed)
	refill(&rl.state.RPDTokens, float64(rl.config.RPD), float64(rl.config.RPD)/86400, elapsed)
	refill(&rl.state.UploadTokens, float64(rl.config.UploadBytesPerSecond), float64(rl.config.UploadBytesPerSecond), elapsed)
	refill(&rl.state.TotalBytesTokens, float64(rl.config.TotalBytesPerDay), float64(rl.config.TotalBytesPerDay)/86400, elapsed)

	refill(&rl.state.NewConnectionsTokens, float64(rl.config.NewConnectionsPerSecond), float64(rl.config.NewConnectionsPerSecond), elapsed)

	rl.state.LastRefillAt = now
}

func refill(tokens *float64, capacity float64, rate float64, elapsed float64) {
	*tokens += elapsed * rate
	if *tokens > capacity {
		*tokens = capacity
	}
}

func consume(tokens *float64, cost float64) bool {
	if *tokens < cost {
		return false
	}
	*tokens -= cost
	return true
}
