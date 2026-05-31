package backend

import (
	"strings"

	"ant-chrome/backend/internal/automation"
)

func buildAutomationImportSourceLabel(source automation.ScriptSource) string {
	switch strings.TrimSpace(source.Type) {
	case "local-file":
		return "本地文件 " + firstNonBlank(source.URI, source.Path)
	case "local-dir":
		return "本地目录 " + firstNonBlank(source.URI, source.Path)
	case "remote-url":
		return "远程地址 " + strings.TrimSpace(source.URI)
	case "git":
		return buildAutomationGitImportLabel(source.URI, source.Ref, source.Path)
	default:
		return firstNonBlank(source.URI, source.Path)
	}
}

func buildAutomationGitImportLabel(repoURL string, ref string, scriptPath string) string {
	label := "Git " + strings.TrimSpace(repoURL)
	if strings.TrimSpace(ref) != "" {
		label += " @ " + strings.TrimSpace(ref)
	}
	if strings.TrimSpace(scriptPath) != "" {
		label += " : " + strings.TrimSpace(scriptPath)
	}
	return label
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
