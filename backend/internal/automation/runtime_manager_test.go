package automation

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync/atomic"
	"testing"

	"ant-chrome/backend/internal/config"
)

func TestEnsureInstalledDownloadsAndExtractsRuntime(t *testing.T) {
	t.Parallel()

	nodeArchive, nodeSHA := buildTestNodeZip(t)
	playwrightArchive, playwrightSHA := buildTestPlaywrightTGZ(t)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v22.15.1/SHASUMS256.txt":
			_, _ = w.Write([]byte(nodeSHA + "  node-v22.15.1-win-x64.zip\n"))
		case "/v22.15.1/node-v22.15.1-win-x64.zip":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(nodeArchive)
		case "/playwright-core/1.59.0":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dist": map[string]any{
					"tarball": server.URL + "/tarballs/playwright-core-1.59.0.tgz",
					"shasum":  playwrightSHA,
				},
			})
		case "/tarballs/playwright-core-1.59.0.tgz":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(playwrightArchive)
		default:
			http.NotFound(w, r)
		}
	})

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceBundled
	cfg.Automation.NodeVersion = "22.15.1"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = config.DefaultAutomationRuntimeVersion(cfg.Automation.NodeVersion, cfg.Automation.PlaywrightCoreVersion)

	manager := NewManager(t.TempDir(), cfg, nil, Options{
		NodeDistBaseURL:    server.URL,
		NPMRegistryBaseURL: server.URL,
		TargetOS:           "windows",
		TargetArch:         "amd64",
	})

	if err := manager.EnsureInstalled(context.Background()); err != nil {
		t.Fatalf("EnsureInstalled returned error: %v", err)
	}

	state := manager.CurrentState()
	if !state.Installed || !state.Ready {
		t.Fatalf("runtime should be ready after install, got %+v", state)
	}
	if _, err := os.Stat(filepath.Join(state.RuntimeDir, "node", "node.exe")); err != nil {
		t.Fatalf("expected node executable to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(state.RuntimeDir, "node_modules", "playwright-core", "package.json")); err != nil {
		t.Fatalf("expected playwright-core package.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(state.RuntimeDir, runnerScriptFileName)); err != nil {
		t.Fatalf("expected runner script to exist: %v", err)
	}
}

func TestEnsureInstalledUsesSystemNodeAndSkipsBundledDownload(t *testing.T) {
	t.Parallel()

	nodeExecPath := lookupNodeExecutable(t)
	playwrightArchive, playwrightSHA := buildTestPlayablePlaywrightTGZ(t, "1.59.0")

	var nodeRequests atomic.Int32

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v22.15.1/") {
			nodeRequests.Add(1)
			http.NotFound(w, r)
			return
		}

		switch r.URL.Path {
		case "/playwright-core/1.59.0":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dist": map[string]any{
					"tarball": server.URL + "/tarballs/playwright-core-1.59.0.tgz",
					"shasum":  playwrightSHA,
				},
			})
		case "/tarballs/playwright-core-1.59.0.tgz":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(playwrightArchive)
		default:
			http.NotFound(w, r)
		}
	})

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceSystem
	cfg.Automation.SystemNodePath = nodeExecPath
	cfg.Automation.NodeVersion = "22.15.1"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = config.DefaultAutomationRuntimeVersion(cfg.Automation.NodeVersion, cfg.Automation.PlaywrightCoreVersion)

	manager := NewManager(t.TempDir(), cfg, nil, Options{
		NodeDistBaseURL:    server.URL,
		NPMRegistryBaseURL: server.URL,
		TargetOS:           goruntime.GOOS,
		TargetArch:         goruntime.GOARCH,
	})

	if err := manager.EnsureInstalled(context.Background()); err != nil {
		t.Fatalf("EnsureInstalled returned error: %v", err)
	}

	state := manager.CurrentState()
	if !state.Installed || !state.Ready {
		t.Fatalf("runtime should be ready after install, got %+v", state)
	}
	if state.NodeSource != config.AutomationNodeSourceSystem {
		t.Fatalf("expected system node source, got %q", state.NodeSource)
	}
	if filepath.Clean(state.NodePath) != filepath.Clean(nodeExecPath) {
		t.Fatalf("expected system node path %q, got %q", nodeExecPath, state.NodePath)
	}
	if nodeRequests.Load() != 0 {
		t.Fatalf("expected bundled node download to be skipped, got %d node requests", nodeRequests.Load())
	}
	if _, err := os.Stat(filepath.Join(state.RuntimeDir, "node_modules", "playwright-core", "package.json")); err != nil {
		t.Fatalf("expected playwright-core package.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(state.RuntimeDir, runnerScriptFileName)); err != nil {
		t.Fatalf("expected runner script to exist: %v", err)
	}
	if _, err := os.Stat(manager.nodeExecutablePath(state.RuntimeDir)); !os.IsNotExist(err) {
		t.Fatalf("expected bundled node to be absent, got err=%v", err)
	}
}

