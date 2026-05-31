package backend

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"ant-chrome/backend/internal/automation"
	"ant-chrome/backend/internal/browser"
)

const defaultAutomationCreateNameTemplate = "${templateName}-${timestamp}"

func (a *App) resolveAutomationEffectiveSelector(script automation.ScriptRecord, input automation.ScriptRunRequest, required bool) (map[string]any, string, error) {
	overrideSelectorText := strings.TrimSpace(input.SelectorText)
	if !input.UseScriptSelector && overrideSelectorText != "" {
		selector, err := parseAutomationJSONObject(overrideSelectorText, required)
		return selector, "", err
	}

	if strings.TrimSpace(script.TargetConfig.Mode) != "" && !strings.EqualFold(script.TargetConfig.Mode, "manual") {
		return a.resolveAutomationScriptTarget(script)
	}

	selectorText := resolveAutomationRunJSONText(input.SelectorText, script.SelectorText, input.UseScriptSelector)
	selector, err := parseAutomationJSONObject(selectorText, required)
	return selector, "", err
}

func (a *App) resolveAutomationScriptTarget(script automation.ScriptRecord) (map[string]any, string, error) {
	switch strings.ToLower(strings.TrimSpace(script.TargetConfig.Mode)) {
	case "existing":
		profile, err := a.resolveAutomationExactTargetProfile(script.TargetConfig.Selector, "使用已有实例")
		if err != nil {
			return nil, "", err
		}
		return automationProfileSelector(profile.ProfileId), automationProfileLabel(profile), nil
	case "rotate":
		profiles, err := a.resolveAutomationTargetProfiles(script.TargetConfig.Selector, "按条件轮询实例")
		if err != nil {
			return nil, "", err
		}
		profile := a.pickAutomationRoundRobinTarget(script.ID, script.TargetConfig.Selector, profiles)
		return automationProfileSelector(profile.ProfileId), fmt.Sprintf("轮询实例 %s", automationProfileLabel(profile)), nil
	case "create":
		templateProfile, err := a.resolveAutomationExactTargetProfile(script.TargetConfig.TemplateSelector, "按模板新建实例")
		if err != nil {
			return nil, "", err
		}
		createdName := buildAutomationCreatedProfileName(script.TargetConfig.CreateNameTemplate, script, templateProfile)
		createdProfile, err := a.browserMgr.Copy(templateProfile.ProfileId, createdName)
		if err != nil {
			return nil, "", fmt.Errorf("按模板新建实例失败: %w", err)
		}
		if createdProfile == nil {
			return nil, "", fmt.Errorf("按模板新建实例失败：未返回新实例")
		}
		return automationProfileSelector(createdProfile.ProfileId), fmt.Sprintf("新建实例 %s", automationProfileLabel(*createdProfile)), nil
	default:
		return map[string]any{}, "", nil
	}
}

func (a *App) resolveAutomationExactTargetProfile(selector automation.ScriptTargetSelector, actionLabel string) (browser.Profile, error) {
	if profile, ok := a.findAutomationTargetProfileByIDOrCode(selector); ok {
		return profile, nil
	}
	return a.resolveAutomationTargetProfile(selector, actionLabel)
}

func (a *App) resolveAutomationTargetProfile(selector automation.ScriptTargetSelector, actionLabel string) (browser.Profile, error) {
	profiles, err := a.resolveAutomationTargetProfiles(selector, actionLabel)
	if err != nil {
		return browser.Profile{}, err
	}
	if len(profiles) > 1 {
		return browser.Profile{}, fmt.Errorf("%s失败：%s", actionLabel, buildAutomationTargetAmbiguousError(profiles))
	}
	return profiles[0], nil
}

