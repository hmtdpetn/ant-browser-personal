package backend

import (
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
	a.config.ProxyCheck = proxy.NormalizeCheckSettings(settings)
	return a.config.Save(a.resolveAppPath("config.yaml"))
}

func (a *App) proxySpeedTestConfig() *proxy.SpeedTestConfig {
	if a == nil || a.config == nil {
		cfg := proxy.DefaultSpeedTestConfig
		return &cfg
	}
	return proxy.BuildSpeedTestConfig(a.config.ProxyCheck)
}

func (a *App) proxyIPHealthConfig() *proxy.IPHealthConfig {
	if a == nil || a.config == nil {
		return &proxy.IPHealthConfig{Source: "ip_health"}
	}
	return proxy.BuildIPHealthConfig(a.config.ProxyCheck)
}
