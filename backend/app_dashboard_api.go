package backend

import (
	goruntime "runtime"

	"ant-chrome/backend/internal/logger"
)

func (a *App) GetDashboardStats() map[string]interface{} {
	profiles := a.browserMgr.List()
	totalInstances := len(profiles)
	runningInstances := 0
	for _, profile := range profiles {
		if profile.Running {
			runningInstances++
		}
	}
	proxyCount := len(a.config.Browser.Proxies)
	coreCount := len(a.config.Browser.Cores)

	var mem goruntime.MemStats
	goruntime.ReadMemStats(&mem)
	memUsedMB := float64(mem.Alloc) / 1024 / 1024

	return map[string]interface{}{
		"totalInstances":   totalInstances,
		"runningInstances": runningInstances,
		"proxyCount":       proxyCount,
		"coreCount":        coreCount,
		"memUsedMB":        int(memUsedMB),
		"appVersion":       a.appVersion(),
	}
}

func (a *App) GetAppConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":    a.appName(),
		"version": a.appVersion(),
	}
}

func (a *App) GetMemoryStats() map[string]interface{} {
	var mem goruntime.MemStats
	goruntime.ReadMemStats(&mem)
	return map[string]interface{}{
		"alloc_mb":       float64(mem.Alloc) / 1024 / 1024,
		"total_alloc_mb": float64(mem.TotalAlloc) / 1024 / 1024,
		"sys_mb":         float64(mem.Sys) / 1024 / 1024,
		"num_gc":         mem.NumGC,
		"limit_mb":       a.config.Runtime.MaxMemoryMB,
		"gc_percent":     a.config.Runtime.GCPercent,
	}
}

func (a *App) TriggerGC()               { goruntime.GC() }
func (a *App) SetLogLevel(level string) { logger.SetGlobalLevelString(level) }
func (a *App) GetLogLevel() string      { return logger.New("App").GetLevel().String() }

// GetAppLogs 获取内存缓冲日志
func (a *App) GetAppLogs() []logger.MemoryLogEntry {
	return logger.GetMemoryWriter().GetEntries()
}

// ClearAppLogs 清空内存缓冲日志
func (a *App) ClearAppLogs() {
	logger.GetMemoryWriter().Clear()
}

// GetRunningInstances 获取运行中实例的详细信息
func (a *App) GetRunningInstances() []BrowserProfile {
	all := a.browserMgr.List()
	result := make([]BrowserProfile, 0)
	for _, profile := range all {
		if profile.Running {
			result = append(result, profile)
		}
	}
	return result
}
