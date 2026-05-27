package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ConfigDuration time.Duration

func (d ConfigDuration) Duration() time.Duration {
	return time.Duration(d)
}

func (d ConfigDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *ConfigDuration) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		return d.parse(text)
	}

	var number int64
	if err := json.Unmarshal(data, &number); err == nil {
		*d = ConfigDuration(time.Duration(number))
		return nil
	}

	return fmt.Errorf("duration must be a string or nanoseconds")
}

func (d *ConfigDuration) UnmarshalText(text []byte) error {
	return d.parse(string(text))
}

func (d *ConfigDuration) parse(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		*d = 0
		return nil
	}

	if number, err := strconv.ParseInt(value, 10, 64); err == nil {
		*d = ConfigDuration(time.Duration(number))
		return nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value, err)
	}

	*d = ConfigDuration(duration)
	return nil
}

type AppConfig struct {
	App        AppInfoConfig        `json:"app" yaml:"app" toml:"app"`
	Log        LogConfig            `json:"log" yaml:"log" toml:"log"`
	Server     ServerConfig         `json:"server" yaml:"server" toml:"server"`
	Proxy      ProxyConfig          `json:"proxy" yaml:"proxy" toml:"proxy"`
	IPAccess   IPAccessConfig       `json:"ip_access" yaml:"ip_access" toml:"ip_access"`
	RateLimit  RateLimitConfigFile  `json:"rate_limit" yaml:"rate_limit" toml:"rate_limit"`
	Cache      CacheConfigFile      `json:"cache" yaml:"cache" toml:"cache"`
	Monitoring MonitoringConfigFile `json:"monitoring" yaml:"monitoring" toml:"monitoring"`
}

type AppInfoConfig struct {
	Name               string `json:"name" yaml:"name" toml:"name"`
	Version            string `json:"version" yaml:"version" toml:"version"`
	ReloadTimerSeconds int    `json:"reload_timer_seconds" yaml:"reload_timer_seconds" toml:"reload_timer_seconds"`
}

func (c *AppInfoConfig) UnmarshalYAML(unmarshal func(any) error) error {
	var raw struct {
		Name                    string `yaml:"name"`
		Version                 string `yaml:"version"`
		ReloadTimerSeconds      int    `yaml:"reload_timer_seconds"`
		ReloadTimerSecondsAlias int    `yaml:"reload-timer"`
	}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	c.Name = raw.Name
	c.Version = raw.Version
	c.ReloadTimerSeconds = raw.ReloadTimerSeconds
	if c.ReloadTimerSeconds == 0 {
		c.ReloadTimerSeconds = raw.ReloadTimerSecondsAlias
	}
	return nil
}

type LogConfig struct {
	Level string `json:"level" yaml:"level" toml:"level"`
}

type ServerConfig struct {
	Address      string         `json:"address" yaml:"address" toml:"address"`
	ReadTimeout  ConfigDuration `json:"read_timeout" yaml:"read_timeout" toml:"read_timeout"`
	WriteTimeout ConfigDuration `json:"write_timeout" yaml:"write_timeout" toml:"write_timeout"`
	IdleTimeout  ConfigDuration `json:"idle_timeout" yaml:"idle_timeout" toml:"idle_timeout"`
}

type ProxyConfig struct {
	BaseURL      string `json:"base_url" yaml:"base_url" toml:"base_url"`
	DefaultAllow bool   `json:"default_allow" yaml:"default_allow" toml:"default_allow"`
	AllowFile    string `json:"allow_file" yaml:"allow_file" toml:"allow_file"`
	DenyFile     string `json:"deny_file" yaml:"deny_file" toml:"deny_file"`
}

