package usecases

import (
	"context"
	"testing"
	"time"

	"proxy/internal/domain"
	"proxy/internal/testutils"
)

func serviceConfig() domain.RateLimitConfig {
	return domain.RateLimitConfig{
		Enabled:                  true,
		RPS:                      5,
		RPM:                      50,
		RPH:                      500,
		RPD:                      5000,
		UploadBytesPerSecond:     1024,
		DownloadBytesPerSecond:   1024,
		TotalBytesPerDay:         10_000,
		NewConnectionsPerSecond:  2,
		MaxConcurrentConnections: 2,
	}
}

func newRateLimitService(
	store *testutils.MockRateLimitStore,
) *rateLimitService {
	svc := NewRateLimitService(
		store,
		serviceConfig(),
		&testutils.MockLogger{},
		nil,
	)

	return svc.(*rateLimitService)
}

func TestEvaluateRequest_Disabled(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	cfg := serviceConfig()
	cfg.Enabled = false

	svc := NewRateLimitService(
		store,
		cfg,
		&testutils.MockLogger{},
		nil,
	)

	decision, err := svc.EvaluateRequest(
		context.Background(),
		"1.1.1.1",
		100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !decision.Allowed {
		t.Fatal("expected allowed")
	}
}

func TestEvaluateRequest_Success(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	svc := newRateLimitService(store)

	decision, err := svc.EvaluateRequest(
		context.Background(),
		"1.1.1.1",
		100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !decision.Allowed {
		t.Fatal("expected allowed")
	}

	state, err := store.Get(
		context.Background(),
		"1.1.1.1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state == nil {
		t.Fatal("state not initialized")
	}

	if state.TotalRequests != 1 {
		t.Fatal("request counter not incremented")
	}
}

func TestEvaluateRequest_RateLimited(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	store.Store["1.1.1.1"] =
		&domain.RateLimitState{
			Key:       "1.1.1.1",
			RPSTokens: 0,
		}

	svc := newRateLimitService(store)

	decision, err := svc.EvaluateRequest(
		context.Background(),
		"1.1.1.1",
		100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Allowed {
		t.Fatal("expected deny")
	}

	if len(svc.violations) != 1 {
		t.Fatal("violation not recorded")
	}
}

func TestAddConnection_Success(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	svc := newRateLimitService(store)

	ok, err := svc.AddConnection(
		context.Background(),
		"2.2.2.2",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ok {
		t.Fatal("expected allowed")
	}

	state, _ := store.Get(
		context.Background(),
		"2.2.2.2",
	)

	if state.ActiveConnections != 1 {
		t.Fatal("connection not added")
	}
}

func TestAddConnection_LimitExceeded(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	store.Store["1.1.1.1"] =
		&domain.RateLimitState{
			Key:                  "1.1.1.1",
			NewConnectionsTokens: 1,
			ActiveConnections:    2,
		}

	svc := newRateLimitService(store)

	ok, err := svc.AddConnection(
		context.Background(),
		"1.1.1.1",
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if ok {
		t.Fatal("expected denied")
	}

	if len(svc.violations) != 1 {
		t.Fatal("violation not recorded")
	}
}

func TestRemoveConnection(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	store.Store["1.1.1.1"] =
		&domain.RateLimitState{
			Key:               "1.1.1.1",
			ActiveConnections: 2,
		}

	svc := newRateLimitService(store)

	err := svc.RemoveConnection(
		context.Background(),
		"1.1.1.1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, _ := store.Get(
		context.Background(),
		"1.1.1.1",
	)

	if state.ActiveConnections != 1 {
		t.Fatal("connection not removed")
	}
}

func TestGetStatus(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	store.Store["1.1.1.1"] =
		&domain.RateLimitState{
			Key:       "1.1.1.1",
			RPSTokens: 5,
		}

	svc := newRateLimitService(store)

	status, err := svc.GetStatus(
		context.Background(),
		"1.1.1.1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Key != "1.1.1.1" {
		t.Fatal("wrong key")
	}
}

func TestGetStatus_NotFound(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	svc := newRateLimitService(store)

	_, err := svc.GetStatus(
		context.Background(),
		"1.1.1.1",
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetMetrics(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	store.Store["ip1"] =
		&domain.RateLimitState{
			TotalRequests:   10,
			LimitedRequests: 2,
		}

	store.Store["ip2"] =
		&domain.RateLimitState{
			TotalRequests:   20,
			LimitedRequests: 3,
		}

	svc := newRateLimitService(store)

	svc.recordViolation(
		"ip1",
		"RPS",
		"limit",
	)

	metrics := svc.GetMetrics(
		context.Background(),
	)

	if metrics.UniqueClients != 2 {
		t.Fatal("wrong unique clients")
	}

	if metrics.TotalRequests != 30 {
		t.Fatal("wrong total requests")
	}

	if metrics.LimitedRequests != 5 {
		t.Fatal("wrong limited requests")
	}
}

func TestResetIP(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	store.Store["1.1.1.1"] =
		&domain.RateLimitState{}

	svc := newRateLimitService(store)

	err := svc.ResetIP(
		context.Background(),
		"1.1.1.1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, exists := store.Store["1.1.1.1"]
	if exists {
		t.Fatal("ip not deleted")
	}
}

func TestGetTopViolators(
	t *testing.T,
) {
	store := testutils.NewRateLimitStore()

	svc := newRateLimitService(store)

	svc.recordViolation(
		"ip1",
		"RPS",
		"limit",
	)

	time.Sleep(time.Millisecond)

	svc.recordViolation(
		"ip1",
		"RPM",
		"limit",
	)

	svc.recordViolation(
		"ip2",
		"RPS",
		"limit",
	)

	result :=
		svc.GetTopViolators(
			context.Background(),
			1,
		)

	if len(result) != 1 {
		t.Fatal("wrong limit")
	}

	if result[0].Key != "ip1" {
		t.Fatal("wrong top violator")
	}

	if result[0].Violations != 2 {
		t.Fatal("wrong violation count")
	}
}

func TestGetKey(t *testing.T) {
	store := testutils.NewRateLimitStore()

	svc := newRateLimitService(store)

	key := svc.getKey("8.8.8.8")

	if key != "8.8.8.8" {
		t.Fatal("wrong key")
	}
}

func TestInitState(
	t *testing.T,
) {
	state := &domain.RateLimitState{}

	initState(
		state,
		"ip-1",
		serviceConfig(),
	)

	if state.Key != "ip-1" {
		t.Fatal("wrong key")
	}

	if state.RPSTokens == 0 {
		t.Fatal("tokens not initialized")
	}

	if state.CreatedAt.IsZero() {
		t.Fatal("created at empty")
	}
}
