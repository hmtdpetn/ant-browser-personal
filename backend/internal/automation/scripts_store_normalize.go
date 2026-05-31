package automation

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func normalizeScriptRecord(input ScriptRecord, existing ScriptRecord) (ScriptRecord, error) {
	now := time.Now().Format(time.RFC3339)

	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = uuid.NewString()
	}
	if !isSafeScriptID(id) {
		return ScriptRecord{}, fmt.Errorf("script id is invalid")
	}

	entryFile := normalizeScriptEntryFile(input.EntryFile)
	recordType := normalizeScriptType(input.Type)
	recordStatus := normalizeScriptStatus(input.Status)
	packageFormat := normalizeScriptPackageFormat(firstNonEmpty(strings.TrimSpace(input.PackageFormat), strings.TrimSpace(existing.PackageFormat)))
	manifestVersion := normalizeScriptManifestVersion(input.ManifestVersion, existing.ManifestVersion)
	createdAt := firstNonEmpty(strings.TrimSpace(existing.CreatedAt), strings.TrimSpace(input.CreatedAt), now)
	updatedAt := firstNonEmpty(strings.TrimSpace(input.UpdatedAt), now)

	if strings.TrimSpace(input.Name) == "" {
		return ScriptRecord{}, fmt.Errorf("script name is required")
	}

	return ScriptRecord{
		PackageFormat:   packageFormat,
		ManifestVersion: manifestVersion,
		ID:              id,
		Name:            strings.TrimSpace(input.Name),
		Description:     strings.TrimSpace(input.Description),
		Type:            recordType,
		Status:          recordStatus,
		EntryFile:       entryFile,
		Tags:            normalizeScriptTags(input.Tags),
		SelectorText:    normalizeScriptJSONText(input.SelectorText),
		ParamsText:      normalizeScriptJSONText(input.ParamsText),
		ScriptText:      normalizeScriptText(input.ScriptText),
		Notes:           strings.TrimSpace(input.Notes),
		TargetConfig:    normalizeScriptTargetConfig(input.TargetConfig),
		Source:          normalizeScriptSource(input.Source, existing.Source),
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}

func normalizeScriptPackageFormat(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return defaultScriptPackageFormat
	}
	return normalized
}

func normalizeScriptManifestVersion(value int, fallback int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return defaultScriptManifestVersion
}

func normalizeScriptType(value string) string {
	switch strings.TrimSpace(value) {
	case "launch-api":
		return "launch-api"
	default:
		return "playwright-cdp"
	}
}

func normalizeScriptStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "ready":
		return "ready"
	case "disabled":
		return "disabled"
	default:
		return "draft"
	}
}

func normalizeScriptEntryFile(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return defaultScriptEntryFile
	}
	normalized = filepath.ToSlash(filepath.Clean(normalized))
	if normalized == "." || normalized == "/" || normalized == scriptStoreConfigFileName {
		return defaultScriptEntryFile
	}
	if strings.HasPrefix(normalized, "../") || normalized == ".." || filepath.IsAbs(normalized) {
		return defaultScriptEntryFile
	}
	return normalized
}

func normalizeBundleFilePath(value string) (string, error) {
	normalized := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if normalized == "." || normalized == "/" || normalized == "" {
		return "", fmt.Errorf("bundle file path is invalid")
	}
	if strings.HasPrefix(normalized, "../") || normalized == ".." || filepath.IsAbs(normalized) {
		return "", fmt.Errorf("bundle file path is invalid")
	}
	return normalized, nil
}

func normalizeScriptTags(tags []string) []string {
	deduped := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		normalized := strings.TrimSpace(tag)
		if normalized == "" {
			continue
		}
		if _, exists := deduped[normalized]; exists {
			continue
		}
		deduped[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizeScriptJSONText(value string) string {
	return strings.TrimSpace(value)
}

func normalizeScriptText(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

func normalizeScriptSource(input ScriptSource, existing ScriptSource) ScriptSource {
	source := ScriptSource{
		Type:       firstNonEmpty(strings.TrimSpace(input.Type), strings.TrimSpace(existing.Type)),
		URI:        firstNonEmpty(strings.TrimSpace(input.URI), strings.TrimSpace(existing.URI)),
		Ref:        firstNonEmpty(strings.TrimSpace(input.Ref), strings.TrimSpace(existing.Ref)),
		Path:       firstNonEmpty(strings.TrimSpace(input.Path), strings.TrimSpace(existing.Path)),
		ImportedAt: firstNonEmpty(strings.TrimSpace(input.ImportedAt), strings.TrimSpace(existing.ImportedAt)),
	}
	if source.Type == "" && (source.URI != "" || source.Ref != "" || source.Path != "" || source.ImportedAt != "") {
		source.Type = "manual"
	}
	return source
}

func normalizeScriptTargetConfig(input ScriptTargetConfig) ScriptTargetConfig {
	mode := normalizeScriptTargetMode(input.Mode)
	createNameTemplate := strings.TrimSpace(input.CreateNameTemplate)
	if createNameTemplate == "" && mode == "create" {
		createNameTemplate = defaultScriptCreateNameTemplate
	}

	return ScriptTargetConfig{
		Mode:               mode,
		Selector:           normalizeScriptTargetSelector(input.Selector),
		TemplateSelector:   normalizeScriptTargetSelector(input.TemplateSelector),
		CreateNameTemplate: createNameTemplate,
	}
}

func normalizeScriptTargetMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "existing":
		return "existing"
	case "create":
		return "create"
	case "rotate":
		return "rotate"
	default:
		return "manual"
	}
}

func normalizeScriptTargetSelector(input ScriptTargetSelector) ScriptTargetSelector {
	return ScriptTargetSelector{
		Code:        strings.ToUpper(strings.TrimSpace(input.Code)),
		ProfileID:   strings.TrimSpace(input.ProfileID),
		ProfileName: strings.TrimSpace(input.ProfileName),
		GroupID:     strings.TrimSpace(input.GroupID),
		Keywords:    normalizeScriptTags(input.Keywords),
		Tags:        normalizeScriptTags(input.Tags),
	}
}

func isSafeScriptID(value string) bool {
	for _, ch := range value {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '-', ch == '_', ch == '.':
		default:
			return false
		}
	}
	return true
}

func parseRFC3339OrZero(value string) time.Time {
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return ts
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
