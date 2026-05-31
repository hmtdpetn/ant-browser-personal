package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ant-chrome/backend/internal/config"
)

func TestRunScriptTaskExecutesCustomRunner(t *testing.T) {
	nodeExecPath := lookupNodeExecutable(t)

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceSystem
	cfg.Automation.SystemNodePath = nodeExecPath
	cfg.Automation.NodeVersion = "test-node"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = "test-runtime"

	manager := NewManager(t.TempDir(), cfg, nil, Options{})

	state := manager.CurrentState()
	if err := writeRunnerScript(state.RunnerPath); err != nil {
		t.Fatalf("write runner script failed: %v", err)
	}
	if err := writeMockPlaywrightModule(state.RuntimeDir, cfg.Automation.PlaywrightCoreVersion); err != nil {
		t.Fatalf("write mock playwright module failed: %v", err)
	}

	receivedBody := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/launch" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":        true,
			"profileId": "profile-script",
			"debugPort": 9333,
			"cdpUrl":    "http://127.0.0.1:9333",
		})
	}))
	defer server.Close()

	scriptDir := filepath.Join(state.RuntimeDir, "tmp", "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("create script dir failed: %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "script.cjs")
	scriptSource := `const fs = require('fs');

module.exports.run = async ({ launch, connect, selector, params, log, artifact }) => {
  const session = await launch({
    selector,
    startUrls: params.startUrls,
    skipDefaultStartUrls: true,
  })

  const { browser } = await connect(session)
  const context = browser.contexts()[0]
  const page = context.pages()[0] || await context.newPage()
  await page.goto(params.url, { waitUntil: 'domcontentloaded', timeout: params.timeoutMs || 30000 })

  const filePath = artifact('script-output.txt')
  fs.writeFileSync(filePath, 'artifact-ready')
  log('profile', session.profileId)

  return {
    ok: true,
    summary: '脚本执行成功',
    profileId: session.profileId,
    url: page.url(),
    artifactPath: filePath,
  }
}`
	if err := os.WriteFile(scriptPath, []byte(scriptSource), 0o644); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	artifactDir := filepath.Join(t.TempDir(), "artifacts")
	result, err := manager.RunScriptTask(context.Background(), ScriptTaskRequest{
		TaskKey:       "script:test",
		ScriptPath:    scriptPath,
		Selector:      map[string]any{"code": "BUYER_001"},
		Params:        map[string]any{"url": "https://example.com/script", "startUrls": []string{"https://example.com/script"}},
		LaunchBaseURL: server.URL,
		ArtifactDir:   artifactDir,
	})
	if err != nil {
		t.Fatalf("RunScriptTask returned error: %v", err)
	}

	if !result.OK {
		t.Fatalf("expected script task to succeed, got %+v", result)
	}
	if result.Summary != "脚本执行成功" {
		t.Fatalf("unexpected summary: %s", result.Summary)
	}
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if !strings.Contains(result.ResultText, `"profileId":"profile-script"`) {
		t.Fatalf("expected result text to contain profileId, got %s", result.ResultText)
	}
	if !strings.Contains(result.ResultText, `"artifactPath":"`) {
		t.Fatalf("expected result text to contain artifact path, got %s", result.ResultText)
	}

	if selector, ok := receivedBody["selector"].(map[string]any); !ok || selector["code"] != "BUYER_001" {
		t.Fatalf("unexpected selector payload: %+v", receivedBody)
	}

	artifactData, err := os.ReadFile(filepath.Join(artifactDir, "script-output.txt"))
	if err != nil {
		t.Fatalf("read script artifact failed: %v", err)
	}
	if string(artifactData) != "artifact-ready" {
		t.Fatalf("unexpected script artifact payload: %s", string(artifactData))
	}
}