func (c *ProxyConfig) UnmarshalYAML(unmarshal func(any) error) error {
	var raw struct {
		BaseURL           string `yaml:"base_url"`
		BaseURLAlias      string `yaml:"base-url"`
		DefaultAllow      bool   `yaml:"default_allow"`
		DefaultAllowAlias bool   `yaml:"default-allow"`
		AllowFile         string `yaml:"allow_file"`
		AllowFileAlias    string `yaml:"allow-file"`
		DenyFile          string `yaml:"deny_file"`
		DenyFileAlias     string `yaml:"deny-file"`
	}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	c.BaseURL = raw.BaseURL
	if c.BaseURL == "" {
		c.BaseURL = raw.BaseURLAlias
	}
	c.DefaultAllow = raw.DefaultAllow || raw.DefaultAllowAlias
	c.AllowFile = raw.AllowFile
	if c.AllowFile == "" {
		c.AllowFile = raw.AllowFileAlias
	}
	c.DenyFile = raw.DenyFile
	if c.DenyFile == "" {
		c.DenyFile = raw.DenyFileAlias
	}
	return nil
}

type IPAccessConfig struct {
	Enabled           bool                 `json:"enabled" yaml:"enabled" toml:"enabled"`
	DefaultPolicy     string               `json:"default_policy" yaml:"default_policy" toml:"default_policy"`
	CacheTTL          ConfigDuration       `json:"cache_ttl" yaml:"cache_ttl" toml:"cache_ttl"`
	EnableLogging     bool                 `json:"enable_logging" yaml:"enable_logging" toml:"enable_logging"`
	EnableMetrics     bool                 `json:"enable_metrics" yaml:"enable_metrics" toml:"enable_metrics"`
	MaxListSize       int                  `json:"max_list_size" yaml:"max_list_size" toml:"max_list_size"`
	EnableGreyList    bool                 `json:"enable_grey_list" yaml:"enable_grey_list" toml:"enable_grey_list"`
	GreyListThreshold int                  `json:"grey_list_threshold" yaml:"grey_list_threshold" toml:"grey_list_threshold"`
	AllowList         []IPAccessRuleConfig `json:"allow_list" yaml:"allow_list" toml:"allow_list"`
	DenyList          []IPAccessRuleConfig `json:"deny_list" yaml:"deny_list" toml:"deny_list"`
	GreyList          []IPAccessRuleConfig `json:"grey_list" yaml:"grey_list" toml:"grey_list"`
}

type IPAccessRuleConfig struct {
	ID        string         `json:"id" yaml:"id" toml:"id"`
	Value     string         `json:"value" yaml:"value" toml:"value"`
	Comment   string         `json:"comment" yaml:"comment" toml:"comment"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty" yaml:"expires_at,omitempty" toml:"expires_at,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty" toml:"metadata,omitempty"`
}

type RateLimitConfigFile struct {
	Enabled                  bool   `json:"enabled" yaml:"enabled" toml:"enabled"`
	RPS                      int64  `json:"rps" yaml:"rps" toml:"rps"`
	RPM                      int64  `json:"rpm" yaml:"rpm" toml:"rpm"`
	RPH                      int64  `json:"rph" yaml:"rph" toml:"rph"`
	RPD                      int64  `json:"rpd" yaml:"rpd" toml:"rpd"`
	DownloadBytesPerSecond   int64  `json:"download_bytes_per_second" yaml:"download_bytes_per_second" toml:"download_bytes_per_second"`
	UploadBytesPerSecond     int64  `json:"upload_bytes_per_second" yaml:"upload_bytes_per_second" toml:"upload_bytes_per_second"`
	TotalBytesPerDay         int64  `json:"total_bytes_per_day" yaml:"total_bytes_per_day" toml:"total_bytes_per_day"`
	MaxConcurrentConnections int64  `json:"max_concurrent_connections" yaml:"max_concurrent_connections" toml:"max_concurrent_connections"`
	NewConnectionsPerSecond  int64  `json:"new_connections_per_second" yaml:"new_connections_per_second" toml:"new_connections_per_second"`
	SubnetMask               string `json:"subnet_mask" yaml:"subnet_mask" toml:"subnet_mask"`
}

