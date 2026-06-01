package proxy

import (
	"ant-chrome/backend/internal/apppath"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func (m *XrayManager) buildRuntimeConfig(key string, outbound map[string]interface{}, port int, dnsServers string) (string, error) {
	return m.buildRuntimeConfigWithRoute(
		key,
		[]interface{}{outbound},
		[]interface{}{
			map[string]interface{}{
				"type":        "field",
				"inboundTag":  []string{"socks-in"},
				"outboundTag": "proxy-out",
			},
		},
		port,
		dnsServers,
	)
}

func (m *XrayManager) buildRuntimeConfigWithRoute(key string, outbounds []interface{}, rules []interface{}, port int, dnsServers string) (string, error) {
	baseDir := m.resolveWorkdir(key)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", err
	}

	preferIPv4 := m.Config.PreferIPv4Enabled()
	if preferIPv4 {
		// 主方案：把出站节点 server 域名预解析为 IPv4 字面量再写入配置，
		// 任何内核都不会再去查 AAAA。SNI/serverName 由 rewrite 内部保留。
		rewriteXrayOutboundsToIPv4(outbounds)
	}

	cfgPath := filepath.Join(baseDir, "xray-config.json")
	cfg := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "info",
			"error":    filepath.Join(baseDir, "xray-error.log"),
		},
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "socks-in",
				"port":     port,
				"listen":   "127.0.0.1",
				"protocol": "socks",
				"settings": map[string]interface{}{
					"udp": true,
				},
				"sniffing": map[string]interface{}{
					"enabled": false,
				},
			},
		},
		"outbounds": append(outbounds,
			map[string]interface{}{
				"protocol": "direct",
				"tag":      "direct",
			},
			map[string]interface{}{
				"protocol": "blackhole",
				"tag":      "block",
			},
		),
		"routing": map[string]interface{}{
			"rules": rules,
		},
	}
	dnsCfg := parseDnsConfig(dnsServers)
	if preferIPv4 {
		// 加固：顶层 queryStrategy=UseIPv4 让 xray 内部 DNS 只查 A 记录。
		// 与用户自定义 dns servers 合并，不覆盖用户配置。
		if dnsCfg == nil {
			dnsCfg = map[string]interface{}{}
		}
		dnsCfg["queryStrategy"] = "UseIPv4"
	}
	if dnsCfg != nil {
		cfg["dns"] = dnsCfg
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		return "", err
	}
	return cfgPath, nil
}

func (m *XrayManager) resolveWorkdir(key string) string {
	root := strings.TrimSpace(m.Config.Browser.UserDataRoot)
	if root == "" {
		root = "data"
	}
	if !filepath.IsAbs(root) {
		root = apppath.Resolve(m.AppRoot, root)
	}
	return filepath.Join(root, "_xray", key)
}
