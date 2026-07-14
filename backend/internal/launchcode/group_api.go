package launchcode

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"ant-chrome/backend/internal/browser"
)

const (
	instanceGroupRoutePrefix = "/api/groups/"
	proxyGroupRoutePrefix    = "/api/proxy-groups/"
)

type profileGroupMoveRequest struct {
	ProfileIDs []string `json:"profileIds"`
	GroupID    string   `json:"groupId"`
}

type proxyGroupMoveRequest struct {
	ProxyIDs []string `json:"proxyIds"`
	GroupID  string   `json:"groupId"`
}

type profileGroupMover interface {
	MoveToGroup(profileIDs []string, groupID string) error
}

func (s *LaunchServer) handleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListGroups(w)
	case http.MethodPost:
		s.handleCreateGroup(w, r)
	default:
		writeGroupAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *LaunchServer) handleGroupByID(w http.ResponseWriter, r *http.Request) {
	groupID, ok := parseGroupRouteID(r.URL.Path, instanceGroupRoutePrefix)
	if !ok {
		writeGroupAPIError(w, http.StatusNotFound, "group not found")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handleUpdateGroup(w, r, groupID)
	case http.MethodDelete:
		s.handleDeleteGroup(w, groupID)
	default:
		writeGroupAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *LaunchServer) handleListGroups(w http.ResponseWriter) {
	if !s.instanceGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "instance group api is unavailable")
		return
	}

	groups, err := s.browserMgr.GroupDAO.List()
	if err != nil {
		writeGroupAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}
	profiles, err := s.browserMgr.ProfileDAO.List()
	if err != nil {
		writeGroupAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	counts := make(map[string]int, len(groups))
	for _, profile := range profiles {
		if profile != nil && strings.TrimSpace(profile.GroupId) != "" {
			counts[profile.GroupId]++
		}
	}
	items := make([]browser.GroupWithCount, 0, len(groups))
	for _, group := range groups {
		if group != nil {
			items = append(items, browser.GroupWithCount{Group: *group, InstanceCount: counts[group.GroupId]})
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "items": items, "count": len(items)})
}

func (s *LaunchServer) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	if !s.instanceGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "instance group api is unavailable")
		return
	}
	input, err := decodeGroupInput(r)
	if err != nil {
		writeGroupAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	group, err := s.browserMgr.GroupDAO.Create(input)
	if err != nil {
		writeGroupDAOError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "group": group})
}

func (s *LaunchServer) handleUpdateGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	if !s.instanceGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "instance group api is unavailable")
		return
	}
	input, err := decodeGroupInput(r)
	if err != nil {
		writeGroupAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	group, err := s.browserMgr.GroupDAO.Update(groupID, input)
	if err != nil {
		writeGroupDAOError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "group": group})
}

func (s *LaunchServer) handleDeleteGroup(w http.ResponseWriter, groupID string) {
	if !s.instanceGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "instance group api is unavailable")
		return
	}
	group, err := s.browserMgr.GroupDAO.GetById(groupID)
	if err != nil {
		writeGroupDAOError(w, err)
		return
	}
	if err := s.browserMgr.GroupDAO.Delete(groupID); err != nil {
		writeGroupDAOError(w, err)
		return
	}
	s.replaceInMemoryProfileGroup(groupID, group.ParentId)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "deleted": true, "groupId": groupID})
}

func (s *LaunchServer) handleMoveProfilesToGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGroupAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.instanceGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "instance group api is unavailable")
		return
	}
	var req profileGroupMoveRequest
	if !decodeGroupRequestBody(r, &req) {
		writeGroupAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	profileIDs := normalizeGroupItemIDs(req.ProfileIDs)
	if len(profileIDs) == 0 {
		writeGroupAPIError(w, http.StatusBadRequest, "profileIds is required")
		return
	}
	groupID := strings.TrimSpace(req.GroupID)
	mover, ok := s.browserMgr.ProfileDAO.(profileGroupMover)
	if !ok {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "profile group move is unavailable")
		return
	}
	if err := mover.MoveToGroup(profileIDs, groupID); err != nil {
		writeGroupDAOError(w, err)
		return
	}
	s.updateInMemoryProfileGroups(profileIDs, groupID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "profileIds": profileIDs, "groupId": groupID})
}

func (s *LaunchServer) handleProxyGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListProxyGroups(w)
	case http.MethodPost:
		s.handleCreateProxyGroup(w, r)
	default:
		writeGroupAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *LaunchServer) handleProxyGroupByID(w http.ResponseWriter, r *http.Request) {
	groupID, ok := parseGroupRouteID(r.URL.Path, proxyGroupRoutePrefix)
	if !ok {
		writeGroupAPIError(w, http.StatusNotFound, "proxy group not found")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handleUpdateProxyGroup(w, r, groupID)
	case http.MethodDelete:
		s.handleDeleteProxyGroup(w, groupID)
	default:
		writeGroupAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *LaunchServer) handleListProxyGroups(w http.ResponseWriter) {
	if !s.proxyGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "proxy group api is unavailable")
		return
	}
	groups, err := s.browserMgr.ProxyGroupDAO.List()
	if err != nil {
		writeGroupAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}
	proxies, err := s.browserMgr.ProxyDAO.List()
	if err != nil {
		writeGroupAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	counts := make(map[string]int, len(groups))
	for _, proxy := range proxies {
		if strings.TrimSpace(proxy.GroupId) != "" {
			counts[proxy.GroupId]++
		}
	}
	items := make([]browser.ProxyGroupWithCount, 0, len(groups))
	for _, group := range groups {
		if group != nil {
			items = append(items, browser.ProxyGroupWithCount{ProxyGroup: *group, ProxyCount: counts[group.GroupId]})
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "items": items, "count": len(items)})
}