type CacheConfigFile struct {
	Enabled     bool           `json:"enabled" yaml:"enabled" toml:"enabled"`
	MaxSize     int64          `json:"max_size" yaml:"max_size" toml:"max_size"`
	DefaultTTL  ConfigDuration `json:"default_ttl" yaml:"default_ttl" toml:"default_ttl"`
	TTL2xx      ConfigDuration `json:"ttl_2xx" yaml:"ttl_2xx" toml:"ttl_2xx"`
	TTL3xx      ConfigDuration `json:"ttl_3xx" yaml:"ttl_3xx" toml:"ttl_3xx"`
	TTL4xx      ConfigDuration `json:"ttl_4xx" yaml:"ttl_4xx" toml:"ttl_4xx"`
	TTL5xx      ConfigDuration `json:"ttl_5xx" yaml:"ttl_5xx" toml:"ttl_5xx"`
	MinBodySize int64          `json:"min_body_size" yaml:"min_body_size" toml:"min_body_size"`
	MaxBodySize int64          `json:"max_body_size" yaml:"max_body_size" toml:"max_body_size"`
}

type MonitoringConfigFile struct {
	RequestWindow        ConfigDuration `json:"request_window" yaml:"request_window" toml:"request_window"`
	MaxLatencySamples    int            `json:"max_latency_samples" yaml:"max_latency_samples" toml:"max_latency_samples"`
	MaxPathStats         int            `json:"max_path_stats" yaml:"max_path_stats" toml:"max_path_stats"`
	TrackPathStats       bool           `json:"track_path_stats" yaml:"track_path_stats" toml:"track_path_stats"`
	EnableRequestLogging bool           `json:"enable_request_logging" yaml:"enable_request_logging" toml:"enable_request_logging"`
}

func NewDefaultAppConfig() AppConfig {
	return AppConfig{
		App: AppInfoConfig{
			Name:               "ProxyApp",
			Version:            "0.1",
			ReloadTimerSeconds: 5,
		},
		Log: LogConfig{
			Level: "INFO",
		},
		Server: ServerConfig{
			Address:      ":8080",
			ReadTimeout:  ConfigDuration(5 * time.Second),
			WriteTimeout: ConfigDuration(30 * time.Second),
			IdleTimeout:  ConfigDuration(time.Minute),
		},
		Proxy: ProxyConfig{
			BaseURL:   "http://127.0.0.1:8081",
			AllowFile: "configs/allow.json",
			DenyFile:  "configs/deny.json",
		},
		IPAccess: IPAccessConfig{
			DefaultPolicy: "allow",
			CacheTTL:      ConfigDuration(5 * time.Minute),
			MaxListSize:   10000,
		},
		Cache: CacheConfigFile{
			MaxSize:     100 * 1024 * 1024,
			DefaultTTL:  ConfigDuration(5 * time.Minute),
			TTL2xx:      ConfigDuration(5 * time.Minute),
			TTL3xx:      ConfigDuration(time.Minute),
			TTL4xx:      ConfigDuration(30 * time.Second),
			TTL5xx:      ConfigDuration(10 * time.Second),
			MinBodySize: 1,
			MaxBodySize: 10 * 1024 * 1024,
		},
		Monitoring: MonitoringConfigFile{
			RequestWindow:     ConfigDuration(time.Minute),
			MaxLatencySamples: 1024,
			MaxPathStats:      100,
			TrackPathStats:    true,
		},
	}
}

