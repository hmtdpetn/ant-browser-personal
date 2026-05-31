package proxy

import (
	"strings"
	"testing"
)

func TestProxyConfigToMappingStandardProxy(t *testing.T) {
	t.Parallel()

	mapping, err := proxyConfigToMapping("http://user:pass@example.com:8080/path")
	if err != nil {
		t.Fatalf("proxyConfigToMapping returned error: %v", err)
	}

	if got := mapping["type"]; got != "http" {
		t.Fatalf("type = %v, want http", got)
	}
	if got := mapping["server"]; got != "example.com" {
		t.Fatalf("server = %v, want example.com", got)
	}
	if got := mapping["port"]; got != 8080 {
		t.Fatalf("port = %v, want 8080", got)
	}
	if got := mapping["username"]; got != "user" {
		t.Fatalf("username = %v, want user", got)
	}
	if got := mapping["password"]; got != "pass" {
		t.Fatalf("password = %v, want pass", got)
	}
}

func TestProxyConfigToMappingEscapedCredentials(t *testing.T) {
	t.Parallel()

	mapping, err := proxyConfigToMapping("http://user%40mail:p%40ss%3Aword@example.com:8080")
	if err != nil {
		t.Fatalf("proxyConfigToMapping returned error: %v", err)
	}
	if got := mapping["username"]; got != "user@mail" {
		t.Fatalf("username = %v, want user@mail", got)
	}
	if got := mapping["password"]; got != "p@ss:word" {
		t.Fatalf("password = %v, want p@ss:word", got)
	}
}

func TestProxyEndpointDropsCredentials(t *testing.T) {
	t.Parallel()

	endpoint, err := proxyEndpoint("http://user:pass@example.com:8080")
	if err != nil {
		t.Fatalf("proxyEndpoint returned error: %v", err)
	}
	if endpoint != "example.com:8080" {
		t.Fatalf("endpoint = %q, want example.com:8080", endpoint)
	}
}

func TestProxyConfigToMappingClashYAML(t *testing.T) {
	t.Parallel()

	src := "proxies:\n  - type: vmess\n    server: test.example.com\n    port: 443\n"
	mapping, err := proxyConfigToMapping(src)
	if err != nil {
		t.Fatalf("proxyConfigToMapping returned error: %v", err)
	}

	if got := mapping["type"]; got != "vmess" {
		t.Fatalf("type = %v, want vmess", got)
	}
	if got := mapping["server"]; got != "test.example.com" {
		t.Fatalf("server = %v, want test.example.com", got)
	}
	if got := mapping["port"]; got != 443 {
		t.Fatalf("port = %v, want 443", got)
	}
	if got := mapping["name"]; got != "speedtest-proxy" {
		t.Fatalf("name = %v, want speedtest-proxy", got)
	}
}

func TestProxyConfigToMappingUnsupportedURI(t *testing.T) {
	t.Parallel()

	if _, err := proxyConfigToMapping("vmess://example"); err == nil {
		t.Fatal("expected unsupported URI error")
	}
}

func TestURLToMeta(t *testing.T) {
	t.Parallel()

	meta, err := urlToMeta("https://1.2.3.4:8443/path")
	if err != nil {
		t.Fatalf("urlToMeta returned error: %v", err)
	}

	if meta.Host != "1.2.3.4" {
		t.Fatalf("host = %q, want 1.2.3.4", meta.Host)
	}
	if meta.DstPort != 8443 {
		t.Fatalf("port = %d, want 8443", meta.DstPort)
	}
	if !meta.DstIP.IsValid() || meta.DstIP.String() != "1.2.3.4" {
		t.Fatalf("DstIP = %v, want 1.2.3.4", meta.DstIP)
	}
}

func TestDefaultProxyCheckURLsAreConfigured(t *testing.T) {
	t.Parallel()

	if strings.TrimSpace(DefaultSpeedTestURL) == "" {
		t.Fatalf("DefaultSpeedTestURL must not be empty")
	}
	if strings.TrimSpace(DefaultIPHealthURL) == "" {
		t.Fatalf("DefaultIPHealthURL must not be empty")
	}
}
