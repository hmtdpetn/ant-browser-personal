package automation

import "context"

func (m *Manager) InstallAsync(ctx context.Context) {
	m.mu.Lock()
	if m.installing {
		m.mu.Unlock()
		return
	}
	m.installing = true
	m.lastError = ""
	m.mu.Unlock()

	go func() {
		_ = m.ensureInstalled(ctx, true)
	}()
}

func (m *Manager) EnsureInstalled(ctx context.Context) error {
	return m.ensureInstalled(ctx, false)
}

func (m *Manager) ensureInstalled(ctx context.Context, flagAlreadySet bool) error {
	ctx = ensureRuntimeInstallContext(ctx)
	if !m.beginRuntimeInstall(flagAlreadySet) {
		return nil
	}
	defer m.finishRuntimeInstall()

	auto := m.currentAutomationConfig()
	nodeMode := runtimeInstallNodeMode(auto)

	reused, err := m.tryUseReadyRuntime(ctx, nodeMode)
	if err != nil {
		return m.installFailed(err)
	}
	if reused {
		return nil
	}

	if err := validateRuntimeVersion(auto.RuntimeVersion); err != nil {
		return m.installFailed(err)
	}

	workspace, err := m.prepareRuntimeInstallWorkspace(auto.RuntimeVersion)
	if err != nil {
		return m.installFailed(err)
	}
	defer workspace.cleanup()

	m.emitProgress("checking", 5, "正在准备自动化运行时", "")

	nodePlan, err := m.prepareRuntimeNodePlan(ctx, auto, nodeMode)
	if err != nil {
		return m.installFailed(err)
	}
	if nodePlan.UseBundledNode {
		if err := m.installBundledNodeRuntime(ctx, workspace.TempRoot, workspace.StagingDir, auto.NodeVersion, 10, 45, 50, "正在下载内建 Node 运行时"); err != nil {
			return m.installFailed(err)
		}
	}

	if err := m.installPlaywrightRuntime(ctx, workspace.TempRoot, workspace.StagingDir, auto.PlaywrightCoreVersion); err != nil {
		return m.installFailed(err)
	}

	nodeSource, nodeVersion, nodePath, err := m.resolveInstalledNodeRuntime(ctx, workspace.TempRoot, workspace.StagingDir, auto, nodeMode, nodePlan)
	if err != nil {
		return m.installFailed(err)
	}

	if err := m.activateRuntimeInstall(workspace.StagingDir, auto, nodeSource, nodeVersion, nodePath); err != nil {
		return m.installFailed(err)
	}

	m.clearRuntimeInstallError()
	m.emitProgress("done", 100, "自动化运行时已安装完成", "")
	return nil
}
