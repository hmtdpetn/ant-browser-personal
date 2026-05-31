package backend

import (
	"fmt"
	"strings"

	"ant-chrome/backend/internal/automation"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) AutomationScriptImportText(text string) (*automation.ScriptRecord, error) {
	bundle, err := automation.ImportBundleFromBytesWithOptions("automation-template.json", []byte(strings.TrimSpace(text)), "文本导入", a.automationScriptImportOptions())
	if err != nil {
		return nil, err
	}
	return a.saveImportedAutomationBundle(bundle)
}

func (a *App) AutomationScriptImportLocalFile() (*automation.ScriptRecord, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("应用上下文未初始化")
	}

	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择脚本文件",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "脚本文件 (*.zip;*.json;*.js;*.cjs;*.mjs;*.ts;*.cts;*.mts)", Pattern: "*.zip;*.json;*.js;*.cjs;*.mjs;*.ts;*.cts;*.mts"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("打开文件对话框失败: %w", err)
	}
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("未选择脚本文件")
	}

	bundle, err := automation.ImportBundleFromFileWithOptions(path, "本地文件 "+path, a.automationScriptImportOptions())
	if err != nil {
		return nil, err
	}
	return a.saveImportedAutomationBundle(bundle)
}

func (a *App) AutomationScriptImportLocalDirectory() (*automation.ScriptRecord, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("应用上下文未初始化")
	}

	path, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择脚本目录",
	})
	if err != nil {
		return nil, fmt.Errorf("打开目录对话框失败: %w", err)
	}
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("未选择脚本目录")
	}

	bundle, err := automation.ImportBundleFromDirectoryWithOptions(path, "", "本地目录 "+path, a.automationScriptImportOptions())
	if err != nil {
		return nil, err
	}
	return a.saveImportedAutomationBundle(bundle)
}

func (a *App) AutomationScriptImportRemote(rawURL string) (*automation.ScriptRecord, error) {
	bundle, err := a.loadAutomationRemoteBundle(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	return a.saveImportedAutomationBundle(bundle)
}

func (a *App) AutomationScriptImportGit(repoURL string, ref string, scriptPath string) (*automation.ScriptRecord, error) {
	bundle, err := a.loadAutomationGitBundle(strings.TrimSpace(repoURL), strings.TrimSpace(ref), strings.TrimSpace(scriptPath))
	if err != nil {
		return nil, err
	}
	return a.saveImportedAutomationBundle(bundle)
}

func (a *App) AutomationScriptRefresh(scriptID string) (*automation.ScriptRecord, error) {
	normalizedID := strings.TrimSpace(scriptID)
	if normalizedID == "" {
		return nil, fmt.Errorf("脚本 ID 不能为空")
	}

	existing, err := a.automationScriptStore().Get(normalizedID)
	if err != nil {
		return nil, fmt.Errorf("读取脚本失败: %w", err)
	}

	bundle, err := a.loadAutomationBundleFromSource(existing.Source)
	if err != nil {
		return nil, err
	}

	bundle.Record.ID = existing.ID
	bundle.Record.CreatedAt = existing.CreatedAt
	bundle.Record.Status = existing.Status

	record, err := a.automationScriptStore().ImportBundle(bundle)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (a *App) saveImportedAutomationBundle(bundle automation.ImportedBundle) (*automation.ScriptRecord, error) {
	record, err := a.automationScriptStore().ImportBundle(bundle)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (a *App) automationScriptImportOptions() automation.ImportOptions {
	if a.config == nil {
		return automation.ImportOptions{}
	}
	return automation.ImportOptions{
		AllowTypeScriptBuild: a.config.Automation.AllowTypeScriptBuild,
	}
}
