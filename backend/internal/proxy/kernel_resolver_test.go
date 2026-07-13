package proxy

import (
	"testing"

	"ant-chrome/backend/internal/config"
)

func TestResolveProxyKernelDefaultPriority(t *testing.T) {
	cases := []struct {
		name       string
		proxy      string
		wantKernel string
	}{
		{name: "vless uses xray", proxy: "vless://00000000-0000-0000-0000-000000000000@example.com:443", wantKernel: ProxyKernelXray},
		{name: "hysteria2 uses sing-box", proxy: "hysteria2://pass@example.com:443", wantKernel: ProxyKernelSingBox},
		{name: "anytls URI uses sing-box", proxy: "anytls://pass@example.com:443?sni=example.com", wantKernel: ProxyKernelSingBox},
		{name: "mieru uses mihomo", proxy: mieruClashNode, wantKernel: ProxyKernelMihomo},
		{name: "http uses native", proxy: "http://127.0.0.1:8080", wantKernel: ProxyKernelNative},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveProxyKernel(tc.proxy, nil, "", "")
			if err != nil {
				t.Fatalf("ResolveProxyKernel returned error: %v", err)
			}
			if got.Kernel != tc.wantKernel {
				t.Fatalf("kernel = %q, want %q; resolution=%+v", got.Kernel, tc.wantKernel, got)
			}
		})
	}
}

func TestResolveProxyKernelRejectsUnsupportedPreferredKernel(t *testing.T) {
	_, err := ResolveProxyKernel(mieruClashNode, nil, "", ProxyKernelXray)
	if err == nil {
		t.Fatal("expected mieru + xray preference to be rejected")
	}
}

func TestResolveProxyKernelReadsPreferredKernelFromProxy(t *testing.T) {
	proxyID := "p1"
	got, err := ResolveProxyKernel("", []config.BrowserProxy{{ProxyId: proxyID, ProxyConfig: mieruClashNode, PreferredKernel: ProxyKernelMihomo}}, proxyID, "")
	if err != nil {
		t.Fatalf("ResolveProxyKernel returned error: %v", err)
	}
	if got.Kernel != ProxyKernelMihomo || got.PreferredKernel != ProxyKernelMihomo {
		t.Fatalf("unexpected resolution: %+v", got)
	}
}

func TestResolveProxyKernelForConnectorPrefersMihomoStack(t *testing.T) {
	got, err := ResolveProxyKernelForConnector("vless://00000000-0000-0000-0000-000000000000@example.com:443", nil, "", config.BrowserConnectorMihomo)
	if err != nil {
		t.Fatalf("ResolveProxyKernelForConnector returned error: %v", err)
	}
	if got.Kernel != ProxyKernelMihomo {
		t.Fatalf("kernel = %q, want %q; resolution=%+v", got.Kernel, ProxyKernelMihomo, got)
	}
}

func TestResolveProxyKernelForConnectorKeepsSingBoxOnlyProtocols(t *testing.T) {
	got, err := ResolveProxyKernelForConnector("hysteria2://pass@example.com:443", nil, "", config.BrowserConnectorXray)
	if err != nil {
		t.Fatalf("ResolveProxyKernelForConnector returned error: %v", err)
	}
	if got.Kernel != ProxyKernelSingBox {
		t.Fatalf("kernel = %q, want %q; resolution=%+v", got.Kernel, ProxyKernelSingBox, got)
	}
}

func TestResolveProxyKernelForConnectorExplicitPreferenceWins(t *testing.T) {
	proxyID := "p1"
	proxies := []config.BrowserProxy{{
		ProxyId:         proxyID,
		ProxyConfig:     "vless://00000000-0000-0000-0000-000000000000@example.com:443",
		PreferredKernel: ProxyKernelXray,
	}}
	got, err := ResolveProxyKernelForConnector("", proxies, proxyID, config.BrowserConnectorMihomo)
	if err != nil {
		t.Fatalf("ResolveProxyKernelForConnector returned error: %v", err)
	}
	if got.Kernel != ProxyKernelXray {
		t.Fatalf("kernel = %q, want explicit %q; resolution=%+v", got.Kernel, ProxyKernelXray, got)
	}
}
