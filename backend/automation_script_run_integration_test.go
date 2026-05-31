package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"ant-chrome/backend/internal/automation"
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/launchcode"
)

func TestAutomationScriptRunWithOptionsExecutesPlaywrightScript(t *testing.T) {
	nodeExecPath := lookupAutomationTestNode(t)

	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()
	app.config.Automation.Enabled = true
	app.config.Automation.NodeSource = config.AutomationNodeSourceSystem
	app.config.Automation.SystemNodePath = nodeExecPath
	app.config.Automation.NodeVersion = "test-node"
	app.config.Automation.PlaywrightCoreVersion = "1.59.0"
	app.config.Automation.RuntimeVersion = "test-runtime"
	app.automationMgr = automation.NewManager(app.appRoot, app.config, nil, automation.Options{})

	prepareAutomationTestRuntime(t, app.automationMgr, app.config.Automation.PlaywrightCoreVersion)

	app.launchServer = launchcode.NewLaunchServer(
		launchcode.NewLaunchCodeService(launchcode.NewMemoryLaunchCodeDAO()),
		nil,
		nil,
		0,
	)
	if err := app.launchServer.Start(); err != nil {
		t.Fatalf("start launch server failed: %v", err)
	}
	defer func() {
		_ = app.launchServer.Stop()
	}()

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "playwright-success",
		Name:       "Playwright 成功脚本",
		Type:       "playwright-cdp",
		Status:     "ready",
		EntryFile:  "scripts/index.cjs",
		ScriptText: "const fs = require('fs')\nmodule.exports.run = async ({ params, artifact }) => {\n  const outputPath = artifact('result.txt')\n  fs.writeFileSync(outputPath, String(params.message || 'default'), 'utf8')\n  return { ok: true, summary: 'artifact ready', outputPath }\n}\n",
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	run, err := app.AutomationScriptRunWithOptions(automation.ScriptRunRequest{
		ScriptID:          saved.ID,
		SelectorText:      `{}`,
		ParamsText:        `{"message":"hello integration"}`,
		UseScriptSelector: false,
		UseScriptParams:   false,
	})
	if err != nil {
		t.Fatalf("AutomationScriptRunWithOptions returned error: %v", err)
	}
	if run == nil {
		t.Fatalf("AutomationScriptRunWithOptions returned nil result")
	}
	if run.Status != "success" {
		t.Fatalf("expected success status, got %+v", run)
	}
	if run.Summary != "artifact ready" {
		t.Fatalf("unexpected run summary: %q", run.Summary)
	}

	var payload struct {
		OK        bool     `json:"ok"`
		Summary   string   `json:"summary"`
		Artifacts []string `json:"artifacts"`
		Result    struct {
			OutputPath string `json:"outputPath"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(run.ResultText), &payload); err != nil {
		t.Fatalf("unmarshal run result failed: %v; result=%s", err, run.ResultText)
	}
	if !payload.OK {
		t.Fatalf("expected payload ok=true, got %+v", payload)
	}
	if payload.Result.OutputPath == "" {
		t.Fatalf("expected outputPath in payload, got %+v result=%s", payload, run.ResultText)
	}
	if len(payload.Artifacts) != 1 || payload.Artifacts[0] != payload.Result.OutputPath {
		t.Fatalf("expected artifacts to contain output path, got %+v", payload)
	}

	data, err := os.ReadFile(payload.Result.OutputPath)
	if err != nil {
		t.Fatalf("read output artifact failed: %v", err)
	}
	if string(data) != "hello integration" {
		t.Fatalf("unexpected artifact content: %q", string(data))
	}
}

func TestAutomationScriptRunWithOptionsPrestartsStoredTargetForConnectOnlyScript(t *testing.T) {
	nodeExecPath := lookupAutomationTestNode(t)

	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()
	app.config.Automation.Enabled = true
	app.config.Automation.NodeSource = config.AutomationNodeSourceSystem
	app.config.Automation.SystemNodePath = nodeExecPath
	app.config.Automation.NodeVersion = "test-node"
	app.config.Automation.PlaywrightCoreVersion = "1.59.0"
	app.config.Automation.RuntimeVersion = "test-runtime"
	app.browserMgr = browser.NewManager(app.config, app.appRoot)
	app.launchCodeSvc = launchcode.NewLaunchCodeService(launchcode.NewMemoryLaunchCodeDAO())
	app.browserMgr.CodeProvider = app.launchCodeSvc
	app.automationMgr = automation.NewManager(app.appRoot, app.config, nil, automation.Options{})

	prepareAutomationTestRuntimeWithPlaywrightModule(
		t,
		app.automationMgr,
		app.config.Automation.PlaywrightCoreVersion,
		automationTestConnectProbePlaywrightModule,
	)

	var debugHits atomic.Int32
	debugServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debugHits.Add(1)
		if r.URL.Path != "/json/version" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Browser": "Chrome/123.0.0.0",
		})
	}))
	defer debugServer.Close()

	debugURL, err := url.Parse(debugServer.URL)
	if err != nil {
		t.Fatalf("parse debug server url failed: %v", err)
	}
	debugPort, err := strconv.Atoi(debugURL.Port())
	if err != nil {
		t.Fatalf("parse debug server port failed: %v", err)
	}

	profile, err := app.browserMgr.Create(browser.ProfileInput{
		ProfileName: "buyer-connect-only",
	})
	if err != nil {
		t.Fatalf("create profile failed: %v", err)
	}
	if profile == nil {
		t.Fatal("create profile returned nil")
	}
	app.browserMgr.Profiles[profile.ProfileId].Running = true
	app.browserMgr.Profiles[profile.ProfileId].DebugReady = true
	app.browserMgr.Profiles[profile.ProfileId].DebugPort = debugPort
	app.browserMgr.Profiles[profile.ProfileId].Pid = 12345

	app.launchServer = launchcode.NewLaunchServer(
		app.launchCodeSvc,
		app,
		app.browserMgr,
		0,
	)
	if err := app.launchServer.Start(); err != nil {
		t.Fatalf("start launch server failed: %v", err)
	}
	defer func() {
		_ = app.launchServer.Stop()
	}()

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:        "playwright-connect-stored-target",
		Name:      "Playwright Connect Stored Target",
		Type:      "playwright-cdp",
		Status:    "ready",
		EntryFile: "scripts/index.cjs",
		ScriptText: "module.exports.run = async ({ connect }) => {\n" +
			"  const { browser } = await connect()\n" +
			"  return { ok: true, summary: 'connected through stored target', contextCount: browser.contexts().length }\n" +
			"}\n",
		TargetConfig: automation.ScriptTargetConfig{
			Mode: "existing",
			Selector: automation.ScriptTargetSelector{
				ProfileID: profile.ProfileId,
			},
		},
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	run, err := app.AutomationScriptRunWithOptions(automation.ScriptRunRequest{
		ScriptID:          saved.ID,
		UseScriptSelector: true,
		UseScriptParams:   true,
	})
	if err != nil {
		t.Fatalf("AutomationScriptRunWithOptions returned error: %v", err)
	}
	if run == nil {
		t.Fatalf("AutomationScriptRunWithOptions returned nil result")
	}
	if run.Status != "success" {
		t.Fatalf("expected success status, got %+v", run)
	}
	if !strings.Contains(run.Summary, "connected through stored target") {
		t.Fatalf("unexpected run summary: %q", run.Summary)
	}
	if !strings.Contains(run.ResultText, `"contextCount":1`) {
		t.Fatalf("expected connect result payload, got %s", run.ResultText)
	}
	if debugHits.Load() == 0 {
		t.Fatalf("expected connect() to hit active debug endpoint through launch server")
	}
}

func lookupAutomationTestNode(t *testing.T) string {
	t.Helper()

	nodeExecPath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	return nodeExecPath
}

func prepareAutomationTestRuntime(t *testing.T, manager *automation.Manager, playwrightVersion string) {
	t.Helper()

	prepareAutomationTestRuntimeWithPlaywrightModule(
		t,
		manager,
		playwrightVersion,
		"module.exports = { chromium: {} }\n",
	)
}

func prepareAutomationTestRuntimeWithPlaywrightModule(t *testing.T, manager *automation.Manager, playwrightVersion string, playwrightModuleSource string) {
	t.Helper()

	state := manager.CurrentState()

	playwrightCoreDir := filepath.Join(state.RuntimeDir, "node_modules", "playwright-core")
	if err := os.MkdirAll(playwrightCoreDir, 0o755); err != nil {
		t.Fatalf("create playwright-core dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(playwrightCoreDir, "package.json"), []byte("{\"name\":\"playwright-core\",\"version\":\""+playwrightVersion+"\"}\n"), 0o644); err != nil {
		t.Fatalf("write playwright-core package failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(playwrightCoreDir, "index.js"), []byte(playwrightModuleSource), 0o644); err != nil {
		t.Fatalf("write playwright-core stub failed: %v", err)
	}
	if err := os.WriteFile(state.RunnerPath, []byte(automationTestRunnerScript), 0o755); err != nil {
		t.Fatalf("write runner script failed: %v", err)
	}
}

const automationTestConnectProbePlaywrightModule = `const http = require('http')

module.exports = {
  chromium: {
    connectOverCDP: async (endpoint) => {
      const target = new URL('/json/version', endpoint)
      await new Promise((resolve, reject) => {
        const req = http.get(target, (res) => {
          res.resume()
          res.on('end', () => {
            const status = res.statusCode || 0
            if (status >= 200 && status < 300) {
              resolve()
              return
            }
            reject(new Error('cdp connect probe failed with http ' + String(status)))
          })
        })
        req.on('error', reject)
      })

      return {
        contexts: () => [{
          pages: () => [],
          newPage: async () => ({})
        }],
        close: async () => {}
      }
    }
  }
}
`

const automationTestRunnerScript = `const fs = require('fs')
const path = require('path')

async function main() {
  const payloadPath = process.argv[2]
  const payload = JSON.parse(fs.readFileSync(payloadPath, 'utf8'))
  const script = require(payload.ScriptPath)
  const startedAt = new Date().toISOString()
  const result = await script.run({
    selector: payload.Selector || {},
    params: payload.Params || {},
    artifact: (name) => {
      const dir = payload.ArtifactDir || path.dirname(payload.ScriptPath)
      fs.mkdirSync(dir, { recursive: true })
      return path.join(dir, name)
    },
    log: () => {},
    launch: async () => ({ ok: true }),
    connect: async () => ({
      browser: { contexts: () => [] },
      context: {
        pages: () => [],
        newPage: async () => ({})
      },
      page: null
    })
  })

  console.log(JSON.stringify({
    ok: result && result.ok !== false,
    summary: result && result.summary ? String(result.summary) : '',
    error: result && result.error ? String(result.error) : '',
    startedAt,
    finishedAt: new Date().toISOString(),
    ...result
  }))
}

main().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error))
  process.exit(1)
})
`
