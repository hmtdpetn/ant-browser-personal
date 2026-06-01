package proxy

import (
	"os"
	"path/filepath"
	"testing"

	"ant-chrome/backend/internal/config"
)

// TestGenerateSampleConfigs 不是常规单测,而是供人工验证用的配置生成器。
// 用真实 DNS 解析(不打桩)从代表性 anytls / vless 节点生成 config,
// 写到 GEN_CONFIG_DIR 指定目录,供 sing-box check / xray -test 校验。
// 仅在设置了 GEN_CONFIG_DIR 时运行。
func TestGenerateSampleConfigs(t *testing.T) {
	outDir := os.Getenv("GEN_CONFIG_DIR")
	if outDir == "" {
		t.Skip("未设置 GEN_CONFIG_DIR,跳过样例配置生成")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("创建输出目录失败: %v", err)
	}

	// ---- anytls (sing-box) ----
	anytlsYAML := `proxies:
  - name: anytls-problem
    type: anytls
    server: dns.google
    port: 443
    password: REDACTED-PASSWORD
    sni: dns.google
    skip-cert-verify: false
`
	sbOut, err := BuildSingBoxOutbound(anytlsYAML)
	if err != nil {
		t.Fatalf("anytls 解析失败: %v", err)
	}
	sbm := &SingBoxManager{
		Config:  &config.Config{Browser: config.BrowserConfig{UserDataRoot: outDir}},
		AppRoot: outDir,
	}
	sbPath, err := sbm.buildConfig("anytls-sample", sbOut, 41080, "")
	if err != nil {
		t.Fatalf("sing-box buildConfig 失败: %v", err)
	}
	t.Logf("sing-box config 已生成: %s", sbPath)
	copyTo(t, sbPath, filepath.Join(outDir, "singbox-config.json"))

	// ---- vless reality (xray) ----
	vlessNode := map[string]interface{}{
		"type":    "vless",
		"server":  "dns.google",
		"port":    49602,
		"uuid":    "11111111-2222-3333-4444-555555555555",
		"flow":    "xtls-rprx-vision",
		"network": "tcp",
		"sni":     "dns.google",
		"reality-opts": map[string]interface{}{
			"public-key": "iiMPm_l9kSjrEMxjXv15d1cIvDgAOfJFPBIJwx9sfRY",
			"short-id":   nil,
		},
	}
	vlessOut, _, err := buildOutboundFromClashVless(vlessNode)
	if err != nil {
		t.Fatalf("vless 解析失败: %v", err)
	}
	xm := &XrayManager{
		Config:  &config.Config{Browser: config.BrowserConfig{UserDataRoot: outDir}},
		AppRoot: outDir,
	}
	routes := []interface{}{map[string]interface{}{"type": "field", "inboundTag": []string{"socks-in"}, "outboundTag": "proxy-out"}}
	xPath, err := xm.buildRuntimeConfigWithRoute("vless-sample", []interface{}{vlessOut}, routes, 41081, "")
	if err != nil {
		t.Fatalf("xray buildRuntimeConfigWithRoute 失败: %v", err)
	}
	t.Logf("xray config 已生成: %s", xPath)
	copyTo(t, xPath, filepath.Join(outDir, "xray-config.json"))
}

func copyTo(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("写入 %s 失败: %v", dst, err)
	}
}
