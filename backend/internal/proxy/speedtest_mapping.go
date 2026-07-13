package proxy

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func proxyConfigToMapping(src string) (map[string]any, error) {
	src = strings.TrimSpace(src)
	l := strings.ToLower(src)

	if strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") {
		return parseStandardProxy(src, "http")
	}
	if strings.HasPrefix(l, "socks5://") {
		return parseStandardProxy(src, "socks5")
	}
	if strings.HasPrefix(l, "ss://") {
		return parseSSURIToMapping(src)
	}

	if strings.Contains(l, "://") && !strings.Contains(l, "type:") {
		return nil, fmt.Errorf("URI 格式暂不支持: %s", l[:min(30, len(l))])
	}

	return parseClashYAMLToMapping(src)
}

func parseStandardProxy(src string, proxyType string) (map[string]any, error) {
	parsed, err := url.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("代理地址解析失败: %w", err)
	}
	host := strings.TrimSpace(parsed.Hostname())
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		port = 0
	}
	if host == "" || port == 0 {
		return nil, fmt.Errorf("无法解析地址: %s", src)
	}
	username := ""
	password := ""
	if parsed.User != nil {
		username = parsed.User.Username()
		password, _ = parsed.User.Password()
	}

	mapping := map[string]any{
		"name":   "speedtest-proxy",
		"type":   proxyType,
		"server": host,
		"port":   port,
	}
	if username != "" {
		mapping["username"] = username
		mapping["password"] = password
	}
	return mapping, nil
}

func parseSSURIToMapping(src string) (map[string]any, error) {
	outbound, err := buildOutboundSS(src)
	if err != nil {
		return nil, err
	}
	settings, ok := outbound["settings"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("ss 节点缺少 settings")
	}
	server, err := firstShadowsocksServer(settings)
	if err != nil {
		return nil, err
	}
	mapping := map[string]any{
		"name":     "speedtest-proxy",
		"type":     "ss",
		"server":   server["address"],
		"port":     server["port"],
		"cipher":   server["method"],
		"password": server["password"],
	}
	return mapping, nil
}

func firstShadowsocksServer(settings map[string]interface{}) (map[string]interface{}, error) {
	servers, ok := settings["servers"].([]interface{})
	if !ok || len(servers) == 0 {
		return nil, fmt.Errorf("ss 节点缺少 servers")
	}
	server, ok := servers[0].(map[string]interface{})
	if !ok || server == nil {
		return nil, fmt.Errorf("ss server 格式无效")
	}
	return server, nil
}

func parseClashYAMLToMapping(src string) (map[string]any, error) {
	var payload interface{}
	if err := yaml.Unmarshal([]byte(src), &payload); err != nil {
		return nil, fmt.Errorf("YAML 解析失败: %v", err)
	}

	node := pickClashNode(payload)
	if node == nil {
		return nil, fmt.Errorf("无法提取 Clash 节点")
	}

	if _, ok := node["name"]; !ok {
		node["name"] = "speedtest-proxy"
	}

	return node, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
