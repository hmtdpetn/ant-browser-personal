package automation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ant-chrome/backend/internal/config"
)

func runtimeInstallNodeMode(auto config.AutomationConfig) string {
	nodeMode := config.DefaultAutomationNodeSource
	if auto.NodeSource != "" {
		nodeMode = auto.NodeSource
	}
	return nodeMode
}

func (m *Manager) tryUseReadyRuntime(ctx context.Context, nodeMode string) (bool, error) {
	state := m.CurrentState()
	if !state.Ready {
		return false, nil
	}

	if err := syncRunnerScript(state.RunnerPath); err != nil {
		return false, fmt.Errorf("更新自动化 runner 失败: %w", err)
	}

	if !strings.EqualFold(state.NodeSource, config.AutomationNodeSourceSystem) {
		m.emitProgress("done", 100, "自动化运行时已就绪", "")
		return true, nil
	}

	m.emitProgress("checking", 5, "正在验证系统 Node 与 playwright-core", "node")
	if _, err := m.verifyNodeWithPlaywright(ctx, state.NodePath, state.RuntimeDir); err == nil {
		m.emitProgress("done", 100, "自动化运行时已就绪", "")
		return true, nil
	} else if strings.EqualFold(nodeMode, config.AutomationNodeSourceSystem) {
		return false, fmt.Errorf("系统 Node 与 playwright-core 不兼容: %w", err)
	}

	m.emitProgress("checking", 5, "系统 Node 与 playwright-core 不兼容，正在修复运行时", "node")
	return false, nil
}

func (m *Manager) prepareRuntimeInstallWorkspace(runtimeVersion string) (runtimeInstallWorkspace, error) {
	tempRoot := filepath.Join(m.runtimeRoot(), "tmp")
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return runtimeInstallWorkspace{}, fmt.Errorf("创建自动化运行时临时目录失败: %w", err)
	}

	stagingDir := filepath.Join(tempRoot, fmt.Sprintf("%s.staging-%d", runtimeVersion, time.Now().UnixNano()))
	if err := removeRuntimeInstallWorkspace(stagingDir); err != nil {
		return runtimeInstallWorkspace{}, fmt.Errorf("清理自动化运行时临时目录失败: %w", err)
	}
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return runtimeInstallWorkspace{}, fmt.Errorf("创建自动化运行时临时目录失败: %w", err)
	}

	return runtimeInstallWorkspace{
		TempRoot:   tempRoot,
		StagingDir: stagingDir,
	}, nil
}

func removeRuntimeInstallWorkspace(stagingDir string) error {
	if strings.TrimSpace(stagingDir) == "" {
		return nil
	}
	return os.RemoveAll(stagingDir)
}
