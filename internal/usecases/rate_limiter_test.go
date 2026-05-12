package usecases

import (
	"proxy/internal/domain"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_InitializesTime(t *testing.T) {
	state := &domain.RateLimitState{
		Key: "test-key",
	}

	config := domain.RateLimitConfig{
		RPS: 10,
		RPM: 600,
	}

	limiter := NewRateLimiter(state, config)

	assert.NotNil(t, limiter)
	assert.False(t, state.CreatedAt.IsZero())
	assert.False(t, state.LastRefillAt.IsZero())
}

