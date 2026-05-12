package usecases

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"proxy/internal/domain"
	"proxy/internal/infrastructure/logger"
	"proxy/internal/repository"
)

type IPFilterService interface {
	CheckIP(ctx context.Context, ip string) (bool, error)
	ClearCache(ctx context.Context) error
	CheckIPBatch(ctx context.Context, ips []string) ([]domain.IPCheckResult, error)
	AddToAllowList(ctx context.Context, pattern string, comment string) (*domain.IPEntry, error)
	AddToDenyList(ctx context.Context, pattern string, comment string) (*domain.IPEntry, error)
	AddToGreyList(ctx context.Context, pattern string, comment string) (*domain.IPEntry, error)
	RemoveEntry(ctx context.Context, id string) error
	GetPolicy(ctx context.Context) (*domain.IPAccessPolicy, error)
	SetPolicy(ctx context.Context, policy *domain.IPAccessPolicy) error
	ReloadPolicy(ctx context.Context) error
	GetStats(ctx context.Context) (*domain.IPFilterStats, error)
}

// ipFilterService основной сервис IP фильтра
type ipFilterService struct {
	repo            repository.IPAccessRepository
	cache           domain.IPFilterCache
	parser          domain.IPParser
	matcher         domain.IPMatcher
	logger          logger.Logger
	config          domain.IPFilterConfig
	policy          *domain.IPAccessPolicy
	policyMu        sync.RWMutex
	stats           *ipFilterStats
}

type ipFilterStats struct {
	totalChecks      int64
	allowedRequests  int64
	deniedRequests   int64
	greyListRequests int64
	cacheHits        int64
	cacheMisses      int64
}

func NewipFilterService(repo repository.IPAccessRepository,	cache domain.IPFilterCache, parser domain.IPParser, matcher domain.IPMatcher, logger logger.Logger, config domain.IPFilterConfig,) *ipFilterService {
	return &ipFilterService{
		repo:    repo,
		cache:   cache,
		parser:  parser,
		matcher: matcher,
		logger:  logger,
		config:  config,
		stats:   &ipFilterStats{},
	}
}

// CheckIP проверяет IP адрес против политики доступа
func (s *ipFilterService) CheckIP(ctx context.Context, ip string) (*domain.IPCheckResult, error) {
	if !s.config.Enabled {
		return &domain.IPCheckResult{
			IP:        ip,
			IsAllowed: true,
			Reason:    "IP filter disabled",
			CheckedAt: time.Now(),
		}, nil
	}

	// Проверяем кеш
	if cached, ok := s.cache.Get(ip); ok {
		atomic.AddInt64(&s.stats.cacheHits, 1)
		s.logger.Debugf("Cache hit for IP: %s", ip)
		return cached, nil
	}

	atomic.AddInt64(&s.stats.cacheMisses, 1)

	// Парсим IP
	parsedIP, err := s.parser.Parse(ip)
	if err != nil {
		s.logger.Warnf("Invalid IP format: %s, error: %v", ip, err)
		return nil, &domain.IPFilterError{
			Code:    "INVALID_IP",
			Message: fmt.Sprintf("Invalid IP format: %s", ip),
			IP:      ip,
		}
	}

	// Получаем политику
	policy, err := s.getPolicy(ctx)
	if err != nil {
		s.logger.Errorf("Failed to get policy: %v", err)
		return nil, err
	}

	// Проверяем списки
	result := s.performCheck(parsedIP, policy)
	result.CheckedAt = time.Now()

	// Кешируем результат
	s.cache.Set(ip, result)

	// Обновляем статистику
	atomic.AddInt64(&s.stats.totalChecks, 1)
	if result.IsAllowed {
		atomic.AddInt64(&s.stats.allowedRequests, 1)
	} else {
		atomic.AddInt64(&s.stats.deniedRequests, 1)
	}

	if result.ListType == domain.GreyList {
		atomic.AddInt64(&s.stats.greyListRequests, 1)
	}

	if s.config.EnableLogging {
		s.logger.Infof("IP check: %s, allowed: %v, reason: %s", ip, result.IsAllowed, result.Reason)
	}

	return result, nil
}

