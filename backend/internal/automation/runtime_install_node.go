package automation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"ant-chrome/backend/internal/config"
)

type runtimeNodePlan struct {
	UseBundledNode bool
	SystemNode     resolvedNodeRuntime
}

func (m *Manager) prepareRuntimeNodePlan(ctx context.Context, auto config.AutomationConfig, nodeMode string) (runtimeNodePlan, error) {
	plan := runtimeNodePlan{
		UseBundledNode: strings.EqualFold(nodeMode, config.AutomationNodeSourceBundled),
	}
	if plan.UseBundledNode {
		return plan, nil
	}

	m.emitProgress("checking", 8, "正在检测系统 Node", "node")
	resolved, err := m.resolveSystemNode(ctx, auto.SystemNodePath)
	if err == nil {
		plan.SystemNode = resolved
		m.emitProgress("checking", 10, fmt.Sprintf("已检测到系统 Node %s，跳过 Node 下载", resolved.Version), "node")
		return plan, nil
	}

	if strings.EqualFold(nodeMode, config.AutomationNodeSourceSystem) {
		return runtimeNodePlan{}, fmt.Errorf("系统 Node 不可用: %w", err)
	}

	plan.UseBundledNode = true
	m.emitProgress("checking", 10, "未检测到可用的系统 Node，准备回退内建 Node", "node")
	return plan, nil
}

func (m *Manager) installBundledNodeRuntime(ctx context.Context, tempRoot string, stagingDir string, nodeVersion string, startProgress int, endProgress int, extractProgress int, message string) error {
	spec, err := m.nodeArchive(nodeVersion)
	if err != nil {
		return err
	}

	nodeArchiveURL := fmt.Sprintf("%s/v%s/%s", strings.TrimRight(m.options.NodeDistBaseURL, "/"), nodeVersion, spec.FileName)
	nodeShasumURL := fmt.Sprintf("%s/v%s/SHASUMS256.txt", strings.TrimRight(m.options.NodeDistBaseURL, "/"), nodeVersion)
	nodeArchivePath := filepath.Join(tempRoot, spec.FileName)

	expectedNodeSHA, err := m.fetchNodeSHA256(ctx, nodeShasumURL, spec.FileName)
	if err != nil {
		return fmt.Errorf("获取 Node 校验信息失败: %w", err)
	}
	if err := m.downloadFile(ctx, nodeArchiveURL, nodeArchivePath, "node", startProgress, endProgress, message); err != nil {
		return fmt.Errorf("下载 Node 运行时失败: %w", err)
	}
	if actual, err := sha256File(nodeArchivePath); err != nil {
		return fmt.Errorf("校验 Node 运行时失败: %w", err)
	} else if !strings.EqualFold(actual, expectedNodeSHA) {
		return fmt.Errorf("Node 运行时校验失败: expected %s got %s", expectedNodeSHA, actual)
	}

	if extractProgress >= 0 {
		m.emitProgress("extracting", extractProgress, "正在解压内建 Node 运行时", "node")
	}
	if err := extractArchive(nodeArchivePath, filepath.Join(stagingDir, "node"), spec.Format, spec.StripPrefix); err != nil {
		return fmt.Errorf("解压 Node 运行时失败: %w", err)
	}
	return nil
}

func (m *Manager) resolveInstalledNodeRuntime(ctx context.Context, tempRoot string, stagingDir string, auto config.AutomationConfig, nodeMode string, plan runtimeNodePlan) (string, string, string, error) {
	if plan.UseBundledNode {
		return config.AutomationNodeSourceBundled, strings.TrimSpace(auto.NodeVersion), m.nodeExecutablePath(stagingDir), nil
	}

	m.emitProgress("checking", 90, "正在验证系统 Node 与 playwright-core", "node")
	check, err := m.verifyNodeWithPlaywright(ctx, plan.SystemNode.Path, stagingDir)
	if err == nil {
		return config.AutomationNodeSourceSystem, check.NodeVersion, plan.SystemNode.Path, nil
	}

	if strings.EqualFold(nodeMode, config.AutomationNodeSourceSystem) {
		return "", "", "", fmt.Errorf("系统 Node 与 playwright-core 不兼容: %w", err)
	}

	m.emitProgress("checking", 92, "系统 Node 与 playwright-core 不兼容，正在回退内建 Node", "node")
	if installErr := m.installBundledNodeRuntime(ctx, tempRoot, stagingDir, auto.NodeVersion, 92, 97, -1, "正在下载内建 Node 运行时"); installErr != nil {
		return "", "", "", installErr
	}
	return config.AutomationNodeSourceBundled, strings.TrimSpace(auto.NodeVersion), m.nodeExecutablePath(stagingDir), nil
}
