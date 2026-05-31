package proxy

import (
	"strings"
	"time"

	"github.com/metacubex/mihomo/adapter"

	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/logger"
)

// ─── Clash 标准测速 URL ───
// 使用 HTTP 与 Clash 客户端保持一致

const DefaultSpeedTestURL = "http://www.gstatic.com/generate_204"

// SpeedTestConfig 测速参数
type SpeedTestConfig struct {
	Timeout    time.Duration
	TCPTimeout time.Duration
	URLs       []string
}

var DefaultSpeedTestConfig = SpeedTestConfig{
	Timeout:    10 * time.Second,
	TCPTimeout: 5 * time.Second,
}

// ─── 对外入口 ───

// SpeedTest 使用 mihomo 代理适配器进行测速。
// 采用 unified-delay 策略：先建立连接（预热），再单独计时 HTTP 往返，
// 与 Clash 客户端 unified-delay: true 的延迟结果一致。
func SpeedTest(
	proxyId string,
	proxies []config.BrowserProxy,
	xrayMgr *XrayManager,
	singboxMgr *SingBoxManager,
	cfg *SpeedTestConfig,
) TestResult {
	log := logger.New("SpeedTest")

	if cfg == nil {
		c := DefaultSpeedTestConfig
		cfg = &c
	}

	src := resolveProxyConfig("", proxies, proxyId)
	if src == "" {
		return TestResult{ProxyId: proxyId, Ok: false, Error: "代理配置为空"}
	}

	if strings.ToLower(src) == "direct://" {
		return TestResult{ProxyId: proxyId, Ok: true, LatencyMs: 0}
	}

	testURL := strings.TrimSpace(DefaultSpeedTestURL)
	if len(cfg.URLs) > 0 {
		testURL = strings.TrimSpace(cfg.URLs[0])
	}
	if testURL == "" {
		return TestResult{ProxyId: proxyId, Ok: false, Error: "测速目标 URL 为空"}
	}

	resolvedSrc := src
	if IsChainSocks5Proxy(src) {
		if xrayMgr == nil {
			log.Warn("链式代理测速缺少 Xray 管理器，降级到 TCP ping",
				logger.F("proxy_id", proxyId),
			)
			return tcpPingFallback(proxyId, src, cfg.TCPTimeout, log)
		}
		bridgeSocksURL, bridgeErr := xrayMgr.EnsureBridge(src, proxies, proxyId)
		if bridgeErr != nil {
			log.Warn("链式代理桥接失败，降级到 TCP ping",
				logger.F("proxy_id", proxyId),
				logger.F("error", bridgeErr.Error()),
			)
			return tcpPingFallback(proxyId, src, cfg.TCPTimeout, log)
		}
		resolvedSrc = strings.TrimSpace(bridgeSocksURL)
	}

	mapping, err := proxyConfigToMapping(resolvedSrc)
	if err != nil {
		log.Warn("代理配置解析失败，降级到 TCP ping",
			logger.F("proxy_id", proxyId),
			logger.F("error", err.Error()),
		)
		return tcpPingFallback(proxyId, resolvedSrc, cfg.TCPTimeout, log)
	}

	proxyInstance, err := adapter.ParseProxy(mapping)
	if err != nil {
		log.Warn("mihomo 代理创建失败，降级到 TCP ping",
			logger.F("proxy_id", proxyId),
			logger.F("error", err.Error()),
			logger.F("type", mapping["type"]),
		)
		return tcpPingFallback(proxyId, resolvedSrc, cfg.TCPTimeout, log)
	}

	return unifiedDelayTest(proxyId, proxyInstance, testURL, cfg.Timeout)
}
