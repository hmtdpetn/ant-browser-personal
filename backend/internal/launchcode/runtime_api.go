package launchcode

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"ant-chrome/backend/internal/browser"
)

type RuntimeRequest struct {
	Code        string          `json:"code"`
	Key         string          `json:"key"`
	ProfileID   string          `json:"profileId"`
	ProfileName string          `json:"profileName"`
	Keyword     string          `json:"keyword"`
	Keywords    []string        `json:"keywords"`
	Tag         string          `json:"tag"`
	Tags        []string        `json:"tags"`
	GroupID     string          `json:"groupId"`
	MatchMode   string          `json:"matchMode"`
	Selector    *LaunchSelector `json:"selector"`
}

func decodeRuntimeRequest(r *http.Request) (RuntimeRequest, int, string) {
	if r.Method != http.MethodPost {
		return RuntimeRequest{}, http.StatusMethodNotAllowed, "method not allowed"
	}

	var req RuntimeRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return RuntimeRequest{}, http.StatusBadRequest, "invalid request body"
	}
	return req, http.StatusOK, ""
}

func mergeRuntimeSelector(req RuntimeRequest) LaunchSelector {
	var nested LaunchSelector
	if req.Selector != nil {
		nested = *req.Selector
	}

	return normalizeRuntimeSelector(buildMergedSelector(selectorMergeInput{
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

func validateRuntimeSelector(selector LaunchSelector) error {
	if err := selector.Validate(); err != nil {
		return err
	}
	if selector.MatchMode == launchMatchModeAll {
		return httpError("matchMode must be unique or first for runtime control")
	}
	return nil
}

func (s *LaunchServer) handleRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	s.handleRuntimeControl(w, r, "status")
}

func (s *LaunchServer) handleRuntimeActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": "method not allowed",
		})
		return
	}

	activePort, activeProfileID, activeProfileName := s.activeTarget()
	if activePort <= 0 || strings.TrimSpace(activeProfileID) == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok":             true,
			"active":         false,
			"profileId":      "",
			"profileName":    "",
			"launchCode":     "",
			"running":        false,
			"pid":            0,
			"debugPort":      0,
			"debugReady":     false,
			"runtimeWarning": "",
			"lastError":      "",
			"lastStartAt":    "",
			"lastStopAt":     "",
			"cdpPort":        0,
			"cdpUrl":         "",
			"directDebugUrl": "",
			"profile":        nil,
		})
		return
	}

	profile, status, errMsg := s.statusProfile(activeProfileID)
	if errMsg != "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"ok":                false,
			"error":             errMsg,
			"activeProfileId":   activeProfileID,
			"activeProfileName": activeProfileName,
			"activeDebugPort":   activePort,
			"statusCode":        status,
		})
		return
	}

	writeJSON(w, http.StatusOK, s.profileRuntimePayload(profile))
}

func (s *LaunchServer) handleRuntimeStop(w http.ResponseWriter, r *http.Request) {
	s.handleRuntimeControl(w, r, "stop")
}

func (s *LaunchServer) handleRuntimeControl(w http.ResponseWriter, r *http.Request, action string) {
	startAt := time.Now()
	clientIP := remoteIP(r.RemoteAddr)
	selector := LaunchSelector{}

	req, status, errMsg := decodeRuntimeRequest(r)
	if errMsg != "" {
		writeJSON(w, status, map[string]interface{}{
			"ok":    false,
			"error": errMsg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, "", selector, LaunchRequestParams{}, false, status, errMsg, "", "", startAt)
		return
	}

	selector = mergeRuntimeSelector(req)
	if selector.IsEmpty() {
		errMsg = "selector is required"
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok":    false,
			"error": errMsg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, "", selector, LaunchRequestParams{}, false, http.StatusBadRequest, errMsg, "", "", startAt)
		return
	}
	if err := validateRuntimeSelector(selector); err != nil {
		errMsg = err.Error()
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok":    false,
			"error": errMsg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, selector.Code, selector, LaunchRequestParams{}, false, http.StatusBadRequest, errMsg, "", "", startAt)
		return
	}

	var (
		profile    *browser.Profile
		launchCode string
	)
	switch action {
	case "status":
		profile, launchCode, status, errMsg = s.statusBySelector(selector)
	case "stop":
		profile, launchCode, status, errMsg = s.stopBySelector(selector)
	default:
		errMsg = "unsupported runtime action"
		status = http.StatusInternalServerError
	}
	if errMsg != "" {
		writeJSON(w, status, map[string]interface{}{
			"ok":    false,
			"error": errMsg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, launchCode, selector, LaunchRequestParams{}, false, status, errMsg, "", "", startAt)
		return
	}

	payload := s.profileRuntimePayload(profile)
	if strings.TrimSpace(launchCode) != "" {
		payload["launchCode"] = launchCode
		if nested, ok := payload["profile"].(*browser.Profile); ok && nested != nil && strings.TrimSpace(nested.LaunchCode) == "" {
			nested.LaunchCode = launchCode
		}
	}
	if action == "stop" {
		payload["stopped"] = true
	}
	writeJSON(w, http.StatusOK, payload)
	s.appendLaunchLog(r.Method, r.URL.Path, clientIP, launchCode, selector, LaunchRequestParams{}, true, http.StatusOK, "", profile.ProfileId, profile.ProfileName, startAt)
}

