package launchcode

import (
	"fmt"
	"strings"
)

const (
	launchMatchModeUnique = "unique"
	launchMatchModeFirst  = "first"
	launchMatchModeAll    = "all"
)

// LaunchSelector 定义实例选择条件。
// 推荐在 POST /api/launch 中通过 selector 传入，兼容旧版 top-level code 用法。
type LaunchSelector struct {
	Code        string   `json:"code,omitempty"`
	Key         string   `json:"key,omitempty"`
	ProfileID   string   `json:"profileId,omitempty"`
	ProfileName string   `json:"profileName,omitempty"`
	Keyword     string   `json:"keyword,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Tag         string   `json:"tag,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	GroupID     string   `json:"groupId,omitempty"`
	MatchMode   string   `json:"matchMode,omitempty"`
}

func mergeLaunchSelector(req LaunchRequest) LaunchSelector {
	var nested LaunchSelector
	if req.Selector != nil {
		nested = *req.Selector
	}

	return normalizeLaunchSelector(buildMergedSelector(selectorMergeInput{
		Code:        firstNonEmpty(nested.Code, req.Code),
		Key:         firstNonEmpty(nested.Key, req.Key),
		ProfileID:   firstNonEmpty(nested.ProfileID, req.ProfileID),
		ProfileName: firstNonEmpty(nested.ProfileName, req.ProfileName),
		Keywords:    appendSelectorTerms(nil, "", nested.Keywords, nested.Keyword, req.Keyword, req.Keywords),
		Tags:        appendSelectorTerms(nil, nested.Tag, nested.Tags, req.Tag, req.Tags),
		GroupID:     firstNonEmpty(nested.GroupID, req.GroupID),
		MatchMode:   firstNonEmpty(nested.MatchMode, req.MatchMode),
	}))
}

func normalizeLaunchSelector(selector LaunchSelector) LaunchSelector {
	return normalizeSelectorWithDefault(selector, defaultLaunchMatchMode)
}

func normalizeRuntimeSelector(selector LaunchSelector) LaunchSelector {
	return normalizeSelectorWithDefault(selector, defaultRuntimeMatchMode)
}

type selectorMergeInput struct {
	Code        string
	Key         string
	ProfileID   string
	ProfileName string
	Keywords    []string
	Tags        []string
	GroupID     string
	MatchMode   string
}

func buildMergedSelector(input selectorMergeInput) LaunchSelector {
	return LaunchSelector{
		Code:        input.Code,
		Key:         input.Key,
		ProfileID:   input.ProfileID,
		ProfileName: input.ProfileName,
		Keywords:    input.Keywords,
		Tags:        input.Tags,
		GroupID:     input.GroupID,
		MatchMode:   input.MatchMode,
	}
}

func normalizeSelectorWithDefault(selector LaunchSelector, defaultMode func(LaunchSelector) string) LaunchSelector {
	selector.Code = normalizeCode(selector.Code)
	selector.Key = strings.TrimSpace(selector.Key)
	selector.Keywords = normalizeSelectorTerms(appendSelectorTerms(nil, "", selector.Keywords, selector.Keyword))
	selector.Tags = normalizeSelectorTerms(appendSelectorTerms(nil, selector.Tag, selector.Tags))
	selector.ProfileID = strings.TrimSpace(selector.ProfileID)
	selector.ProfileName = strings.TrimSpace(selector.ProfileName)
	selector.GroupID = strings.TrimSpace(selector.GroupID)
	selector.MatchMode = strings.ToLower(strings.TrimSpace(selector.MatchMode))
	if selector.MatchMode == "" {
		selector.MatchMode = defaultMode(selector)
	}
	selector.Keyword = ""
	selector.Tag = ""
	return selector
}

func (selector LaunchSelector) IsEmpty() bool {
	return selector.Code == "" &&
		selector.Key == "" &&
		selector.ProfileID == "" &&
		selector.ProfileName == "" &&
		selector.GroupID == "" &&
		len(selector.Keywords) == 0 &&
		len(selector.Tags) == 0
}

func (selector LaunchSelector) OnlyCode() bool {
	return selector.Code != "" &&
		selector.Key == "" &&
		selector.ProfileID == "" &&
		selector.ProfileName == "" &&
		selector.GroupID == "" &&
		len(selector.Keywords) == 0 &&
		len(selector.Tags) == 0
}

func (selector LaunchSelector) Validate() error {
	switch selector.MatchMode {
	case "", launchMatchModeUnique, launchMatchModeFirst, launchMatchModeAll:
		return nil
	default:
		return fmt.Errorf("matchMode must be unique, first or all")
	}
}

func defaultLaunchMatchMode(selector LaunchSelector) string {
	if selector.Code != "" || selector.Key != "" || len(selector.Keywords) > 0 {
		return launchMatchModeFirst
	}
	return launchMatchModeUnique
}

func defaultRuntimeMatchMode(_ LaunchSelector) string {
	return launchMatchModeUnique
}
