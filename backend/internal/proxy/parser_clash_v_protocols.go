package proxy

import "strings"

func buildOutboundFromClashVless(node map[string]interface{}) (map[string]interface{}, string, error) {
	host := getMapString(node, "server")
	port := getMapInt(node, "port")
	id := getMapString(node, "uuid")
	flow := getMapString(node, "flow")
	sni := getMapString(node, "sni")
	if sni == "" {
		sni = getMapString(node, "servername")
	}
	network := getMapString(node, "network")
	out := map[string]interface{}{
		"protocol": "vless",
		"tag":      "proxy-out",
		"settings": map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": host,
					"port":    port,
					"users": []interface{}{
						map[string]interface{}{
							"id":         id,
							"flow":       flow,
							"encryption": "none",
						},
					},
				},
			},
		},
	}
	stream := map[string]interface{}{}
	tlsVal := strings.ToLower(getMapString(node, "tls"))
	_, hasRealityOpts := node["reality-opts"]

	if hasRealityOpts {
		stream["network"] = "tcp"
		realityOpts := map[string]interface{}{
			"spiderX": "",
		}
		if sni != "" {
			realityOpts["serverName"] = sni
		}
		fingerprint := getMapString(node, "client-fingerprint")
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		realityOpts["fingerprint"] = fingerprint
		if rm := toStringMap(node["reality-opts"]); rm != nil {
			if pbk := getMapString(rm, "public-key"); pbk != "" {
				realityOpts["publicKey"] = pbk
			}
			if sid := getMapString(rm, "short-id"); sid != "" {
				realityOpts["shortId"] = sid
			}
		}
		stream["security"] = "reality"
		stream["realitySettings"] = realityOpts
	} else if getMapBool(node, "tls") || tlsVal == "true" || tlsVal == "tls" {
		tlsSettings := map[string]interface{}{}
		if sni != "" {
			tlsSettings["serverName"] = sni
		}
		tlsSettings["allowInsecure"] = getMapBool(node, "skip-cert-verify")
		stream["security"] = "tls"
		stream["tlsSettings"] = tlsSettings
	}
	if network == "ws" {
		stream["network"] = "ws"
		ws := map[string]interface{}{}
		if wsOpts, ok := node["ws-opts"]; ok {
			if wsMap := toStringMap(wsOpts); wsMap != nil {
				path := getMapString(wsMap, "path")
				if path != "" {
					ws["path"] = path
				}
				if headers, ok := wsMap["headers"]; ok {
					if headerMap := toStringMap(headers); headerMap != nil {
						if hostH := getMapString(headerMap, "Host"); hostH != "" {
							ws["headers"] = map[string]interface{}{"Host": hostH}
						}
					}
				}
			}
		}
		stream["wsSettings"] = ws
	}
	if network == "grpc" {
		stream["network"] = "grpc"
		if grpcOpts, ok := node["grpc-opts"]; ok {
			if grpcMap := toStringMap(grpcOpts); grpcMap != nil {
				serviceName := getMapString(grpcMap, "grpc-service-name")
				if serviceName != "" {
					stream["grpcSettings"] = map[string]interface{}{"serviceName": serviceName}
				}
			}
		}
	}
	if len(stream) > 0 {
		out["streamSettings"] = stream
	}
	return out, "", nil
}

func buildOutboundFromClashVmess(node map[string]interface{}) (map[string]interface{}, string, error) {
	host := getMapString(node, "server")
	port := getMapInt(node, "port")
	id := getMapString(node, "uuid")
	cipher := getMapString(node, "cipher")
	if cipher == "" {
		cipher = "auto"
	}
	network := getMapString(node, "network")
	sni := getMapString(node, "sni")
	if sni == "" {
		sni = getMapString(node, "servername")
	}
	out := map[string]interface{}{
		"protocol": "vmess",
		"tag":      "proxy-out",
		"settings": map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": host,
					"port":    port,
					"users": []interface{}{
						map[string]interface{}{
							"id":       id,
							"security": cipher,
						},
					},
				},
			},
		},
	}
	stream := map[string]interface{}{}
	if getMapBool(node, "tls") || strings.ToLower(getMapString(node, "tls")) == "true" {
		tlsSettings := map[string]interface{}{}
		if sni != "" {
			tlsSettings["serverName"] = sni
		}
		tlsSettings["allowInsecure"] = getMapBool(node, "skip-cert-verify")
		stream["security"] = "tls"
		stream["tlsSettings"] = tlsSettings
	}
	if network == "ws" {
		stream["network"] = "ws"
		ws := map[string]interface{}{}
		if wsOpts, ok := node["ws-opts"]; ok {
			if wsMap := toStringMap(wsOpts); wsMap != nil {
				path := getMapString(wsMap, "path")
				if path != "" {
					ws["path"] = path
				}
				if headers, ok := wsMap["headers"]; ok {
					if headerMap := toStringMap(headers); headerMap != nil {
						if hostH := getMapString(headerMap, "Host"); hostH != "" {
							ws["headers"] = map[string]interface{}{"Host": hostH}
						}
					}
				}
			}
		}
		stream["wsSettings"] = ws
	}
	if network == "grpc" {
		stream["network"] = "grpc"
		if grpcOpts, ok := node["grpc-opts"]; ok {
			if grpcMap := toStringMap(grpcOpts); grpcMap != nil {
				serviceName := getMapString(grpcMap, "grpc-service-name")
				if serviceName != "" {
					stream["grpcSettings"] = map[string]interface{}{"serviceName": serviceName}
				}
			}
		}
	}
	if len(stream) > 0 {
		out["streamSettings"] = stream
	}
	return out, "", nil
}
