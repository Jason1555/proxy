package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"proxy/internal/domain"
	"proxy/internal/testutils"
)

func basePolicy() *domain.IPAccessPolicy {
	return &domain.IPAccessPolicy{
		DefaultPolicy: "deny",
		AllowList: []domain.IPEntry{
			{
				Value: "10.0.0.1",
				Type:  domain.AllowList,
			},
		},
		DenyList: []domain.IPEntry{
			{
				Value: "192.168.1.1",
				Type:  domain.DenyList,
			},
		},
		GreyList: []domain.IPEntry{
			{
				Value: "172.16.0.1",
				Type:  domain.GreyList,
			},
		},
		UpdatedAt: time.Now(),
		Version:   1,
	}
}

func newService(
	repo *testutils.MockRepo,
	cache *testutils.MockCache,
	parser *testutils.MockParser,
	matcher *testutils.MockMatcher,
) *ipFilterService {
	return NewipFilterService(
		repo,
		cache,
		parser,
		matcher,
		&testutils.MockLogger{},
		domain.IPFilterConfig{
			Enabled:        true,
			EnableGreyList: true,
		},
		nil,
	)
}

func TestCheckIP_FilterDisabled(t *testing.T) {
	svc := newService(
		&testutils.MockRepo{},
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	svc.config.Enabled = false

	result, err := svc.CheckIP(
		context.Background(),
		"1.1.1.1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsAllowed {
		t.Fatal("expected allowed")
	}

	if result.Reason != "IP filter disabled" {
		t.Fatalf("unexpected reason: %s", result.Reason)
	}
}

func TestCheckIP_CacheHit(t *testing.T) {
	cache := testutils.NewMockCache()

	expected := &domain.IPCheckResult{
		IP:        "1.1.1.1",
		IsAllowed: true,
		Reason:    "cached",
	}

	cache.Set("1.1.1.1", expected)

	svc := newService(
		&testutils.MockRepo{},
		cache,
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	result, err := svc.CheckIP(
		context.Background(),
		"1.1.1.1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Fatal("expected cached result")
	}
}

func TestCheckIP_InvalidIP(t *testing.T) {
	svc := newService(
		&testutils.MockRepo{},
		testutils.NewMockCache(),
		&testutils.MockParser{
			ParseErr: errors.New("invalid ip"),
		},
		&testutils.MockMatcher{},
	)

	_, err := svc.CheckIP(
		context.Background(),
		"bad-ip",
	)

	if err == nil {
		t.Fatal("expected error")
	}

	var ipErr *domain.IPFilterError
	if !errors.As(err, &ipErr) {
		t.Fatalf(
			"expected IPFilterError got %T",
			err,
		)
	}

	if ipErr.Code != "INVALID_IP" {
		t.Fatalf(
			"unexpected error code: %s",
			ipErr.Code,
		)
	}
}

func TestCheckIP_GetPolicyError(t *testing.T) {
	svc := newService(
		&testutils.MockRepo{
			GetPolicyErr: errors.New("db error"),
		},
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	_, err := svc.CheckIP(
		context.Background(),
		"8.8.8.8",
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckIP_DefaultAllow(t *testing.T) {
	repo := &testutils.MockRepo{
		Policy: &domain.IPAccessPolicy{
			DefaultPolicy: "allow",
		},
	}

	svc := newService(
		repo,
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	result, err := svc.CheckIP(
		context.Background(),
		"8.8.8.8",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsAllowed {
		t.Fatal("expected allowed")
	}
}

func TestCheckIP_DefaultDeny(t *testing.T) {
	repo := &testutils.MockRepo{
		Policy: &domain.IPAccessPolicy{
			DefaultPolicy: "deny",
		},
	}

	svc := newService(
		repo,
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	result, err := svc.CheckIP(
		context.Background(),
		"8.8.8.8",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsAllowed {
		t.Fatal("expected denied")
	}
}

func TestCheckIPBatch(t *testing.T) {
	repo := &testutils.MockRepo{
		Policy: basePolicy(),
	}

	svc := newService(
		repo,
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	results, err := svc.CheckIPBatch(
		context.Background(),
		[]string{
			"1.1.1.1",
			"2.2.2.2",
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf(
			"expected 2 results got %d",
			len(results),
		)
	}
}

func TestAddToAllowList(t *testing.T) {
	repo := &testutils.MockRepo{}
	cache := testutils.NewMockCache()

	svc := newService(
		repo,
		cache,
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	entry, err := svc.AddToAllowList(
		context.Background(),
		"1.1.1.1",
		"test",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.Type != domain.AllowList {
		t.Fatal("wrong type")
	}

	if len(repo.AddedEntries) != 1 {
		t.Fatal("entry not added")
	}

	if !cache.ClearCalled {
		t.Fatal("cache not cleared")
	}
}

func TestAddToList_InvalidPattern(t *testing.T) {
	svc := newService(
		&testutils.MockRepo{},
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	_, err := svc.AddToAllowList(
		context.Background(),
		"invalid-ip",
		"comment",
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddToList_RepoError(t *testing.T) {
	svc := newService(
		&testutils.MockRepo{
			AddEntryErr: errors.New("repo error"),
		},
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	_, err := svc.AddToAllowList(
		context.Background(),
		"1.1.1.1",
		"test",
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRemoveEntry(t *testing.T) {
	repo := &testutils.MockRepo{}
	cache := testutils.NewMockCache()

	svc := newService(
		repo,
		cache,
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	err := svc.RemoveEntry(
		context.Background(),
		"id-1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.RemovedIDs) != 1 {
		t.Fatal("entry not removed")
	}

	if !cache.ClearCalled {
		t.Fatal("cache not cleared")
	}
}

func TestSetPolicy(t *testing.T) {
	repo := &testutils.MockRepo{}
	cache := testutils.NewMockCache()

	svc := newService(
		repo,
		cache,
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	err := svc.SetPolicy(
		context.Background(),
		basePolicy(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svc.policy == nil {
		t.Fatal("policy not set")
	}

	if !cache.ClearCalled {
		t.Fatal("cache not cleared")
	}
}

func TestReloadPolicy(t *testing.T) {
	repo := &testutils.MockRepo{
		Policy: basePolicy(),
	}

	cache := testutils.NewMockCache()

	svc := newService(
		repo,
		cache,
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	err := svc.ReloadPolicy(
		context.Background(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svc.policy == nil {
		t.Fatal("policy nil")
	}
}

func TestGetStats(t *testing.T) {
	repo := &testutils.MockRepo{
		Policy: basePolicy(),
	}

	svc := newService(
		repo,
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	_, _ = svc.CheckIP(
		context.Background(),
		"1.1.1.1",
	)

	stats, err := svc.GetStats(
		context.Background(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.TotalChecks != 1 {
		t.Fatalf(
			"expected 1 got %d",
			stats.TotalChecks,
		)
	}
}

func TestValidate(t *testing.T) {
	repo := &testutils.MockRepo{
		Policy: basePolicy(),
	}

	svc := newService(
		repo,
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	err := svc.Validate(
		context.Background(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidDefaultPolicy(t *testing.T) {
	repo := &testutils.MockRepo{
		Policy: &domain.IPAccessPolicy{
			DefaultPolicy: "invalid",
		},
	}

	svc := newService(
		repo,
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	err := svc.Validate(
		context.Background(),
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClearCache(t *testing.T) {
	cache := testutils.NewMockCache()

	svc := newService(
		&testutils.MockRepo{},
		cache,
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	err := svc.ClearCache(
		context.Background(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cache.ClearCalled {
		t.Fatal("cache not cleared")
	}
}

func TestGetPolicy_Cached(t *testing.T) {
	repo := &testutils.MockRepo{
		Policy: basePolicy(),
	}

	svc := newService(
		repo,
		testutils.NewMockCache(),
		&testutils.MockParser{},
		&testutils.MockMatcher{},
	)

	_, _ = svc.getPolicy(
		context.Background(),
	)

	_, _ = svc.getPolicy(
		context.Background(),
	)

	if repo.GetPolicyCalls != 1 {
		t.Fatalf(
			"expected 1 repo call got %d",
			repo.GetPolicyCalls,
		)
	}
}
