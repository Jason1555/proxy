package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLoader_LoadYAML(t *testing.T) {
	isolateConfigEnv(t)

	path := writeConfigFile(t, "config.yaml", `
app:
  name: EdgeProxy
  version: "1.2"
  reload-timer: 7
log:
  level: debug
server:
  address: ":9090"
proxy:
  base-url: http://upstream.local
  default-allow: true
  allow-file: configs/allow.json
  deny-file: configs/deny.json
ip_access:
  enabled: true
  default_policy: deny
  cache_ttl: 2m
  allow_list:
    - value: 10.0.0.0/8
      comment: internal
rate_limit:
  enabled: true
  rps: 25
cache:
  enabled: true
  default_ttl: 30s
monitoring:
  request_window: 15s
  track_path_stats: true
`)

	cfg, err := NewFileLoader().Load(context.Background(), path)

	require.NoError(t, err)
	assert.Equal(t, "EdgeProxy", cfg.App.Name)
	assert.Equal(t, "1.2", cfg.App.Version)
	assert.Equal(t, 7, cfg.App.ReloadTimerSeconds)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, ":9090", cfg.Server.Address)
	assert.Equal(t, "http://upstream.local", cfg.Proxy.BaseURL)
	assert.True(t, cfg.Proxy.DefaultAllow)
	assert.Equal(t, "configs/allow.json", cfg.Proxy.AllowFile)
	assert.True(t, cfg.IPAccess.Enabled)
	assert.Equal(t, "deny", cfg.IPAccess.DefaultPolicy)
	assert.Equal(t, 2*time.Minute, cfg.IPAccess.CacheTTL.Duration())
	assert.Equal(t, int64(25), cfg.RateLimit.RPS)
	assert.Equal(t, 30*time.Second, cfg.Cache.DefaultTTL.Duration())
	assert.Equal(t, 15*time.Second, cfg.Monitoring.RequestWindow.Duration())
	assert.Equal(t, "allow-1", cfg.IPAccess.AllowList[0].ID)
}

func TestFileLoader_LoadJSON(t *testing.T) {
	isolateConfigEnv(t)

	path := writeConfigFile(t, "config.json", `{
  "app": {"name": "JsonProxy", "version": "2.0", "reload_timer_seconds": 8},
  "log": {"level": "warn"},
  "server": {"address": ":8081"},
  "proxy": {"base_url": "http://json-upstream", "allow_file": "allow.json", "deny_file": "deny.json"},
  "ip_access": {
    "enabled": true,
    "default_policy": "allow",
    "cache_ttl": "1m",
    "deny_list": [{"id": "bad", "value": "203.0.113.10"}]
  },
  "rate_limit": {"enabled": true, "rps": 5},
  "cache": {"enabled": true, "default_ttl": "20s"},
  "monitoring": {"request_window": "10s", "track_path_stats": true}
}`)

	cfg, err := NewFileLoader().Load(context.Background(), path)

	require.NoError(t, err)
	assert.Equal(t, "JsonProxy", cfg.App.Name)
	assert.Equal(t, "warn", cfg.Log.Level)
	assert.Equal(t, 8*time.Second, cfg.ReloadInterval())
	assert.Equal(t, ":8081", cfg.Server.Address)
	assert.Equal(t, "http://json-upstream", cfg.Proxy.BaseURL)
	assert.Equal(t, "bad", cfg.IPAccess.DenyList[0].ID)
	assert.Equal(t, time.Minute, cfg.IPAccess.CacheTTL.Duration())
	assert.Equal(t, int64(5), cfg.ToRateLimitConfig().RPS)
	assert.Equal(t, 20*time.Second, cfg.ToCacheConfig().DefaultTTL)
}

