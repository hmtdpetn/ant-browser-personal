package backend

import (
	"os"
	"path/filepath"

	"ant-chrome/backend/internal/automation"
)

const (
	automationScriptDefaultsMarkerName = "defaults-seeded-v11"
)

var automationScriptDefaultsLegacyMarkerNames = []string{
	"defaults-seeded-v10",
	"defaults-seeded-v9",
	"defaults-seeded-v8",
	"defaults-seeded-v7",
	"defaults-seeded-v6",
	"defaults-seeded-v5",
	"defaults-seeded-v4",
	"defaults-seeded-v3",
	"defaults-seeded-v2",
}

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

func (a *App) automationScriptDefaultsInitializedAnyLegacy() bool {
	for _, name := range automationScriptDefaultsLegacyMarkerNames {
		if a.automationScriptDefaultsInitializedByName(name) {
			return true
		}
	}
	return false
}

func (a *App) markAutomationScriptDefaultsInitialized() error {
	markerPath := a.automationScriptDefaultsMarkerPath(automationScriptDefaultsMarkerName)
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(markerPath, []byte("ok\n"), 0o644)
}

func (a *App) ensureAutomationScriptDefaults(store *automation.ScriptStore) error {
	defaults, err := automation.DefaultScriptBundles()
	if err != nil {
		return err
	}
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
		if a.automationScriptDefaultsInitializedAnyLegacy() {
			return a.markAutomationScriptDefaultsInitialized()
		}

		importedCount := 0
		var lastImportErr error
		for _, bundle := range defaults {
			if _, err := store.ImportBundle(bundle); err != nil {
				lastImportErr = err
				continue
			}
			importedCount++
		}
		if importedCount == 0 && lastImportErr != nil {
			return lastImportErr
		}
		return a.markAutomationScriptDefaultsInitialized()
	}

	// Migration: existing scripts are present, refresh built-in baselines and add missing ones once.
	if a.automationScriptDefaultsInitializedAnyLegacy() {
		existingByID := make(map[string]automation.ScriptRecord, len(items))
		for _, item := range items {
			existingByID[item.ID] = item
		}
		for _, bundle := range defaults {
			if existing, exists := existingByID[bundle.Record.ID]; exists {
				if existing.Source.Type == "builtin" {
					bundle.Record = mergeBuiltinDefaultScriptForMigration(existing, bundle.Record)
					_, _ = store.ImportBundle(bundle)
				}
				continue
			}
			_, _ = store.ImportBundle(bundle)
		}
	}
	return a.markAutomationScriptDefaultsInitialized()
}

func mergeBuiltinDefaultScriptForMigration(existing automation.ScriptRecord, next automation.ScriptRecord) automation.ScriptRecord {
	next.ID = existing.ID
	next.CreatedAt = existing.CreatedAt
	next.Status = existing.Status
	next.TargetConfig = existing.TargetConfig
	next.PublicAPI.Enabled = existing.PublicAPI.Enabled
	if existing.PublicAPI.Path != "" {
		next.PublicAPI.Path = existing.PublicAPI.Path
	}
	if existing.PublicAPI.TimeoutMs > 0 {
		next.PublicAPI.TimeoutMs = existing.PublicAPI.TimeoutMs
	}
	next.PublicAPI.Variables = mergeBuiltinDefaultPublicAPIVariables(
		existing.PublicAPI.Variables,
		next.PublicAPI.Variables,
	)
	return next
}

func mergeBuiltinDefaultPublicAPIVariables(existing []automation.ScriptPublicAPIVariable, next []automation.ScriptPublicAPIVariable) []automation.ScriptPublicAPIVariable {
	existingByName := make(map[string]automation.ScriptPublicAPIVariable, len(existing))
	for _, variable := range existing {
		if variable.Name != "" {
			existingByName[variable.Name] = variable
		}
	}

	result := make([]automation.ScriptPublicAPIVariable, 0, len(next))
	for _, variable := range next {
		if existingVariable, ok := existingByName[variable.Name]; ok {
			variable.DefaultValue = existingVariable.DefaultValue
			variable.Required = existingVariable.Required
		}
		result = append(result, variable)
	}
	return result
}
