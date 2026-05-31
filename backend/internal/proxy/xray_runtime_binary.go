package proxy

import (
	"ant-chrome/backend/internal/fsutil"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

func (m *XrayManager) resolveBinary() (string, error) {
	configPath := strings.TrimSpace(m.Config.Browser.XrayBinaryPath)
	if configPath != "" {
		resolved := resolveEnvPath(configPath, m.AppRoot)
		if resolved != "" {
			if _, err := os.Stat(resolved); err == nil {
				if err := fsutil.EnsureExecutable(resolved); err != nil {
					return "", fmt.Errorf("xray 文件不可执行: %s: %w", resolved, err)
				}
				return resolved, nil
			}
		}
	}
	env := strings.TrimSpace(os.Getenv("XRAY_BINARY_PATH"))
	if env != "" {
		if _, err := os.Stat(env); err == nil {
			if err := fsutil.EnsureExecutable(env); err != nil {
				return "", fmt.Errorf("xray 文件不可执行: %s: %w", env, err)
			}
			return env, nil
		}
	}

	binaryNames := []string{"xray"}
	if goruntime.GOOS == "windows" {
		binaryNames = []string{"xray.exe", "xray"}
	}
	platformDir := fmt.Sprintf("%s-%s", goruntime.GOOS, goruntime.GOARCH)

	searchDirs := make([]string, 0, 4)
	if m.AppRoot != "" {
		searchDirs = append(searchDirs,
			filepath.Join(m.AppRoot, "bin", platformDir),
			filepath.Join(m.AppRoot, "bin"),
		)
	}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		searchDirs = append(searchDirs,
			filepath.Join(exeDir, "bin", platformDir),
			filepath.Join(exeDir, "bin"),
		)
	}

	for _, dir := range searchDirs {
		for _, name := range binaryNames {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				if err := fsutil.EnsureExecutable(candidate); err != nil {
					return "", fmt.Errorf("xray 文件不可执行: %s: %w", candidate, err)
				}
				return candidate, nil
			}
		}
	}

	for _, name := range binaryNames {
		if path, err := exec.LookPath(name); err == nil {
			if err := fsutil.EnsureExecutable(path); err != nil {
				return "", fmt.Errorf("xray 文件不可执行: %s: %w", path, err)
			}
			return path, nil
		}
	}

	return "", fmt.Errorf("未找到 xray 可执行文件。请将 xray 放到 bin/%s/ 或 bin/ 目录，或在配置中设置 XrayBinaryPath", platformDir)
}