func TestRunScriptTaskLaunchFiltersNonLaunchParams(t *testing.T) {
	nodeExecPath := lookupNodeExecutable(t)

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceSystem
	cfg.Automation.SystemNodePath = nodeExecPath
	cfg.Automation.NodeVersion = "test-node"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = "test-runtime"

	manager := NewManager(t.TempDir(), cfg, nil, Options{})

	state := manager.CurrentState()
	if err := writeRunnerScript(state.RunnerPath); err != nil {
		t.Fatalf("write runner script failed: %v", err)
	}
	if err := writeMockPlaywrightModule(state.RuntimeDir, cfg.Automation.PlaywrightCoreVersion); err != nil {
		t.Fatalf("write mock playwright module failed: %v", err)
	}

	type launchRequestPayload struct {
		Code                 string         `json:"code"`
		Key                  string         `json:"key"`
		ProfileID            string         `json:"profileId"`
		ProfileName          string         `json:"profileName"`
		Keyword              string         `json:"keyword"`
		Keywords             []string       `json:"keywords"`
		Tag                  string         `json:"tag"`
		Tags                 []string       `json:"tags"`
		GroupID              string         `json:"groupId"`
		MatchMode            string         `json:"matchMode"`
		Selector             map[string]any `json:"selector"`
		LaunchArgs           []string       `json:"launchArgs"`
		StartURLs            []string       `json:"startUrls"`
		SkipDefaultStartURLs bool           `json:"skipDefaultStartUrls"`
	}

	receivedBody := launchRequestPayload{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/launch" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&receivedBody); err != nil {
			t.Fatalf("decode launch request body failed: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":        true,
			"profileId": "profile-script",
			"debugPort": 9333,
			"cdpUrl":    "http://127.0.0.1:9333",
		})
	}))
	defer server.Close()

	scriptDir := filepath.Join(state.RuntimeDir, "tmp", "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("create script dir failed: %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "script-launch-filter.cjs")
	scriptSource := `module.exports.run = async ({ launch, selector, params }) => {
  const session = await launch({
    selector,
    startUrls: params.startUrls,
    skipDefaultStartUrls: true,
  })

  return {
    ok: true,
    summary: '脚本执行成功',
    profileId: session.profileId,
  }
}`
	if err := os.WriteFile(scriptPath, []byte(scriptSource), 0o644); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	result, err := manager.RunScriptTask(context.Background(), ScriptTaskRequest{
		TaskKey:       "script:launch-filter",
		ScriptPath:    scriptPath,
		Selector:      map[string]any{"code": "DEMO_READY"},
		Params:        map[string]any{"url": "https://www.baidu.com", "keyword": "OpenAI", "captureScreenshot": true, "waitAfterSearchMs": 1500, "startUrls": []string{"https://www.baidu.com"}},
		LaunchBaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunScriptTask returned error: %v", err)
	}

	if !result.OK {
		t.Fatalf("expected script task to succeed, got %+v", result)
	}
	if receivedBody.Selector["code"] != "DEMO_READY" {
		t.Fatalf("unexpected selector payload: %+v", receivedBody)
	}
	if len(receivedBody.StartURLs) != 1 || receivedBody.StartURLs[0] != "https://www.baidu.com" {
		t.Fatalf("unexpected startUrls payload: %+v", receivedBody.StartURLs)
	}
	if !receivedBody.SkipDefaultStartURLs {
		t.Fatalf("expected skipDefaultStartUrls to be true")
	}
	if receivedBody.Keyword != "" {
		t.Fatalf("expected non-launch params to be filtered, got keyword=%q", receivedBody.Keyword)
	}
}

