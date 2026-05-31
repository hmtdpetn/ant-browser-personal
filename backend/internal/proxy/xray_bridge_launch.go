package proxy

import (
	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/logger"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (m *XrayManager) ensureBridge(proxyConfig string, proxies []config.BrowserProxy, proxyId string, pin bool) (string, string, error) {
	log := logger.New("Xray")
	src := resolveProxyConfig(proxyConfig, proxies, proxyId)
	dnsServers := ""
	if proxyId != "" {
		for _, item := range proxies {
			if strings.EqualFold(item.ProxyId, proxyId) {
				dnsServers = item.DnsServers
				break
			}
		}
	}
	if src == "" {
		return "", "", fmt.Errorf("未找到代理节点")
	}
	src = normalizeNodeScheme(src)

	var (
		outbounds     []interface{}
		routes        []interface{}
		preferredPort int
	)

	if IsChainSocks5Proxy(src) {
		chainCfg, err := ParseChainSocks5Config(src)
		if err != nil {
			log.Error("链式节点解析失败", logger.F("error", err))
			return "", "", err
		}
		outbounds = []interface{}{
			chainSocks5Outbound(chainCfg.First, "first-hop", ""),
			chainSocks5Outbound(chainCfg.Second, "second-hop", "first-hop"),
		}
		routes = []interface{}{
			map[string]interface{}{
				"type":        "field",
				"inboundTag":  []string{"socks-in"},
				"outboundTag": "second-hop",
			},
		}
		preferredPort = chainCfg.LocalPort
	} else {
		directOutbound, shouldBridgeDirectProxy, err := buildDirectProxyBridgeOutbound(src)
		if err != nil {
			log.Error("直连代理桥接配置解析失败", logger.F("error", err))
			return "", "", err
		}
		if shouldBridgeDirectProxy {
			outbounds = []interface{}{directOutbound}
			routes = []interface{}{
				map[string]interface{}{
					"type":        "field",
					"inboundTag":  []string{"socks-in"},
					"outboundTag": "proxy-out",
				},
			}
		} else {
			standardProxy, outbound, err := ParseProxyNode(src)
			if err != nil {
				log.Error("节点解析失败", logger.F("error", err))
				return "", "", err
			}
			if standardProxy != "" {
				return standardProxy, "", nil
			}
			if outbound == nil {
				return "", "", fmt.Errorf("节点解析失败")
			}
			outbounds = []interface{}{outbound}
			routes = []interface{}{
				map[string]interface{}{
					"type":        "field",
					"inboundTag":  []string{"socks-in"},
					"outboundTag": "proxy-out",
				},
			}
		}
	}
	key := computeNodeKey(src + "\x00" + dnsServers)

	if socksURL, reused := m.tryReuseBridge(key, pin); reused {
		log.Info("复用桥接进程", logger.F("key", key), logger.F("socks_url", socksURL))
		return socksURL, key, nil
	}
	m.launchMu.Lock()
	defer m.launchMu.Unlock()
	if socksURL, reused := m.tryReuseBridge(key, pin); reused {
		log.Info("复用桥接进程", logger.F("key", key), logger.F("socks_url", socksURL))
		return socksURL, key, nil
	}

	binaryPath, err := m.resolveBinary()
	if err != nil {
		log.Error("xray 不可用", logger.F("error", err))
		return "", "", err
	}

	maxLaunchRetries := 3
	if preferredPort > 0 {
		maxLaunchRetries = 1
	}
	var lastErr error
	for attempt := 1; attempt <= maxLaunchRetries; attempt++ {
		socksURL, bridge, err := m.launchBridgeAttempt(log, key, binaryPath, outbounds, routes, preferredPort, dnsServers, pin, attempt)
		if err == nil {
			return socksURL, key, nil
		}
		if bridge != nil && bridge.Running {
			go m.watchBridge(bridge, key)
		}
		lastErr = err
	}
	return "", "", fmt.Errorf("xray 启动失败（已重试 %d 次）: %w", maxLaunchRetries, lastErr)
}

func (m *XrayManager) launchBridgeAttempt(log *logger.Logger, key string, binaryPath string, outbounds []interface{}, routes []interface{}, preferredPort int, dnsServers string, pin bool, attempt int) (string, *XrayBridge, error) {
	port := preferredPort
	if port <= 0 {
		var err error
		port, err = nextAvailablePort()
		if err != nil {
			log.Error("端口分配失败", logger.F("error", err), logger.F("attempt", attempt))
			return "", nil, err
		}
	}
	cfgPath, err := m.buildRuntimeConfigWithRoute(key, outbounds, routes, port, dnsServers)
	if err != nil {
		log.Error("xray 配置生成失败", logger.F("error", err))
		return "", nil, err
	}
	cmd := exec.Command(binaryPath, "run", "-c", cfgPath)
	hideWindow(cmd)
	cmd.Dir = filepath.Dir(cfgPath)

	stderrPath := filepath.Join(filepath.Dir(cfgPath), "xray-stderr.log")
	stderrFile, _ := os.Create(stderrPath)
	if stderrFile != nil {
		cmd.Stderr = stderrFile
	}

	if err := cmd.Start(); err != nil {
		if stderrFile != nil {
			stderrFile.Close()
		}
		log.Error("xray 启动失败", logger.F("error", err), logger.F("attempt", attempt))
		return "", nil, err
	}

	bridge := &XrayBridge{
		NodeKey:    key,
		Port:       port,
		Cmd:        cmd,
		Pid:        cmd.Process.Pid,
		Running:    true,
		RefCount:   0,
		LastUsedAt: time.Now(),
	}
	log.Info("xray 启动", logger.F("key", key), logger.F("pid", bridge.Pid), logger.F("port", bridge.Port), logger.F("attempt", attempt))

	if err := m.waitBridgeReady(log, bridge, cfgPath, stderrPath, stderrFile, attempt); err != nil {
		return "", nil, err
	}

	if socksURL, reused := m.registerBridge(key, bridge, pin); reused {
		log.Info("复用已就绪桥接进程", logger.F("key", key), logger.F("socks_url", socksURL))
		bridge.Stopping = true
		m.stopBridgeProcess(bridge)
		return socksURL, nil, nil
	}

	return fmt.Sprintf("socks5://127.0.0.1:%d", port), bridge, nil
}

