package proxy

import (
	"encoding/json"
	"os"
	"testing"

	"ant-chrome/backend/internal/config"
)

// stubLookupIPv4 在测试期间把 A 记录解析打桩为固定结果，避免依赖真实网络。
func stubLookupIPv4(t *testing.T, mapping map[string][]string) {
	t.Helper()
	prev := lookupIPv4Fn
	lookupIPv4Fn = func(host string) ([]string, error) {
		if ips, ok := mapping[host]; ok {
			return ips, nil
		}
		return nil, nil // 无 A 记录
	}
	t.Cleanup(func() { lookupIPv4Fn = prev })
}

func TestResolveDomainToIPv4(t *testing.T) {
	stubLookupIPv4(t, map[string][]string{
		"node.example.com": {"203.0.113.7"},
	})

	if ip, ok := resolveDomainToIPv4("node.example.com"); !ok || ip != "203.0.113.7" {
		t.Fatalf("域名解析期望 203.0.113.7,得到 ip=%q ok=%v", ip, ok)
	}
	// 已是 IP 字面量,不应改写。
	if _, ok := resolveDomainToIPv4("198.51.100.9"); ok {
		t.Fatalf("IPv4 字面量不应被再次解析")
	}
	if _, ok := resolveDomainToIPv4("2001:db8::1"); ok {
		t.Fatalf("IPv6 字面量不应被解析为 IPv4")
	}
	// 无 A 记录时保留原域名。
	if _, ok := resolveDomainToIPv4("noaaaa.example.org"); ok {
		t.Fatalf("无 A 记录应返回 false")
	}
}

// TestRewriteSingBoxOutboundPreservesSNI 验证 anytls/sing-box 出站 server 被替换为
// IPv4 的同时,空的 tls.server_name 被回填为原域名(保留正确 SNI)。
func TestRewriteSingBoxOutboundPreservesSNI(t *testing.T) {
	stubLookupIPv4(t, map[string][]string{
		"anytls.example.com": {"203.0.113.20"},
	})

	out := map[string]interface{}{
		"type":   "anytls",
		"server": "anytls.example.com",
		"tls": map[string]interface{}{
			"enabled": true,
			// server_name 故意留空,模拟节点未显式给 sni 的情况。
		},
	}
	rewriteSingBoxOutboundToIPv4(out)

	if out["server"] != "203.0.113.20" {
		t.Fatalf("server 应替换为 IPv4,得到 %v", out["server"])
	}
	tls := out["tls"].(map[string]interface{})
	if tls["server_name"] != "anytls.example.com" {
		t.Fatalf("空 server_name 应回填原域名,得到 %v", tls["server_name"])
	}
}

func TestRewriteSingBoxOutboundKeepsExplicitSNI(t *testing.T) {
	stubLookupIPv4(t, map[string][]string{
		"anytls.example.com": {"203.0.113.20"},
	})
	out := map[string]interface{}{
		"type":   "anytls",
		"server": "anytls.example.com",
		"tls": map[string]interface{}{
			"enabled":     true,
			"server_name": "custom.sni.example",
		},
	}
	rewriteSingBoxOutboundToIPv4(out)
	tls := out["tls"].(map[string]interface{})
	if tls["server_name"] != "custom.sni.example" {
		t.Fatalf("显式 server_name 不应被覆盖,得到 %v", tls["server_name"])
	}
}

// TestRewriteXrayOutboundPreservesServerName 验证 vless/reality 出站 address 被替换为
// IPv4 的同时,空的 realitySettings.serverName 被回填为原域名。
func TestRewriteXrayOutboundPreservesServerName(t *testing.T) {
	stubLookupIPv4(t, map[string][]string{
		"node.example.com": {"203.0.113.30"},
	})
	out := map[string]interface{}{
		"protocol": "vless",
		"settings": map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{"address": "node.example.com", "port": 443},
			},
		},
		"streamSettings": map[string]interface{}{
			"realitySettings": map[string]interface{}{
				"publicKey": "k",
				// serverName 留空
			},
		},
	}
	rewriteXrayOutboundsToIPv4([]interface{}{out})

	vnext := out["settings"].(map[string]interface{})["vnext"].([]interface{})
	addr := vnext[0].(map[string]interface{})["address"]
	if addr != "203.0.113.30" {
		t.Fatalf("vnext address 应替换为 IPv4,得到 %v", addr)
	}
	reality := out["streamSettings"].(map[string]interface{})["realitySettings"].(map[string]interface{})
	if reality["serverName"] != "node.example.com" {
		t.Fatalf("空 serverName 应回填原域名,得到 %v", reality["serverName"])
	}
}