func TestProbeSystemNodeUsesExplicitPath(t *testing.T) {
	t.Parallel()

	nodeExecPath := lookupNodeExecutable(t)
	manager := NewManager(t.TempDir(), config.DefaultConfig(), nil, Options{})

	result, err := manager.ProbeSystemNode(context.Background(), nodeExecPath)
	if err != nil {
		t.Fatalf("ProbeSystemNode returned error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected probe result to be ok, got %+v", result)
	}
	if filepath.Clean(result.Path) != filepath.Clean(nodeExecPath) {
		t.Fatalf("expected probe path %q, got %q", nodeExecPath, result.Path)
	}
	if strings.TrimSpace(result.Version) == "" {
		t.Fatalf("expected probe version to be set, got %+v", result)
	}
}

func TestProbeSystemNodeMissingReturnsError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	manager := NewManager(t.TempDir(), config.DefaultConfig(), nil, Options{})
	_, err := manager.ProbeSystemNode(context.Background(), filepath.Join(t.TempDir(), "missing-node.exe"))
	if err == nil {
		t.Fatalf("expected ProbeSystemNode to fail for missing node path")
	}
}

func TestCurrentStateReportsBundledFallbackReasonWhenSystemNodeMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceAuto
	cfg.Automation.SystemNodePath = filepath.Join(t.TempDir(), "missing-node.exe")

	manager := NewManager(t.TempDir(), cfg, nil, Options{
		TargetOS:   "windows",
		TargetArch: "amd64",
	})

	state := manager.CurrentState()
	if state.NodeSource != config.AutomationNodeSourceBundled {
		t.Fatalf("expected bundled node source, got %q", state.NodeSource)
	}
	if !strings.Contains(state.NodeResolution, "回退") {
		t.Fatalf("expected fallback resolution message, got %q", state.NodeResolution)
	}
	if strings.TrimSpace(state.SystemNodeError) == "" {
		t.Fatalf("expected system node error to be set, got %+v", state)
	}
}

func TestCurrentStateReportsSystemResolutionWhenExplicitNodeSucceeds(t *testing.T) {
	t.Parallel()

	nodeExecPath := lookupNodeExecutable(t)

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceAuto
	cfg.Automation.SystemNodePath = nodeExecPath

	manager := NewManager(t.TempDir(), cfg, nil, Options{
		TargetOS:   goruntime.GOOS,
		TargetArch: goruntime.GOARCH,
	})

	state := manager.CurrentState()
	if state.NodeSource != config.AutomationNodeSourceSystem {
		t.Fatalf("expected system node source, got %q", state.NodeSource)
	}
	if !strings.Contains(state.NodeResolution, "配置的系统 Node 路径") {
		t.Fatalf("expected explicit system node resolution, got %q", state.NodeResolution)
	}
}

