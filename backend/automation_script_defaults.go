package backend

import (
	"os"
	"path/filepath"

	"ant-chrome/backend/internal/automation"
)

const (
	automationScriptDefaultsMarkerName       = "defaults-seeded-v2"
	automationScriptDefaultsLegacyMarkerName = "defaults-seeded-v1"
)

func (a *App) automationScriptDefaultsMarkerPath(name string) string {
	return a.resolveAppPath(filepath.ToSlash(filepath.Join("data", "automation", name)))
}

func (a *App) automationScriptDefaultsInitializedByName(name string) bool {
	info, err := os.Stat(a.automationScriptDefaultsMarkerPath(name))
	return err == nil && !info.IsDir()
}

func (a *App) automationScriptDefaultsInitialized() bool {
	return a.automationScriptDefaultsInitializedByName(automationScriptDefaultsMarkerName)
}

func (a *App) automationScriptDefaultsInitializedLegacy() bool {
	return a.automationScriptDefaultsInitializedByName(automationScriptDefaultsLegacyMarkerName)
}

func (a *App) markAutomationScriptDefaultsInitialized() error {
	markerPath := a.automationScriptDefaultsMarkerPath(automationScriptDefaultsMarkerName)
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(markerPath, []byte("ok\n"), 0o644)
}

func (a *App) ensureAutomationScriptDefaults(store *automation.ScriptStore) error {
	defaults := automation.DefaultScripts()
	items, err := store.List()
	if err != nil {
		return err
	}

	// v2 marker exists: defaults were already initialized or user intentionally removed them.
	if a.automationScriptDefaultsInitialized() {
		return nil
	}

	if len(items) == 0 {
		// Keep legacy behavior for users that had deleted all defaults under v1.
		if a.automationScriptDefaultsInitializedLegacy() {
			return a.markAutomationScriptDefaultsInitialized()
		}

		for _, record := range defaults {
			if _, err := store.Save(record); err != nil {
				return err
			}
		}
		return a.markAutomationScriptDefaultsInitialized()
	}

	// Migration from v1: existing scripts are present, add any missing built-in baselines once.
	if a.automationScriptDefaultsInitializedLegacy() {
		existingIDs := make(map[string]struct{}, len(items))
		for _, item := range items {
			existingIDs[item.ID] = struct{}{}
		}
		for _, record := range defaults {
			if _, exists := existingIDs[record.ID]; exists {
				continue
			}
			if _, err := store.Save(record); err != nil {
				return err
			}
		}
	}
	return a.markAutomationScriptDefaultsInitialized()
}
