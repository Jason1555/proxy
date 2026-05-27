package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"proxy/internal/domain"
)

func applyEnvOverrides(cfg *domain.AppConfig) error {
	if value, ok := os.LookupEnv("APP_NAME"); ok {
		cfg.App.Name = value
	}
	if value, ok := os.LookupEnv("APP_VERSION"); ok {
		cfg.App.Version = value
	}
	if value, ok := os.LookupEnv("RELOAD_TIMER"); ok {
		seconds, err := parseEnvInt(value, "RELOAD_TIMER")
		if err != nil {
			return err
		}
		cfg.App.ReloadTimerSeconds = seconds
	}
	if value, ok := os.LookupEnv("LOG_LEVEL"); ok {
		cfg.Log.Level = strings.ToUpper(strings.TrimSpace(value))
	}
	if value, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.Server.Address = value
	}
	if value, ok := os.LookupEnv("SERVER_PORT"); ok {
		cfg.Server.Address = normalizePortAddress(value)
	}
	if value, ok := os.LookupEnv("PROXY_URL"); ok {
		cfg.Proxy.BaseURL = value
	}
	if value, ok := os.LookupEnv("DEFAULT_ALLOW"); ok {
		defaultAllow, err := parseEnvBool(value, "DEFAULT_ALLOW")
		if err != nil {
			return err
		}
		cfg.Proxy.DefaultAllow = defaultAllow
		if defaultAllow {
			cfg.IPAccess.DefaultPolicy = "allow"
		} else {
			cfg.IPAccess.DefaultPolicy = "deny"
		}
	}
	if value, ok := os.LookupEnv("IP_ACCESS_DEFAULT_POLICY"); ok {
		cfg.IPAccess.DefaultPolicy = strings.ToLower(strings.TrimSpace(value))
	}
	if value, ok := os.LookupEnv("ALLOW_FILE"); ok {
		cfg.Proxy.AllowFile = value
	}
	if value, ok := os.LookupEnv("DENY_FILE"); ok {
		cfg.Proxy.DenyFile = value
	}

	return nil
}

func parseEnvBool(value, name string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("invalid %s bool value %q: %w", name, value, err)
	}
	return parsed, nil
}

func parseEnvInt(value, name string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid %s int value %q: %w", name, value, err)
	}
	return parsed, nil
}

func normalizePortAddress(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return port
	}
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}
