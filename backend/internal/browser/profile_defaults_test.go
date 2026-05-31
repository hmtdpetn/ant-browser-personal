package browser

import (
	"ant-chrome/backend/internal/config"
	"testing"
)

func TestApplyDefaultsDoesNotFallbackToDirectAfterPoolBindByProxyConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	mgr := NewManager(cfg, "")
	mgr.ProxyDAO = &proxyDAOStub{
		list: []Proxy{
			{ProxyId: directProxyID, ProxyName: "直连（不走代理）", ProxyConfig: "direct://"},
			{ProxyId: "pool-1", ProxyName: "节点-01", ProxyConfig: "socks5://127.0.0.1:1080"},
		},
	}

	profile := &Profile{
		ProfileId:   "pf-apply-defaults-1",
		ProxyId:     "",
		ProxyConfig: "socks5://127.0.0.1:1080",
	}

	changed := mgr.ApplyDefaults(profile)
	if !changed {
		t.Fatalf("expected proxy binding to change")
	}
	if profile.ProxyId != "pool-1" {
		t.Fatalf("expected proxyId to bind to pool-1, got=%q", profile.ProxyId)
	}
	if profile.ProxyId == directProxyID {
		t.Fatalf("expected not to fallback to direct proxy")
	}
}

func TestApplyDefaultsKeepsCustomProxyConfigWhenNotInPool(t *testing.T) {
	cfg := config.DefaultConfig()
	mgr := NewManager(cfg, "")
	mgr.ProxyDAO = &proxyDAOStub{
		list: []Proxy{
			{ProxyId: directProxyID, ProxyName: "直连（不走代理）", ProxyConfig: "direct://"},
			{ProxyId: "pool-2", ProxyName: "日本-01", ProxyConfig: "socks5://127.0.0.1:2080"},
		},
	}

	profile := &Profile{
		ProfileId:   "pf-apply-defaults-2",
		ProxyId:     "",
		ProxyConfig: "http://127.0.0.1:9090",
	}

	_ = mgr.ApplyDefaults(profile)
	if profile.ProxyId != "" {
		t.Fatalf("expected proxyId to stay empty for custom proxyConfig, got=%q", profile.ProxyId)
	}
	if profile.ProxyConfig != "http://127.0.0.1:9090" {
		t.Fatalf("expected proxyConfig to be preserved, got=%q", profile.ProxyConfig)
	}
}

func TestApplyDefaultsClearsMissingProxyIdButPreservesProxyConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	mgr := NewManager(cfg, "")
	mgr.ProxyDAO = &proxyDAOStub{
		list: []Proxy{
			{ProxyId: directProxyID, ProxyName: "直连（不走代理）", ProxyConfig: "direct://"},
		},
	}

	profile := &Profile{
		ProfileId:   "pf-apply-defaults-3",
		ProxyId:     "missing-proxy-id",
		ProxyConfig: "http://127.0.0.1:9090",
	}

	changed := mgr.ApplyDefaults(profile)
	if !changed {
		t.Fatalf("expected proxy binding to change when clearing missing proxyId")
	}
	if profile.ProxyId != "" {
		t.Fatalf("expected missing proxyId to be cleared, got=%q", profile.ProxyId)
	}
	if profile.ProxyConfig != "http://127.0.0.1:9090" {
		t.Fatalf("expected proxyConfig to be preserved, got=%q", profile.ProxyConfig)
	}
	if profile.ProxyId == directProxyID {
		t.Fatalf("expected not to fallback to direct proxy when proxyConfig is present")
	}
}

func TestApplyDefaultsFallsBackToDirectWhenProxyMissing(t *testing.T) {
	cfg := config.DefaultConfig()
	mgr := NewManager(cfg, "")
	mgr.ProxyDAO = &proxyDAOStub{
		list: []Proxy{
			{ProxyId: directProxyID, ProxyName: "直连（不走代理）", ProxyConfig: "direct://"},
		},
	}

	profile := &Profile{
		ProfileId:   "pf-apply-defaults-4",
		ProxyId:     "",
		ProxyConfig: "",
	}

	changed := mgr.ApplyDefaults(profile)
	if !changed {
		t.Fatalf("expected direct proxy fallback to change profile")
	}
	if profile.ProxyId != directProxyID {
		t.Fatalf("expected fallback to direct proxy id, got=%q", profile.ProxyId)
	}
	if profile.ProxyConfig != "direct://" {
		t.Fatalf("expected fallback proxy config to be direct://, got=%q", profile.ProxyConfig)
	}
}