func TestDnsServerAddrs(t *testing.T) {
	got := dnsServerAddrs("1.1.1.1, 8.8.8.8")
	if len(got) != 2 || got[0] != "1.1.1.1" || got[1] != "8.8.8.8" {
		t.Fatalf("dnsServerAddrs 解析逗号列表失败: %#v", got)
	}
	if addrs := dnsServerAddrs(""); len(addrs) != 0 {
		t.Fatalf("空输入应返回空,得到 %#v", addrs)
	}
}

// TestSingBoxBuildConfigForcesIPv4 用 IP 字面量 server(避免真实 DNS)端到端验证
// buildConfig 写出的 singbox-config.json 包含 ipv4_only 策略,且默认 DNS 服务器。
func TestSingBoxBuildConfigForcesIPv4(t *testing.T) {
	dir := t.TempDir()
	m := &SingBoxManager{
		Config:  &config.Config{Browser: config.BrowserConfig{UserDataRoot: dir}},
		AppRoot: dir,
	}
	outbound := map[string]interface{}{
		"type":        "anytls",
		"tag":         "proxy-out",
		"server":      "198.51.100.5", // 已是 IP,buildConfig 不会触发 DNS
		"server_port": 443,
		"tls":         map[string]interface{}{"enabled": true, "server_name": "anytls.example.com"},
	}

	cfgPath, err := m.buildConfig("k", outbound, 12345, "")
	if err != nil {
		t.Fatalf("buildConfig 失败: %v", err)
	}
	cfg := readJSONConfig(t, cfgPath)

	outbounds := cfg["outbounds"].([]interface{})
	proxy := outbounds[0].(map[string]interface{})
	if proxy["domain_strategy"] != "ipv4_only" {
		t.Fatalf("proxy outbound domain_strategy 应为 ipv4_only,得到 %v", proxy["domain_strategy"])
	}
	if proxy["server"] != "198.51.100.5" {
		t.Fatalf("IP 字面量 server 不应被改动,得到 %v", proxy["server"])
	}
	if sni := proxy["tls"].(map[string]interface{})["server_name"]; sni != "anytls.example.com" {
		t.Fatalf("server_name 应保留原域名,得到 %v", sni)
	}
	dns := cfg["dns"].(map[string]interface{})
	if dns["strategy"] != "ipv4_only" {
		t.Fatalf("dns.strategy 应为 ipv4_only,得到 %v", dns["strategy"])
	}
	servers := dns["servers"].([]interface{})
	if len(servers) != 2 {
		t.Fatalf("默认 DNS 服务器应为 2 个,得到 %d", len(servers))
	}
}

// TestSingBoxBuildConfigToggleOff 验证 prefer_ipv4=false 时恢复原行为。
func TestSingBoxBuildConfigToggleOff(t *testing.T) {
	dir := t.TempDir()
	off := false
	m := &SingBoxManager{
		Config: &config.Config{
			App:     config.AppConfig{PreferIPv4: &off},
			Browser: config.BrowserConfig{UserDataRoot: dir},
		},
		AppRoot: dir,
	}
	outbound := map[string]interface{}{
		"type":        "anytls",
		"tag":         "proxy-out",
		"server":      "198.51.100.5",
		"server_port": 443,
	}
	cfgPath, err := m.buildConfig("k", outbound, 12346, "")
	if err != nil {
		t.Fatalf("buildConfig 失败: %v", err)
	}
	cfg := readJSONConfig(t, cfgPath)
	if _, ok := cfg["dns"]; ok {
		t.Fatalf("关闭开关时不应写入 dns 块")
	}
	proxy := cfg["outbounds"].([]interface{})[0].(map[string]interface{})
	if _, ok := proxy["domain_strategy"]; ok {
		t.Fatalf("关闭开关时不应写入 domain_strategy")
	}
}

func readJSONConfig(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("解析配置 JSON 失败: %v", err)
	}
	return cfg
}