func (s *LaunchServer) handleCreateProxyGroup(w http.ResponseWriter, r *http.Request) {
	if !s.proxyGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "proxy group api is unavailable")
		return
	}
	input, err := decodeProxyGroupInput(r)
	if err != nil {
		writeGroupAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	group, err := s.browserMgr.ProxyGroupDAO.Create(input)
	if err != nil {
		writeGroupDAOError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "group": group})
}

func (s *LaunchServer) handleUpdateProxyGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	if !s.proxyGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "proxy group api is unavailable")
		return
	}
	input, err := decodeProxyGroupInput(r)
	if err != nil {
		writeGroupAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	group, err := s.browserMgr.ProxyGroupDAO.Update(groupID, input)
	if err != nil {
		writeGroupDAOError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "group": group})
}

func (s *LaunchServer) handleDeleteProxyGroup(w http.ResponseWriter, groupID string) {
	if !s.proxyGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "proxy group api is unavailable")
		return
	}
	if err := s.browserMgr.ProxyGroupDAO.Delete(groupID); err != nil {
		writeGroupDAOError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "deleted": true, "groupId": groupID})
}

func (s *LaunchServer) handleMoveProxiesToGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGroupAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.proxyGroupAPIReady() {
		writeGroupAPIError(w, http.StatusServiceUnavailable, "proxy group api is unavailable")
		return
	}
	var req proxyGroupMoveRequest
	if !decodeGroupRequestBody(r, &req) {
		writeGroupAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	proxyIDs := normalizeGroupItemIDs(req.ProxyIDs)
	if len(proxyIDs) == 0 {
		writeGroupAPIError(w, http.StatusBadRequest, "proxyIds is required")
		return
	}
	groupID := strings.TrimSpace(req.GroupID)
	if err := s.browserMgr.ProxyGroupDAO.MoveProxies(proxyIDs, groupID); err != nil {
		writeGroupDAOError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "proxyIds": proxyIDs, "groupId": groupID})
}

func (s *LaunchServer) instanceGroupAPIReady() bool {
	return s.browserMgr != nil && s.browserMgr.GroupDAO != nil && s.browserMgr.ProfileDAO != nil
}

func (s *LaunchServer) proxyGroupAPIReady() bool {
	return s.browserMgr != nil && s.browserMgr.ProxyGroupDAO != nil && s.browserMgr.ProxyDAO != nil
}

func (s *LaunchServer) updateInMemoryProfileGroups(profileIDs []string, groupID string) {
	if s.browserMgr == nil {
		return
	}
	now := time.Now().Format(time.RFC3339)
	s.browserMgr.Mutex.Lock()
	for _, profileID := range profileIDs {
		if profile := s.browserMgr.Profiles[profileID]; profile != nil {
			profile.GroupId = groupID
			profile.UpdatedAt = now
		}
	}
	s.browserMgr.Mutex.Unlock()
}

func (s *LaunchServer) replaceInMemoryProfileGroup(fromGroupID string, toGroupID string) {
	if s.browserMgr == nil {
		return
	}
	now := time.Now().Format(time.RFC3339)
	s.browserMgr.Mutex.Lock()
	for _, profile := range s.browserMgr.Profiles {
		if profile != nil && profile.GroupId == fromGroupID {
			profile.GroupId = toGroupID
			profile.UpdatedAt = now
		}
	}
	s.browserMgr.Mutex.Unlock()
}

func decodeGroupInput(r *http.Request) (browser.GroupInput, error) {
	var input browser.GroupInput
	if !decodeGroupRequestBody(r, &input) {
		return browser.GroupInput{}, io.ErrUnexpectedEOF
	}
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.ParentId = strings.TrimSpace(input.ParentId)
	return input, nil
}

func decodeProxyGroupInput(r *http.Request) (browser.ProxyGroupInput, error) {
	var input browser.ProxyGroupInput
	if !decodeGroupRequestBody(r, &input) {
		return browser.ProxyGroupInput{}, io.ErrUnexpectedEOF
	}
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.ParentId = strings.TrimSpace(input.ParentId)
	return input, nil
}

func decodeGroupRequestBody(r *http.Request, target interface{}) bool {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(target) == nil
}

func parseGroupRouteID(path string, prefix string) (string, bool) {
	groupID := strings.TrimSpace(strings.Trim(strings.TrimPrefix(path, prefix), "/"))
	if groupID == "" || strings.Contains(groupID, "/") {
		return "", false
	}
	return groupID, true
}

func normalizeGroupItemIDs(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func writeGroupDAOError(w http.ResponseWriter, err error) {
	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "not found") || strings.Contains(message, "不存在"):
		writeGroupAPIError(w, http.StatusNotFound, message)
	case strings.Contains(lower, "already exists") || strings.Contains(lower, "duplicate") || strings.Contains(message, "重复"):
		writeGroupAPIError(w, http.StatusConflict, message)
	default:
		writeGroupAPIError(w, http.StatusBadRequest, message)
	}
}

func writeGroupAPIError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{"ok": false, "error": message})
}
