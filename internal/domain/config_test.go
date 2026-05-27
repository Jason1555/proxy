package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDuration_UnmarshalJSON_String(t *testing.T) {
	var duration ConfigDuration

	err := json.Unmarshal([]byte(`"150ms"`), &duration)

	require.NoError(t, err)
	assert.Equal(t, 150*time.Millisecond, duration.Duration())
}

func TestConfigDuration_UnmarshalJSON_Number(t *testing.T) {
	var duration ConfigDuration

	err := json.Unmarshal([]byte(`1000000000`), &duration)

	require.NoError(t, err)
	assert.Equal(t, time.Second, duration.Duration())
}

func TestAppConfig_NormalizeAndConvert(t *testing.T) {
	cfg := AppConfig{
		IPAccess: IPAccessConfig{
			Enabled:       true,
			DefaultPolicy: "deny",
			AllowList: []IPAccessRuleConfig{
				{Value: "10.0.0.0/8", Comment: "internal"},
			},
		},
		RateLimit: RateLimitConfigFile{
			Enabled: true,
			RPS:     10,
		},
		Cache: CacheConfigFile{
			Enabled: true,
		},
	}

	require.NoError(t, cfg.Normalize())

	ipFilterConfig := cfg.ToIPFilterConfig()
	rateLimitConfig := cfg.ToRateLimitConfig()
	cacheConfig := cfg.ToCacheConfig()
	policy := cfg.ToIPAccessPolicy(time.Unix(10, 0))

	assert.Equal(t, ":8080", cfg.Server.Address)
	assert.Equal(t, "ProxyApp", cfg.App.Name)
	assert.Equal(t, "0.1", cfg.App.Version)
	assert.Equal(t, 5*time.Second, cfg.ReloadInterval())
	assert.Equal(t, "INFO", cfg.Log.Level)
	assert.Equal(t, "http://127.0.0.1:8081", cfg.Proxy.BaseURL)
	assert.Equal(t, DenyList, ipFilterConfig.DefaultPolicy)
	assert.Equal(t, int64(10), rateLimitConfig.RPS)
	assert.True(t, cacheConfig.Enabled)
	assert.Equal(t, "allow-1", policy.AllowList[0].ID)
	assert.Equal(t, "10.0.0.0/8", policy.AllowList[0].Value)
	assert.Equal(t, "deny", policy.DefaultPolicy)
}

func TestAppConfig_NormalizeRejectsInvalidPolicy(t *testing.T) {
	cfg := AppConfig{
		IPAccess: IPAccessConfig{
			DefaultPolicy: "captcha",
		},
	}

	assert.Error(t, cfg.Normalize())
}
