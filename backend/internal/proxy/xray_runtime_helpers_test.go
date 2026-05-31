package proxy

import (
	"reflect"
	"testing"
)

func TestParseDnsConfigFromClashYAML(t *testing.T) {
	t.Parallel()

	raw := `
dns:
  enable: true
  nameserver:
    - 8.8.8.8
    - tls://1.1.1.1
  fallback:
    - https://dns.google/dns-query
`

	got := parseDnsConfig(raw)
	want := map[string]interface{}{
		"servers": []interface{}{"8.8.8.8", "https://dns.google/dns-query"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseDnsConfig() = %#v, want %#v", got, want)
	}
}

func TestParseDnsConfigFromCommaList(t *testing.T) {
	t.Parallel()

	got := parseDnsConfig("8.8.8.8, tls://1.1.1.1, 127.0.0.1:53")
	want := map[string]interface{}{
		"servers": []interface{}{"8.8.8.8", "127.0.0.1:53"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseDnsConfig() = %#v, want %#v", got, want)
	}
}

func TestNormalizeNodeScheme(t *testing.T) {
	t.Parallel()

	if got := normalizeNodeScheme("hysteria://example"); got != "hysteria2://example" {
		t.Fatalf("normalizeNodeScheme() = %q", got)
	}
	if got := normalizeNodeScheme("vmess://example"); got != "vmess://example" {
		t.Fatalf("normalizeNodeScheme() unexpectedly changed vmess: %q", got)
	}
}
