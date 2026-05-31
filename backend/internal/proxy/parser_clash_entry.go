package proxy

import (
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

func parseClashNode(src string) (map[string]interface{}, string, error) {
	data := strings.TrimSpace(src)
	if strings.HasPrefix(strings.ToLower(data), "clash://") {
		raw := strings.TrimPrefix(data, "clash://")
		raw, _ = url.QueryUnescape(raw)
		decoded, err := decodeBase64String(raw)
		if err != nil {
			return nil, "", err
		}
		data = string(decoded)
	}
	var payload interface{}
	if err := yaml.Unmarshal([]byte(data), &payload); err != nil {
		return nil, "", err
	}
	nodeMap := pickClashNode(payload)
	if nodeMap == nil {
		return nil, "", fmt.Errorf("clash 节点解析失败")
	}
	nodeType := strings.ToLower(getMapString(nodeMap, "type"))
	switch nodeType {
	case "socks5", "http", "https":
		return nil, buildStandardProxyFromClash(nodeMap, nodeType), nil
	case "vmess":
		return buildOutboundFromClashVmess(nodeMap)
	case "vless":
		return buildOutboundFromClashVless(nodeMap)
	case "trojan":
		return buildOutboundFromClashTrojan(nodeMap)
	case "ss", "shadowsocks":
		return buildOutboundFromClashSS(nodeMap)
	case "ssr":
		return nil, "", fmt.Errorf("不支持 ShadowsocksR 协议，Xray 不支持 SSR，请使用 SS/vmess/vless/trojan")
	case "hysteria2", "hysteria":
		return buildOutboundFromClashHysteria2(nodeMap)
	}
	return nil, "", fmt.Errorf("不支持的节点类型")
}

func pickClashNode(payload interface{}) map[string]interface{} {
	if m := toStringMap(payload); m != nil {
		if proxies, ok := m["proxies"]; ok {
			if arr, ok := proxies.([]interface{}); ok && len(arr) > 0 {
				return toStringMap(arr[0])
			}
		}
		if proxyItem, ok := m["proxy"]; ok {
			if node := toStringMap(proxyItem); node != nil {
				return node
			}
		}
		return m
	}
	if arr, ok := payload.([]interface{}); ok && len(arr) > 0 {
		return toStringMap(arr[0])
	}
	return nil
}

func buildStandardProxyFromClash(node map[string]interface{}, scheme string) string {
	host := getMapString(node, "server")
	port := getMapInt(node, "port")
	username := getMapString(node, "username")
	password := getMapString(node, "password")
	if host == "" || port == 0 {
		return ""
	}
	address := fmt.Sprintf("%s:%d", host, port)
	if username != "" {
		user := url.UserPassword(username, password)
		return fmt.Sprintf("%s://%s@%s", scheme, user.String(), address)
	}
	return fmt.Sprintf("%s://%s", scheme, address)
}
