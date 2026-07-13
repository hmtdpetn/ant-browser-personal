package proxy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ant-chrome/backend/internal/config"
)

const mieruClashNode = `
name: 可乐云-HK01 Mieru
type: mieru
server: cursor.kaolun.cn
port: 40001
username: user-1
password: pass-1
transport: TCP
`

func TestMieruClashNodeIsMihomoOnlyProtocol(t *testing.T) {
	if !IsMihomoOnlyProtocol(mieruClashNode) {
		t.Fatalf("expected mieru Clash node to be treated as mihomo-only")
	}
	if IsSingBoxProtocol(mieruClashNode) {
		t.Fatalf("mieru must not be treated as sing-box protocol")
	}
	if RequiresBridge(mieruClashNode, nil, "") {
		t.Fatalf("mieru must not require xray bridge")
	}
	ok, msg := ValidateProxyConfig(mieruClashNode, nil, "")
	if !ok {
		t.Fatalf("ValidateProxyConfig rejected mieru node: %s", msg)
	}
}

func TestMieruClashNodeBuildsMihomoNode(t *testing.T) {
	node, err := buildMihomoNode(mieruClashNode)
	if err != nil {
		t.Fatalf("buildMihomoNode returned error: %v", err)
	}
	if node["type"] != "mieru" {
		t.Fatalf("type = %v, want mieru", node["type"])
	}
	if node["server"] != "cursor.kaolun.cn" {
		t.Fatalf("server = %v, want cursor.kaolun.cn", node["server"])
	}
	if node["port"] != 40001 {
		t.Fatalf("port = %v, want 40001", node["port"])
	}
}

func TestMieruSpeedTestRequiresMihomoConnector(t *testing.T) {
	proxyID := "mieru-proxy"
	result := SpeedTestWithConnector(
		proxyID,
		[]config.BrowserProxy{{ProxyId: proxyID, ProxyConfig: mieruClashNode}},
		nil,
		nil,
		nil,
		config.BrowserConnectorXray,
		&SpeedTestConfig{Timeout: 10, URLs: []string{"http://latency.test/generate_204"}},
	)
	if result.Ok {
		t.Fatalf("speed test should fail without mihomo connector, got success: %+v", result)
	}
	if result.Engine != config.BrowserConnectorMihomo {
		t.Fatalf("engine = %q, want mihomo; result=%+v", result.Engine, result)
	}
	if !strings.Contains(result.Error, "Mihomo") {
		t.Fatalf("error = %q, want Mihomo guidance", result.Error)
	}
}

func TestResolveMihomoBinaryFindsDownloadedCoreLayout(t *testing.T) {
	root := t.TempDir()
	binPath := filepath.Join(root, "bin", runtime.GOOS+"-"+runtime.GOARCH, "mihomo", "mihomo.exe")
	if runtime.GOOS != "windows" {
		binPath = filepath.Join(root, "bin", runtime.GOOS+"-"+runtime.GOARCH, "mihomo", "mihomo")
	}
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binPath, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	manager := &ClashManager{Config: &config.Config{}, AppRoot: root}
	got, err := manager.resolveMihomoBinary()
	if err != nil {
		t.Fatalf("resolveMihomoBinary returned error: %v", err)
	}
	if got != binPath {
		t.Fatalf("resolveMihomoBinary = %q, want %q", got, binPath)
	}
}
