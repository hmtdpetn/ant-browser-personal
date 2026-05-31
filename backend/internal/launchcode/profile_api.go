package launchcode

import (
	"net/http"

	"ant-chrome/backend/internal/browser"
)

// ProfileWriteRequest 用于创建/更新实例配置。
// profile 为持久化配置；start 为本次自动启动的临时参数。
type ProfileWriteRequest struct {
	Profile    *browser.ProfileInput `json:"profile"`
	LaunchCode string                `json:"launchCode"`
	AutoLaunch bool                  `json:"autoLaunch"`
	Start      *LaunchRequestParams  `json:"start"`
}

type profileCreator interface {
	CreateProfile(input browser.ProfileInput) (*browser.Profile, error)
}

type profileUpdater interface {
	UpdateProfile(profileID string, input browser.ProfileInput) (*browser.Profile, error)
}

type profileDeleter interface {
	DeleteProfile(profileID string) error
}

func (s *LaunchServer) handleProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListProfiles(w, r)
	case http.MethodPost:
		s.handleCreateProfile(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": "method not allowed",
		})
	}
}

func (s *LaunchServer) handleProfileByID(w http.ResponseWriter, r *http.Request) {
	profileID, action, ok := parseProfilePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"ok":    false,
			"error": "profile not found",
		})
		return
	}

	switch action {
	case "status":
		s.handleProfileStatus(w, r, profileID)
		return
	case "stop":
		s.handleStopProfile(w, r, profileID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetProfile(w, r, profileID)
	case http.MethodPut:
		s.handleUpdateProfile(w, r, profileID)
	case http.MethodDelete:
		s.handleDeleteProfile(w, r, profileID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": "method not allowed",
		})
	}
}
