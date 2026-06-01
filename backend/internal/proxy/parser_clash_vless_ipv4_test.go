package proxy

import "testing"

// TestClashVlessRealityForcesIPv4 验证 Clash VLESS Reality 节点生成的 Xray 出站
// streamSettings.sockopt.domainStrategy 被设置为 UseIPv4，避免域名解析到不可达的
// IPv6(AAAA) 端点导致出站超时。
func TestClashVlessRealityForcesIPv4(t *testing.T) {
	node := map[string]interface{}{
		"type":    "vless",
		"server":  "node.example.com",
		"port":    49602,
		"uuid":    "11111111-2222-3333-4444-555555555555",
		"flow":    "xtls-rprx-vision",
		"network": "tcp",
		"sni":     "example.com",
		"reality-opts": map[string]interface{}{
			"public-key": "iiMPm_l9kSjrEMxjXv15d1cIvDgAOfJFPBIJwx9sfRY",
			"short-id":   nil, // 上一阶段修复：null short-id 不应生成 "<nil>"
		},
	}

	out, _, err := buildOutboundFromClashVless(node)
	if err != nil {
		t.Fatalf("buildOutboundFromClashVless 失败: %v", err)
	}

	stream, ok := out["streamSettings"].(map[string]interface{})
	if !ok {
		t.Fatalf("streamSettings 缺失或类型错误: %#v", out["streamSettings"])
	}

	sockopt, ok := stream["sockopt"].(map[string]interface{})
	if !ok {
		t.Fatalf("streamSettings.sockopt 缺失或类型错误: %#v", stream["sockopt"])
	}
	if got := sockopt["domainStrategy"]; got != "UseIPv4" {
		t.Fatalf("domainStrategy = %v, 期望 UseIPv4", got)
	}

	// 回归：null short-id 不得写出 "<nil>"。
	reality, ok := stream["realitySettings"].(map[string]interface{})
	if !ok {
		t.Fatalf("realitySettings 缺失: %#v", stream["realitySettings"])
	}
	if sid, exists := reality["shortId"]; exists {
		t.Fatalf("空 short-id 不应写出 shortId，得到: %v", sid)
	}
}