func (s *LaunchServer) statusBySelector(selector LaunchSelector) (*browser.Profile, string, int, string) {
	target, status, errMsg := s.resolveRuntimeTarget(selector)
	if errMsg != "" {
		return nil, target.LaunchCode, status, errMsg
	}

	profile, status, errMsg := s.statusProfile(target.ProfileID)
	if errMsg != "" {
		return nil, target.LaunchCode, status, errMsg
	}
	if profile != nil && target.LaunchCode != "" {
		profile.LaunchCode = target.LaunchCode
	}
	return profile, target.LaunchCode, http.StatusOK, ""
}

func (s *LaunchServer) stopBySelector(selector LaunchSelector) (*browser.Profile, string, int, string) {
	target, status, errMsg := s.resolveRuntimeTarget(selector)
	if errMsg != "" {
		return nil, target.LaunchCode, status, errMsg
	}

	profile, status, errMsg := s.stopProfile(target.ProfileID)
	if errMsg != "" {
		return nil, target.LaunchCode, status, errMsg
	}
	if profile != nil && target.LaunchCode != "" {
		profile.LaunchCode = target.LaunchCode
	}
	return profile, target.LaunchCode, http.StatusOK, ""
}

type runtimeTarget struct {
	ProfileID  string
	LaunchCode string
}

func (s *LaunchServer) resolveRuntimeTarget(selector LaunchSelector) (runtimeTarget, int, string) {
	selector = normalizeRuntimeSelector(selector)
	if selector.IsEmpty() {
		return runtimeTarget{}, http.StatusBadRequest, "selector is required"
	}
	if err := validateRuntimeSelector(selector); err != nil {
		return runtimeTarget{}, http.StatusBadRequest, err.Error()
	}
	selector = s.withCodeKeywordFallback(selector, true)

	if selector.OnlyCode() {
		if s.service == nil {
			return runtimeTarget{}, http.StatusServiceUnavailable, "launch code service is unavailable"
		}
		profileID, err := s.service.Resolve(selector.Code)
		if err != nil {
			return runtimeTarget{LaunchCode: selector.Code}, http.StatusNotFound, "launch code not found"
		}
		return runtimeTarget{ProfileID: profileID, LaunchCode: selector.Code}, http.StatusOK, ""
	}

	if selector.ProfileID != "" &&
		selector.Key == "" &&
		selector.ProfileName == "" &&
		selector.GroupID == "" &&
		len(selector.Keywords) == 0 &&
		len(selector.Tags) == 0 {
		return runtimeTarget{
			ProfileID:  selector.ProfileID,
			LaunchCode: s.resolveProfileLaunchCode(selector.ProfileID, ""),
		}, http.StatusOK, ""
	}

	profile, status, errMsg := s.findProfileBySelector(selector)
	if errMsg != "" {
		if selector.Code != "" {
			return runtimeTarget{LaunchCode: selector.Code}, status, errMsg
		}
		return runtimeTarget{}, status, errMsg
	}

	return runtimeTarget{
		ProfileID:  profile.ProfileId,
		LaunchCode: profile.LaunchCode,
	}, http.StatusOK, ""
}

type runtimeRequestError string

func (e runtimeRequestError) Error() string {
	return string(e)
}

func httpError(message string) error {
	return runtimeRequestError(message)
}
