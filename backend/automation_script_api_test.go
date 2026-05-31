package backend

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ant-chrome/backend/internal/automation"
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
)

func TestAutomationScriptListSeedsDefaultScriptsOnFreshApp(t *testing.T) {
	app := NewApp(t.TempDir())

	items, err := app.AutomationScriptList()
	if err != nil {
		t.Fatalf("AutomationScriptList returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two default scripts, got %d", len(items))
	}

	byID := make(map[string]automation.ScriptRecord, len(items))
	for _, script := range items {
		byID[script.ID] = script
	}

	expectedNames := map[string]string{
		"dual-instance-runtime-switch": "双实例启动与 Runtime 切换",
		"news-query-txt":               "查询新闻并写 TXT",
	}

	for scriptID, expectedName := range expectedNames {
		script, ok := byID[scriptID]
		if !ok {
			t.Fatalf("missing default script %q", scriptID)
		}
		if script.Name != expectedName {
			t.Fatalf("unexpected default script name for %q: %q", scriptID, script.Name)
		}
		if script.EntryFile != "index.cjs" {
			t.Fatalf("unexpected default entry file for %q: %q", scriptID, script.EntryFile)
		}

		scriptDir := filepath.Join(app.resolveAppPath(filepath.ToSlash(filepath.Join("data", "automation", "scripts"))), script.ID)
		if _, err := os.Stat(filepath.Join(scriptDir, "config")); err != nil {
			t.Fatalf("expected default config to exist for %q: %v", scriptID, err)
		}
		if _, err := os.Stat(filepath.Join(scriptDir, script.EntryFile)); err != nil {
			t.Fatalf("expected default entry file to exist for %q: %v", scriptID, err)
		}
	}

	dualScript := byID[automation.DualInstanceRuntimeScriptID]
	if !strings.Contains(dualScript.ParamsText, `"browsers"`) {
		t.Fatalf("expected dual-instance default params to use browsers array, got %s", dualScript.ParamsText)
	}
	if strings.Contains(dualScript.ParamsText, `"primaryCode"`) {
		t.Fatalf("expected dual-instance default params to drop legacy primaryCode fields, got %s", dualScript.ParamsText)
	}

	for scriptID := range expectedNames {
		if err := app.AutomationScriptDelete(scriptID); err != nil {
			t.Fatalf("AutomationScriptDelete returned error for %q: %v", scriptID, err)
		}
	}

	items, err = app.AutomationScriptList()
	if err != nil {
		t.Fatalf("AutomationScriptList returned error after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected deleted default script not to be re-seeded, got %d items", len(items))
	}
}

func TestAutomationScriptSaveListAndDelete(t *testing.T) {
	app := NewApp(t.TempDir())

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "app-script",
		Name:       "App 脚本",
		Type:       "playwright-cdp",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: true })",
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}
	if saved == nil {
		t.Fatalf("AutomationScriptSave returned nil result")
	}
	if saved.ID != "app-script" {
		t.Fatalf("expected saved id app-script, got %q", saved.ID)
	}

	items, err := app.AutomationScriptList()
	if err != nil {
		t.Fatalf("AutomationScriptList returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one script, got %d", len(items))
	}

	if err := app.AutomationScriptDelete(saved.ID); err != nil {
		t.Fatalf("AutomationScriptDelete returned error: %v", err)
	}

	items, err = app.AutomationScriptList()
	if err != nil {
		t.Fatalf("AutomationScriptList returned error after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected zero scripts after delete, got %d", len(items))
	}
}

func TestAutomationScriptSaveHydratesExactTargetSelectorWithCode(t *testing.T) {
	app := newAutomationTargetTestApp(t)
	profile := createAutomationTargetProfile(t, app, browser.ProfileInput{
		ProfileName: "buyer-001",
	})
	code, err := app.launchCodeSvc.SetCode(profile.ProfileId, "BUYER_001")
	if err != nil {
		t.Fatalf("set code failed: %v", err)
	}

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "app-script",
		Name:       "App 脚本",
		Type:       "playwright-cdp",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: true })",
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
	if saved == nil {
		t.Fatalf("AutomationScriptSave returned nil result")
	}
	if saved.TargetConfig.Selector.ProfileID != profile.ProfileId {
		t.Fatalf("expected profileId to be preserved, got %+v", saved.TargetConfig.Selector)
	}
	if saved.TargetConfig.Selector.Code != code {
		t.Fatalf("expected code snapshot %q, got %+v", code, saved.TargetConfig.Selector)
	}
}