func (a *App) resolveAutomationTargetProfiles(selector automation.ScriptTargetSelector, actionLabel string) ([]browser.Profile, error) {
	if a.browserMgr == nil {
		return nil, fmt.Errorf("%s失败：实例管理器未初始化", actionLabel)
	}

	normalized := normalizeAutomationTargetSelector(selector)
	if automationTargetSelectorEmpty(normalized) {
		return nil, fmt.Errorf("%s失败：请至少填写一个实例条件", actionLabel)
	}

	snapshots := a.browserMgr.List()
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("%s失败：当前没有可用实例", actionLabel)
	}

	if normalized.Code != "" {
		snapshots = filterAutomationProfiles(snapshots, func(item browser.Profile) bool {
			return strings.EqualFold(strings.TrimSpace(item.LaunchCode), normalized.Code)
		})
	}
	if normalized.ProfileID != "" {
		snapshots = filterAutomationProfiles(snapshots, func(item browser.Profile) bool {
			return strings.TrimSpace(item.ProfileId) == normalized.ProfileID
		})
	}
	if normalized.ProfileName != "" {
		snapshots = filterAutomationProfiles(snapshots, func(item browser.Profile) bool {
			return strings.EqualFold(strings.TrimSpace(item.ProfileName), normalized.ProfileName)
		})
	}
	if normalized.GroupID != "" {
		snapshots = filterAutomationProfiles(snapshots, func(item browser.Profile) bool {
			return strings.TrimSpace(item.GroupId) == normalized.GroupID
		})
	}
	if len(normalized.Tags) > 0 {
		snapshots = filterAutomationProfiles(snapshots, func(item browser.Profile) bool {
			return automationProfileHasAllTags(item, normalized.Tags)
		})
	}
	if len(normalized.Keywords) > 0 {
		snapshots = filterAutomationProfiles(snapshots, func(item browser.Profile) bool {
			return automationProfileMatchesAllKeywordQueries(item, normalized.Keywords)
		})
	}

	if len(snapshots) == 0 {
		return nil, fmt.Errorf("%s失败：没有匹配到实例", actionLabel)
	}

	sortAutomationProfilesForTarget(snapshots)
	return snapshots, nil
}

func normalizeAutomationTargetSelector(selector automation.ScriptTargetSelector) automation.ScriptTargetSelector {
	return automation.ScriptTargetSelector{
		Code:        strings.ToUpper(strings.TrimSpace(selector.Code)),
		ProfileID:   strings.TrimSpace(selector.ProfileID),
		ProfileName: strings.TrimSpace(selector.ProfileName),
		GroupID:     strings.TrimSpace(selector.GroupID),
		Keywords:    normalizeAutomationTargetTerms(selector.Keywords),
		Tags:        normalizeAutomationTargetTerms(selector.Tags),
	}
}

func (a *App) findAutomationTargetProfileByIDOrCode(selector automation.ScriptTargetSelector) (browser.Profile, bool) {
	if a.browserMgr == nil {
		return browser.Profile{}, false
	}

	normalizedProfileID := strings.TrimSpace(selector.ProfileID)
	normalizedCode := strings.ToUpper(strings.TrimSpace(selector.Code))
	if normalizedProfileID == "" && normalizedCode == "" {
		return browser.Profile{}, false
	}

	snapshots := a.browserMgr.List()
	if normalizedProfileID != "" {
		for _, item := range snapshots {
			if strings.TrimSpace(item.ProfileId) == normalizedProfileID {
				return item, true
			}
		}
	}

	if normalizedCode != "" {
		for _, item := range snapshots {
			if strings.EqualFold(strings.TrimSpace(item.LaunchCode), normalizedCode) {
				return item, true
			}
		}
	}

	return browser.Profile{}, false
}

func (a *App) enrichAutomationExactTargetSelector(selector automation.ScriptTargetSelector) automation.ScriptTargetSelector {
	normalized := normalizeAutomationTargetSelector(selector)
	profile, ok := a.findAutomationTargetProfileByIDOrCode(normalized)
	if !ok {
		return normalized
	}

	normalized.ProfileID = strings.TrimSpace(profile.ProfileId)
	if code := strings.ToUpper(strings.TrimSpace(profile.LaunchCode)); code != "" {
		normalized.Code = code
	}
	return normalized
}