// CheckIPBatch проверяет несколько IP адресов
func (s *ipFilterService) CheckIPBatch(ctx context.Context, ips []string) ([]domain.IPCheckResult, error) {
	results := make([]domain.IPCheckResult, 0, len(ips))
	
	for _, ip := range ips {
		result, err := s.CheckIP(ctx, ip)
		if err != nil {
			s.logger.Warnf("Failed to check IP %s: %v", ip, err)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// AddToAllowList добавляет IP в белый список
func (s *ipFilterService) AddToAllowList(ctx context.Context, pattern string, comment string) (*domain.IPEntry, error) {
	return s.addToList(ctx, domain.AllowList, pattern, comment)
}

// AddToDenyList добавляет IP в чёрный список
func (s *ipFilterService) AddToDenyList(ctx context.Context, pattern string, comment string) (*domain.IPEntry, error) {
	return s.addToList(ctx, domain.DenyList, pattern, comment)
}

// AddToGreyList добавляет IP в серый список
func (s *ipFilterService) AddToGreyList(ctx context.Context, pattern string, comment string) (*domain.IPEntry, error) {
	return s.addToList(ctx, domain.GreyList, pattern, comment)
}

func (s *ipFilterService) addToList(ctx context.Context, listType domain.IPListType, pattern string, comment string) (*domain.IPEntry, error) {
	// Валидируем паттерн
	if !s.parser.IsValidIP(pattern) && !s.parser.IsValidCIDR(pattern) {
		return nil, &domain.IPFilterError{
			Code:    "INVALID_PATTERN",
			Message: fmt.Sprintf("Invalid IP or CIDR pattern: %s", pattern),
		}
	}

	entry := &domain.IPEntry{
		ID:        fmt.Sprintf("%s-%d", listType, time.Now().UnixNano()),
		Type:      listType,
		Value:     pattern,
		Comment:   comment,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Сохраняем в репозиторий
	err := s.repo.AddEntry(ctx, entry)
	if err != nil {
		s.logger.Errorf("Failed to add entry to %s: %v", listType, err)
		return nil, err
	}

	// Очищаем кеш
	s.cache.Clear()

	s.logger.Infof("Added to %s: %s (comment: %s)", listType, pattern, comment)

	return entry, nil
}

// RemoveEntry удаляет запись из списка
func (s *ipFilterService) RemoveEntry(ctx context.Context, id string) error {
	err := s.repo.RemoveEntry(ctx, id)
	if err != nil {
		s.logger.Errorf("Failed to remove entry: %v", err)
		return err
	}

	// Очищаем кеш
	s.cache.Clear()

	s.logger.Infof("Removed entry: %s", id)

	return nil
}

// GetPolicy получает текущую политику
func (s *ipFilterService) GetPolicy(ctx context.Context) (*domain.IPAccessPolicy, error) {
	return s.getPolicy(ctx)
}

// SetPolicy устанавливает новую политику
func (s *ipFilterService) SetPolicy(ctx context.Context, policy *domain.IPAccessPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	err := s.repo.SavePolicy(ctx, policy)
	if err != nil {
		s.logger.Errorf("Failed to save policy: %v", err)
		return err
	}

	s.policyMu.Lock()
	s.policy = policy
	s.policyMu.Unlock()

	// Очищаем кеш
	s.cache.Clear()

	s.logger.Infof("Policy updated, version: %d", policy.Version)

	return nil
}

// ReloadPolicy перезагружает политику из репозитория
func (s *ipFilterService) ReloadPolicy(ctx context.Context) error {
	policy, err := s.repo.GetPolicy(ctx)
	if err != nil {
		s.logger.Errorf("Failed to reload policy: %v", err)
		return err
	}

	s.policyMu.Lock()
	s.policy = policy
	s.policyMu.Unlock()

	// Очищаем кеш
	s.cache.Clear()

	s.logger.Infof("Policy reloaded, version: %d", policy.Version)

	return nil
}

// GetStats получает статистику фильтра
func (s *ipFilterService) GetStats(ctx context.Context) (*domain.IPFilterStats, error) {
	policy, err := s.getPolicy(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.IPFilterStats{
		TotalChecks:      atomic.LoadInt64(&s.stats.totalChecks),
		AllowedRequests:  atomic.LoadInt64(&s.stats.allowedRequests),
		DeniedRequests:   atomic.LoadInt64(&s.stats.deniedRequests),
		GreyListRequests: atomic.LoadInt64(&s.stats.greyListRequests),
		CacheHits:        atomic.LoadInt64(&s.stats.cacheHits),
		CacheMisses:      atomic.LoadInt64(&s.stats.cacheMisses),
		AllowListSize:    len(policy.AllowList),
		DenyListSize:     len(policy.DenyList),
		GreyListSize:     len(policy.GreyList),
		LastUpdated:      policy.UpdatedAt,
	}, nil
}

// ClearCache очищает кеш
func (s *ipFilterService) ClearCache(ctx context.Context) error {
	s.cache.Clear()
	s.logger.Infof("Cache cleared")
	return nil
}

// Validate валидирует конфигурацию и политику
func (s *ipFilterService) Validate(ctx context.Context) error {
	policy, err := s.getPolicy(ctx)
	if err != nil {
		return fmt.Errorf("failed to get policy for validation: %w", err)
	}

	// Валидируем default policy
	if policy.DefaultPolicy != "allow" && policy.DefaultPolicy != "deny" {
		return fmt.Errorf("invalid default policy: %s", policy.DefaultPolicy)
	}

	// Валидируем все записи
	for _, entry := range append(append(policy.AllowList, policy.DenyList...), policy.GreyList...) {
		if !s.parser.IsValidIP(entry.Value) && !s.parser.IsValidCIDR(entry.Value) {
			return fmt.Errorf("invalid entry value: %s", entry.Value)
		}
	}

	s.logger.Infof("IP filter validation passed")

	return nil
}

// Приватные методы

func (s *ipFilterService) getPolicy(ctx context.Context) (*domain.IPAccessPolicy, error) {
	s.policyMu.RLock()
	if s.policy != nil {
		defer s.policyMu.RUnlock()
		return s.policy, nil
	}
	s.policyMu.RUnlock()

	policy, err := s.repo.GetPolicy(ctx)
	if err != nil {
		return nil, err
	}

	s.policyMu.Lock()
	s.policy = policy
	s.policyMu.Unlock()

	return policy, nil
}

func (s *ipFilterService) performCheck(ip net.IP, policy *domain.IPAccessPolicy) *domain.IPCheckResult {
	result := &domain.IPCheckResult{
		IP:        ip.String(),
		IsAllowed: policy.DefaultPolicy == "allow",
		Reason:    fmt.Sprintf("Default policy: %s", policy.DefaultPolicy),
	}

	// Проверяем чёрный список (имеет приоритет)
	if matched, entry := s.matcher.Match(ip, policy.DenyList); matched {
		result.IsAllowed = false
		result.Reason = "Matched deny list"
		result.ListType = domain.DenyList
		result.MatchedRule = entry.Value
		return result
	}

	// Проверяем белый список
	if matched, entry := s.matcher.Match(ip, policy.AllowList); matched {
		result.IsAllowed = true
		result.Reason = "Matched allow list"
		result.ListType = domain.AllowList
		result.MatchedRule = entry.Value
		return result
	}

	// Проверяем серый список (если включен)
	if s.config.EnableGreyList {
		if matched, entry := s.matcher.Match(ip, policy.GreyList); matched {
			result.IsAllowed = false
			result.Reason = "Matched grey list"
			result.ListType = domain.GreyList
			result.MatchedRule = entry.Value
			return result
		}
	}

	return result
}
