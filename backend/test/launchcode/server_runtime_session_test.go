package launchcode_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/launchcode"
)

type sessionStarter struct {
	mgr        *browser.Manager
	waitReady  bool
	waitErr    error
	lastParams launchcode.LaunchRequestParams
}

func newSessionStarter(mgr *browser.Manager, waitReady bool) *sessionStarter {
	return &sessionStarter{
		mgr:       mgr,
		waitReady: waitReady,
	}
}

func (m *sessionStarter) StartInstance(profileID string) (*browser.Profile, error) {
	return m.StartInstanceWithParams(profileID, launchcode.LaunchRequestParams{})
}

func (m *sessionStarter) StartInstanceWithParams(profileID string, params launchcode.LaunchRequestParams) (*browser.Profile, error) {
	m.lastParams = params
	m.mgr.Mutex.Lock()
	defer m.mgr.Mutex.Unlock()

	profile, ok := m.mgr.Profiles[profileID]
	if !ok || profile == nil {
		return nil, fmt.Errorf("profile not found")
	}

	profile.Running = true
	profile.DebugReady = false
	profile.DebugPort = 9666
	profile.Pid = 4321
	profile.RuntimeWarning = "debug pending"
	profile.LastError = ""
	return profile, nil
}

func (m *sessionStarter) StatusInstance(profileID string) (*browser.Profile, error) {
	m.mgr.Mutex.Lock()
	defer m.mgr.Mutex.Unlock()

	profile, ok := m.mgr.Profiles[profileID]
	if !ok || profile == nil {
		return nil, fmt.Errorf("profile not found")
	}
	return profile, nil
}

func (m *sessionStarter) WaitInstanceDebugReady(profileID string, debugPort int, timeout time.Duration) (*browser.Profile, bool, error) {
	if m.waitErr != nil {
		return nil, false, m.waitErr
	}

	m.mgr.Mutex.Lock()
	defer m.mgr.Mutex.Unlock()

	profile, ok := m.mgr.Profiles[profileID]
	if !ok || profile == nil {
		return nil, false, fmt.Errorf("profile not found")
	}

	if m.waitReady {
		profile.Running = true
		profile.DebugReady = true
		profile.DebugPort = debugPort
		profile.RuntimeWarning = ""
		return profile, true, nil
	}

	return profile, false, nil
}

func TestRuntimeSessionWaitsUntilDebugReady(t *testing.T) {
	svc := newInMemoryService()
	profile := &browser.Profile{
		ProfileId:   "runtime-session-ready",
		ProfileName: "Runtime Session Ready",
	}
	manager := newSelectorTestManager(profile)
	starter := newSessionStarter(manager, true)

	if _, err := svc.SetCode(profile.ProfileId, "runtime-session-ready"); err != nil {
		t.Fatalf("SetCode 失败: %v", err)
	}

	handler := buildTestHandlerWithManager(svc, starter, manager)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/session", bytes.NewBufferString(`{
		"code":"runtime-session-ready",
		"timeoutMs":5000,
		"launchArgs":["--window-size=1400,900"],
		"startUrls":["https://example.com"],
		"skipDefaultStartUrls":true
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if len(starter.lastParams.LaunchArgs) != 1 || starter.lastParams.LaunchArgs[0] != "--window-size=1400,900" {
		t.Fatalf("launchArgs 透传错误: %+v", starter.lastParams)
	}
	if len(starter.lastParams.StartURLs) != 1 || starter.lastParams.StartURLs[0] != "https://example.com" {
		t.Fatalf("startUrls 透传错误: %+v", starter.lastParams)
	}
	if !starter.lastParams.SkipDefaultStartURLs {
		t.Fatalf("skipDefaultStartUrls 透传错误: %+v", starter.lastParams)
	}

	var resp struct {
		OK             bool   `json:"ok"`
		Ready          bool   `json:"ready"`
		WaitTimedOut   bool   `json:"waitTimedOut"`
		Retryable      bool   `json:"retryable"`
		Active         bool   `json:"active"`
		ProfileID      string `json:"profileId"`
		LaunchCode     string `json:"launchCode"`
		DebugReady     bool   `json:"debugReady"`
		CDPURL         string `json:"cdpUrl"`
		DirectDebugURL string `json:"directDebugUrl"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !resp.OK || !resp.Ready || resp.WaitTimedOut || resp.Retryable {
		t.Fatalf("ready 响应不正确: %+v", resp)
	}
	if !resp.Active || !resp.DebugReady || resp.ProfileID != profile.ProfileId {
		t.Fatalf("会话状态不正确: %+v", resp)
	}
	if resp.LaunchCode != "RUNTIME-SESSION-READY" {
		t.Fatalf("launchCode 不正确: %+v", resp)
	}
	if resp.CDPURL == "" || resp.DirectDebugURL == "" {
		t.Fatalf("应返回可接管地址: %+v", resp)
	}
}

func TestRuntimeSessionReturnsAcceptedWhileDebugIsPending(t *testing.T) {
	svc := newInMemoryService()
	profile := &browser.Profile{
		ProfileId:   "runtime-session-pending",
		ProfileName: "Runtime Session Pending",
	}
	manager := newSelectorTestManager(profile)
	starter := newSessionStarter(manager, false)

	if _, err := svc.SetCode(profile.ProfileId, "runtime-session-pending"); err != nil {
		t.Fatalf("SetCode 失败: %v", err)
	}

	handler := buildTestHandlerWithManager(svc, starter, manager)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/session", bytes.NewBufferString(`{
		"code":"runtime-session-pending",
		"timeoutMs":1000
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("期望 202，实际 %d，body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		OK             bool   `json:"ok"`
		Ready          bool   `json:"ready"`
		WaitTimedOut   bool   `json:"waitTimedOut"`
		Retryable      bool   `json:"retryable"`
		Active         bool   `json:"active"`
		DebugReady     bool   `json:"debugReady"`
		RuntimeWarning string `json:"runtimeWarning"`
		CDPURL         string `json:"cdpUrl"`
		DirectDebugURL string `json:"directDebugUrl"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !resp.OK || resp.Ready || !resp.WaitTimedOut || !resp.Retryable {
		t.Fatalf("pending 响应不正确: %+v", resp)
	}
	if resp.Active || resp.DebugReady {
		t.Fatalf("pending 会话不应标记为 active/debugReady: %+v", resp)
	}
	if resp.RuntimeWarning != "debug pending" {
		t.Fatalf("runtimeWarning 不正确: %+v", resp)
	}
	if resp.CDPURL != "" || resp.DirectDebugURL != "" {
		t.Fatalf("pending 会话不应返回可接管地址: %+v", resp)
	}
}

func TestRuntimeSessionRejectsMatchModeAll(t *testing.T) {
	manager := newSelectorTestManager()
	handler := buildTestHandlerWithManager(newInMemoryService(), newSessionStarter(manager, true), manager)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/session", bytes.NewBufferString(`{"keyword":"shop","matchMode":"all"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d，body=%s", w.Code, w.Body.String())
	}
}