func TestAutomationScriptRunRecordsUnsupportedType(t *testing.T) {
	app := NewApp(t.TempDir())

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "playwright-script",
		Name:       "Playwright 脚本",
		Type:       "playwright-cdp",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: true })",
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	run, err := app.AutomationScriptRun(saved.ID)
	if err != nil {
		t.Fatalf("AutomationScriptRun returned error: %v", err)
	}
	if run == nil {
		t.Fatalf("AutomationScriptRun returned nil result")
	}
	if run.Status != "failed" {
		t.Fatalf("expected unsupported script to fail, got %q", run.Status)
	}
	if run.Error == "" {
		t.Fatalf("expected unsupported script run to contain error")
	}

	runs, err := app.AutomationScriptRunList(10)
	if err != nil {
		t.Fatalf("AutomationScriptRunList returned error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one run record, got %d", len(runs))
	}
}

func TestAutomationScriptRunWithOptionsInvalidSelector(t *testing.T) {
	app := NewApp(t.TempDir())

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:           "launch-script",
		Name:         "Launch 脚本",
		Type:         "launch-api",
		Status:       "ready",
		EntryFile:    "index.cjs",
		SelectorText: `{"code":"BUYER_001"}`,
		ParamsText:   `{"startUrls":["https://example.com"]}`,
		ScriptText:   "export async function run() {}",
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	run, err := app.AutomationScriptRunWithOptions(automation.ScriptRunRequest{
		ScriptID:          saved.ID,
		SelectorText:      "{invalid",
		UseScriptSelector: false,
		UseScriptParams:   true,
	})
	if err != nil {
		t.Fatalf("AutomationScriptRunWithOptions returned error: %v", err)
	}
	if run == nil {
		t.Fatalf("AutomationScriptRunWithOptions returned nil result")
	}
	if run.Status != "failed" {
		t.Fatalf("expected invalid selector run to fail, got %q", run.Status)
	}
	if run.Error == "" {
		t.Fatalf("expected invalid selector run to contain error")
	}
	if run.Summary != "脚本执行失败" {
		t.Fatalf("expected invalid selector summary, got %q", run.Summary)
	}
}

func TestAutomationScriptRunWithOptionsAllowsEmptySelectorForDualInstanceRuntimeScript(t *testing.T) {
	app := NewApp(t.TempDir())

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         automation.DualInstanceRuntimeScriptID,
		Name:       "双实例启动与 Runtime 切换",
		Type:       "launch-api",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ParamsText: `{"browsers":[{"code":"BUYER_001"},{"code":"BUYER_002"}],"timeoutMs":45000}`,
		ScriptText: "export async function run() {}",
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	run, err := app.AutomationScriptRunWithOptions(automation.ScriptRunRequest{
		ScriptID:          saved.ID,
		SelectorText:      "",
		UseScriptSelector: false,
		UseScriptParams:   true,
	})
	if err != nil {
		t.Fatalf("AutomationScriptRunWithOptions returned error: %v", err)
	}
	if run == nil {
		t.Fatalf("AutomationScriptRunWithOptions returned nil result")
	}
	if run.Summary != "双实例流程执行失败" {
		t.Fatalf("expected dual-instance flow to bypass selector validation, got %+v", run)
	}
	if strings.Contains(run.Error, "selector is required") {
		t.Fatalf("expected dual-instance script to allow empty selector, got %+v", run)
	}
}

func TestAutomationScriptRefreshFromLocalFile(t *testing.T) {
	app := NewApp(t.TempDir())

	sourcePath := filepath.Join(t.TempDir(), "demo-script.cjs")
	if err := os.WriteFile(sourcePath, []byte("module.exports.run = async () => ({ ok: true, source: 'local-file' })"), 0o644); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "refresh-local-file",
		Name:       "本地文件脚本",
		Type:       "launch-api",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: false })",
		Source: automation.ScriptSource{
			Type: "local-file",
			URI:  sourcePath,
		},
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	refreshed, err := app.AutomationScriptRefresh(saved.ID)
	if err != nil {
		t.Fatalf("AutomationScriptRefresh returned error: %v", err)
	}
	if refreshed == nil {
		t.Fatalf("AutomationScriptRefresh returned nil result")
	}
	if refreshed.ID != saved.ID {
		t.Fatalf("expected same script id, got %q want %q", refreshed.ID, saved.ID)
	}
	if refreshed.Status != "ready" {
		t.Fatalf("expected status to be preserved, got %q", refreshed.Status)
	}
	if refreshed.Type != "playwright-cdp" {
		t.Fatalf("expected type to follow imported source, got %q", refreshed.Type)
	}
	if refreshed.EntryFile != "demo-script.cjs" {
		t.Fatalf("expected entry file from source bundle, got %q", refreshed.EntryFile)
	}
	if !strings.Contains(refreshed.ScriptText, "source: 'local-file'") {
		t.Fatalf("expected refreshed script text from local file, got %q", refreshed.ScriptText)
	}
	if refreshed.Source.Type != "local-file" || refreshed.Source.URI != sourcePath {
		t.Fatalf("unexpected refreshed source: %+v", refreshed.Source)
	}
	if refreshed.Source.ImportedAt == "" {
		t.Fatalf("expected refreshed source importedAt to be populated")
	}
}

