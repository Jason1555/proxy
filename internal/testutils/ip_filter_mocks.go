package testutils

import (
	"context"
	"net"
	"sync"

	"proxy/internal/domain"
)

type MockRepo struct {
	Policy *domain.IPAccessPolicy

	GetPolicyErr  error
	SavePolicyErr error
	AddEntryErr   error
	RemoveErr     error
	GetEntryErr   error
	GetAllErr     error
	ClearListErr  error

	AddedEntries []*domain.IPEntry
	RemovedIDs   []string

	GetPolicyCalls int
	SaveCalls      int
	AddCalls       int
	RemoveCalls    int
}

func (m *MockRepo) GetPolicy(
	ctx context.Context,
) (*domain.IPAccessPolicy, error) {
	m.GetPolicyCalls++

	if m.GetPolicyErr != nil {
		return nil, m.GetPolicyErr
	}

	return m.Policy, nil
}

func (m *MockRepo) SavePolicy(
	ctx context.Context,
	policy *domain.IPAccessPolicy,
) error {
	m.SaveCalls++

	if m.SavePolicyErr != nil {
		return m.SavePolicyErr
	}

	m.Policy = policy
	return nil
}

func (m *MockRepo) AddEntry(
	ctx context.Context,
	entry *domain.IPEntry,
) error {
	m.AddCalls++

	if m.AddEntryErr != nil {
		return m.AddEntryErr
	}

	m.AddedEntries = append(
		m.AddedEntries,
		entry,
	)

	return nil
}

func (m *MockRepo) RemoveEntry(
	ctx context.Context,
	id string,
) error {
	m.RemoveCalls++

	if m.RemoveErr != nil {
		return m.RemoveErr
	}

	m.RemovedIDs = append(
		m.RemovedIDs,
		id,
	)

	return nil
}

func (m *MockRepo) GetEntry(
	ctx context.Context,
	id string,
) (*domain.IPEntry, error) {
	if m.GetEntryErr != nil {
		return nil, m.GetEntryErr
	}

	return nil, nil
}

func (m *MockRepo) GetAllEntries(
	ctx context.Context,
	listType domain.IPListType,
) ([]domain.IPEntry, error) {
	if m.GetAllErr != nil {
		return nil, m.GetAllErr
	}

	return nil, nil
}

func (m *MockRepo) ClearList(
	ctx context.Context,
	listType domain.IPListType,
) error {
	return m.ClearListErr
}

type MockCache struct {
	Data map[string]*domain.IPCheckResult

	ClearCalled bool

	GetCalls int
	SetCalls int

	Mutex sync.RWMutex
}

func NewMockCache() *MockCache {
	return &MockCache{
		Data: make(
			map[string]*domain.IPCheckResult,
		),
	}
}

func (m *MockCache) Get(
	ip string,
) (*domain.IPCheckResult, bool) {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	m.GetCalls++

	result, ok := m.Data[ip]

	return result, ok
}

func (m *MockCache) Set(
	ip string,
	result *domain.IPCheckResult,
) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	m.SetCalls++

	if m.Data == nil {
		m.Data = make(
			map[string]*domain.IPCheckResult,
		)
	}

	m.Data[ip] = result
}

func (m *MockCache) Clear() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	m.ClearCalled = true
	m.Data = make(
		map[string]*domain.IPCheckResult,
	)
}

func (m *MockCache) GetStats() map[string]any {
	return map[string]any{
		"entries": len(m.Data),
		"gets":    m.GetCalls,
		"sets":    m.SetCalls,
	}
}

type MockParser struct {
	ParseErr error

	ValidIP   bool
	ValidCIDR bool
}

func (m *MockParser) Parse(
	ip string,
) (net.IP, error) {
	if m.ParseErr != nil {
		return nil, m.ParseErr
	}

	return net.ParseIP(ip), nil
}

func (m *MockParser) ParseCIDR(
	cidr string,
) (*net.IPNet, error) {
	_, network, err := net.ParseCIDR(cidr)

	return network, err
}

func (m *MockParser) IsValidIP(
	ip string,
) bool {
	if m.ValidIP {
		return true
	}

	return net.ParseIP(ip) != nil
}

func (m *MockParser) IsValidCIDR(
	cidr string,
) bool {
	if m.ValidCIDR {
		return true
	}

	_, _, err := net.ParseCIDR(cidr)

	return err == nil
}

type MockMatcher struct {
	MatchResult  bool
	MatchedEntry *domain.IPEntry
}

func (m *MockMatcher) Match(
	ip net.IP,
	entries []domain.IPEntry,
) (bool, *domain.IPEntry) {
	if m.MatchResult {
		return true, m.MatchedEntry
	}

	return false, nil
}

func (m *MockMatcher) MatchCIDR(
	ip net.IP,
	cidr *net.IPNet,
) bool {
	return false
}

func (m *MockMatcher) MatchRange(
	ip net.IP,
	start,
	end net.IP,
) bool {
	return false
}
