package proxy

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"ant-chrome/backend/internal/config"
)

// TestConnectivity 通过 TCP 握手测试代理服务器的可达性和延迟
// 直接对 server:port 建立 TCP 连接测量 RTT，无需启动外部进程
func TestConnectivity(proxyId string, proxyConfig string, proxies []config.BrowserProxy, _ interface{}) TestResult {
	src := strings.TrimSpace(proxyConfig)
	if proxyId != "" {
		for _, item := range proxies {
			if strings.EqualFold(item.ProxyId, proxyId) {
				src = strings.TrimSpace(item.ProxyConfig)
				break
			}
		}
	}
	if src == "" {
		return TestResult{ProxyId: proxyId, Ok: false, Error: "代理配置为空"}
	}

	endpoint, err := proxyEndpoint(src)
	if err != nil {
		return TestResult{ProxyId: proxyId, Ok: false, Error: fmt.Sprintf("地址解析失败: %v", err)}
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", endpoint, 10*time.Second)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return TestResult{ProxyId: proxyId, Ok: false, LatencyMs: latency, Error: err.Error()}
	}
	conn.Close()
	return TestResult{ProxyId: proxyId, Ok: true, LatencyMs: latency}
}

// TestRealConnectivity 通过代理链路发起真实 HTTP 请求测量端到端延迟。
// - DirectProxy (http/https/socks5)：直接通过该代理发送请求
// - BridgeProxy (vmess/vless/Clash)：调用 EnsureBridge 获取 socks5 地址后发送请求
// - SingBoxProxy (hysteria2/tuic)：调用 SingBoxManager.EnsureBridge 后发送请求
func TestRealConnectivity(
	proxyId string,
	proxies []config.BrowserProxy,
	xrayMgr *XrayManager,
) TestResult {
	return TestRealConnectivityWithSingBox(proxyId, proxies, xrayMgr, nil)
}

// TestRealConnectivityWithSingBox 支持 sing-box 的真实连通性测试
func TestRealConnectivityWithSingBox(
	proxyId string,
	proxies []config.BrowserProxy,
	xrayMgr *XrayManager,
	singboxMgr *SingBoxManager,
) TestResult {
	src := resolveProxyConfig("", proxies, proxyId)
	if src == "" {
		return TestResult{ProxyId: proxyId, Ok: false, Error: "代理配置为空"}
	}

	targetURL := strings.TrimSpace(DefaultSpeedTestURL)
	if targetURL == "" {
		return TestResult{ProxyId: proxyId, Ok: false, Error: "真实连通性测试目标 URL 为空"}
	}
	const timeout = 15 * time.Second

	client, err := buildProxyHTTPClient(src, proxyId, proxies, xrayMgr, singboxMgr, timeout)
	if err != nil {
		return TestResult{ProxyId: proxyId, Ok: false, Error: err.Error()}
	}

	start := time.Now()
	resp, err := client.Get(targetURL)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return TestResult{ProxyId: proxyId, Ok: false, LatencyMs: latency, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return TestResult{ProxyId: proxyId, Ok: false, LatencyMs: latency, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	return TestResult{ProxyId: proxyId, Ok: true, LatencyMs: latency}
}
