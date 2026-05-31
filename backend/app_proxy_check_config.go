package backend

import (
	"strings"
	"time"

	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/proxy"
)

type ProxyCheckSettings = config.ProxyCheckConfig
type ProxyCheckTarget = config.ProxyCheckTarget

func (a *App) GetProxyCheckSettings() ProxyCheckSettings {
	if a.config == nil {
		return config.DefaultConfig().ProxyCheck
	}
	settings := a.config.ProxyCheck
	settings.Targets = append([]config.ProxyCheckTarget{}, settings.Targets...)
	return settings
}

func (a *App) SaveProxyCheckSettings(settings ProxyCheckSettings) error {
	if a.config == nil {
		return nil
	}
	settings.BridgeStartTimeoutMs = normalizePositiveInt(settings.BridgeStartTimeoutMs, 15000)
	settings.SpeedTargetID = strings.TrimSpace(settings.SpeedTargetID)
	settings.IPHealthTargetID = strings.TrimSpace(settings.IPHealthTargetID)
	settings.Targets = normalizeProxyCheckTargets(settings.Targets)
	if len(settings.Targets) == 0 {
		settings.Targets = config.DefaultConfig().ProxyCheck.Targets
	}
	if settings.SpeedTargetID == "" {
		settings.SpeedTargetID = firstProxyCheckTargetID(settings.Targets, "speed", "")
	}
	if settings.IPHealthTargetID == "" {
		settings.IPHealthTargetID = firstProxyCheckTargetID(settings.Targets, "ip_health", "")
	}
	a.config.ProxyCheck = settings
	return a.config.Save(a.resolveAppPath("config.yaml"))
}

func (a *App) proxySpeedTestConfig() *proxy.SpeedTestConfig {
	cfg := proxy.DefaultSpeedTestConfig
	if a == nil || a.config == nil {
		return &cfg
	}
	target := a.proxyCheckTarget(a.config.ProxyCheck.SpeedTargetID, "speed")
	if strings.TrimSpace(target.URL) != "" {
		cfg.URLs = []string{strings.TrimSpace(target.URL)}
	}
	if target.TimeoutMs > 0 {
		cfg.Timeout = time.Duration(target.TimeoutMs) * time.Millisecond
	}
	return &cfg
}

func (a *App) proxyIPHealthConfig() *proxy.IPHealthConfig {
	cfg := &proxy.IPHealthConfig{Source: "ip_health"}
	if a == nil || a.config == nil {
		return cfg
	}
	target := a.proxyCheckTarget(a.config.ProxyCheck.IPHealthTargetID, "ip_health")
	if strings.TrimSpace(target.URL) != "" {
		cfg.URL = strings.TrimSpace(target.URL)
	}
	if strings.TrimSpace(target.ID) != "" {
		cfg.Source = strings.TrimSpace(target.ID)
	}
	if strings.TrimSpace(target.Parser) != "" {
		cfg.Parser = strings.TrimSpace(target.Parser)
	}
	if target.TimeoutMs > 0 {
		cfg.Timeout = time.Duration(target.TimeoutMs) * time.Millisecond
	}
	return cfg
}

func (a *App) proxyCheckTarget(id string, targetType string) config.ProxyCheckTarget {
	if a == nil || a.config == nil {
		return config.ProxyCheckTarget{}
	}
	normalizedID := strings.TrimSpace(id)
	normalizedType := strings.TrimSpace(targetType)
	for _, target := range a.config.ProxyCheck.Targets {
		if normalizedID != "" && strings.EqualFold(strings.TrimSpace(target.ID), normalizedID) {
			return target
		}
	}
	for _, target := range a.config.ProxyCheck.Targets {
		if normalizedType != "" && strings.EqualFold(strings.TrimSpace(target.Type), normalizedType) {
			return target
		}
	}
	return config.ProxyCheckTarget{}
}

func normalizeProxyCheckTargets(targets []config.ProxyCheckTarget) []config.ProxyCheckTarget {
	result := make([]config.ProxyCheckTarget, 0, len(targets))
	seen := map[string]struct{}{}
	for _, target := range targets {
		target.ID = strings.TrimSpace(target.ID)
		target.Name = strings.TrimSpace(target.Name)
		target.Type = strings.TrimSpace(target.Type)
		target.URL = strings.TrimSpace(target.URL)
		target.Parser = strings.TrimSpace(target.Parser)
		if target.ID == "" || target.URL == "" {
			continue
		}
		key := strings.ToLower(target.ID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if target.Name == "" {
			target.Name = target.ID
		}
		if target.Type == "" {
			target.Type = "speed"
		}
		if target.TimeoutMs <= 0 {
			target.TimeoutMs = 10000
		}
		result = append(result, target)
	}
	return result
}

func firstProxyCheckTargetID(targets []config.ProxyCheckTarget, targetType string, fallback string) string {
	for _, target := range targets {
		if strings.EqualFold(strings.TrimSpace(target.Type), targetType) {
			return strings.TrimSpace(target.ID)
		}
	}
	return fallback
}

func normalizePositiveInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
