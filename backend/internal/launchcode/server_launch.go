package launchcode

import (
	"fmt"
	"net/http"
	"strings"

	"ant-chrome/backend/internal/browser"
)

func (s *LaunchServer) launchSuccessPayload(profile *browser.Profile, launchCode string) map[string]interface{} {
	cdpURL := s.CDPURL()
	cdpPort := s.Port()
	if cdpURL == "" && profile != nil && profile.DebugReady && profile.DebugPort > 0 {
		cdpPort = profile.DebugPort
		cdpURL = fmt.Sprintf("http://127.0.0.1:%d", profile.DebugPort)
	}

	return map[string]interface{}{
		"ok":             true,
		"profileId":      profile.ProfileId,
		"profileName":    profile.ProfileName,
		"launchCode":     launchCode,
		"pid":            profile.Pid,
		"debugPort":      profile.DebugPort,
		"debugReady":     profile.DebugReady,
		"runtimeWarning": profile.RuntimeWarning,
		"cdpPort":        cdpPort,
		"cdpUrl":         cdpURL,
	}
}

func (s *LaunchServer) launchByCode(code string, params LaunchRequestParams) (*browser.Profile, string, int, string) {
	return s.launchBySelectorInternal(normalizeLaunchSelector(LaunchSelector{Code: code}), params, false)
}

func (s *LaunchServer) launchBySelector(selector LaunchSelector, params LaunchRequestParams) (*browser.Profile, string, int, string) {
	return s.launchBySelectorInternal(selector, params, true)
}

func (s *LaunchServer) launchProfile(profileID string, params LaunchRequestParams) (*browser.Profile, error) {
	if starterWithParams, ok := s.starter.(BrowserStarterWithParams); ok {
		profile, err := starterWithParams.StartInstanceWithParams(profileID, params)
		return normalizeLaunchedProfileRuntime(profile), err
	}
	profile, err := s.starter.StartInstance(profileID)
	return normalizeLaunchedProfileRuntime(profile), err
}

func normalizeLaunchedProfileRuntime(profile *browser.Profile) *browser.Profile {
	if profile == nil {
		return nil
	}

	// Backward compatibility: older starter implementations only filled pid/debugPort.
	if !profile.Running && (profile.Pid > 0 || profile.DebugPort > 0) {
		profile.Running = true
	}
	if !profile.DebugReady &&
		profile.DebugPort > 0 &&
		strings.TrimSpace(profile.RuntimeWarning) == "" &&
		(profile.Running || profile.Pid > 0) {
		profile.DebugReady = true
	}

	return profile
}

func (s *LaunchServer) launchBySelectorInternal(selector LaunchSelector, params LaunchRequestParams, allowCodeKeywordFallback bool) (*browser.Profile, string, int, string) {
	var (
		profileID  string
		launchCode string
		err        error
	)

	selector = normalizeLaunchSelector(selector)
	if selector.IsEmpty() {
		return nil, "", http.StatusBadRequest, "selector is required"
	}
	if err = selector.Validate(); err != nil {
		return nil, "", http.StatusBadRequest, err.Error()
	}
	selector = s.withCodeKeywordFallback(selector, allowCodeKeywordFallback)

	if selector.OnlyCode() {
		profileID, err = s.service.Resolve(selector.Code)
		if err != nil {
			return nil, "", http.StatusNotFound, "launch code not found"
		}
		launchCode = selector.Code
	} else {
		profileSnapshot, status, errMsg := s.findProfileBySelector(selector)
		if errMsg != "" {
			if selector.Code != "" {
				launchCode = selector.Code
			}
			return nil, launchCode, status, errMsg
		}
		profileID = profileSnapshot.ProfileId
		launchCode = profileSnapshot.LaunchCode
	}

	profile, err := s.launchProfile(profileID, params)
	if err != nil {
		return nil, launchCode, http.StatusInternalServerError, err.Error()
	}

	if launchCode == "" && s.service != nil && profile != nil {
		if code, codeErr := s.service.EnsureCode(profile.ProfileId); codeErr == nil {
			launchCode = code
		}
	}
	if profile != nil && launchCode != "" {
		profile.LaunchCode = launchCode
	}

	return profile, launchCode, http.StatusOK, ""
}