func normalizeAutomationTargetTerms(items []string) []string {
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

func automationTargetSelectorEmpty(selector automation.ScriptTargetSelector) bool {
	return selector.Code == "" &&
		selector.ProfileID == "" &&
		selector.ProfileName == "" &&
		selector.GroupID == "" &&
		len(selector.Keywords) == 0 &&
		len(selector.Tags) == 0
}

func filterAutomationProfiles(items []browser.Profile, keep func(browser.Profile) bool) []browser.Profile {
	filtered := make([]browser.Profile, 0, len(items))
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func automationProfileHasAllTags(profile browser.Profile, required []string) bool {
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

func automationProfileMatchesAllKeywordQueries(profile browser.Profile, queries []string) bool {
	if len(queries) == 0 {
		return true
	}
	if len(profile.Keywords) == 0 {
		return false
	}

	for _, query := range queries {
		queryLower := strings.ToLower(strings.TrimSpace(query))
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

func sortAutomationProfilesForTarget(items []browser.Profile) {
	sort.Slice(items, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(items[i].ProfileName))
		rightName := strings.ToLower(strings.TrimSpace(items[j].ProfileName))
		if leftName != rightName {
			return leftName < rightName
		}
		return items[i].ProfileId < items[j].ProfileId
	})
}

func buildAutomationTargetAmbiguousError(items []browser.Profile) string {
	const maxPreview = 5
	parts := make([]string, 0, minAutomationInt(len(items), maxPreview))
	for i := 0; i < len(items) && i < maxPreview; i++ {
		parts = append(parts, automationProfileLabel(items[i]))
	}
	suffix := ""
	if len(items) > maxPreview {
		suffix = fmt.Sprintf(" 等 %d 个实例", len(items))
	}
	return fmt.Sprintf("命中了多个实例：%s%s。请改用 code/profileId，或继续加分组、标签、关键字缩小范围", strings.Join(parts, "，"), suffix)
}

func automationProfileLabel(profile browser.Profile) string {
	label := strings.TrimSpace(profile.ProfileName)
	if label == "" {
		label = strings.TrimSpace(profile.ProfileId)
	}
	if code := strings.TrimSpace(profile.LaunchCode); code != "" {
		return fmt.Sprintf("%s[id=%s, code=%s]", label, profile.ProfileId, code)
	}
	return fmt.Sprintf("%s[id=%s]", label, profile.ProfileId)
}

func automationProfileSelector(profileID string) map[string]any {
	return map[string]any{
		"profileId": strings.TrimSpace(profileID),
	}
}

func (a *App) pickAutomationRoundRobinTarget(scriptID string, selector automation.ScriptTargetSelector, profiles []browser.Profile) browser.Profile {
	rotationKey := buildAutomationTargetRotationKey(scriptID, selector)

	a.automationTargetMu.Lock()
	defer a.automationTargetMu.Unlock()

	lastProfileID := strings.TrimSpace(a.automationTargetCursor[rotationKey])
	nextIndex := 0
	if lastProfileID != "" {
		for idx, profile := range profiles {
			if profile.ProfileId == lastProfileID {
				nextIndex = (idx + 1) % len(profiles)
				break
			}
		}
	}

	selected := profiles[nextIndex]
	a.automationTargetCursor[rotationKey] = selected.ProfileId
	return selected
}

func buildAutomationTargetRotationKey(scriptID string, selector automation.ScriptTargetSelector) string {
	payload := normalizeAutomationTargetSelector(selector)
	data, err := json.Marshal(payload)
	if err != nil {
		return strings.TrimSpace(scriptID)
	}
	return strings.TrimSpace(scriptID) + ":" + string(data)
}

func buildAutomationCreatedProfileName(template string, script automation.ScriptRecord, source browser.Profile) string {
	now := time.Now()
	pattern := strings.TrimSpace(template)
	if pattern == "" {
		pattern = defaultAutomationCreateNameTemplate
	}

	replacements := map[string]string{
		"${timestamp}":    now.Format("20060102-150405"),
		"${date}":         now.Format("20060102"),
		"${time}":         now.Format("150405"),
		"${templateName}": strings.TrimSpace(source.ProfileName),
		"${scriptName}":   strings.TrimSpace(script.Name),
	}
	for placeholder, value := range replacements {
		pattern = strings.ReplaceAll(pattern, placeholder, value)
	}
	pattern = strings.TrimSpace(pattern)
	if pattern != "" {
		return pattern
	}

	templateName := strings.TrimSpace(source.ProfileName)
	if templateName == "" {
		templateName = "自动化实例"
	}
	return fmt.Sprintf("%s-%s", templateName, now.Format("20060102-150405"))
}

func appendAutomationRunSummary(summary string, targetSummary string) string {
	summary = strings.TrimSpace(summary)
	targetSummary = strings.TrimSpace(targetSummary)
	if targetSummary == "" {
		return summary
	}
	if summary == "" {
		return targetSummary
	}
	return summary + " · " + targetSummary
}

func minAutomationInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
