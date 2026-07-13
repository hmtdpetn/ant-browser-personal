package proxy

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func IsMihomoOnlyProtocol(proxyConfig string) bool {
	return mihomoOnlyProtocolType(proxyConfig) != ""
}

func mihomoOnlyProtocolType(proxyConfig string) string {
	nodeType := clashNodeType(proxyConfig)
	switch nodeType {
	case "mieru":
		return nodeType
	default:
		return ""
	}
}

func validateMihomoOnlyProtocol(proxyConfig string) error {
	src := strings.TrimSpace(proxyConfig)
	var payload interface{}
	if err := yaml.Unmarshal([]byte(src), &payload); err != nil {
		return fmt.Errorf("YAML 解析失败: %w", err)
	}
	node := pickClashNode(payload)
	if node == nil {
		return fmt.Errorf("mihomo 节点解析失败")
	}
	if getMapString(node, "server") == "" {
		return fmt.Errorf("mieru 节点缺少 server")
	}
	if getMapInt(node, "port") == 0 {
		return fmt.Errorf("mieru 节点缺少 port")
	}
	return nil
}

func clashNodeType(proxyConfig string) string {
	src := strings.TrimSpace(proxyConfig)
	if src == "" || (!strings.Contains(strings.ToLower(src), "type:") && !strings.Contains(strings.ToLower(src), "proxies:")) {
		return ""
	}
	var payload interface{}
	if err := yaml.Unmarshal([]byte(src), &payload); err != nil {
		return ""
	}
	node := pickClashNode(payload)
	if node == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(getMapString(node, "type")))
}