func chainSocks5Outbound(hop chainSocks5Hop, tag string, nextTag string) map[string]interface{} {
	protocol := normalizeChainHopProtocol(hop.Protocol)
	if protocol == "http" {
		return chainHTTPOutbound(hop, tag, nextTag)
	}

	user := map[string]interface{}{}
	if strings.TrimSpace(hop.Username) != "" {
		user["user"] = strings.TrimSpace(hop.Username)
		if strings.TrimSpace(hop.Password) != "" {
			user["pass"] = hop.Password
		}
	}

	server := map[string]interface{}{
		"address": strings.TrimSpace(hop.Server),
		"port":    hop.Port,
	}
	if len(user) > 0 {
		server["users"] = []interface{}{user}
	}

	outbound := map[string]interface{}{
		"protocol": "socks",
		"tag":      tag,
		"settings": map[string]interface{}{
			"servers": []interface{}{server},
		},
	}
	if strings.TrimSpace(nextTag) != "" {
		outbound["proxySettings"] = map[string]interface{}{
			"tag": strings.TrimSpace(nextTag),
		}
	}
	return outbound
}

func chainHTTPOutbound(hop chainSocks5Hop, tag string, nextTag string) map[string]interface{} {
	user := map[string]interface{}{}
	if strings.TrimSpace(hop.Username) != "" {
		user["user"] = strings.TrimSpace(hop.Username)
		if strings.TrimSpace(hop.Password) != "" {
			user["pass"] = hop.Password
		}
	}

	server := map[string]interface{}{
		"address": strings.TrimSpace(hop.Server),
		"port":    hop.Port,
	}
	if len(user) > 0 {
		server["users"] = []interface{}{user}
	}

	outbound := map[string]interface{}{
		"protocol": "http",
		"tag":      tag,
		"settings": map[string]interface{}{
			"servers": []interface{}{server},
		},
	}
	if strings.TrimSpace(nextTag) != "" {
		outbound["proxySettings"] = map[string]interface{}{
			"tag": strings.TrimSpace(nextTag),
		}
	}
	return outbound
}

func (m *XrayManager) waitBridgeReady(log *logger.Logger, bridge *XrayBridge, cfgPath string, stderrPath string, stderrFile *os.File, attempt int) error {
	if err := waitPortReady("127.0.0.1", bridge.Port, m.bridgeStartTimeout()); err != nil {
		if stderrFile != nil {
			stderrFile.Close()
		}
		m.logBridgeStartupError(log, cfgPath, stderrPath)
		bridge.Stopping = true
		m.stopBridgeProcess(bridge)
		bridge.Running = false
		bridge.Pid = 0
		bridge.LastError = m.describeBridgeReadyError(err, cfgPath, stderrPath)
		log.Error("xray 端口不可用，重试", logger.F("key", bridge.NodeKey), logger.F("error", err), logger.F("port", bridge.Port), logger.F("attempt", attempt))
		time.Sleep(200 * time.Millisecond)
		return fmt.Errorf("%s", bridge.LastError)
	}
	if stderrFile != nil {
		stderrFile.Close()
	}
	return nil
}

func (m *XrayManager) bridgeStartTimeout() time.Duration {
	if m != nil && m.Config != nil && m.Config.ProxyCheck.BridgeStartTimeoutMs > 0 {
		return time.Duration(m.Config.ProxyCheck.BridgeStartTimeoutMs) * time.Millisecond
	}
	return 15 * time.Second
}

func (m *XrayManager) describeBridgeReadyError(err error, cfgPath string, stderrPath string) string {
	parts := []string{err.Error()}
	if strings.TrimSpace(cfgPath) != "" {
		parts = append(parts, "配置文件: "+cfgPath)
	}
	if tail := readLogTail(stderrPath, 1200); tail != "" {
		parts = append(parts, "stderr: "+tail)
	} else if cfgPath != "" {
		if tail := readLogTail(filepath.Join(filepath.Dir(cfgPath), "xray-error.log"), 1200); tail != "" {
			parts = append(parts, "error.log: "+tail)
		}
	}
	return strings.Join(parts, "；")
}

func readLogTail(path string, max int) string {
	if strings.TrimSpace(path) == "" || max <= 0 {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	text := strings.TrimSpace(string(data))
	if len(text) <= max {
		return text
	}
	return text[len(text)-max:]
}

func (m *XrayManager) logBridgeStartupError(log *logger.Logger, cfgPath string, stderrPath string) {
	if stderrContent, readErr := os.ReadFile(stderrPath); readErr == nil && len(stderrContent) > 0 {
		log.Error("xray stderr", logger.F("output", string(stderrContent)))
		return
	}

	errLogPath := filepath.Join(filepath.Dir(cfgPath), "xray-error.log")
	if errContent, readErr := os.ReadFile(errLogPath); readErr == nil && len(errContent) > 0 {
		log.Error("xray error.log", logger.F("output", string(errContent)))
	}
}
