package proxy

import (
	"fmt"
	"strings"

	"ant-chrome/backend/internal/config"
)

const (
	ProxyKernelAuto    = "auto"
	ProxyKernelNative  = "native"
	ProxyKernelXray    = "xray"
	ProxyKernelSingBox = "sing-box"
	ProxyKernelMihomo  = "mihomo"
)

type ProxyKernelResolution struct {
	Protocol         string   `json:"protocol"`
	PreferredKernel  string   `json:"preferredKernel"`
	Kernel           string   `json:"kernel"`
	SupportedKernels []string `json:"supportedKernels"`
	MissingCore      string   `json:"missingCore,omitempty"`
	Reason           string   `json:"reason"`
}

func NormalizePreferredKernel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", ProxyKernelAuto:
		return ""
	case ProxyKernelXray:
		return ProxyKernelXray
	case ProxyKernelSingBox, "singbox", "sing_box":
		return ProxyKernelSingBox
	case ProxyKernelMihomo, "clash", "clash-meta":
		return ProxyKernelMihomo
	case ProxyKernelNative:
		return ProxyKernelNative
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func ResolveProxyKernel(proxyConfig string, proxies []config.BrowserProxy, proxyId string, preferredKernel string) (ProxyKernelResolution, error) {
	src := strings.TrimSpace(resolveProxyConfig(proxyConfig, proxies, proxyId))
	if strings.TrimSpace(preferredKernel) == "" && strings.TrimSpace(proxyId) != "" {
		for _, item := range proxies {
			if strings.EqualFold(strings.TrimSpace(item.ProxyId), strings.TrimSpace(proxyId)) {
				preferredKernel = item.PreferredKernel
				break
			}
		}
	}
	preferred := NormalizePreferredKernel(preferredKernel)
	if preferred == "" {
		preferred = ProxyKernelAuto
	}
	resolution := ProxyKernelResolution{PreferredKernel: preferred}
	if src == "" || strings.EqualFold(src, "direct://") {
		resolution.Protocol = "direct"
		resolution.Kernel = ProxyKernelNative
		resolution.SupportedKernels = []string{ProxyKernelNative}
		resolution.Reason = "直连无需代理内核"
		return resolution, validatePreferredKernel(resolution, preferred)
	}

	protocol := DetectProxyProtocol(src)
	resolution.Protocol = protocol
	resolution.SupportedKernels = SupportedKernelsForProtocol(protocol, src, proxies, proxyId)
	if len(resolution.SupportedKernels) == 0 {
		return resolution, fmt.Errorf("不支持的代理协议: %s", protocol)
	}
	if preferred != ProxyKernelAuto {
		if !containsKernel(resolution.SupportedKernels, preferred) {
			return resolution, fmt.Errorf("协议 %s 不支持指定内核 %s", protocol, preferred)
		}
		resolution.Kernel = preferred
		resolution.Reason = "使用代理指定内核"
		return resolution, nil
	}
	resolution.Kernel = resolution.SupportedKernels[0]
	resolution.Reason = "按默认内核优先级自动选择"
	return resolution, nil
}

func ResolveProxyKernelForConnector(proxyConfig string, proxies []config.BrowserProxy, proxyId string, connectorType string) (ProxyKernelResolution, error) {
	src := strings.TrimSpace(resolveProxyConfig(proxyConfig, proxies, proxyId))
	preferredKernel := preferredKernelForConnector(src, proxies, proxyId, connectorType)
	return ResolveProxyKernel(src, proxies, proxyId, preferredKernel)
}

func DetectProxyProtocol(proxyConfig string) string {
	src := strings.TrimSpace(proxyConfig)
	l := strings.ToLower(src)
	if src == "" || strings.EqualFold(src, "direct://") {
		return "direct"
	}
	if strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") {
		return "http"
	}
	if strings.HasPrefix(l, "socks5://") {
		return "socks5"
	}
	if IsChainSocks5Proxy(src) {
		return "chain+socks5"
	}
	if nodeType := clashNodeType(src); nodeType != "" {
		return nodeType
	}
	for _, prefix := range []string{"vmess://", "vless://", "trojan://", "ss://", "ssr://", "hysteria2://", "hysteria://", "tuic://", "anytls://"} {
		if strings.HasPrefix(l, prefix) {
			return strings.TrimSuffix(prefix, "://")
		}
	}
	return "unknown"
}

func SupportedKernelsForProtocol(protocol string, proxyConfig string, proxies []config.BrowserProxy, proxyId string) []string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "direct", "http", "https", "socks5":
		return []string{ProxyKernelNative}
	case "vmess", "vless", "trojan", "ss", "shadowsocks", "chain+socks5":
		return []string{ProxyKernelXray, ProxyKernelMihomo}
	case "hysteria", "hysteria2", "tuic", "anytls":
		return []string{ProxyKernelSingBox, ProxyKernelMihomo}
	case "mieru":
		return []string{ProxyKernelMihomo}
	default:
		if RequiresLocalProxyBridgeForBrowser(proxyConfig) || RequiresBridge(proxyConfig, proxies, proxyId) {
			return []string{ProxyKernelXray, ProxyKernelMihomo}
		}
		if IsSingBoxProtocol(proxyConfig) {
			return []string{ProxyKernelSingBox, ProxyKernelMihomo}
		}
		if IsMihomoOnlyProtocol(proxyConfig) {
			return []string{ProxyKernelMihomo}
		}
		return nil
	}
}

func validatePreferredKernel(resolution ProxyKernelResolution, preferred string) error {
	if preferred == "" || preferred == ProxyKernelAuto {
		return nil
	}
	if !containsKernel(resolution.SupportedKernels, preferred) {
		return fmt.Errorf("协议 %s 不支持指定内核 %s", resolution.Protocol, preferred)
	}
	return nil
}

func containsKernel(kernels []string, kernel string) bool {
	kernel = NormalizePreferredKernel(kernel)
	for _, item := range kernels {
		if item == kernel {
			return true
		}
	}
	return false
}

func preferredKernelForConnector(src string, proxies []config.BrowserProxy, proxyId string, connectorType string) string {
	if config.NormalizeBrowserConnectorType(connectorType) != config.BrowserConnectorMihomo {
		return ""
	}
	if proxyHasExplicitPreferredKernel(proxies, proxyId) {
		return ""
	}
	src = strings.TrimSpace(resolveProxyConfig(src, proxies, proxyId))
	protocol := DetectProxyProtocol(src)
	if containsKernel(SupportedKernelsForProtocol(protocol, src, proxies, proxyId), ProxyKernelMihomo) {
		return ProxyKernelMihomo
	}
	return ""
}

func proxyHasExplicitPreferredKernel(proxies []config.BrowserProxy, proxyId string) bool {
	proxyId = strings.TrimSpace(proxyId)
	if proxyId == "" {
		return false
	}
	for _, item := range proxies {
		if strings.EqualFold(strings.TrimSpace(item.ProxyId), proxyId) {
			return NormalizePreferredKernel(item.PreferredKernel) != ""
		}
	}
	return false
}