func TestRunScriptTaskFallsBackToLaunchBaseURLWhenSessionEndpointIsInvalid(t *testing.T) {
	nodeExecPath := lookupNodeExecutable(t)

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceSystem
	cfg.Automation.SystemNodePath = nodeExecPath
	cfg.Automation.NodeVersion = "test-node"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = "test-runtime"

	manager := NewManager(t.TempDir(), cfg, nil, Options{})

	state := manager.CurrentState()
	if err := writeRunnerScript(state.RunnerPath); err != nil {
		t.Fatalf("write runner script failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/launch" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         true,
			"profileId":  "profile-script",
			"debugPort":  0,
			"debugReady": false,
			"cdpUrl":     "http://127.0.0.1:0",
		})
	}))
	defer server.Close()

	if err := writeMockPlaywrightModuleWithExpectedEndpoint(state.RuntimeDir, cfg.Automation.PlaywrightCoreVersion, server.URL); err != nil {
		t.Fatalf("write mock playwright module failed: %v", err)
	}

	scriptDir := filepath.Join(state.RuntimeDir, "tmp", "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("create script dir failed: %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "script-fallback.cjs")
	scriptSource := `module.exports.run = async ({ launch, connect, selector }) => {
  const session = await launch({ selector })
  const connection = await connect(session)

  return {
    ok: true,
    summary: '脚本已通过 Launch 地址回退连接',
    connectedEndpoint: connection.session.cdpUrl,
    profileId: session.profileId,
  }
}`
	if err := os.WriteFile(scriptPath, []byte(scriptSource), 0o644); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	result, err := manager.RunScriptTask(context.Background(), ScriptTaskRequest{
		TaskKey:       "script:fallback",
		ScriptPath:    scriptPath,
		Selector:      map[string]any{"code": "DEMO_READY"},
		LaunchBaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunScriptTask returned error: %v", err)
	}

	if !result.OK {
		t.Fatalf("expected script task to succeed, got %+v", result)
	}
	if result.Summary != "脚本已通过 Launch 地址回退连接" {
		t.Fatalf("unexpected summary: %s", result.Summary)
	}
	if !strings.Contains(result.ResultText, `"connectedEndpoint":"`+server.URL+`"`) {
		t.Fatalf("expected result text to contain fallback endpoint, got %s", result.ResultText)
	}
}

func TestRunScriptTaskClosesBrowserConnections(t *testing.T) {
	nodeExecPath := lookupNodeExecutable(t)

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceSystem
	cfg.Automation.SystemNodePath = nodeExecPath
	cfg.Automation.NodeVersion = "test-node"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = "test-runtime"

	manager := NewManager(t.TempDir(), cfg, nil, Options{})

	state := manager.CurrentState()
	if err := writeRunnerScript(state.RunnerPath); err != nil {
		t.Fatalf("write runner script failed: %v", err)
	}
	if err := writeMockPlaywrightModuleWithPersistentConnection(state.RuntimeDir, cfg.Automation.PlaywrightCoreVersion, ""); err != nil {
		t.Fatalf("write mock playwright module failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":        true,
			"profileId": "profile-script-close",
			"debugPort": 9333,
			"cdpUrl":    "http://127.0.0.1:9333",
		})
	}))
	defer server.Close()

	scriptDir := filepath.Join(state.RuntimeDir, "tmp", "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("create script dir failed: %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "script-close.cjs")
	scriptSource := `module.exports.run = async ({ launch, connect, selector }) => {
  const session = await launch({ selector })
  const connection = await connect(session)

  return {
    ok: true,
    summary: '脚本执行成功',
    connectedEndpoint: connection.session.cdpUrl,
  }
}`
	if err := os.WriteFile(scriptPath, []byte(scriptSource), 0o644); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := manager.RunScriptTask(ctx, ScriptTaskRequest{
		TaskKey:       "script:close",
		ScriptPath:    scriptPath,
		Selector:      map[string]any{"code": "DEMO_READY"},
		LaunchBaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunScriptTask returned error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected script task to succeed, got %+v", result)
	}
}

