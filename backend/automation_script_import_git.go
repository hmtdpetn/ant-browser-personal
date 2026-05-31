package backend

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ant-chrome/backend/internal/automation"
)

func cloneAutomationGitRepository(repoURL string, ref string) (string, func(), error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", nil, fmt.Errorf("未找到 git，可先安装 git 后再导入仓库脚本")
	}

	tempDir, err := os.MkdirTemp("", "ant-automation-git-*")
	if err != nil {
		return "", nil, fmt.Errorf("创建 Git 临时目录失败: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	if strings.TrimSpace(ref) == "" {
		if err := runGitCommand("", "clone", "--depth", "1", repoURL, tempDir); err != nil {
			cleanup()
			return "", nil, err
		}
		return tempDir, cleanup, nil
	}

	if err := runGitCommand("", "clone", "--depth", "1", "--branch", ref, "--single-branch", repoURL, tempDir); err == nil {
		return tempDir, cleanup, nil
	}

	_ = os.RemoveAll(tempDir)
	tempDir, err = os.MkdirTemp("", "ant-automation-git-*")
	if err != nil {
		return "", nil, fmt.Errorf("创建 Git 临时目录失败: %w", err)
	}
	cleanup = func() {
		_ = os.RemoveAll(tempDir)
	}

	if err := runGitCommand("", "clone", repoURL, tempDir); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := runGitCommand(tempDir, "checkout", ref); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("切换 Git 引用失败: %w", err)
	}
	return tempDir, cleanup, nil
}

func runGitCommand(workdir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if strings.TrimSpace(workdir) != "" {
		cmd.Dir = workdir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("git %s 失败: %s", strings.Join(args, " "), message)
	}
	return nil
}

func (a *App) loadAutomationGitBundle(repoURL string, ref string, scriptPath string) (automation.ImportedBundle, error) {
	normalizedRepoURL := strings.TrimSpace(repoURL)
	if normalizedRepoURL == "" {
		return automation.ImportedBundle{}, fmt.Errorf("Git 仓库地址不能为空")
	}

	normalizedRef := strings.TrimSpace(ref)
	normalizedScriptPath := strings.TrimSpace(scriptPath)

	repoDir, cleanup, err := cloneAutomationGitRepository(normalizedRepoURL, normalizedRef)
	if err != nil {
		return automation.ImportedBundle{}, err
	}
	defer cleanup()

	bundle, err := automation.ImportBundleFromDirectoryWithOptions(repoDir, normalizedScriptPath, buildAutomationGitImportLabel(normalizedRepoURL, normalizedRef, normalizedScriptPath), a.automationScriptImportOptions())
	if err != nil {
		return automation.ImportedBundle{}, err
	}
	return bundle, nil
}
