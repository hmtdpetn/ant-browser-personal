package launchcode

import (
	"ant-chrome/backend/internal/browser"
	"fmt"
	"strings"
)

func profileHasAllTags(profile browser.Profile, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(profile.Tags) == 0 {
		return false
	}

	for _, want := range required {
		found := false
		for _, tag := range profile.Tags {
			if strings.EqualFold(strings.TrimSpace(tag), want) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func profileHasExactKeyword(profile browser.Profile, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" || len(profile.Keywords) == 0 {
		return false
	}

	for _, keyword := range profile.Keywords {
		if strings.EqualFold(strings.TrimSpace(keyword), expected) {
			return true
		}
	}
	return false
}

func profileMatchesAllKeywordQueries(profile browser.Profile, queries []string) bool {
	if len(queries) == 0 {
		return true
	}
	if len(profile.Keywords) == 0 {
		return false
	}

	for _, query := range queries {
		queryLower := strings.ToLower(query)
		found := false
		for _, keyword := range profile.Keywords {
			if strings.Contains(strings.ToLower(strings.TrimSpace(keyword)), queryLower) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func buildAmbiguousSelectorError(items []browser.Profile) string {
	const maxPreview = 5
	parts := make([]string, 0, minInt(len(items), maxPreview))
	for i := 0; i < len(items) && i < maxPreview; i++ {
		label := strings.TrimSpace(items[i].ProfileName)
		if label == "" {
			label = items[i].ProfileId
		}
		if items[i].LaunchCode != "" {
			parts = append(parts, fmt.Sprintf("%s[id=%s, code=%s]", label, items[i].ProfileId, items[i].LaunchCode))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s[id=%s]", label, items[i].ProfileId))
	}
	suffix := ""
	if len(items) > maxPreview {
		suffix = fmt.Sprintf(" ... and %d more", len(items)-maxPreview)
	}
	return fmt.Sprintf("selector matched %d profiles: %s%s; use code/profileId or add groupId/tags/keywords, or set matchMode=first", len(items), strings.Join(parts, ", "), suffix)
}

func appendSelectorTerms(dst []string, single string, many []string, moreSinglesAndSlices ...interface{}) []string {
	if trimmed := strings.TrimSpace(single); trimmed != "" {
		dst = append(dst, trimmed)
	}
	dst = append(dst, many...)
	for _, item := range moreSinglesAndSlices {
		switch v := item.(type) {
		case string:
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				dst = append(dst, trimmed)
			}
		case []string:
			dst = append(dst, v...)
		}
	}
	return dst
}

func normalizeSelectorTerms(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
