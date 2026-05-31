package backend

import (
	"path/filepath"
	"testing"

	"ant-chrome/backend/internal/config"
)

func TestSaveAutomationRuntimeSettingsNormalizesAndPersists(t *testing.T) {
	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()

	state, err := app.SaveAutomationRuntimeSettings(" SYSTEM ", "  C:/tools/node/node.exe  ")
	if err != nil {
		t.Fatalf("SaveAutomationRuntimeSettings returned error: %v", err)
	}

	if app.config.Automation.NodeSource != config.AutomationNodeSourceSystem {
		t.Fatalf("expected node source %q, got %q", config.AutomationNodeSourceSystem, app.config.Automation.NodeSource)
	}
	if app.config.Automation.SystemNodePath != "C:/tools/node/node.exe" {
		t.Fatalf("expected trimmed system node path, got %q", app.config.Automation.SystemNodePath)
	}

	settings, ok := state["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("state.settings should be a map, got %T", state["settings"])
	}
	if settings["nodeSource"] != config.AutomationNodeSourceSystem {
		t.Fatalf("expected settings.nodeSource %q, got %#v", config.AutomationNodeSourceSystem, settings["nodeSource"])
	}
	if settings["systemNodePath"] != "C:/tools/node/node.exe" {
		t.Fatalf("expected settings.systemNodePath to be trimmed, got %#v", settings["systemNodePath"])
	}

	loaded, err := LoadConfig(filepath.Join(app.appRoot, "config.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if loaded.Automation.NodeSource != config.AutomationNodeSourceSystem {
		t.Fatalf("expected persisted node source %q, got %q", config.AutomationNodeSourceSystem, loaded.Automation.NodeSource)
	}
	if loaded.Automation.SystemNodePath != "C:/tools/node/node.exe" {
		t.Fatalf("expected persisted system node path, got %q", loaded.Automation.SystemNodePath)
	}
}

func TestSaveAutomationRuntimeSettingsFallsBackToAutoForUnknownSource(t *testing.T) {
	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()

	if _, err := app.SaveAutomationRuntimeSettings("custom-source", ""); err != nil {
		t.Fatalf("SaveAutomationRuntimeSettings returned error: %v", err)
	}

	if app.config.Automation.NodeSource != config.AutomationNodeSourceAuto {
		t.Fatalf("expected unknown source to normalize to %q, got %q", config.AutomationNodeSourceAuto, app.config.Automation.NodeSource)
	}
}

func TestSaveAutomationSettingsPreservesRuntimeStrategy(t *testing.T) {
	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()
	app.config.Automation.NodeSource = config.AutomationNodeSourceSystem
	app.config.Automation.SystemNodePath = "C:/tools/node/node.exe"

	if _, err := app.SaveAutomationSettings(true, true); err != nil {
		t.Fatalf("SaveAutomationSettings returned error: %v", err)
	}

	if app.config.Automation.NodeSource != config.AutomationNodeSourceSystem {
		t.Fatalf("expected node source to be preserved, got %q", app.config.Automation.NodeSource)
	}
	if app.config.Automation.SystemNodePath != "C:/tools/node/node.exe" {
		t.Fatalf("expected system node path to be preserved, got %q", app.config.Automation.SystemNodePath)
	}
}

func TestSaveAutomationScriptPackageSettingsPersists(t *testing.T) {
	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()

	state, err := app.SaveAutomationScriptPackageSettings(true)
	if err != nil {
		t.Fatalf("SaveAutomationScriptPackageSettings returned error: %v", err)
	}

	if !app.config.Automation.AllowTypeScriptBuild {
		t.Fatalf("expected allowTypeScriptBuild to be enabled in memory")
	}

	settings, ok := state["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("state.settings should be a map, got %T", state["settings"])
	}
	if settings["allowTypeScriptBuild"] != true {
		t.Fatalf("expected settings.allowTypeScriptBuild true, got %#v", settings["allowTypeScriptBuild"])
	}

	loaded, err := LoadConfig(filepath.Join(app.appRoot, "config.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if !loaded.Automation.AllowTypeScriptBuild {
		t.Fatalf("expected persisted allowTypeScriptBuild to be true")
	}
}
