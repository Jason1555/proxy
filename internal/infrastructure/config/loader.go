package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"proxy/internal/domain"
)

type Loader interface {
	Load(ctx context.Context, path string) (*domain.AppConfig, error)
}

type FileLoader struct{}

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (l *FileLoader) Load(ctx context.Context, path string) (*domain.AppConfig, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg domain.AppConfig
	if err := decodeConfig(path, data, &cfg); err != nil {
		return nil, err
	}
	if err := applyEnvOverrides(&cfg); err != nil {
		return nil, err
	}
	if err := cfg.Normalize(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func decodeConfig(path string, data []byte, cfg *domain.AppConfig) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("parse yaml config %s: %w", path, err)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("parse json config %s: %w", path, err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("parse toml config %s: %w", path, err)
		}
	default:
		return fmt.Errorf("unsupported config extension %q", filepath.Ext(path))
	}

	return nil
}
