package launchcode_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ant-chrome/backend/internal/browser"
)

type lifecycleStarter struct {
	mgr     *browser.Manager
	started []string
	stopped []string
}

func newLifecycleStarter(mgr *browser.Manager) *lifecycleStarter {
	return &lifecycleStarter{mgr: mgr}
}

func (m *lifecycleStarter) StartInstance(profileID string) (*browser.Profile, error) {
	m.mgr.Mutex.Lock()
	defer m.mgr.Mutex.Unlock()

	profile, ok := m.mgr.Profiles[profileID]
	if !ok || profile == nil {
		return nil, fmt.Errorf("profile not found")
	}

	m.started = append(m.started, profileID)
	profile.Running = true
	profile.DebugReady = true
	profile.Pid = 7000 + len(m.started)
	profile.DebugPort = 9600 + len(m.started)
	profile.RuntimeWarning = ""
	profile.LastError = ""
	profile.LastStartAt = time.Now().Format(time.RFC3339)
	return profile, nil
}

func (m *lifecycleStarter) StatusInstance(profileID string) (*browser.Profile, error) {
	m.mgr.Mutex.Lock()
	defer m.mgr.Mutex.Unlock()

	profile, ok := m.mgr.Profiles[profileID]
	if !ok || profile == nil {
		return nil, fmt.Errorf("profile not found")
	}
	return profile, nil
}

func (m *lifecycleStarter) StopInstance(profileID string) (*browser.Profile, error) {
	m.mgr.Mutex.Lock()
	defer m.mgr.Mutex.Unlock()

	profile, ok := m.mgr.Profiles[profileID]
	if !ok || profile == nil {
		return nil, fmt.Errorf("profile not found")
	}

	m.stopped = append(m.stopped, profileID)
	profile.Running = false
	profile.DebugReady = false
	profile.Pid = 0
	profile.DebugPort = 0
	profile.RuntimeWarning = ""
	profile.LastStopAt = time.Now().Format(time.RFC3339)
	return profile, nil
}

func TestProfileStatusEndpointReturnsRuntimePayload(t *testing.T) {
	svc := newInMemoryService()
	profile := &browser.Profile{
		ProfileId:   "profile-runtime-status",
		ProfileName: "Runtime Status",
	}
	manager := newSelectorTestManager(profile)
	starter := newLifecycleStarter(manager)

	code, err := svc.SetCode(profile.ProfileId, "runtime_status")
	if err != nil {
		t.Fatalf("SetCode 失败: %v", err)
	}

	handler := buildTestHandlerWithManager(svc, starter, manager)

	reqLaunch := httptest.NewRequest(http.MethodGet, "/api/launch/"+code, nil)
	wLaunch := httptest.NewRecorder()
	handler.ServeHTTP(wLaunch, reqLaunch)
	if wLaunch.Code != http.StatusOK {
		t.Fatalf("启动实例失败: status=%d body=%s", wLaunch.Code, wLaunch.Body.String())
	}

	reqStatus := httptest.NewRequest(http.MethodGet, "/api/profiles/"+profile.ProfileId+"/status", nil)
	wStatus := httptest.NewRecorder()
	handler.ServeHTTP(wStatus, reqStatus)

	if wStatus.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", wStatus.Code, wStatus.Body.String())
	}

	var resp struct {
		OK             bool             `json:"ok"`
		ProfileID      string           `json:"profileId"`
		LaunchCode     string           `json:"launchCode"`
		Running        bool             `json:"running"`
		Active         bool             `json:"active"`
		DebugReady     bool             `json:"debugReady"`
		CDPURL         string           `json:"cdpUrl"`
		DirectDebugURL string           `json:"directDebugUrl"`
		Profile        *browser.Profile `json:"profile"`
	}
	if err := json.NewDecoder(wStatus.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !resp.OK || resp.ProfileID != profile.ProfileId {
		t.Fatalf("响应不正确: %+v", resp)
	}
	if resp.LaunchCode != "RUNTIME_STATUS" {
		t.Fatalf("launchCode 不正确: %+v", resp)
	}
	if !resp.Running || !resp.Active || !resp.DebugReady {
		t.Fatalf("运行态字段不正确: %+v", resp)
	}
	if resp.CDPURL == "" || resp.DirectDebugURL == "" {
		t.Fatalf("应返回可连接的 CDP 信息: %+v", resp)
	}
	if resp.Profile == nil || !resp.Profile.Running || !resp.Profile.DebugReady {
		t.Fatalf("嵌套 profile 运行态不正确: %+v", resp)
	}
}

