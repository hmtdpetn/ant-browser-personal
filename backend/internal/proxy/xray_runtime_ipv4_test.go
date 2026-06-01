package proxy

import (
	"testing"

	"ant-chrome/backend/internal/config"
)

// TestXrayBuildConfigForcesIPv4 端到端验证 buildRuntimeConfigWithRoute 写出的
// xray-config.json 顶层含 dns.queryStrategy=UseIPv4,且 vnext address 被替换为 IPv4,
// 而空 serverName 被回填为原域名(保留 SNI)。
func TestXrayBuildConfigForcesIPv4(t *testing.T) {
	stubLookupIPv4(t, map[string][]string{
		"node.example.com": {"203.0.113.40"},
	})
	dir := t.TempDir()
	m := &XrayManager{
		Config:  &config.Config{Browser: config.BrowserConfig{UserDataRoot: dir}},
		AppRoot: dir,
	}
	outbound := map[string]interface{}{
		"protocol": "vless",
		"tag":      "proxy-out",
		"settings": map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{"address": "node.example.com", "port": 443},
			},
		},
		"streamSettings": map[string]interface{}{
			"security":        "reality",
			"realitySettings": map[string]interface{}{"publicKey": "k"},
		},
	}
	routes := []interface{}{map[string]interface{}{"type": "field", "inboundTag": []string{"socks-in"}, "outboundTag": "proxy-out"}}

	cfgPath, err := m.buildRuntimeConfigWithRoute("k", []interface{}{outbound}, routes, 23456, "")
	if err != nil {
		t.Fatalf("buildRuntimeConfigWithRoute 失败: %v", err)
	}
	cfg := readJSONConfig(t, cfgPath)

	dns := cfg["dns"].(map[string]interface{})
	if dns["queryStrategy"] != "UseIPv4" {
		t.Fatalf("dns.queryStrategy 应为 UseIPv4,得到 %v", dns["queryStrategy"])
	}
	ob := cfg["outbounds"].([]interface{})[0].(map[string]interface{})
	addr := ob["settings"].(map[string]interface{})["vnext"].([]interface{})[0].(map[string]interface{})["address"]
	if addr != "203.0.113.40" {
		t.Fatalf("vnext address 应替换为 IPv4,得到 %v", addr)
	}
	reality := ob["streamSettings"].(map[string]interface{})["realitySettings"].(map[string]interface{})
	if reality["serverName"] != "node.example.com" {
		t.Fatalf("空 serverName 应回填原域名,得到 %v", reality["serverName"])
	}
}

// TestXrayBuildConfigMergesUserDns 验证用户自定义 dns_servers 与 queryStrategy 合并,
// 不互相覆盖。
func TestXrayBuildConfigMergesUserDns(t *testing.T) {
	dir := t.TempDir()
	m := &XrayManager{
		Config:  &config.Config{Browser: config.BrowserConfig{UserDataRoot: dir}},
		AppRoot: dir,
	}
	outbound := map[string]interface{}{"protocol": "vless", "tag": "proxy-out", "settings": map[string]interface{}{}}
	routes := []interface{}{map[string]interface{}{"type": "field", "inboundTag": []string{"socks-in"}, "outboundTag": "proxy-out"}}

	cfgPath, err := m.buildRuntimeConfigWithRoute("k", []interface{}{outbound}, routes, 23457, "8.8.4.4")
	if err != nil {
		t.Fatalf("buildRuntimeConfigWithRoute 失败: %v", err)
	}
	cfg := readJSONConfig(t, cfgPath)
	dns := cfg["dns"].(map[string]interface{})
	if dns["queryStrategy"] != "UseIPv4" {
		t.Fatalf("queryStrategy 应为 UseIPv4,得到 %v", dns["queryStrategy"])
	}
	servers := dns["servers"].([]interface{})
	if len(servers) != 1 || servers[0] != "8.8.4.4" {
		t.Fatalf("用户自定义 dns servers 应保留,得到 %#v", servers)
	}
}
