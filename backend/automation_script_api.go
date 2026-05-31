package backend

import (
	"path/filepath"
	"strings"

	"ant-chrome/backend/internal/automation"
)

func (a *App) automationScriptStore() *automation.ScriptStore {
	return automation.NewScriptStore(a.resolveAppPath(filepath.ToSlash(filepath.Join("data", "automation", "scripts"))))
}

func (a *App) AutomationScriptList() ([]automation.ScriptRecord, error) {
	store := a.automationScriptStore()
	if err := a.ensureAutomationScriptDefaults(store); err != nil {
		return nil, err
	}

	items, err := store.List()
	if err != nil {
		return nil, err
	}
	return a.enrichAutomationScriptRecords(items), nil
}

func (a *App) AutomationScriptGet(scriptID string) (*automation.ScriptRecord, error) {
	store := a.automationScriptStore()
	if err := a.ensureAutomationScriptDefaults(store); err != nil {
		return nil, err
	}

	record, err := store.Get(scriptID)
	if err != nil {
		return nil, err
	}
	enriched := a.enrichAutomationScriptRecord(record)
	return &enriched, nil
}

func (a *App) AutomationScriptSave(input automation.ScriptRecord) (*automation.ScriptRecord, error) {
	record, err := a.automationScriptStore().Save(a.enrichAutomationScriptRecord(input))
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (a *App) AutomationScriptDelete(scriptID string) error {
	return a.automationScriptStore().Delete(scriptID)
}

func (a *App) enrichAutomationScriptRecords(items []automation.ScriptRecord) []automation.ScriptRecord {
	if len(items) == 0 {
		return []automation.ScriptRecord{}
	}

	result := make([]automation.ScriptRecord, 0, len(items))
	for _, item := range items {
		result = append(result, a.enrichAutomationScriptRecord(item))
	}
	return result
}

func (a *App) enrichAutomationScriptRecord(record automation.ScriptRecord) automation.ScriptRecord {
	switch strings.ToLower(strings.TrimSpace(record.TargetConfig.Mode)) {
	case "existing":
		record.TargetConfig.Selector = a.enrichAutomationExactTargetSelector(record.TargetConfig.Selector)
	case "create":
		record.TargetConfig.TemplateSelector = a.enrichAutomationExactTargetSelector(record.TargetConfig.TemplateSelector)
	}
	return record
}