func TestRuntimeActiveEndpointReportsCurrentTarget(t *testing.T) {
	svc := newInMemoryService()
	profile := &browser.Profile{
		ProfileId:   "profile-runtime-active",
		ProfileName: "Runtime Active",
	}
	manager := newSelectorTestManager(profile)
	starter := newLifecycleStarter(manager)

	code, err := svc.SetCode(profile.ProfileId, "runtime_active")
	if err != nil {
		t.Fatalf("SetCode 失败: %v", err)
	}

	handler := buildTestHandlerWithManager(svc, starter, manager)

	reqBefore := httptest.NewRequest(http.MethodGet, "/api/runtime/active", nil)
	wBefore := httptest.NewRecorder()
	handler.ServeHTTP(wBefore, reqBefore)
	if wBefore.Code != http.StatusOK {
		t.Fatalf("未激活前查询失败: status=%d body=%s", wBefore.Code, wBefore.Body.String())
	}

	var before struct {
		OK     bool `json:"ok"`
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(wBefore.Body).Decode(&before); err != nil {
		t.Fatalf("解析未激活响应失败: %v", err)
	}
	if !before.OK || before.Active {
		t.Fatalf("未激活响应不正确: %+v", before)
	}

	reqLaunch := httptest.NewRequest(http.MethodGet, "/api/launch/"+code, nil)
	wLaunch := httptest.NewRecorder()
	handler.ServeHTTP(wLaunch, reqLaunch)
	if wLaunch.Code != http.StatusOK {
		t.Fatalf("启动实例失败: status=%d body=%s", wLaunch.Code, wLaunch.Body.String())
	}

	reqAfter := httptest.NewRequest(http.MethodGet, "/api/runtime/active", nil)
	wAfter := httptest.NewRecorder()
	handler.ServeHTTP(wAfter, reqAfter)
	if wAfter.Code != http.StatusOK {
		t.Fatalf("激活后查询失败: status=%d body=%s", wAfter.Code, wAfter.Body.String())
	}

	var after struct {
		OK         bool   `json:"ok"`
		Active     bool   `json:"active"`
		ProfileID  string `json:"profileId"`
		LaunchCode string `json:"launchCode"`
		CDPURL     string `json:"cdpUrl"`
	}
	if err := json.NewDecoder(wAfter.Body).Decode(&after); err != nil {
		t.Fatalf("解析激活响应失败: %v", err)
	}
	if !after.OK || !after.Active || after.ProfileID != profile.ProfileId {
		t.Fatalf("激活响应不正确: %+v", after)
	}
	if after.LaunchCode != "RUNTIME_ACTIVE" || after.CDPURL == "" {
		t.Fatalf("激活响应缺少 launchCode/CDP 地址: %+v", after)
	}
}

func TestProfileStopEndpointStopsAndClearsActiveTarget(t *testing.T) {
	svc := newInMemoryService()
	profile := &browser.Profile{
		ProfileId:   "profile-runtime-stop",
		ProfileName: "Runtime Stop",
	}
	manager := newSelectorTestManager(profile)
	starter := newLifecycleStarter(manager)

	code, err := svc.SetCode(profile.ProfileId, "runtime_stop")
	if err != nil {
		t.Fatalf("SetCode 失败: %v", err)
	}

	handler := buildTestHandlerWithManager(svc, starter, manager)

	reqLaunch := httptest.NewRequest(http.MethodGet, "/api/launch/"+code, nil)
	wLaunch := httptest.NewRecorder()
	handler.ServeHTTP(wLaunch, reqLaunch)
	if wLaunch.Code != http.StatusOK {
		t.Fatalf("启动实例失败: status=%d body=%s", wLaunch.Code, wLaunch.Body.String())
	}

	reqStop := httptest.NewRequest(http.MethodPost, "/api/profiles/"+profile.ProfileId+"/stop", nil)
	wStop := httptest.NewRecorder()
	handler.ServeHTTP(wStop, reqStop)

	if wStop.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", wStop.Code, wStop.Body.String())
	}

	var resp struct {
		OK             bool             `json:"ok"`
		Stopped        bool             `json:"stopped"`
		Running        bool             `json:"running"`
		Active         bool             `json:"active"`
		CDPURL         string           `json:"cdpUrl"`
		DirectDebugURL string           `json:"directDebugUrl"`
		Profile        *browser.Profile `json:"profile"`
	}
	if err := json.NewDecoder(wStop.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !resp.OK || !resp.Stopped {
		t.Fatalf("停止响应不正确: %+v", resp)
	}
	if resp.Running || resp.Active {
		t.Fatalf("停止后运行态应关闭: %+v", resp)
	}
	if resp.CDPURL != "" || resp.DirectDebugURL != "" {
		t.Fatalf("停止后不应再暴露调试地址: %+v", resp)
	}
	if resp.Profile == nil || resp.Profile.Running || resp.Profile.DebugReady {
		t.Fatalf("嵌套 profile 停止态不正确: %+v", resp)
	}

	reqProxy := httptest.NewRequest(http.MethodGet, "/json/version", nil)
	wProxy := httptest.NewRecorder()
	handler.ServeHTTP(wProxy, reqProxy)
	if wProxy.Code != http.StatusServiceUnavailable {
		t.Fatalf("停止后应清空 active target: status=%d body=%s", wProxy.Code, wProxy.Body.String())
	}

	reqActive := httptest.NewRequest(http.MethodGet, "/api/runtime/active", nil)
	wActive := httptest.NewRecorder()
	handler.ServeHTTP(wActive, reqActive)
	if wActive.Code != http.StatusOK {
		t.Fatalf("停止后查询 active 失败: status=%d body=%s", wActive.Code, wActive.Body.String())
	}

	var activeResp struct {
		OK     bool `json:"ok"`
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(wActive.Body).Decode(&activeResp); err != nil {
		t.Fatalf("解析停止后 active 响应失败: %v", err)
	}
	if !activeResp.OK || activeResp.Active {
		t.Fatalf("停止后 active 响应不正确: %+v", activeResp)
	}
}

func TestProfileStopEndpointReturnsServiceUnavailableWhenRuntimeControlIsMissing(t *testing.T) {
	svc := newInMemoryService()
	profile := &browser.Profile{
		ProfileId:   "profile-runtime-unsupported",
		ProfileName: "Runtime Unsupported",
	}
	manager := newSelectorTestManager(profile)
	starter := newMockStarterWithParams()
	starter.addProfile(profile)

	handler := buildTestHandlerWithManager(svc, starter, manager)
	req := httptest.NewRequest(http.MethodPost, "/api/profiles/"+profile.ProfileId+"/stop", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("期望 503，实际 %d，body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["error"] != "profile runtime control is not available" {
		t.Fatalf("错误信息不正确: %+v", resp)
	}
}