func TestEnsureInstalledRepairsBrokenReadyAutoRuntime(t *testing.T) {
	t.Parallel()

	nodeExecPath := lookupNodeExecutable(t)
	playwrightArchive, playwrightSHA := buildTestPlayablePlaywrightTGZ(t, "1.59.0")

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/playwright-core/1.59.0":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dist": map[string]any{
					"tarball": server.URL + "/tarballs/playwright-core-1.59.0.tgz",
					"shasum":  playwrightSHA,
				},
			})
		case "/tarballs/playwright-core-1.59.0.tgz":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(playwrightArchive)
		default:
			http.NotFound(w, r)
		}
	})

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceAuto
	cfg.Automation.SystemNodePath = nodeExecPath
	cfg.Automation.NodeVersion = "22.15.1"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = config.DefaultAutomationRuntimeVersion(cfg.Automation.NodeVersion, cfg.Automation.PlaywrightCoreVersion)

	manager := NewManager(t.TempDir(), cfg, nil, Options{
		NPMRegistryBaseURL: server.URL,
		TargetOS:           goruntime.GOOS,
		TargetArch:         goruntime.GOARCH,
	})

	initialState := manager.CurrentState()
	if err := writeRunnerScript(initialState.RunnerPath); err != nil {
		t.Fatalf("write runner script failed: %v", err)
	}
	if err := writeBrokenPlaywrightModule(initialState.RuntimeDir, cfg.Automation.PlaywrightCoreVersion); err != nil {
		t.Fatalf("write broken playwright module failed: %v", err)
	}

	readyState := manager.CurrentState()
	if !readyState.Ready {
		t.Fatalf("expected broken runtime to appear ready before verification, got %+v", readyState)
	}

	if err := manager.EnsureInstalled(context.Background()); err != nil {
		t.Fatalf("EnsureInstalled returned error: %v", err)
	}

	check, err := manager.SelfCheck(context.Background())
	if err != nil {
		t.Fatalf("SelfCheck returned error after repair: %v", err)
	}
	if !check.OK {
		t.Fatalf("expected repaired runtime to pass self-check, got %+v", check)
	}
}

func TestEnsureInstalledAutoFallsBackToBundledWhenSystemNodeMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	nodeArchive, nodeSHA := buildTestNodeZip(t)
	playwrightArchive, playwrightSHA := buildTestPlaywrightTGZ(t)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v22.15.1/SHASUMS256.txt":
			_, _ = w.Write([]byte(nodeSHA + "  node-v22.15.1-win-x64.zip\n"))
		case "/v22.15.1/node-v22.15.1-win-x64.zip":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(nodeArchive)
		case "/playwright-core/1.59.0":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dist": map[string]any{
					"tarball": server.URL + "/tarballs/playwright-core-1.59.0.tgz",
					"shasum":  playwrightSHA,
				},
			})
		case "/tarballs/playwright-core-1.59.0.tgz":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(playwrightArchive)
		default:
			http.NotFound(w, r)
		}
	})

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceAuto
	cfg.Automation.SystemNodePath = filepath.Join(t.TempDir(), "missing-node.exe")
	cfg.Automation.NodeVersion = "22.15.1"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = config.DefaultAutomationRuntimeVersion(cfg.Automation.NodeVersion, cfg.Automation.PlaywrightCoreVersion)

	manager := NewManager(t.TempDir(), cfg, nil, Options{
		NodeDistBaseURL:    server.URL,
		NPMRegistryBaseURL: server.URL,
		TargetOS:           "windows",
		TargetArch:         "amd64",
	})

	if err := manager.EnsureInstalled(context.Background()); err != nil {
		t.Fatalf("EnsureInstalled returned error: %v", err)
	}

	state := manager.CurrentState()
	if !state.Installed || !state.Ready {
		t.Fatalf("runtime should be ready after install, got %+v", state)
	}
	if state.NodeSource != config.AutomationNodeSourceBundled {
		t.Fatalf("expected bundled node source after fallback, got %q", state.NodeSource)
	}
}

