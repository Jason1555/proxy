package usecases

import (
	"testing"
	"time"

	"proxy/internal/domain"
)

func testConfig() domain.RateLimitConfig {
	return domain.RateLimitConfig{
		Enabled:                  true,
		RPS:                      10,
		RPM:                      100,
		RPH:                      1000,
		RPD:                      10000,
		UploadBytesPerSecond:     1024,
		TotalBytesPerDay:         10000,
		NewConnectionsPerSecond:  5,
		MaxConcurrentConnections: 2,
	}
}

func testState() *domain.RateLimitState {
	now := time.Now()

	return &domain.RateLimitState{
		Key: "user-1",

		RPSTokens: 10,
		RPMTokens: 100,
		RPHTokens: 1000,
		RPDTokens: 10000,

		UploadTokens:     1024,
		DownloadTokens:   1024,
		TotalBytesTokens: 10000,

		NewConnectionsTokens: 5,

		CreatedAt:    now,
		LastRefillAt: now,
	}
}

func TestNewRateLimiter_InitializesTimestamps(
	t *testing.T,
) {
	state := &domain.RateLimitState{}

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	if rl == nil {
		t.Fatal("expected limiter")
	}

	if state.CreatedAt.IsZero() {
		t.Fatal("CreatedAt not initialized")
	}

	if state.LastRefillAt.IsZero() {
		t.Fatal("LastRefillAt not initialized")
	}
}

func TestEvaluate_Disabled(t *testing.T) {
	cfg := testConfig()
	cfg.Enabled = false

	rl := NewRateLimiter(
		testState(),
		cfg,
	)

	decision := rl.Evaluate(100)

	if !decision.Allowed {
		t.Fatal("expected allowed")
	}
}

func TestEvaluate_Success(t *testing.T) {
	state := testState()

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	decision := rl.Evaluate(100)

	if !decision.Allowed {
		t.Fatalf(
			"expected allowed got %+v",
			decision,
		)
	}

	if state.TotalRequests != 1 {
		t.Fatal("request counter not incremented")
	}

	if state.RPSTokens >= 10 {
		t.Fatal("tokens were not consumed")
	}
}

func TestEvaluate_RPSExceeded(t *testing.T) {
	state := testState()
	state.RPSTokens = 0

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	decision := rl.Evaluate(10)

	if decision.Allowed {
		t.Fatal("expected reject")
	}

	if decision.FailedOn != "RPS" {
		t.Fatalf(
			"expected RPS got %s",
			decision.FailedOn,
		)
	}
}

func TestEvaluate_RPMExceeded(t *testing.T) {
	state := testState()
	state.RPMTokens = 0

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	decision := rl.Evaluate(10)

	if decision.Allowed {
		t.Fatal("expected reject")
	}

	if decision.FailedOn != "RPM" {
		t.Fatal("expected RPM failure")
	}
}

func TestEvaluate_RPHExceeded(t *testing.T) {
	state := testState()
	state.RPHTokens = 0

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	decision := rl.Evaluate(10)

	if decision.Allowed {
		t.Fatal("expected reject")
	}

	if decision.FailedOn != "RPH" {
		t.Fatal("expected RPH failure")
	}
}

func TestEvaluate_RPDExceeded(t *testing.T) {
	state := testState()
	state.RPDTokens = 0

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	decision := rl.Evaluate(10)

	if decision.Allowed {
		t.Fatal("expected reject")
	}

	if decision.FailedOn != "RPD" {
		t.Fatal("expected RPD failure")
	}
}

func TestEvaluate_UploadExceeded(t *testing.T) {
	state := testState()
	state.UploadTokens = 0

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	decision := rl.Evaluate(100)

	if decision.Allowed {
		t.Fatal("expected reject")
	}

	if decision.FailedOn != "UPLOAD" {
		t.Fatal("expected upload failure")
	}
}

func TestEvaluate_TotalExceeded(t *testing.T) {
	state := testState()
	state.TotalBytesTokens = 0

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	decision := rl.Evaluate(100)

	if decision.Allowed {
		t.Fatal("expected reject")
	}

	if decision.FailedOn != "TOTAL" {
		t.Fatal("expected total failure")
	}
}

func TestAllowConnection_Success(t *testing.T) {
	state := testState()

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	ok := rl.AllowConnection()

	if !ok {
		t.Fatal("expected allowed")
	}

	if state.ActiveConnections != 1 {
		t.Fatal("connection counter wrong")
	}
}

func TestAllowConnection_TokenExceeded(
	t *testing.T,
) {
	state := testState()
	state.NewConnectionsTokens = 0

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	ok := rl.AllowConnection()

	if ok {
		t.Fatal("expected deny")
	}

	if state.LimitedRequests != 1 {
		t.Fatal("limited counter not incremented")
	}
}

func TestAllowConnection_MaxConcurrent(
	t *testing.T,
) {
	state := testState()
	state.ActiveConnections = 2

	cfg := testConfig()
	cfg.MaxConcurrentConnections = 2

	rl := NewRateLimiter(
		state,
		cfg,
	)

	ok := rl.AllowConnection()

	if ok {
		t.Fatal("expected deny")
	}
}

func TestCloseConnection(t *testing.T) {
	state := testState()
	state.ActiveConnections = 2

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	rl.CloseConnection()

	if state.ActiveConnections != 1 {
		t.Fatal("expected decrement")
	}
}

func TestCloseConnection_NoNegative(
	t *testing.T,
) {
	state := testState()

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	rl.CloseConnection()

	if state.ActiveConnections < 0 {
		t.Fatal("negative connections")
	}
}

func TestRefill(t *testing.T) {
	tokens := 0.0

	refill(
		&tokens,
		10,
		5,
		2,
	)

	if tokens != 10 {
		t.Fatalf(
			"expected 10 got %f",
			tokens,
		)
	}
}

func TestConsume_Success(t *testing.T) {
	tokens := 10.0

	ok := consume(&tokens, 5)

	if !ok {
		t.Fatal("expected consume")
	}

	if tokens != 5 {
		t.Fatal("wrong balance")
	}
}

func TestConsume_Fail(t *testing.T) {
	tokens := 1.0

	ok := consume(&tokens, 5)

	if ok {
		t.Fatal("expected fail")
	}
}

func TestRefillAll(t *testing.T) {
	state := testState()

	state.RPSTokens = 0
	state.LastRefillAt =
		time.Now().Add(-2 * time.Second)

	rl := NewRateLimiter(
		state,
		testConfig(),
	)

	rl.refillAll(time.Now())

	if state.RPSTokens <= 0 {
		t.Fatal("tokens not refilled")
	}
}
