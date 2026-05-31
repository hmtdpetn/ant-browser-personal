package proxy

import "fmt"

func buildOutboundFromClashTrojan(node map[string]interface{}) (map[string]interface{}, string, error) {
	host := getMapString(node, "server")
	port := getMapInt(node, "port")
	password := getMapString(node, "password")
	sni := getMapString(node, "sni")
	if sni == "" {
		sni = getMapString(node, "servername")
	}
	network := getMapString(node, "network")
	skipVerify := getMapBool(node, "skip-cert-verify")

	out := map[string]interface{}{
		"protocol": "trojan",
		"tag":      "proxy-out",
		"settings": map[string]interface{}{
			"address":  host,
			"port":     port,
			"password": password,
		},
	}
	stream := map[string]interface{}{
		"security": "tls",
		"tlsSettings": map[string]interface{}{
			"serverName":    sni,
			"allowInsecure": skipVerify,
		},
	}
	if network == "ws" {
		stream["network"] = "ws"
		ws := map[string]interface{}{}
		if wsOpts, ok := node["ws-opts"]; ok {
			if wsMap := toStringMap(wsOpts); wsMap != nil {
				if path := getMapString(wsMap, "path"); path != "" {
					ws["path"] = path
				}
				if headers := toStringMap(wsMap["headers"]); headers != nil {
					if h := getMapString(headers, "Host"); h != "" {
						ws["headers"] = map[string]interface{}{"Host": h}
					}
				}
			}
		}
		stream["wsSettings"] = ws
	} else if network == "grpc" {
		stream["network"] = "grpc"
		if grpcOpts, ok := node["grpc-opts"]; ok {
			if grpcMap := toStringMap(grpcOpts); grpcMap != nil {
				if svcName := getMapString(grpcMap, "grpc-service-name"); svcName != "" {
					stream["grpcSettings"] = map[string]interface{}{"serviceName": svcName}
				}
			}
		}
	}
	out["streamSettings"] = stream
	return out, "", nil
}

func buildOutboundFromClashHysteria2(node map[string]interface{}) (map[string]interface{}, string, error) {
	return nil, "", fmt.Errorf("Xray 不支持 hysteria2 协议，请使用 vless/vmess/socks5/http 格式的代理")
}

// buildOutboundFromClashSS 从 Clash YAML 格式解析 Shadowsocks outbound
func buildOutboundFromClashSS(node map[string]interface{}) (map[string]interface{}, string, error) {
	host := getMapString(node, "server")
	port := getMapInt(node, "port")
	password := getMapString(node, "password")
	cipher := getMapString(node, "cipher")
	if cipher == "" {
		cipher = getMapString(node, "method")
	}
	if cipher == "" {
		cipher = "aes-256-gcm"
	}
	out := map[string]interface{}{
		"protocol": "shadowsocks",
		"tag":      "proxy-out",
		"settings": map[string]interface{}{
			"address":  host,
			"port":     port,
			"method":   cipher,
			"password": password,
		},
	}
	if plugin := getMapString(node, "plugin"); plugin != "" {
		pluginOpts := getMapString(node, "plugin-opts")
		_ = pluginOpts
	}
	return out, "", nil
}
