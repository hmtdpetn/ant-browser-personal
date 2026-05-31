package backend

import (
	"ant-chrome/backend/internal/backup"
	"path/filepath"
	"strings"
)

func backupResolveEntryComponentName(entry backup.ScopeEntry) string {
	if desc := strings.TrimSpace(entry.Description); desc != "" {
		return desc
	}
	if entry.ID != "" {
		return entry.ID
	}
	switch entry.Category {
	case backup.CategorySystemConfig:
		return "系统配置"
	case backup.CategoryAppData:
		return "应用数据"
	case backup.CategoryBrowserData:
		return "浏览器数据"
	case backup.CategoryCoreData:
		return "内核数据"
	case backup.CategoryLogs:
		return "日志数据"
	default:
		return "未知组件"
	}
}

func backupResolveManifestComponentName(entry backup.ManifestEntry) string {
	if desc := strings.TrimSpace(entry.Description); desc != "" {
		return desc
	}
	if id := strings.TrimSpace(entry.ID); id != "" {
		return id
	}
	return "未知模块"
}

func backupEnsureZipSuffix(path string) string {
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		return path
	}
	return path + ".zip"
}
