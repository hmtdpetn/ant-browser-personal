package backend

import (
	"fmt"
	"strings"

	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/logger"
)

func (a *App) SaveAutomationSettings(enabled bool, headlessDefault bool) (map[string]interface{}, error) {
	if a.config == nil {
		return nil, fmt.Errorf("automation config is not initialized")
	}

	a.config.Automation.Enabled = enabled
	a.config.Automation.HeadlessDefault = headlessDefault
	applyAutomationConfigDefaults(&a.config.Automation)

	if err := a.config.Save(a.resolveAppPath("config.yaml")); err != nil {
		logger.New("Automation").Error("自动化配置保存失败", logger.F("error", err.Error()))
		return nil, err
	}

	if a.automationMgr != nil {
		a.automationMgr.SetConfig(a.config)
		state := a.automationMgr.CurrentState()
		if enabled && !state.Ready && strings.EqualFold(a.config.Automation.InstallPolicy, config.DefaultAutomationInstallPolicy) {
			a.automationMgr.InstallAsync(a.ctx)
		}
	}

	return a.automationStatePayload(), nil
}

func (a *App) SaveAutomationRuntimeSettings(nodeSource string, systemNodePath string) (map[string]interface{}, error) {
	if a.config == nil {
		return nil, fmt.Errorf("automation config is not initialized")
	}

	a.config.Automation.NodeSource = normalizeAutomationNodeSourceInput(nodeSource)
	a.config.Automation.SystemNodePath = strings.TrimSpace(systemNodePath)
	applyAutomationConfigDefaults(&a.config.Automation)

	if err := a.config.Save(a.resolveAppPath("config.yaml")); err != nil {
		logger.New("Automation").Error("自动化运行时策略保存失败", logger.F("error", err.Error()))
		return nil, err
	}

	if a.automationMgr != nil {
		a.automationMgr.SetConfig(a.config)
		if a.config.Automation.Enabled && strings.EqualFold(a.config.Automation.InstallPolicy, config.DefaultAutomationInstallPolicy) {
			a.automationMgr.InstallAsync(a.ctx)
		}
	}

	return a.automationStatePayload(), nil
}

func (a *App) SaveAutomationScriptPackageSettings(allowTypeScriptBuild bool) (map[string]interface{}, error) {
	if a.config == nil {
		return nil, fmt.Errorf("automation config is not initialized")
	}

	a.config.Automation.AllowTypeScriptBuild = allowTypeScriptBuild
	applyAutomationConfigDefaults(&a.config.Automation)

	if err := a.config.Save(a.resolveAppPath("config.yaml")); err != nil {
		logger.New("Automation").Error("自动化脚本包配置保存失败", logger.F("error", err.Error()))
		return nil, err
	}

	if a.automationMgr != nil {
		a.automationMgr.SetConfig(a.config)
	}

	return a.automationStatePayload(), nil
}

func (a *App) InstallAutomationRuntime() (map[string]interface{}, error) {
	if a.automationMgr == nil {
		return nil, fmt.Errorf("automation runtime manager is not initialized")
	}
	a.automationMgr.InstallAsync(a.ctx)
	return a.automationStatePayload(), nil
}

func (a *App) AutomationProbeSystemNode(systemNodePath string) (map[string]interface{}, error) {
	if a.automationMgr == nil {
		return nil, fmt.Errorf("automation runtime manager is not initialized")
	}

	explicitPath := strings.TrimSpace(systemNodePath)
	if explicitPath == "" && a.config != nil {
		explicitPath = strings.TrimSpace(a.config.Automation.SystemNodePath)
	}

	result, err := a.automationMgr.ProbeSystemNode(a.ctx, explicitPath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"ok":      result.OK,
		"path":    result.Path,
		"version": result.Version,
	}, nil
}

func (a *App) AutomationRuntimeSelfCheck() (map[string]interface{}, error) {
	if a.automationMgr == nil {
		return nil, fmt.Errorf("automation runtime manager is not initialized")
	}
	result, err := a.automationMgr.SelfCheck(a.ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"ok":                result.OK,
		"nodeSource":        result.NodeSource,
		"nodeVersion":       result.NodeVersion,
		"playwrightVersion": result.PlaywrightVersion,
	}, nil
}