func (c *AppConfig) Normalize() error {
	defaults := NewDefaultAppConfig()

	if c.App.Name == "" {
		c.App.Name = defaults.App.Name
	}
	if c.App.Version == "" {
		c.App.Version = defaults.App.Version
	}
	if c.App.ReloadTimerSeconds == 0 {
		c.App.ReloadTimerSeconds = defaults.App.ReloadTimerSeconds
	}

	if c.Log.Level == "" {
		c.Log.Level = defaults.Log.Level
	}

	if c.Server.Address == "" {
		c.Server.Address = defaults.Server.Address
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = defaults.Server.ReadTimeout
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = defaults.Server.WriteTimeout
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = defaults.Server.IdleTimeout
	}

	if c.Proxy.BaseURL == "" {
		c.Proxy.BaseURL = defaults.Proxy.BaseURL
	}
	if c.Proxy.AllowFile == "" {
		c.Proxy.AllowFile = defaults.Proxy.AllowFile
	}
	if c.Proxy.DenyFile == "" {
		c.Proxy.DenyFile = defaults.Proxy.DenyFile
	}

	if c.IPAccess.DefaultPolicy == "" {
		c.IPAccess.DefaultPolicy = defaults.IPAccess.DefaultPolicy
	}
	if c.IPAccess.DefaultPolicy != "allow" && c.IPAccess.DefaultPolicy != "deny" {
		return fmt.Errorf("invalid ip_access.default_policy: %s", c.IPAccess.DefaultPolicy)
	}
	if c.IPAccess.CacheTTL == 0 {
		c.IPAccess.CacheTTL = defaults.IPAccess.CacheTTL
	}
	if c.IPAccess.MaxListSize == 0 {
		c.IPAccess.MaxListSize = defaults.IPAccess.MaxListSize
	}

	totalRules := len(c.IPAccess.AllowList) + len(c.IPAccess.DenyList) + len(c.IPAccess.GreyList)
	if totalRules > c.IPAccess.MaxListSize {
		return fmt.Errorf("ip access lists contain %d rules, max is %d", totalRules, c.IPAccess.MaxListSize)
	}

	if err := normalizeRules(c.IPAccess.AllowList, AllowList); err != nil {
		return err
	}
	if err := normalizeRules(c.IPAccess.DenyList, DenyList); err != nil {
		return err
	}
	if err := normalizeRules(c.IPAccess.GreyList, GreyList); err != nil {
		return err
	}

	if c.Cache.MaxSize == 0 {
		c.Cache.MaxSize = defaults.Cache.MaxSize
	}
	if c.Cache.DefaultTTL == 0 {
		c.Cache.DefaultTTL = defaults.Cache.DefaultTTL
	}
	if c.Cache.TTL2xx == 0 {
		c.Cache.TTL2xx = defaults.Cache.TTL2xx
	}
	if c.Cache.TTL3xx == 0 {
		c.Cache.TTL3xx = defaults.Cache.TTL3xx
	}
	if c.Cache.TTL4xx == 0 {
		c.Cache.TTL4xx = defaults.Cache.TTL4xx
	}
	if c.Cache.TTL5xx == 0 {
		c.Cache.TTL5xx = defaults.Cache.TTL5xx
	}
	if c.Cache.MinBodySize == 0 {
		c.Cache.MinBodySize = defaults.Cache.MinBodySize
	}
	if c.Cache.MaxBodySize == 0 {
		c.Cache.MaxBodySize = defaults.Cache.MaxBodySize
	}

	if c.Monitoring.RequestWindow == 0 {
		c.Monitoring.RequestWindow = defaults.Monitoring.RequestWindow
	}
	if c.Monitoring.MaxLatencySamples == 0 {
		c.Monitoring.MaxLatencySamples = defaults.Monitoring.MaxLatencySamples
	}
	if c.Monitoring.MaxPathStats == 0 {
		c.Monitoring.MaxPathStats = defaults.Monitoring.MaxPathStats
	}

	return nil
}

func normalizeRules(rules []IPAccessRuleConfig, listType IPListType) error {
	for i := range rules {
		if strings.TrimSpace(rules[i].Value) == "" {
			return fmt.Errorf("empty %s rule at index %d", listType, i)
		}
		if rules[i].ID == "" {
			rules[i].ID = fmt.Sprintf("%s-%d", listType, i+1)
		}
	}
	return nil
}

