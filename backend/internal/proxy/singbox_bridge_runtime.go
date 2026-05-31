package proxy

import (
	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/logger"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// EnsureBridge 确保 sing-box 桥接进程运行，返回 socks5://127.0.0.1:port
func (m *SingBoxManager) EnsureBridge(proxyConfig string, proxies []config.BrowserProxy, proxyId string) (string, error) {
	log := logger.New("SingBox")
	src := resolveProxyConfig(proxyConfig, proxies, proxyId)
	if src == "" {
		return "", fmt.Errorf("未找到代理节点")
	}

	src = normalizeNodeScheme(src)
	outbound, err := BuildSingBoxOutbound(src)
	if err != nil {
		log.Error("节点解析失败", logger.F("error", err))
		return "", err
	}

	key := computeNodeKey(src)

	if socksURL, reused := m.tryReuseBridge(key); reused {
		log.Info("复用 sing-box 桥接", logger.F("key", key[:8]), logger.F("socks_url", socksURL))
		return socksURL, nil
	}

	binaryPath, err := m.resolveBinary()
	if err != nil {
		log.Error("sing-box 不可用", logger.F("error", err), logger.F("appRoot", m.AppRoot))
		return "", err
	}
	log.Debug("sing-box binary", logger.F("path", binaryPath))

	const maxRetries = 3
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		port, err := nextAvailablePort()
		if err != nil {
			lastErr = err
			continue
		}

		cfgPath, err := m.buildConfig(key, outbound, port)
		if err != nil {
			return "", fmt.Errorf("sing-box 配置生成失败: %w", err)
		}

		cmd := exec.Command(binaryPath, "run", "-c", cfgPath)
		hideWindow(cmd)
		cmd.Dir = filepath.Dir(cfgPath)
		stderrPath := filepath.Join(filepath.Dir(cfgPath), "singbox-stderr.log")
		stderrFile, _ := os.Create(stderrPath)
		if stderrFile != nil {
			cmd.Stderr = stderrFile
		}

		if err := cmd.Start(); err != nil {
			if stderrFile != nil {
				stderrFile.Close()
			}
			log.Error("sing-box 启动失败", logger.F("error", err), logger.F("attempt", attempt))
			lastErr = err
			continue
		}

		bridge := &SingBoxBridge{
			NodeKey: key,
			Port:    port,
			Cmd:     cmd,
			Pid:     cmd.Process.Pid,
			Running: true,
		}
		log.Info("sing-box 启动", logger.F("key", key[:8]), logger.F("pid", bridge.Pid), logger.F("port", port))

		if err := waitPortReady("127.0.0.1", port, 10*time.Second); err != nil {
			if stderrFile != nil {
				stderrFile.Close()
			}
			if content, readErr := os.ReadFile(stderrPath); readErr == nil && len(content) > 0 {
				log.Error("sing-box stderr", logger.F("output", string(content)))
			}
			bridge.Stopping = true
			m.stopBridgeProcess(bridge)
			bridge.Running = false
			bridge.Pid = 0
			bridge.LastError = err.Error()
			log.Error("sing-box 端口不可用，重试", logger.F("error", err), logger.F("attempt", attempt))
			lastErr = err
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if stderrFile != nil {
			stderrFile.Close()
		}

		if socksURL, reused := m.registerBridge(key, bridge); reused {
			log.Info("复用已就绪 sing-box 桥接", logger.F("key", key[:8]), logger.F("socks_url", socksURL))
			bridge.Stopping = true
			m.stopBridgeProcess(bridge)
			return socksURL, nil
		}

		go m.watchBridge(bridge, key)
		return fmt.Sprintf("socks5://127.0.0.1:%d", port), nil
	}

	return "", fmt.Errorf("sing-box 启动失败（已重试 %d 次）: %w", maxRetries, lastErr)
}

// StopAll 关闭所有 sing-box 桥接进程
func (m *SingBoxManager) StopAll() {
	m.mu.Lock()
	bridges := make([]*SingBoxBridge, 0, len(m.Bridges))
	for key, bridge := range m.Bridges {
		if bridge != nil {
			bridge.Stopping = true
			bridges = append(bridges, bridge)
		}
		delete(m.Bridges, key)
	}
	m.mu.Unlock()

	for _, bridge := range bridges {
		m.stopBridgeProcess(bridge)
	}
}

func (m *SingBoxManager) tryReuseBridge(key string) (string, bool) {
	var stale *SingBoxBridge

	m.mu.Lock()
	if bridge, ok := m.Bridges[key]; ok && bridge != nil {
		alive := bridge.Running && bridge.Cmd != nil && bridge.Cmd.Process != nil && bridge.Cmd.ProcessState == nil
		if alive && waitPortReady("127.0.0.1", bridge.Port, 800*time.Millisecond) == nil {
			socksURL := fmt.Sprintf("socks5://127.0.0.1:%d", bridge.Port)
			m.mu.Unlock()
			return socksURL, true
		}

		bridge.Stopping = true
		stale = bridge
		delete(m.Bridges, key)
	}
	m.mu.Unlock()

	if stale != nil {
		m.stopBridgeProcess(stale)
	}
	return "", false
}

func (m *SingBoxManager) registerBridge(key string, bridge *SingBoxBridge) (string, bool) {
	var duplicate *SingBoxBridge

	m.mu.Lock()
	if existing, ok := m.Bridges[key]; ok && existing != nil {
		if existing == bridge {
			m.mu.Unlock()
			return "", false
		}

		alive := existing.Running && existing.Cmd != nil && existing.Cmd.Process != nil && existing.Cmd.ProcessState == nil
		if alive && waitPortReady("127.0.0.1", existing.Port, 800*time.Millisecond) == nil {
			duplicate = bridge
			socksURL := fmt.Sprintf("socks5://127.0.0.1:%d", existing.Port)
			m.mu.Unlock()
			if duplicate != nil {
				duplicate.Stopping = true
				m.stopBridgeProcess(duplicate)
			}
			return socksURL, true
		}

		existing.Stopping = true
		delete(m.Bridges, key)
		duplicate = existing
	}
	m.Bridges[key] = bridge
	m.mu.Unlock()

	if duplicate != nil {
		m.stopBridgeProcess(duplicate)
	}
	return "", false
}

func (m *SingBoxManager) watchBridge(bridge *SingBoxBridge, key string) {
	if bridge == nil || bridge.Cmd == nil {
		return
	}
	_ = bridge.Cmd.Wait()

	m.mu.Lock()
	if current, ok := m.Bridges[key]; ok && current == bridge {
		delete(m.Bridges, key)
	}
	bridge.Running = false
	stopping := bridge.Stopping
	m.mu.Unlock()

	if !stopping && m.OnBridgeDied != nil {
		m.OnBridgeDied(key, fmt.Errorf("sing-box 桥接进程意外退出"))
	}
}

func (m *SingBoxManager) stopBridgeProcess(bridge *SingBoxBridge) {
	if bridge == nil || bridge.Cmd == nil || bridge.Cmd.Process == nil {
		return
	}
	_ = bridge.Cmd.Process.Kill()
}
