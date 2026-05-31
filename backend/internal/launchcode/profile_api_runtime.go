package launchcode

import (
	"fmt"
	"net/http"
	"strings"

	"ant-chrome/backend/internal/browser"
)

func (s *LaunchServer) handleProfileStatus(w http.ResponseWriter, r *http.Request, profileID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": "method not allowed",
		})
		return
	}

	profile, status, errMsg := s.statusProfile(profileID)
	if errMsg != "" {
		writeJSON(w, status, map[string]interface{}{
			"ok":    false,
			"error": errMsg,
		})
		return
	}

	writeJSON(w, http.StatusOK, s.profileRuntimePayload(profile))
}

func (s *LaunchServer) handleStopProfile(w http.ResponseWriter, r *http.Request, profileID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": "method not allowed",
		})
		return
	}

	profile, status, errMsg := s.stopProfile(profileID)
	if errMsg != "" {
		writeJSON(w, status, map[string]interface{}{
			"ok":    false,
			"error": errMsg,
		})
		return
	}

	payload := s.profileRuntimePayload(profile)
	payload["stopped"] = true
	writeJSON(w, http.StatusOK, payload)
}

func (s *LaunchServer) statusProfile(profileID string) (*browser.Profile, int, string) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil, http.StatusNotFound, "profile not found"
	}

	if provider, ok := s.starter.(BrowserStatusProvider); ok {
		profile, err := provider.StatusInstance(profileID)
		if err != nil {
			return nil, mapProfileWriteErrorStatus(err), err.Error()
		}
		return s.normalizeRuntimeProfile(profile), http.StatusOK, ""
	}

	return s.profileSnapshotByID(profileID)
}

func (s *LaunchServer) stopProfile(profileID string) (*browser.Profile, int, string) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil, http.StatusNotFound, "profile not found"
	}

	stopper, ok := s.starter.(BrowserStopper)
	if !ok {
		return nil, http.StatusServiceUnavailable, "profile runtime control is not available"
	}

	profile, err := stopper.StopInstance(profileID)
	if err != nil {
		return nil, mapProfileWriteErrorStatus(err), err.Error()
	}

	snapshot := s.normalizeRuntimeProfile(profile)
	if snapshot == nil {
		return nil, http.StatusInternalServerError, "profile stop returned nil profile"
	}
	if !snapshot.Running {
		s.ClearActiveProfile(snapshot.ProfileId)
	}
	return snapshot, http.StatusOK, ""
}

func (s *LaunchServer) normalizeRuntimeProfile(profile *browser.Profile) *browser.Profile {
	if profile == nil {
		return nil
	}

	snapshot := *profile
	snapshot.LaunchCode = s.resolveProfileLaunchCode(snapshot.ProfileId, snapshot.LaunchCode)
	return &snapshot
}

func (s *LaunchServer) profileRuntimePayload(profile *browser.Profile) map[string]interface{} {
	normalized := s.normalizeRuntimeProfile(profile)
	if normalized == nil {
		return map[string]interface{}{
			"ok":    false,
			"error": "profile runtime is not available",
		}
	}

	activePort, activeID, _ := s.activeTarget()
	active := strings.TrimSpace(normalized.ProfileId) != "" && normalized.ProfileId == activeID && activePort > 0

	directDebugURL := ""
	if normalized.DebugReady && normalized.DebugPort > 0 {
		directDebugURL = fmt.Sprintf("http://127.0.0.1:%d", normalized.DebugPort)
	}

	cdpPort := 0
	cdpURL := ""
	if active {
		cdpPort = s.Port()
		cdpURL = s.CDPURL()
		if cdpURL == "" && directDebugURL != "" {
			cdpPort = normalized.DebugPort
			cdpURL = directDebugURL
		}
	}

	return map[string]interface{}{
		"ok":             true,
		"profileId":      normalized.ProfileId,
		"profileName":    normalized.ProfileName,
		"launchCode":     normalized.LaunchCode,
		"running":        normalized.Running,
		"pid":            normalized.Pid,
		"debugPort":      normalized.DebugPort,
		"debugReady":     normalized.DebugReady,
		"runtimeWarning": normalized.RuntimeWarning,
		"lastError":      normalized.LastError,
		"lastStartAt":    normalized.LastStartAt,
		"lastStopAt":     normalized.LastStopAt,
		"active":         active,
		"cdpPort":        cdpPort,
		"cdpUrl":         cdpURL,
		"directDebugUrl": directDebugURL,
		"profile":        normalized,
	}
}