func TestAutomationScriptRefreshFromLocalDirectory(t *testing.T) {
	app := NewApp(t.TempDir())

	sourceDir := filepath.Join(t.TempDir(), "local-dir-script")
	if err := os.MkdirAll(filepath.Join(sourceDir, "scripts", "helpers"), 0o755); err != nil {
		t.Fatalf("create local dir source failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "automation.script.json"), []byte(`{
  "name": "本地目录脚本",
  "type": "playwright-cdp",
  "entryFile": "scripts/index.cjs"
}`), 0o644); err != nil {
		t.Fatalf("write local dir manifest failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "scripts", "index.cjs"), []byte("const helper = require('./helpers/helper.cjs')\nmodule.exports.run = async () => helper.run()"), 0o644); err != nil {
		t.Fatalf("write local dir entry failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "scripts", "helpers", "helper.cjs"), []byte("module.exports.run = async () => ({ ok: true, source: 'local-dir' })"), 0o644); err != nil {
		t.Fatalf("write local dir helper failed: %v", err)
	}

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "refresh-local-dir",
		Name:       "旧本地目录脚本",
		Type:       "launch-api",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: false })",
		Source: automation.ScriptSource{
			Type: "local-dir",
			URI:  sourceDir,
		},
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	refreshed, err := app.AutomationScriptRefresh(saved.ID)
	if err != nil {
		t.Fatalf("AutomationScriptRefresh returned error: %v", err)
	}
	if refreshed == nil {
		t.Fatalf("AutomationScriptRefresh returned nil result")
	}
	if refreshed.ID != saved.ID {
		t.Fatalf("expected same script id, got %q want %q", refreshed.ID, saved.ID)
	}
	if refreshed.Status != "ready" {
		t.Fatalf("expected status to be preserved, got %q", refreshed.Status)
	}
	if refreshed.EntryFile != "scripts/index.cjs" {
		t.Fatalf("expected nested entry file, got %q", refreshed.EntryFile)
	}
	if !strings.Contains(refreshed.ScriptText, "helper.run()") {
		t.Fatalf("expected refreshed script text from local directory, got %q", refreshed.ScriptText)
	}
}

func TestAutomationScriptRefreshFromRemote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
  "manifest": {
    "name": "远程刷新脚本",
    "description": "来自远程",
    "type": "playwright-cdp",
    "entryFile": "index.cjs"
  },
  "script": "module.exports.run = async () => ({ ok: true, source: 'remote' })"
}`))
	}))
	defer server.Close()

	app := NewApp(t.TempDir())
	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "refresh-remote",
		Name:       "旧远程脚本",
		Type:       "launch-api",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: false })",
		Source: automation.ScriptSource{
			Type: "remote-url",
			URI:  server.URL + "/script.json",
		},
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	refreshed, err := app.AutomationScriptRefresh(saved.ID)
	if err != nil {
		t.Fatalf("AutomationScriptRefresh returned error: %v", err)
	}
	if refreshed == nil {
		t.Fatalf("AutomationScriptRefresh returned nil result")
	}
	if refreshed.ID != saved.ID {
		t.Fatalf("expected same script id, got %q want %q", refreshed.ID, saved.ID)
	}
	if refreshed.Name != "远程刷新脚本" {
		t.Fatalf("expected remote manifest name, got %q", refreshed.Name)
	}
	if refreshed.Status != "ready" {
		t.Fatalf("expected status to be preserved, got %q", refreshed.Status)
	}
	if !strings.Contains(refreshed.ScriptText, "source: 'remote'") {
		t.Fatalf("expected refreshed remote script text, got %q", refreshed.ScriptText)
	}
	if refreshed.Source.Type != "remote-url" || refreshed.Source.URI != server.URL+"/script.json" {
		t.Fatalf("unexpected refreshed source: %+v", refreshed.Source)
	}
}

