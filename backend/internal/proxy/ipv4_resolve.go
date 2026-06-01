package proxy

import (
	"context"
	"net"
	"strings"
	"time"

	"ant-chrome/backend/internal/logger"
)

// ipv4LookupTimeout 限制单次 A 记录解析耗时，避免阻塞桥接启动。
const ipv4LookupTimeout = 5 * time.Second

// lookupIPv4Fn 是 A 记录解析的可注入入口，便于测试在无网络环境下打桩。
var lookupIPv4Fn = lookupIPv4

// resolveDomainToIPv4 把域名解析为它的第一个 IPv4(A 记录)字面量。
//
// 返回值：
//   - (ipv4, true)  解析成功，调用方可用 ipv4 替换节点 server 字段；
//   - ("", false)   host 为空、本身已是 IP 字面量、解析超时或无 A 记录。
//     调用方应保留原域名，不做替换。
//
// 注意：本函数只负责解析 server 拨号地址，SNI / Host / Reality serverName
// 等字段由调用方单独保留为原域名，不得用本函数的结果覆盖。
func resolveDomainToIPv4(host string) (string, bool) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", false
	}
	// 已经是 IP 字面量（含 IPv6），无需解析。
	if ip := net.ParseIP(host); ip != nil {
		return "", false
	}

	ips, err := lookupIPv4Fn(host)
	if err != nil || len(ips) == 0 {
		log := logger.New("ProxyDNS")
		errMsg := "no A record"
		if err != nil {
			errMsg = err.Error()
		}
		log.Warn("IPv4 预解析失败，保留原域名", logger.F("host", host), logger.F("error", errMsg))
		return "", false
	}
	return ips[0], true
}

// lookupIPv4 仅查询 A 记录，带超时控制。
func lookupIPv4(host string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ipv4LookupTimeout)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if v4 := a.To4(); v4 != nil {
			out = append(out, v4.String())
		}
	}
	return out, nil
}

// rewriteXrayOutboundsToIPv4 遍历 xray 出站列表，把节点 server 域名替换为 IPv4。
// 覆盖 vless/vmess(settings.vnext[].address) 与 trojan/ss/socks/http(settings.servers[].address)。
// 替换后若对应的 TLS/Reality serverName 为空，则回填原域名以保留正确 SNI。
func rewriteXrayOutboundsToIPv4(outbounds []interface{}) {
	for _, ob := range outbounds {
		if m, ok := ob.(map[string]interface{}); ok {
			rewriteXrayOutboundToIPv4(m)
		}
	}
}

func rewriteXrayOutboundToIPv4(ob map[string]interface{}) {
	settings, ok := ob["settings"].(map[string]interface{})
	if !ok {
		return
	}

	origDomain := ""
	replace := func(list []interface{}) {
		for _, item := range list {
			entry, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			addr, ok := entry["address"].(string)
			if !ok {
				continue
			}
			if ipv4, ok := resolveDomainToIPv4(addr); ok {
				if origDomain == "" {
					origDomain = strings.TrimSpace(addr)
				}
				entry["address"] = ipv4
			}
		}
	}

	if v, ok := settings["vnext"].([]interface{}); ok {
		replace(v)
	}
	if v, ok := settings["servers"].([]interface{}); ok {
		replace(v)
	}

	if origDomain != "" {
		ensureXrayServerName(ob, origDomain)
	}
}

// ensureXrayServerName 在 server 被替换成 IP 后，确保 streamSettings 里的
// serverName(SNI) 仍是原域名；仅在 serverName 缺省时回填，不覆盖用户显式配置。
func ensureXrayServerName(ob map[string]interface{}, domain string) {
	stream, ok := ob["streamSettings"].(map[string]interface{})
	if !ok {
		return
	}
	for _, key := range []string{"tlsSettings", "realitySettings"} {
		s, ok := stream[key].(map[string]interface{})
		if !ok {
			continue
		}
		if name, _ := s["serverName"].(string); strings.TrimSpace(name) == "" {
			s["serverName"] = domain
		}
	}
}

// rewriteSingBoxOutboundToIPv4 把 sing-box 出站的 server 域名替换为 IPv4。
// 替换前若 tls.server_name 为空，则回填原域名以保留正确 SNI。
func rewriteSingBoxOutboundToIPv4(ob map[string]interface{}) {
	server, ok := ob["server"].(string)
	if !ok {
		return
	}
	ipv4, ok := resolveDomainToIPv4(server)
	if !ok {
		return
	}
	if tls, ok := ob["tls"].(map[string]interface{}); ok {
		if enabled, _ := tls["enabled"].(bool); enabled {
			if name, _ := tls["server_name"].(string); strings.TrimSpace(name) == "" {
				tls["server_name"] = strings.TrimSpace(server)
			}
		}
	}
	ob["server"] = ipv4
}

// dnsServerAddrs 从用户填写的 DnsServers 原始字符串中提取可用的 DNS 地址列表。
// 复用 xray 侧的解析逻辑（兼容 Clash dns YAML 与逗号分隔 IP 列表）。
func dnsServerAddrs(raw string) []string {
	cfg := parseDnsConfig(raw)
	if cfg == nil {
		return nil
	}
	servers, ok := cfg["servers"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(servers))
	for _, s := range servers {
		if str, ok := s.(string); ok && strings.TrimSpace(str) != "" {
			out = append(out, str)
		}
	}
	return out
}
