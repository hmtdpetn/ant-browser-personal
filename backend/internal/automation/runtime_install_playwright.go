package automation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ant-chrome/backend/internal/config"
)

func (m *Manager) installPlaywrightRuntime(ctx context.Context, tempRoot string, stagingDir string, version string) error {
	playwrightMeta, err := m.fetchPlaywrightMetadata(ctx, version)
	if err != nil {
		return fmt.Errorf("获取 playwright-core 元数据失败: %w", err)
	}

	playwrightArchivePath := filepath.Join(tempRoot, fmt.Sprintf("playwright-core-%s.tgz", version))
	if err := m.downloadFile(ctx, playwrightMeta.TarballURL, playwrightArchivePath, "playwright", 55, 80, "正在下载 playwright-core"); err != nil {
		return fmt.Errorf("下载 playwright-core 失败: %w", err)
	}
	if actual, err := sha1File(playwrightArchivePath); err != nil {
		return fmt.Errorf("校验 playwright-core 失败: %w", err)
	} else if playwrightMeta.Shasum != "" && !strings.EqualFold(actual, playwrightMeta.Shasum) {
		return fmt.Errorf("playwright-core 校验失败: expected %s got %s", playwrightMeta.Shasum, actual)
	}

	m.emitProgress("extracting", 85, "正在解压 playwright-core", "playwright")
	if err := extractArchive(playwrightArchivePath, filepath.Join(stagingDir, "node_modules", "playwright-core"), "tar.gz", "package/"); err != nil {
		return fmt.Errorf("解压 playwright-core 失败: %w", err)
	}
	return nil
}

func (m *Manager) activateRuntimeInstall(stagingDir string, auto config.AutomationConfig, nodeSource string, nodeVersion string, nodePath string) error {
	if err := writeRuntimeManifest(
		filepath.Join(stagingDir, "manifest.json"),
		nodeVersion,
		auto.PlaywrightCoreVersion,
		auto.RuntimeVersion,
		nodeSource,
		nodePath,
	); err != nil {
		return fmt.Errorf("写入自动化运行时清单失败: %w", err)
	}
	if err := writeRunnerScript(filepath.Join(stagingDir, runnerScriptFileName)); err != nil {
		return fmt.Errorf("写入自动化 runner 失败: %w", err)
	}

	runtimeDir := m.runtimeDir(auto.RuntimeVersion)
	if err := os.MkdirAll(filepath.Dir(runtimeDir), 0o755); err != nil {
		return fmt.Errorf("创建自动化运行时目录失败: %w", err)
	}
	if err := os.RemoveAll(runtimeDir); err != nil {
		return fmt.Errorf("替换自动化运行时目录失败: %w", err)
	}
	if err := os.Rename(stagingDir, runtimeDir); err != nil {
		return fmt.Errorf("启用自动化运行时失败: %w", err)
	}
	return nil
}
