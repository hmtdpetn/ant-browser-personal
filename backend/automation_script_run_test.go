package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ant-chrome/backend/internal/automation"
)

func TestPreparePlaywrightScriptWorkspaceCopiesScriptDirectory(t *testing.T) {
	app := NewApp(t.TempDir())

	saved, err := app.AutomationScriptSave(automation.ScriptRecord{
		ID:         "workspace-script",
		Name:       "工作区脚本",
		Type:       "playwright-cdp",
		Status:     "ready",
		EntryFile:  "scripts/index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: true, source: 'workspace' })",
	})
	if err != nil {
		t.Fatalf("AutomationScriptSave returned error: %v", err)
	}

	scriptDir, err := app.automationScriptStore().Dir(saved.ID)
	if err != nil {
		t.Fatalf("Dir returned error: %v", err)
	}

	extraHelperPath := filepath.Join(scriptDir, "scripts", "helpers", "format.cjs")
	if err := os.MkdirAll(filepath.Dir(extraHelperPath), 0o755); err != nil {
		t.Fatalf("create helper dir failed: %v", err)
	}
	if err := os.WriteFile(extraHelperPath, []byte("module.exports.format = () => 'helper-ready'"), 0o644); err != nil {
		t.Fatalf("write helper file failed: %v", err)
	}

	assetPath := filepath.Join(scriptDir, "assets", "seed.txt")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatalf("create asset dir failed: %v", err)
	}
	if err := os.WriteFile(assetPath, []byte("seed-ready"), 0o644); err != nil {
		t.Fatalf("write asset file failed: %v", err)
	}

	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	scriptPath, artifactDir, cleanup, err := app.preparePlaywrightScriptWorkspace(runtimeDir, *saved)
	if err != nil {
		t.Fatalf("preparePlaywrightScriptWorkspace returned error: %v", err)
	}
	defer cleanup()

	execRoot := workspaceRootFromScriptPath(t, scriptPath, saved.EntryFile)

	assertFileContent(t, scriptPath, saved.ScriptText)
	assertFileContent(t, filepath.Join(execRoot, "config"), `"id": "workspace-script"`)
	assertFileContent(t, filepath.Join(execRoot, "scripts", "helpers", "format.cjs"), "helper-ready")
	assertFileContent(t, filepath.Join(execRoot, "assets", "seed.txt"), "seed-ready")
	assertFileContent(t, filepath.Join(execRoot, "node_modules", "playwright", "index.js"), "playwright-core")
	assertFileContent(t, filepath.Join(execRoot, "node_modules", "playwright-core", "package.json"), `"name":"playwright-core"`)

	if info, err := os.Stat(artifactDir); err != nil || !info.IsDir() {
		t.Fatalf("expected artifact dir to exist, got err=%v info=%v", err, info)
	}
}

func TestPreparePlaywrightScriptWorkspaceFallsBackWhenScriptDirMissing(t *testing.T) {
	app := NewApp(t.TempDir())

	script := automation.ScriptRecord{
		ID:         "orphan-script",
		Name:       "孤立脚本",
		Type:       "playwright-cdp",
		Status:     "ready",
		EntryFile:  "nested/index.cjs",
		ScriptText: "module.exports.run = async () => ({ ok: true, source: 'orphan' })",
	}

	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	scriptPath, _, cleanup, err := app.preparePlaywrightScriptWorkspace(runtimeDir, script)
	if err != nil {
		t.Fatalf("preparePlaywrightScriptWorkspace returned error: %v", err)
	}

	execRoot := workspaceRootFromScriptPath(t, scriptPath, script.EntryFile)
	assertFileContent(t, scriptPath, script.ScriptText)
	assertFileContent(t, filepath.Join(execRoot, "node_modules", "playwright", "index.js"), "playwright-core")

	cleanup()
	if _, err := os.Stat(execRoot); !os.IsNotExist(err) {
		t.Fatalf("expected cleanup to remove execRoot, got %v", err)
	}
}

func workspaceRootFromScriptPath(t *testing.T, scriptPath string, entryFile string) string {
	t.Helper()

	entryPath := filepath.FromSlash(entryFile)
	if !strings.HasSuffix(scriptPath, entryPath) {
		t.Fatalf("script path %q does not end with entry file %q", scriptPath, entryPath)
	}

	execRoot := strings.TrimSuffix(scriptPath, entryPath)
	return strings.TrimRight(execRoot, `\/`)
}

func assertFileContent(t *testing.T, path string, expectedSubstring string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s failed: %v", path, err)
	}
	if !strings.Contains(string(data), expectedSubstring) {
		t.Fatalf("file %s does not contain %q; got %q", path, expectedSubstring, string(data))
	}
}
