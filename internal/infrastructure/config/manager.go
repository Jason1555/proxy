package config

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"proxy/internal/domain"
	"proxy/internal/infrastructure/logger"
)

type Change struct {
	Previous *domain.AppConfig
	Current  domain.AppConfig
	Version  int64
	LoadedAt time.Time
}

type Subscriber func(Change)

type Manager struct {
	mu           sync.RWMutex
	path         string
	interval     time.Duration
	loader       Loader
	logger       logger.Logger
	current      domain.AppConfig
	loaded       bool
	version      int64
	loadedAt     time.Time
	lastFileInfo fileInfo
	nextSubID    int
	subscribers  map[int]Subscriber
}

type fileInfo struct {
	modTime time.Time
	size    int64
}

func NewManager(path string, interval time.Duration, loader Loader, logger logger.Logger) *Manager {
	if interval <= 0 {
		interval = time.Second
	}
	if loader == nil {
		loader = NewFileLoader()
	}

	return &Manager{
		path:        path,
		interval:    interval,
		loader:      loader,
		logger:      logger,
		subscribers: make(map[int]Subscriber),
	}
}

func (m *Manager) Load(ctx context.Context) error {
	cfg, info, err := m.loadFromDisk(ctx)
	if err != nil {
		return err
	}

	m.apply(*cfg, info)
	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	if err := m.Load(ctx); err != nil {
		return err
	}

	go m.watch(ctx)
	return nil
}

func (m *Manager) Current() (domain.AppConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.current, m.loaded
}

func (m *Manager) Version() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.version
}

func (m *Manager) LoadedAt() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.loadedAt
}

func (m *Manager) Subscribe(subscriber Subscriber) func() {
	if subscriber == nil {
		return func() {}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextSubID
	m.nextSubID++
	m.subscribers[id] = subscriber

	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		delete(m.subscribers, id)
	}
}

func (m *Manager) watch(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.reloadIfChanged(ctx); err != nil && m.logger != nil {
				m.logger.Warnf("config reload failed: %v", err)
			}
		}
	}
}

func (m *Manager) reloadIfChanged(ctx context.Context) error {
	info, err := statConfig(m.path)
	if err != nil {
		return err
	}

	m.mu.RLock()
	loaded := m.loaded
	last := m.lastFileInfo
	m.mu.RUnlock()

	if loaded && last == info {
		return nil
	}

	cfg, loadedInfo, err := m.loadFromDisk(ctx)
	if err != nil {
		return err
	}

	m.apply(*cfg, loadedInfo)
	return nil
}

func (m *Manager) loadFromDisk(ctx context.Context) (*domain.AppConfig, fileInfo, error) {
	if m.path == "" {
		return nil, fileInfo{}, fmt.Errorf("config path is empty")
	}

	info, err := statConfig(m.path)
	if err != nil {
		return nil, fileInfo{}, err
	}

	cfg, err := m.loader.Load(ctx, m.path)
	if err != nil {
		return nil, fileInfo{}, err
	}

	return cfg, info, nil
}

func (m *Manager) apply(cfg domain.AppConfig, info fileInfo) {
	now := time.Now()

	m.mu.Lock()
	var previous *domain.AppConfig
	if m.loaded {
		prev := m.current
		previous = &prev
	}

	m.current = cfg
	m.loaded = true
	m.version++
	m.loadedAt = now
	m.lastFileInfo = info

	change := Change{
		Previous: previous,
		Current:  cfg,
		Version:  m.version,
		LoadedAt: now,
	}

	subscribers := make([]Subscriber, 0, len(m.subscribers))
	for _, subscriber := range m.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	m.mu.Unlock()

	if m.logger != nil {
		m.logger.Infof("config loaded from %s, version: %d", m.path, change.Version)
	}

	for _, subscriber := range subscribers {
		subscriber(change)
	}
}

func statConfig(path string) (fileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return fileInfo{}, fmt.Errorf("stat config %s: %w", path, err)
	}

	return fileInfo{
		modTime: stat.ModTime(),
		size:    stat.Size(),
	}, nil
}