func TestRunScriptTaskTerminatesHungScriptOnTimeout(t *testing.T) {
	nodeExecPath := lookupNodeExecutable(t)

	cfg := config.DefaultConfig()
	cfg.Automation.Enabled = true
	cfg.Automation.NodeSource = config.AutomationNodeSourceSystem
	cfg.Automation.SystemNodePath = nodeExecPath
	cfg.Automation.NodeVersion = "test-node"
	cfg.Automation.PlaywrightCoreVersion = "1.59.0"
	cfg.Automation.RuntimeVersion = "test-runtime"

	manager := NewManager(t.TempDir(), cfg, nil, Options{})

	state := manager.CurrentState()
	if err := writeRunnerScript(state.RunnerPath); err != nil {
		t.Fatalf("write runner script failed: %v", err)
	}
	if err := writeMockPlaywrightModule(state.RuntimeDir, cfg.Automation.PlaywrightCoreVersion); err != nil {
		t.Fatalf("write mock playwright module failed: %v", err)
	}

	scriptDir := filepath.Join(state.RuntimeDir, "tmp", "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("create script dir failed: %v", err)
	}
	scriptPath := filepath.Join(scriptDir, "script-timeout.cjs")
	scriptSource := `module.exports.run = async () => {
  await new Promise(() => setInterval(() => {}, 1000))
}`
	if err := os.WriteFile(scriptPath, []byte(scriptSource), 0o644); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	startedAt := time.Now()
	_, err := manager.RunScriptTask(context.Background(), ScriptTaskRequest{
		TaskKey:       "script:timeout",
		ScriptPath:    scriptPath,
		LaunchBaseURL: "http://127.0.0.1",
		Timeout:       150 * time.Millisecond,
	})
	elapsed := time.Since(startedAt)
	if err == nil {
		t.Fatalf("expected RunScriptTask to fail on timeout")
	}
	if !strings.Contains(err.Error(), "超时") {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("expected timeout to terminate quickly, took %s", elapsed)
	}

	manager.mu.Lock()
	activeTaskCount := len(manager.activeTasks)
	profileTaskCount := len(manager.profileTask)
	manager.mu.Unlock()
	if activeTaskCount != 0 || profileTaskCount != 0 {
		t.Fatalf("expected timed out task to be unregistered, active=%d profile=%d", activeTaskCount, profileTaskCount)
	}
}

func lookupNodeExecutable(t *testing.T) string {
	t.Helper()

	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("node is not available: %v", err)
	}

	cmd := exec.Command(nodePath, "-p", "process.execPath")
	output, err := cmd.Output()
	if err != nil {
		return nodePath
	}

	resolved := strings.TrimSpace(string(output))
	if resolved == "" {
		return nodePath
	}
	return resolved
}

func writeMockPlaywrightModule(runtimeDir, version string) error {
	return writeMockPlaywrightModuleWithExpectedEndpoint(runtimeDir, version, "")
}

func writeMockPlaywrightModuleWithExpectedEndpoint(runtimeDir, version, expectedEndpoint string) error {
	return writeMockPlaywrightModuleWithOptions(runtimeDir, version, expectedEndpoint, false)
}

func writeMockPlaywrightModuleWithPersistentConnection(runtimeDir, version, expectedEndpoint string) error {
	return writeMockPlaywrightModuleWithOptions(runtimeDir, version, expectedEndpoint, true)
}

func writeMockPlaywrightModuleWithOptions(runtimeDir, version, expectedEndpoint string, persistentConnection bool) error {
	moduleDir := filepath.Join(runtimeDir, "node_modules", "playwright-core")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		return err
	}

	packageJSON := fmt.Sprintf("{\"name\":\"playwright-core\",\"version\":\"%s\",\"main\":\"index.js\"}", version)
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		return err
	}

	expectedEndpointJSON, err := json.Marshal(expectedEndpoint)
	if err != nil {
		return err
	}
	persistentConnectionJSON, err := json.Marshal(persistentConnection)
	if err != nil {
		return err
	}

	indexJS := fmt.Sprintf(`const fs = require('fs');

const expectedEndpoint = %s;
const persistentConnection = %s;

function createPage() {
  let currentURL = 'about:blank';
  return {
    async goto(url) {
      currentURL = url;
    },
    async waitForTimeout() {},
    async screenshot(options) {
      fs.writeFileSync(options.path, 'mock-screenshot');
    },
    async title() {
      return 'Mock Page Title';
    },
    url() {
      return currentURL;
    },
    async close() {},
  };
}

const context = {
  async newPage() {
    return createPage();
  },
  pages() {
    return [];
  },
};

exports.chromium = {
  async connectOverCDP(endpoint) {
    if (String(endpoint).includes(':0')) {
      throw new Error('invalid cdp endpoint');
    }
    if (expectedEndpoint && endpoint !== expectedEndpoint) {
      throw new Error('unexpected cdp endpoint: ' + endpoint);
    }
    const hold = persistentConnection ? setInterval(() => {}, 1000) : null;
    return {
      contexts() {
        return [context];
      },
      async close() {
        if (hold) {
          clearInterval(hold);
        }
      },
    };
  },
};
`, string(expectedEndpointJSON), string(persistentConnectionJSON))
	return os.WriteFile(filepath.Join(moduleDir, "index.js"), []byte(indexJS), 0o644)
}
