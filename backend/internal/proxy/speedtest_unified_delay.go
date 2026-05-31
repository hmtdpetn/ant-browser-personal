package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	C "github.com/metacubex/mihomo/constant"
)

// unifiedDelayTest 模拟 Clash unified-delay 模式：
// 1. 通过代理建立到目标的 TCP 连接（预热，不计入延迟）
// 2. 发送第一次 HTTP 请求预热连接（不计入延迟）
// 3. 在已建立的连接上发送第二次 HTTP 请求，只计这次的 RTT
// 这样测出的延迟 = 纯 HTTP 往返时间，和 Clash unified-delay: true 一致。
func unifiedDelayTest(proxyId string, px C.Proxy, testURL string, timeout time.Duration) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	addr, err := urlToMeta(testURL)
	if err != nil {
		return TestResult{ProxyId: proxyId, Ok: false, Error: fmt.Sprintf("URL 解析失败: %v", err)}
	}

	conn, err := px.DialContext(ctx, &addr)
	if err != nil {
		return TestResult{ProxyId: proxyId, Ok: false, Error: fmt.Sprintf("代理连接失败: %v", err)}
	}
	defer conn.Close()

	transport := &http.Transport{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			return conn, nil
		},
		DisableKeepAlives: false,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer client.CloseIdleConnections()

	req1, _ := http.NewRequestWithContext(ctx, http.MethodHead, testURL, nil)
	resp1, err := client.Do(req1)
	if err != nil {
		return TestResult{ProxyId: proxyId, Ok: false, Error: err.Error()}
	}
	resp1.Body.Close()

	start := time.Now()
	req2, _ := http.NewRequestWithContext(ctx, http.MethodHead, testURL, nil)
	resp2, err := client.Do(req2)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return TestResult{ProxyId: proxyId, Ok: false, LatencyMs: latency, Error: err.Error()}
	}
	resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK && resp2.StatusCode != http.StatusNoContent {
		return TestResult{
			ProxyId:   proxyId,
			Ok:        false,
			LatencyMs: latency,
			Error:     fmt.Sprintf("HTTP %d", resp2.StatusCode),
		}
	}

	return TestResult{ProxyId: proxyId, Ok: true, LatencyMs: latency}
}

// urlToMeta 将 URL 转换为 mihomo Metadata
func urlToMeta(rawURL string) (C.Metadata, error) {
	var host string
	var portNum uint16
	if strings.HasPrefix(rawURL, "https://") {
		host = rawURL[len("https://"):]
		portNum = 443
	} else if strings.HasPrefix(rawURL, "http://") {
		host = rawURL[len("http://"):]
		portNum = 80
	} else {
		return C.Metadata{}, fmt.Errorf("不支持的 URL scheme")
	}

	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	if h, p, err := net.SplitHostPort(host); err == nil {
		host = h
		fmt.Sscanf(p, "%d", &portNum)
	}

	meta := C.Metadata{
		Host:    host,
		DstPort: portNum,
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		meta.DstIP = addr
	}
	return meta, nil
}
