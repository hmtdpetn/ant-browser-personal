package launchcode

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleLaunch GET /api/launch/{code}
func (s *LaunchServer) handleLaunch(w http.ResponseWriter, r *http.Request) {
	startAt := time.Now()
	clientIP := remoteIP(r.RemoteAddr)
	selector := LaunchSelector{}
	if r.Method != http.MethodGet {
		msg := "method not allowed"
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": msg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, "", selector, LaunchRequestParams{}, false, http.StatusMethodNotAllowed, msg, "", "", startAt)
		return
	}

	code := strings.TrimPrefix(r.URL.Path, "/api/launch/")
	if strings.TrimSpace(code) == "" {
		msg := "launch code not found"
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"ok":    false,
			"error": msg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, "", selector, LaunchRequestParams{}, false, http.StatusNotFound, msg, "", "", startAt)
		return
	}

	selector = normalizeLaunchSelector(LaunchSelector{Code: code})
	profile, launchCode, status, errMsg := s.launchByCode(code, LaunchRequestParams{})
	if errMsg != "" {
		writeJSON(w, status, map[string]interface{}{
			"ok":    false,
			"error": errMsg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, selector.Code, selector, LaunchRequestParams{}, false, status, errMsg, "", "", startAt)
		return
	}

	s.SetActiveProfile(profile)
	writeJSON(w, http.StatusOK, s.launchSuccessPayload(profile, launchCode))
	s.appendLaunchLog(r.Method, r.URL.Path, clientIP, launchCode, selector, LaunchRequestParams{}, true, http.StatusOK, "", profile.ProfileId, profile.ProfileName, startAt)
}

// handleLaunchWithBody POST /api/launch
func (s *LaunchServer) handleLaunchWithBody(w http.ResponseWriter, r *http.Request) {
	startAt := time.Now()
	clientIP := remoteIP(r.RemoteAddr)
	selector := LaunchSelector{}
	if r.Method != http.MethodPost {
		msg := "method not allowed"
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": msg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, "", selector, LaunchRequestParams{}, false, http.StatusMethodNotAllowed, msg, "", "", startAt)
		return
	}

	var req LaunchRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		msg := "invalid request body"
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok":    false,
			"error": msg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, "", selector, LaunchRequestParams{}, false, http.StatusBadRequest, msg, "", "", startAt)
		return
	}

	selector = mergeLaunchSelector(req)
	if selector.IsEmpty() {
		msg := "selector is required"
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok":    false,
			"error": msg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, "", selector, req.LaunchRequestParams, false, http.StatusBadRequest, msg, "", "", startAt)
		return
	}

	req.LaunchRequestParams = normalizeLaunchRequestParams(req.LaunchRequestParams)
	if selector.MatchMode == launchMatchModeAll {
		profiles, status, errMsg := s.launchAllBySelector(selector, req.LaunchRequestParams)
		if errMsg != "" {
			writeJSON(w, status, map[string]interface{}{
				"ok":    false,
				"error": errMsg,
			})
			s.appendLaunchLog(r.Method, r.URL.Path, clientIP, selector.Code, selector, req.LaunchRequestParams, false, status, errMsg, "", "", startAt)
			return
		}

		activeProfile, profileIDs, profileNames := summarizeLaunchedProfiles(profiles)
		if activeProfile != nil {
			s.SetActiveProfile(activeProfile)
		}
		writeJSON(w, http.StatusOK, s.launchBatchSuccessPayload(profiles))
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, selector.Code, selector, req.LaunchRequestParams, true, http.StatusOK, "", profileIDs, profileNames, startAt)
		return
	}

	profile, launchCode, status, errMsg := s.launchBySelector(selector, req.LaunchRequestParams)
	if errMsg != "" {
		writeJSON(w, status, map[string]interface{}{
			"ok":    false,
			"error": errMsg,
		})
		s.appendLaunchLog(r.Method, r.URL.Path, clientIP, launchCode, selector, req.LaunchRequestParams, false, status, errMsg, "", "", startAt)
		return
	}

	s.SetActiveProfile(profile)
	writeJSON(w, http.StatusOK, s.launchSuccessPayload(profile, launchCode))
	s.appendLaunchLog(r.Method, r.URL.Path, clientIP, launchCode, selector, req.LaunchRequestParams, true, http.StatusOK, "", profile.ProfileId, profile.ProfileName, startAt)
}

// handleLaunchLogs GET /api/launch/logs?limit=50
func (s *LaunchServer) handleLaunchLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": "method not allowed",
		})
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			if n < 1 {
				n = 1
			}
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}

	items := s.listLaunchLogs(limit)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":    true,
		"items": items,
	})
}
