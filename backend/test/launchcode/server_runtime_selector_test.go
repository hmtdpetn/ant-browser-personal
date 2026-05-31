package launchcode_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ant-chrome/backend/internal/browser"
)

func TestRuntimeStatusWithCodeFallbackReturnsConflictByDefault(t *testing.T) {
	svc := newInMemoryService()
	profileA := &browser.Profile{
		ProfileId:   "runtime-status-a",
		ProfileName: "A Account",
		Keywords:    []string{"shop"},
		Running:     true,
		DebugReady:  true,
		DebugPort:   9411,
	}
	profileB := &browser.Profile{
		ProfileId:   "runtime-status-b",
		ProfileName: "B Account",
		Keywords:    []string{"shop"},
		Running:     true,
		DebugReady:  true,
		DebugPort:   9412,
	}
	manager := newSelectorTestManager(profileA, profileB)
	starter := newLifecycleStarter(manager)
	handler := buildTestHandlerWithManager(svc, starter, manager)

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/status", bytes.NewBufferString(`{"code":"shop"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("期望 409，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "matchMode=first") {
		t.Fatalf("错误信息未提示 matchMode=first: %s", w.Body.String())
	}
}

func TestRuntimeStatusWithMatchModeFirstReturnsStableTarget(t *testing.T) {
	svc := newInMemoryService()
	profileB := &browser.Profile{
		ProfileId:   "runtime-status-b",
		ProfileName: "B Account",
		Keywords:    []string{"shop"},
		Running:     true,
		DebugReady:  true,
		DebugPort:   9412,
		Pid:         3002,
	}
	profileA := &browser.Profile{
		ProfileId:   "runtime-status-a",
		ProfileName: "A Account",
		Keywords:    []string{"shop"},
		Running:     true,
		DebugReady:  true,
		DebugPort:   9411,
		Pid:         3001,
	}
	manager := newSelectorTestManager(profileB, profileA)
	starter := newLifecycleStarter(manager)
	handler := buildTestHandlerWithManager(svc, starter, manager)

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/status", bytes.NewBufferString(`{"code":"shop","matchMode":"first"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		OK          bool   `json:"ok"`
		ProfileID   string `json:"profileId"`
		ProfileName string `json:"profileName"`
		Running     bool   `json:"running"`
		DebugReady  bool   `json:"debugReady"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !resp.OK || resp.ProfileID != profileA.ProfileId || resp.ProfileName != profileA.ProfileName {
		t.Fatalf("响应不正确: %+v", resp)
	}
	if !resp.Running || !resp.DebugReady {
		t.Fatalf("运行态字段不正确: %+v", resp)
	}
}

func TestRuntimeStopWithExactLaunchCode(t *testing.T) {
	svc := newInMemoryService()
	profile := &browser.Profile{
		ProfileId:   "runtime-stop-code",
		ProfileName: "Runtime Stop By Code",
		Running:     true,
		DebugReady:  true,
		DebugPort:   9511,
		Pid:         4001,
	}
	manager := newSelectorTestManager(profile)
	starter := newLifecycleStarter(manager)

	if _, err := svc.SetCode(profile.ProfileId, "runtime-stop-code"); err != nil {
		t.Fatalf("SetCode 失败: %v", err)
	}

	handler := buildTestHandlerWithManager(svc, starter, manager)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/stop", bytes.NewBufferString(`{"code":"runtime-stop-code"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		OK         bool   `json:"ok"`
		Stopped    bool   `json:"stopped"`
		ProfileID  string `json:"profileId"`
		LaunchCode string `json:"launchCode"`
		Running    bool   `json:"running"`
		DebugReady bool   `json:"debugReady"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !resp.OK || !resp.Stopped || resp.ProfileID != profile.ProfileId {
		t.Fatalf("停止响应不正确: %+v", resp)
	}
	if resp.LaunchCode != "RUNTIME-STOP-CODE" {
		t.Fatalf("launchCode 不正确: %+v", resp)
	}
	if resp.Running || resp.DebugReady {
		t.Fatalf("停止后运行态不正确: %+v", resp)
	}
}

func TestRuntimeStatusRejectsMatchModeAll(t *testing.T) {
	svc := newInMemoryService()
	manager := newSelectorTestManager()
	starter := newLifecycleStarter(manager)
	handler := buildTestHandlerWithManager(svc, starter, manager)

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/status", bytes.NewBufferString(`{"keyword":"shop","matchMode":"all"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "matchMode must be unique or first") {
		t.Fatalf("错误信息不正确: %s", w.Body.String())
	}
}