func TestEnsureInstalledSystemSourceFailsWhenSystemNodeMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceSystem
	cfg.Automation.SystemNodePath = filepath.Join(t.TempDir(), "missing-node.exe")
	cfg.Automation.NodeVersion = "22.15.1"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = config.DefaultAutomationRuntimeVersion(cfg.Automation.NodeVersion, cfg.Automation.PlaywrightCoreVersion)

	manager := NewManager(t.TempDir(), cfg, nil, Options{
		TargetOS:   "windows",
		TargetArch: "amd64",
	})

	err := manager.EnsureInstalled(context.Background())
	if err == nil {
		t.Fatalf("expected EnsureInstalled to fail when system node is missing")
	}
	if !strings.Contains(err.Error(), "系统 Node 不可用") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureInstalledRefreshesExistingRunnerScript(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceBundled
	cfg.Automation.NodeVersion = "22.15.1"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = config.DefaultAutomationRuntimeVersion(cfg.Automation.NodeVersion, cfg.Automation.PlaywrightCoreVersion)

	manager := NewManager(t.TempDir(), cfg, nil, Options{
		TargetOS:   "windows",
		TargetArch: "amd64",
	})

	state := manager.CurrentState()
	if err := os.MkdirAll(filepath.Dir(state.NodePath), 0o755); err != nil {
		t.Fatalf("create node dir failed: %v", err)
	}
	if err := os.WriteFile(state.NodePath, []byte("fake-node-runtime"), 0o755); err != nil {
		t.Fatalf("write fake node failed: %v", err)
	}
	playwrightPkgPath := filepath.Join(state.RuntimeDir, "node_modules", "playwright-core", "package.json")
	if err := os.MkdirAll(filepath.Dir(playwrightPkgPath), 0o755); err != nil {
		t.Fatalf("create playwright dir failed: %v", err)
	}
	if err := os.WriteFile(playwrightPkgPath, []byte(`{"name":"playwright-core","version":"1.59.0"}`), 0o644); err != nil {
		t.Fatalf("write fake playwright package.json failed: %v", err)
	}
	if err := os.WriteFile(state.RunnerPath, []byte("old-runner"), 0o755); err != nil {
		t.Fatalf("write stale runner failed: %v", err)
	}

	if err := manager.EnsureInstalled(context.Background()); err != nil {
		t.Fatalf("EnsureInstalled returned error: %v", err)
	}

	runnerData, err := os.ReadFile(state.RunnerPath)
	if err != nil {
		t.Fatalf("read refreshed runner failed: %v", err)
	}
	if string(runnerData) != string(runnerScriptContent) {
		t.Fatalf("expected runner script to be refreshed")
	}
}

func buildTestNodeZip(t *testing.T) ([]byte, string) {
	t.Helper()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	header := &zip.FileHeader{
		Name:   "node-v22.15.1-win-x64/node.exe",
		Method: zip.Deflate,
	}
	fileWriter, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatalf("create node zip header failed: %v", err)
	}
	if _, err := fileWriter.Write([]byte("fake-node-runtime")); err != nil {
		t.Fatalf("write node zip failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close node zip failed: %v", err)
	}

	hash := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(hash[:])
}

func buildTestPlaywrightTGZ(t *testing.T) ([]byte, string) {
	t.Helper()

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	payload := []byte(`{"name":"playwright-core","version":"1.59.0"}`)
	header := &tar.Header{
		Name: "package/package.json",
		Mode: 0o644,
		Size: int64(len(payload)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("write playwright header failed: %v", err)
	}
	if _, err := tarWriter.Write(payload); err != nil {
		t.Fatalf("write playwright payload failed: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close playwright tar failed: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("close playwright gzip failed: %v", err)
	}

	hash := sha1.Sum(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(hash[:])
}

func buildTestPlayablePlaywrightTGZ(t *testing.T, version string) ([]byte, string) {
	t.Helper()

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	files := map[string][]byte{
		"package/package.json": []byte(`{"name":"playwright-core","version":"` + version + `","main":"index.js"}`),
		"package/index.js":     []byte("exports.chromium = {};"),
	}

	for name, payload := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(payload)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write playable playwright header failed: %v", err)
		}
		if _, err := tarWriter.Write(payload); err != nil {
			t.Fatalf("write playable playwright payload failed: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close playable playwright tar failed: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("close playable playwright gzip failed: %v", err)
	}

	hash := sha1.Sum(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(hash[:])
}

func writeBrokenPlaywrightModule(runtimeDir, version string) error {
	moduleDir := filepath.Join(runtimeDir, "node_modules", "playwright-core")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		return err
	}

	packageJSON := []byte(`{"name":"playwright-core","version":"` + version + `","main":"index.js"}`)
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), packageJSON, 0o644); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(moduleDir, "index.js"), []byte("module.exports = {};"), 0o644)
}