func TestLoadAutomationRemoteBundleSupportsZip(t *testing.T) {
	app := NewApp(t.TempDir())

	zipData := buildAutomationZipBytesForTest(t, map[string]string{
		"automation.script.json": `{
  "name": "远程 ZIP",
  "type": "playwright-cdp",
  "entryFile": "scripts/index.cjs"
}`,
		"scripts/index.cjs": "module.exports.run = async () => ({ ok: true, source: 'remote-zip' })",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(zipData)
	}))
	defer server.Close()

	bundle, err := app.loadAutomationRemoteBundle(server.URL + "/demo.zip")
	if err != nil {
		t.Fatalf("loadAutomationRemoteBundle returned error: %v", err)
	}

	if bundle.Record.Name != "远程 ZIP" {
		t.Fatalf("unexpected bundle name: %s", bundle.Record.Name)
	}
	if bundle.Record.Source.Type != "remote-url" || bundle.Record.Source.URI != server.URL+"/demo.zip" {
		t.Fatalf("unexpected bundle source: %+v", bundle.Record.Source)
	}
	if !strings.Contains(bundle.Record.ScriptText, "remote-zip") {
		t.Fatalf("unexpected script text: %s", bundle.Record.ScriptText)
	}
}

func TestLoadAutomationRemoteBundleBuildsTypeScriptWhenEnabled(t *testing.T) {
	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()
	app.config.Automation.AllowTypeScriptBuild = true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(`export async function run() {
  return { ok: true, source: 'remote-ts' }
}`))
	}))
	defer server.Close()

	bundle, err := app.loadAutomationRemoteBundle(server.URL + "/demo-script.ts")
	if err != nil {
		t.Fatalf("loadAutomationRemoteBundle returned error: %v", err)
	}

	if bundle.Record.EntryFile != "demo-script.cjs" {
		t.Fatalf("unexpected compiled entry file: %s", bundle.Record.EntryFile)
	}
	if !strings.Contains(bundle.Record.ScriptText, "remote-ts") {
		t.Fatalf("unexpected compiled script text: %s", bundle.Record.ScriptText)
	}
	if bundle.Record.Source.Type != "remote-url" || bundle.Record.Source.URI != server.URL+"/demo-script.ts" {
		t.Fatalf("unexpected bundle source: %+v", bundle.Record.Source)
	}
}

func TestAutomationScriptRefreshFromRemoteTypeScriptWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`export async function run() {
  return { ok: true, source: 'remote-ts-refresh' }
}`))
	}))
	defer server.Close()

	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()
	app.config.Automation.AllowTypeScriptBuild = true

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "refresh-remote-ts",
		Name:       "旧远程 TS 脚本",
		Type:       "launch-api",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: false })",
		Source: automation.ScriptSource{
			Type: "remote-url",
			URI:  server.URL + "/refresh-script.ts",
		},
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	refreshed, err := app.AutomationScriptRefresh(saved.ID)
	if err != nil {
		t.Fatalf("AutomationScriptRefresh returned error: %v", err)
	}
	if refreshed.EntryFile != "refresh-script.cjs" {
		t.Fatalf("unexpected refreshed entry file: %s", refreshed.EntryFile)
	}
	if !strings.Contains(refreshed.ScriptText, "remote-ts-refresh") {
		t.Fatalf("unexpected refreshed script text: %s", refreshed.ScriptText)
	}
	if refreshed.Source.Type != "remote-url" || refreshed.Source.URI != server.URL+"/refresh-script.ts" {
		t.Fatalf("unexpected refreshed source: %+v", refreshed.Source)
	}
}