func (s *LaunchServer) launchAllBySelector(selector LaunchSelector, params LaunchRequestParams) ([]*browser.Profile, int, string) {
	selector = normalizeLaunchSelector(selector)
	if selector.IsEmpty() {
		return nil, http.StatusBadRequest, "selector is required"
	}
	if err := selector.Validate(); err != nil {
		return nil, http.StatusBadRequest, err.Error()
	}
	selector = s.withCodeKeywordFallback(selector, true)

	snapshots, status, errMsg := s.findProfilesBySelector(selector)
	if errMsg != "" {
		return nil, status, errMsg
	}

	profiles := make([]*browser.Profile, 0, len(snapshots))
	for _, snapshot := range snapshots {
		profile, err := s.launchProfile(snapshot.ProfileId, params)
		if err != nil {
			label := strings.TrimSpace(snapshot.ProfileName)
			if label == "" {
				label = snapshot.ProfileId
			}
			return profiles, http.StatusInternalServerError, fmt.Sprintf("failed to start profile %s after launching %d profile(s): %v", label, len(profiles), err)
		}

		launchCode := snapshot.LaunchCode
		if launchCode == "" && s.service != nil && profile != nil {
			if code, codeErr := s.service.EnsureCode(profile.ProfileId); codeErr == nil {
				launchCode = code
			}
		}
		if profile != nil && launchCode != "" {
			profile.LaunchCode = launchCode
		}

		profiles = append(profiles, profile)
	}

	return profiles, http.StatusOK, ""
}

func (s *LaunchServer) withCodeKeywordFallback(selector LaunchSelector, allow bool) LaunchSelector {
	if !allow || strings.TrimSpace(selector.Code) == "" {
		return selector
	}
	if s.service != nil {
		if _, err := s.service.Resolve(selector.Code); err == nil {
			return selector
		}
	}

	fallback := selector
	if strings.TrimSpace(fallback.Key) == "" {
		fallback.Key = selector.Code
	}
	fallback.Code = ""
	return fallback
}

func (s *LaunchServer) launchBatchSuccessPayload(profiles []*browser.Profile) map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(profiles))
	for i, profile := range profiles {
		if profile == nil {
			continue
		}
		item := map[string]interface{}{
			"profileId":      profile.ProfileId,
			"profileName":    profile.ProfileName,
			"launchCode":     profile.LaunchCode,
			"pid":            profile.Pid,
			"debugPort":      profile.DebugPort,
			"debugReady":     profile.DebugReady,
			"runtimeWarning": profile.RuntimeWarning,
			"isActive":       i == len(profiles)-1,
		}
		items = append(items, item)
	}

	activeProfile, _, _ := summarizeLaunchedProfiles(profiles)
	cdpURL := s.CDPURL()
	cdpPort := s.Port()
	if cdpURL == "" && activeProfile != nil && activeProfile.DebugReady && activeProfile.DebugPort > 0 {
		cdpPort = activeProfile.DebugPort
		cdpURL = fmt.Sprintf("http://127.0.0.1:%d", activeProfile.DebugPort)
	}

	payload := map[string]interface{}{
		"ok":        true,
		"matchMode": launchMatchModeAll,
		"count":     len(items),
		"items":     items,
		"cdpPort":   cdpPort,
		"cdpUrl":    cdpURL,
	}
	if activeProfile != nil {
		payload["activeProfileId"] = activeProfile.ProfileId
		payload["activeProfileName"] = activeProfile.ProfileName
	}
	return payload
}

func summarizeLaunchedProfiles(profiles []*browser.Profile) (*browser.Profile, string, string) {
	if len(profiles) == 0 {
		return nil, "", ""
	}

	ids := make([]string, 0, len(profiles))
	names := make([]string, 0, len(profiles))
	var active *browser.Profile
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		active = profile
		ids = append(ids, profile.ProfileId)
		if trimmed := strings.TrimSpace(profile.ProfileName); trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return active, strings.Join(ids, ","), strings.Join(names, ",")
}
