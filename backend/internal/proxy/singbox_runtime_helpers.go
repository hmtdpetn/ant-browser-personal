package proxy

import (
	"ant-chrome/backend/internal/apppath"
	"ant-chrome/backend/internal/fsutil"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

func (m *SingBoxManager) resolveBinary() (string, error) {
	configPath := strings.TrimSpace(m.Config.Browser.SingBoxBinaryPath)
	if configPath != "" {
		resolved := resolveEnvPath(configPath, m.AppRoot)
		if resolved != "" {
			if _, err := os.Stat(resolved); err == nil {
				if err := fsutil.EnsureExecutable(resolved); err != nil {
					return "", fmt.Errorf("sing-box 文件不可执行: %s: %w", resolved, err)
				}
				return resolved, nil
			}
		}
	}
	if env := strings.TrimSpace(os.Getenv("SINGBOX_BINARY_PATH")); env != "" {
		if _, err := os.Stat(env); err == nil {
			if err := fsutil.EnsureExecutable(env); err != nil {
				return "", fmt.Errorf("sing-box 文件不可执行: %s: %w", env, err)
			}
			return env, nil
		}
	}

	binaryNames := []string{"sing-box"}
	if goruntime.GOOS == "windows" {
		binaryNames = []string{"sing-box.exe", "sing-box"}
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
					return "", fmt.Errorf("sing-box 文件不可执行: %s: %w", candidate, err)
				}
				return candidate, nil
			}
		}
	}

	for _, name := range binaryNames {
		if path, err := exec.LookPath(name); err == nil {
			if err := fsutil.EnsureExecutable(path); err != nil {
				return "", fmt.Errorf("sing-box 文件不可执行: %s: %w", path, err)
			}
			return path, nil
		}
	}

	return "", fmt.Errorf("未找到 sing-box 可执行文件。请将 sing-box 放到 bin/%s/ 或 bin/ 目录，或在配置中设置 SingBoxBinaryPath", platformDir)
}

func (m *SingBoxManager) buildConfig(key string, outbound map[string]interface{}, port int, dnsServers string) (string, error) {
	baseDir := m.resolveWorkdir(key)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", err
	}

	directOut := map[string]interface{}{
		"type": "direct",
		"tag":  "direct",
	}

	preferIPv4 := m.Config.PreferIPv4Enabled()
	if preferIPv4 {
		// 主方案：把节点 server 域名预解析为 IPv4 字面量；SNI 由 rewrite 内部保留。
		rewriteSingBoxOutboundToIPv4(outbound)
		// 加固：给每个 outbound 加 domain_strategy=ipv4_only（sing-box 1.12 合法值）。
		outbound["domain_strategy"] = "ipv4_only"
		directOut["domain_strategy"] = "ipv4_only"
	}

	cfg := map[string]interface{}{
		"log": map[string]interface{}{
			"level":     "info",
			"output":    filepath.Join(baseDir, "singbox.log"),
			"timestamp": true,
		},
		"inbounds": []interface{}{
			map[string]interface{}{
				"type":        "socks",
				"tag":         "socks-in",
				"listen":      "127.0.0.1",
				"listen_port": port,
			},
		},
		"outbounds": []interface{}{
			outbound,
			directOut,
		},
		"route": map[string]interface{}{
			"rules": []interface{}{
				map[string]interface{}{
					"inbound":  []string{"socks-in"},
					"outbound": "proxy-out",
				},
			},
		},
	}

	if preferIPv4 {
		// 加固：顶层 dns 块 strategy=ipv4_only，让 sing-box 内部解析只用 A 记录。
		addrs := dnsServerAddrs(dnsServers)
		if len(addrs) == 0 {
			addrs = []string{"1.1.1.1", "8.8.8.8"}
		}
		servers := make([]interface{}, 0, len(addrs))
		for _, a := range addrs {
			servers = append(servers, map[string]interface{}{"address": a})
		}
		cfg["dns"] = map[string]interface{}{
			"strategy": "ipv4_only",
			"servers":  servers,
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}

	cfgPath := filepath.Join(baseDir, "singbox-config.json")
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return "", err
	}
	return cfgPath, nil
}

func (m *SingBoxManager) resolveWorkdir(key string) string {
	root := strings.TrimSpace(m.Config.Browser.UserDataRoot)
	if root == "" {
		root = "data"
	}
	if !filepath.IsAbs(root) {
		root = apppath.Resolve(m.AppRoot, root)
	}
	return filepath.Join(root, "_singbox", key)
}
