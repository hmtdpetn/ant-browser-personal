package browser

import (
	"ant-chrome/backend/internal/config"
	"strings"
	"testing"
)

func newProfileProxyInputTestManager(t *testing.T) *Manager {
	t.Helper()
	cfg := config.DefaultConfig()
	mgr := NewManager(cfg, t.TempDir())
	mgr.ProxyDAO = &proxyDAOStub{
		list: []Proxy{
			{ProxyId: directProxyID, ProxyName: "直连（不走代理）", ProxyConfig: "direct://"},
			{ProxyId: "proxy-us", ProxyName: "US", ProxyConfig: "socks5://127.0.0.1:1080"},
		},
	}
	return mgr
}

func TestCreateProfileRejectsMissingProxyIDWithoutProxyConfig(t *testing.T) {
	mgr := newProfileProxyInputTestManager(t)
	_, err := mgr.Create(ProfileInput{
		ProfileName: "buyer-1",
		ProxyId:     "missing-id",
	})
	if err == nil {
		t.Fatalf("expected create to fail for missing proxy id without proxyConfig")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "proxy id not found") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mgr.Profiles) != 0 {
		t.Fatalf("profile should not be created on proxy validation failure")
	}
}

func TestCreateProfileFallsBackToCustomProxyConfigWhenProxyIDMissing(t *testing.T) {
	mgr := newProfileProxyInputTestManager(t)
	profile, err := mgr.Create(ProfileInput{
		ProfileName: "buyer-2",
		ProxyId:     "missing-id",
		ProxyConfig: "http://127.0.0.1:18080",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if profile.ProxyId != "" {
		t.Fatalf("expected proxyId to be cleared, got=%q", profile.ProxyId)
	}
	if profile.ProxyConfig != "http://127.0.0.1:18080" {
		t.Fatalf("expected proxyConfig to be preserved, got=%q", profile.ProxyConfig)
	}
}

func TestCreateProfileFallsBackToDirectWhenProxyInputEmpty(t *testing.T) {
	mgr := newProfileProxyInputTestManager(t)
	profile, err := mgr.Create(ProfileInput{
		ProfileName: "buyer-3",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if profile.ProxyId != directProxyID {
		t.Fatalf("expected direct proxy id, got=%q", profile.ProxyId)
	}
	if profile.ProxyConfig != "direct://" {
		t.Fatalf("expected direct proxy config, got=%q", profile.ProxyConfig)
	}
}

func TestUpdateProfileRejectsMissingProxyIDWithoutProxyConfig(t *testing.T) {
	mgr := newProfileProxyInputTestManager(t)
	profile, err := mgr.Create(ProfileInput{
		ProfileName: "buyer-old",
		ProxyId:     "proxy-us",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	beforeName := profile.ProfileName
	beforeProxyID := profile.ProxyId
	beforeProxyConfig := profile.ProxyConfig

	_, err = mgr.Update(profile.ProfileId, ProfileInput{
		ProfileName: "buyer-new",
		ProxyId:     "missing-id",
	})
	if err == nil {
		t.Fatalf("expected update to fail for missing proxy id without proxyConfig")
	}
	current := mgr.Profiles[profile.ProfileId]
	if current.ProfileName != beforeName {
		t.Fatalf("profile name should stay unchanged on failure, got=%q", current.ProfileName)
	}
	if current.ProxyId != beforeProxyID || current.ProxyConfig != beforeProxyConfig {
		t.Fatalf("proxy fields should stay unchanged on failure, got=%q/%q", current.ProxyId, current.ProxyConfig)
	}
}

func TestUpdateProfileFallsBackToCustomProxyConfigWhenProxyIDMissing(t *testing.T) {
	mgr := newProfileProxyInputTestManager(t)
	profile, err := mgr.Create(ProfileInput{
		ProfileName: "buyer-old",
		ProxyId:     "proxy-us",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updated, err := mgr.Update(profile.ProfileId, ProfileInput{
		ProfileName: "buyer-new",
		ProxyId:     "missing-id",
		ProxyConfig: "http://127.0.0.1:19090",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.ProfileName != "buyer-new" {
		t.Fatalf("expected updated name, got=%q", updated.ProfileName)
	}
	if updated.ProxyId != "" {
		t.Fatalf("expected proxyId to be cleared, got=%q", updated.ProxyId)
	}
	if updated.ProxyConfig != "http://127.0.0.1:19090" {
		t.Fatalf("expected proxyConfig to be updated, got=%q", updated.ProxyConfig)
	}
}