func TestLoadAutomationGitBundleBuildsTypeScriptWhenEnabled(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	repoDir := filepath.Join(t.TempDir(), "automation-ts-repo")
	if err := os.MkdirAll(filepath.Join(repoDir, "scripts", "demo", "helpers"), 0o755); err != nil {
		t.Fatalf("create repo dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "scripts", "demo", "automation.script.json"), []byte(`{
  "name": "Git TS 导入",
  "type": "playwright-cdp",
  "entryFile": "index.ts"
}`), 0o644); err != nil {
		t.Fatalf("write git manifest failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "scripts", "demo", "index.ts"), []byte(`import { flag } from './helpers/flag'

export async function run() {
  return { ok: flag, source: 'git-ts' }
}`), 0o644); err != nil {
		t.Fatalf("write git entry file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "scripts", "demo", "helpers", "flag.ts"), []byte(`export const flag = true`), 0o644); err != nil {
		t.Fatalf("write git helper file failed: %v", err)
	}

	runGitForTest(t, repoDir, "init")
	runGitForTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitForTest(t, repoDir, "config", "user.name", "Test User")
	runGitForTest(t, repoDir, "add", ".")
	runGitForTest(t, repoDir, "commit", "-m", "init")

	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()
	app.config.Automation.AllowTypeScriptBuild = true

	bundle, err := app.loadAutomationGitBundle(repoDir, "", "scripts/demo")
	if err != nil {
		t.Fatalf("loadAutomationGitBundle returned error: %v", err)
	}

	if bundle.Record.Name != "Git TS 导入" {
		t.Fatalf("unexpected bundle name: %s", bundle.Record.Name)
	}
	if bundle.Record.EntryFile != "index.cjs" {
		t.Fatalf("unexpected compiled entry file: %s", bundle.Record.EntryFile)
	}
	if !strings.Contains(bundle.Record.ScriptText, "git-ts") {
		t.Fatalf("unexpected compiled script text: %s", bundle.Record.ScriptText)
	}
	if bundle.Record.Source.Type != "git" || bundle.Record.Source.URI != repoDir || bundle.Record.Source.Path != "scripts/demo" {
		t.Fatalf("unexpected bundle source: %+v", bundle.Record.Source)
	}
}

func TestAutomationScriptRefreshFromGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	repoDir := filepath.Join(t.TempDir(), "automation-repo")
	if err := os.MkdirAll(filepath.Join(repoDir, "scripts", "demo"), 0o755); err != nil {
		t.Fatalf("create repo dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "scripts", "demo", "automation.script.json"), []byte(`{
  "name": "Git 刷新脚本",
  "type": "playwright-cdp",
  "entryFile": "index.cjs"
}`), 0o644); err != nil {
		t.Fatalf("write git manifest failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "scripts", "demo", "index.cjs"), []byte("module.exports.run = async () => ({ ok: true, source: 'git' })"), 0o644); err != nil {
		t.Fatalf("write git entry file failed: %v", err)
	}

	runGitForTest(t, repoDir, "init")
	runGitForTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitForTest(t, repoDir, "config", "user.name", "Test User")
	runGitForTest(t, repoDir, "add", ".")
	runGitForTest(t, repoDir, "commit", "-m", "init")

	app := NewApp(t.TempDir())
	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "refresh-git",
		Name:       "旧 Git 脚本",
		Type:       "launch-api",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: false })",
		Source: automation.ScriptSource{
			Type: "git",
			URI:  repoDir,
			Path: "scripts/demo",
		},
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	refreshed, err := app.AutomationScriptRefresh(saved.ID)
	if err != nil {
		t.Fatalf("AutomationScriptRefresh returned error: %v", err)
	}
	if refreshed == nil {
		t.Fatalf("AutomationScriptRefresh returned nil result")
	}
	if refreshed.ID != saved.ID {
		t.Fatalf("expected same script id, got %q want %q", refreshed.ID, saved.ID)
	}
	if refreshed.Name != "Git 刷新脚本" {
		t.Fatalf("expected git manifest name, got %q", refreshed.Name)
	}
	if refreshed.Status != "ready" {
		t.Fatalf("expected status to be preserved, got %q", refreshed.Status)
	}
	if !strings.Contains(refreshed.ScriptText, "source: 'git'") {
		t.Fatalf("expected refreshed git script text, got %q", refreshed.ScriptText)
	}
	if refreshed.Source.Type != "git" || refreshed.Source.URI != repoDir || refreshed.Source.Path != "scripts/demo" {
		t.Fatalf("unexpected refreshed source: %+v", refreshed.Source)
	}
}

func TestAutomationScriptRefreshRejectsUnsupportedSource(t *testing.T) {
	app := NewApp(t.TempDir())
	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "refresh-manual",
		Name:       "手动脚本",
		Type:       "playwright-cdp",
		Status:     "ready",
		EntryFile:  "index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: true })",
		Source: automation.ScriptSource{
			Type: "manual",
		},
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	if _, err := app.AutomationScriptRefresh(saved.ID); err == nil {
		t.Fatalf("expected unsupported source refresh to fail")
	}
}

func runGitForTest(t *testing.T, workdir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

func buildAutomationZipBytesForTest(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	paths := make([]string, 0, len(files))
	for relativePath := range files {
		paths = append(paths, relativePath)
	}
	sort.Strings(paths)

	for _, relativePath := range paths {
		entry, err := writer.Create(relativePath)
		if err != nil {
			t.Fatalf("create zip entry failed: %v", err)
		}
		if _, err := entry.Write([]byte(files[relativePath])); err != nil {
			t.Fatalf("write zip entry failed: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer failed: %v", err)
	}
	return buf.Bytes()
}