func TestFileLoader_LoadTOML(t *testing.T) {
	isolateConfigEnv(t)

	path := writeConfigFile(t, "config.toml", `
[app]
name = "TomlProxy"
version = "3.0"
reload_timer_seconds = 9

[log]
level = "error"

[server]
address = ":8082"

[proxy]
base_url = "http://toml-upstream"
allow_file = "allow.toml.json"
deny_file = "deny.toml.json"

[ip_access]
enabled = true
default_policy = "deny"
cache_ttl = "45s"

[[ip_access.allow_list]]
value = "192.168.0.0/16"
comment = "lan"

[rate_limit]
enabled = true
rps = 11

[cache]
enabled = true
default_ttl = "25s"

[monitoring]
request_window = "5s"
track_path_stats = true
`)

	cfg, err := NewFileLoader().Load(context.Background(), path)

	require.NoError(t, err)
	assert.Equal(t, "TomlProxy", cfg.App.Name)
	assert.Equal(t, "error", cfg.Log.Level)
	assert.Equal(t, 9*time.Second, cfg.ReloadInterval())
	assert.Equal(t, ":8082", cfg.Server.Address)
	assert.Equal(t, "http://toml-upstream", cfg.Proxy.BaseURL)
	assert.Equal(t, 45*time.Second, cfg.IPAccess.CacheTTL.Duration())
	assert.Equal(t, "192.168.0.0/16", cfg.IPAccess.AllowList[0].Value)
	assert.Equal(t, int64(11), cfg.RateLimit.RPS)
	assert.Equal(t, 5*time.Second, cfg.Monitoring.RequestWindow.Duration())
}

func TestFileLoader_UnsupportedExtension(t *testing.T) {
	isolateConfigEnv(t)

	path := writeConfigFile(t, "config.txt", `server: {address: ":8080"}`)

	_, err := NewFileLoader().Load(context.Background(), path)

	assert.Error(t, err)
}

func TestFileLoader_AppliesEnvOverrides(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("APP_NAME", "EnvProxy")
	t.Setenv("APP_VERSION", "9.9")
	t.Setenv("RELOAD_TIMER", "12")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SERVER_PORT", "7070")
	t.Setenv("PROXY_URL", "http://env-upstream")
	t.Setenv("DEFAULT_ALLOW", "false")
	t.Setenv("ALLOW_FILE", "env-allow.json")
	t.Setenv("DENY_FILE", "env-deny.json")

	path := writeConfigFile(t, "config.yaml", `
server:
  address: ":9090"
ip_access:
  default_policy: allow
`)

	cfg, err := NewFileLoader().Load(context.Background(), path)

	require.NoError(t, err)
	assert.Equal(t, "EnvProxy", cfg.App.Name)
	assert.Equal(t, "9.9", cfg.App.Version)
	assert.Equal(t, 12*time.Second, cfg.ReloadInterval())
	assert.Equal(t, "DEBUG", cfg.Log.Level)
	assert.Equal(t, ":7070", cfg.Server.Address)
	assert.Equal(t, "http://env-upstream", cfg.Proxy.BaseURL)
	assert.False(t, cfg.Proxy.DefaultAllow)
	assert.Equal(t, "deny", cfg.IPAccess.DefaultPolicy)
	assert.Equal(t, "env-allow.json", cfg.Proxy.AllowFile)
	assert.Equal(t, "env-deny.json", cfg.Proxy.DenyFile)
}

func writeConfigFile(t *testing.T, name string, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func isolateConfigEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		"APP_NAME",
		"APP_VERSION",
		"RELOAD_TIMER",
		"LOG_LEVEL",
		"SERVER_ADDRESS",
		"SERVER_PORT",
		"PROXY_URL",
		"DEFAULT_ALLOW",
		"IP_ACCESS_DEFAULT_POLICY",
		"ALLOW_FILE",
		"DENY_FILE",
	}

	for _, key := range keys {
		key := key
		oldValue, hadValue := os.LookupEnv(key)
		require.NoError(t, os.Unsetenv(key))
		t.Cleanup(func() {
			if hadValue {
				_ = os.Setenv(key, oldValue)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}