func (c AppConfig) ToIPFilterConfig() IPFilterConfig {
	return IPFilterConfig{
		Enabled:           c.IPAccess.Enabled,
		DefaultPolicy:     IPListType(c.IPAccess.DefaultPolicy),
		CacheTTL:          c.IPAccess.CacheTTL.Duration(),
		EnableLogging:     c.IPAccess.EnableLogging,
		EnableMetrics:     c.IPAccess.EnableMetrics,
		MaxListSize:       c.IPAccess.MaxListSize,
		EnableGreyList:    c.IPAccess.EnableGreyList,
		GreyListThreshold: c.IPAccess.GreyListThreshold,
	}
}

func (c AppConfig) ToIPAccessPolicy(now time.Time) *IPAccessPolicy {
	return &IPAccessPolicy{
		ID:            "config",
		DefaultPolicy: c.IPAccess.DefaultPolicy,
		AllowList:     rulesToEntries(c.IPAccess.AllowList, AllowList, now),
		DenyList:      rulesToEntries(c.IPAccess.DenyList, DenyList, now),
		GreyList:      rulesToEntries(c.IPAccess.GreyList, GreyList, now),
		CreatedAt:     now,
		UpdatedAt:     now,
		Version:       1,
	}
}

func (c AppConfig) ToRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:                  c.RateLimit.Enabled,
		RPS:                      c.RateLimit.RPS,
		RPM:                      c.RateLimit.RPM,
		RPH:                      c.RateLimit.RPH,
		RPD:                      c.RateLimit.RPD,
		DownloadBytesPerSecond:   c.RateLimit.DownloadBytesPerSecond,
		UploadBytesPerSecond:     c.RateLimit.UploadBytesPerSecond,
		TotalBytesPerDay:         c.RateLimit.TotalBytesPerDay,
		MaxConcurrentConnections: c.RateLimit.MaxConcurrentConnections,
		NewConnectionsPerSecond:  c.RateLimit.NewConnectionsPerSecond,
		SubnetMask:               c.RateLimit.SubnetMask,
	}
}

func (c AppConfig) ToCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:     c.Cache.Enabled,
		MaxSize:     c.Cache.MaxSize,
		DefaultTTL:  c.Cache.DefaultTTL.Duration(),
		TTL2xx:      c.Cache.TTL2xx.Duration(),
		TTL3xx:      c.Cache.TTL3xx.Duration(),
		TTL4xx:      c.Cache.TTL4xx.Duration(),
		TTL5xx:      c.Cache.TTL5xx.Duration(),
		MinBodySize: c.Cache.MinBodySize,
		MaxBodySize: c.Cache.MaxBodySize,
	}
}

func (c AppConfig) ToMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		RequestWindow:        c.Monitoring.RequestWindow.Duration(),
		MaxLatencySamples:    c.Monitoring.MaxLatencySamples,
		MaxPathStats:         c.Monitoring.MaxPathStats,
		TrackPathStats:       c.Monitoring.TrackPathStats,
		EnableRequestLogging: c.Monitoring.EnableRequestLogging,
	}
}

func (c AppConfig) ReloadInterval() time.Duration {
	if c.App.ReloadTimerSeconds <= 0 {
		return time.Second
	}
	return time.Duration(c.App.ReloadTimerSeconds) * time.Second
}

func rulesToEntries(rules []IPAccessRuleConfig, listType IPListType, now time.Time) []IPEntry {
	entries := make([]IPEntry, 0, len(rules))
	for i, rule := range rules {
		id := rule.ID
		if id == "" {
			id = fmt.Sprintf("%s-%d", listType, i+1)
		}

		entries = append(entries, IPEntry{
			ID:        id,
			Type:      listType,
			Value:     strings.TrimSpace(rule.Value),
			Comment:   rule.Comment,
			CreatedAt: now,
			ExpiresAt: rule.ExpiresAt,
			Metadata:  rule.Metadata,
		})
	}
	return entries
}
