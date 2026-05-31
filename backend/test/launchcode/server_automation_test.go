package launchcode_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"ant-chrome/backend/internal/automation"
)

type mockAutomationStarter struct {
	*mockStarterWithParams
	scripts          []automation.ScriptRecord
	runs             []automation.ScriptRunRecord
	lastGetID        string
	lastRunRequest   automation.ScriptRunRequest
	lastRunListLimit int
	runResult        *automation.ScriptRunRecord
	getErr           error
	listErr          error
	runErr           error
	runListErr       error
}

func newMockAutomationStarter() *mockAutomationStarter {
	return &mockAutomationStarter{
		mockStarterWithParams: newMockStarterWithParams(),
	}
}

func (m *mockAutomationStarter) AutomationScriptList() ([]automation.ScriptRecord, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return append([]automation.ScriptRecord(nil), m.scripts...), nil
}

func (m *mockAutomationStarter) AutomationScriptGet(scriptID string) (*automation.ScriptRecord, error) {
	m.lastGetID = scriptID
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, item := range m.scripts {
		if item.ID == scriptID {
			record := item
			return &record, nil
		}
	}
	return nil, os.ErrNotExist
}

func (m *mockAutomationStarter) AutomationScriptRunWithOptions(input automation.ScriptRunRequest) (*automation.ScriptRunRecord, error) {
	m.lastRunRequest = input
	if m.runErr != nil {
		return nil, m.runErr
	}
	if m.runResult == nil {
		return &automation.ScriptRunRecord{
			ID:       "run-default",
			ScriptID: input.ScriptID,
			Status:   "success",
		}, nil
	}

	record := *m.runResult
	return &record, nil
}

func (m *mockAutomationStarter) AutomationScriptRunList(limit int) ([]automation.ScriptRunRecord, error) {
	m.lastRunListLimit = limit
	if m.runListErr != nil {
		return nil, m.runListErr
	}

	items := append([]automation.ScriptRunRecord(nil), m.runs...)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func TestAutomationScriptsEndpointReturnsMetadata(t *testing.T) {
	svc := newInMemoryService()
	starter := newMockAutomationStarter()
	starter.scripts = []automation.ScriptRecord{
		{
			ID:           "news-query-txt",
			Name:         "查询新闻并写 TXT",
			Description:  "测试脚本",
			Type:         "playwright-cdp",
			Status:       "ready",
			EntryFile:    "index.cjs",
			Tags:         []string{"Playwright", "新闻"},
			SelectorText: `{"code":"BUYER_001"}`,
			ParamsText:   `{"keyword":"OpenAI","limit":10}`,
			ScriptText:   `module.exports.run = async () => ({ ok: true })`,
			Notes:        "note",
			CreatedAt:    "2026-04-08T10:00:00Z",
			UpdatedAt:    "2026-04-08T11:00:00Z",
		},
	}

	handler := buildTestHandlerWithManager(svc, starter, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/automation/scripts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "scriptText") {
		t.Fatalf("公共脚本列表不应返回脚本文本: %s", w.Body.String())
	}

	var resp struct {
		OK    bool `json:"ok"`
		Count int  `json:"count"`
		Items []struct {
			ID       string                 `json:"id"`
			Type     string                 `json:"type"`
			Status   string                 `json:"status"`
			Selector map[string]interface{} `json:"selector"`
			Params   map[string]interface{} `json:"params"`
		} `json:"items"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !resp.OK || resp.Count != 1 || len(resp.Items) != 1 {
		t.Fatalf("响应结构错误: %+v", resp)
	}
	if resp.Items[0].ID != "news-query-txt" || resp.Items[0].Type != "playwright-cdp" || resp.Items[0].Status != "ready" {
		t.Fatalf("脚本元数据错误: %+v", resp.Items[0])
	}
	if resp.Items[0].Selector["code"] != "BUYER_001" {
		t.Fatalf("selector 解析错误: %+v", resp.Items[0].Selector)
	}
	if resp.Items[0].Params["keyword"] != "OpenAI" {
		t.Fatalf("params 解析错误: %+v", resp.Items[0].Params)
	}
}

func TestAutomationScriptDetailEndpointReturnsSingleScript(t *testing.T) {
	svc := newInMemoryService()
	starter := newMockAutomationStarter()
	starter.scripts = []automation.ScriptRecord{
		{
			PackageFormat:   "ant-automation-script",
			ManifestVersion: 1,
			ID:              "news-query-txt",
			Name:            "查询新闻并写 TXT",
			Description:     "测试脚本",
			Type:            "playwright-cdp",
			Status:          "ready",
			EntryFile:       "index.cjs",
			Tags:            []string{"Playwright", "新闻"},
			SelectorText:    `{"code":"BUYER_001"}`,
			ParamsText:      `{"keyword":"OpenAI","limit":10}`,
			ScriptText:      `module.exports.run = async () => ({ ok: true })`,
			Notes:           "note",
			Source: automation.ScriptSource{
				Type: "git",
				URI:  "https://example.com/repo.git",
				Ref:  "main",
			},
			CreatedAt: "2026-04-08T10:00:00Z",
			UpdatedAt: "2026-04-08T11:00:00Z",
		},
	}

	handler := buildTestHandlerWithManager(svc, starter, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/automation/scripts/news-query-txt", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if starter.lastGetID != "news-query-txt" {
		t.Fatalf("scriptId 路径解析错误: %s", starter.lastGetID)
	}
	if strings.Contains(w.Body.String(), "scriptText") {
		t.Fatalf("公共脚本详情不应返回脚本文本: %s", w.Body.String())
	}

	var resp struct {
		OK   bool `json:"ok"`
		Item struct {
			ID              string                  `json:"id"`
			PackageFormat   string                  `json:"packageFormat"`
			ManifestVersion int                     `json:"manifestVersion"`
			Source          automation.ScriptSource `json:"source"`
			Selector        map[string]interface{}  `json:"selector"`
		} `json:"item"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !resp.OK || resp.Item.ID != "news-query-txt" {
		t.Fatalf("详情响应错误: %+v", resp)
	}
	if resp.Item.PackageFormat != "ant-automation-script" || resp.Item.ManifestVersion != 1 {
		t.Fatalf("详情元数据错误: %+v", resp.Item)
	}
	if resp.Item.Source.Type != "git" || resp.Item.Source.URI != "https://example.com/repo.git" {
		t.Fatalf("source 返回错误: %+v", resp.Item.Source)
	}
	if resp.Item.Selector["code"] != "BUYER_001" {
		t.Fatalf("selector 解析错误: %+v", resp.Item.Selector)
	}
}

func TestAutomationScriptDetailEndpointReturnsNotFound(t *testing.T) {
	handler := buildTestHandlerWithManager(newInMemoryService(), newMockAutomationStarter(), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/automation/scripts/missing-script", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "script not found") {
		t.Fatalf("错误信息不正确: %s", w.Body.String())
	}
}

func TestAutomationScriptRunEndpointConvertsObjectPayload(t *testing.T) {
	svc := newInMemoryService()
	starter := newMockAutomationStarter()
	starter.runResult = &automation.ScriptRunRecord{
		ID:       "run-1",
		ScriptID: "news-query-txt",
		Status:   "success",
	}

	handler := buildTestHandlerWithManager(svc, starter, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/automation/scripts/run", bytes.NewBufferString(`{
		"scriptId":"news-query-txt",
		"selector":{"code":"BUYER_001"},
		"params":{"keyword":"OpenAI"}
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if starter.lastRunRequest.ScriptID != "news-query-txt" {
		t.Fatalf("scriptId 传递错误: %+v", starter.lastRunRequest)
	}
	if starter.lastRunRequest.UseScriptSelector || starter.lastRunRequest.UseScriptParams {
		t.Fatalf("对象参数应关闭脚本默认 selector/params: %+v", starter.lastRunRequest)
	}
	if starter.lastRunRequest.SelectorText != `{"code":"BUYER_001"}` {
		t.Fatalf("selectorText 转换错误: %s", starter.lastRunRequest.SelectorText)
	}
	if starter.lastRunRequest.ParamsText != `{"keyword":"OpenAI"}` {
		t.Fatalf("paramsText 转换错误: %s", starter.lastRunRequest.ParamsText)
	}

	var resp struct {
		OK  bool `json:"ok"`
		Run struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"run"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !resp.OK || resp.Run.ID != "run-1" || resp.Run.Status != "success" {
		t.Fatalf("run 响应错误: %+v", resp)
	}
}

func TestAutomationScriptRunEndpointUsesScriptDefaultsWhenFieldsOmitted(t *testing.T) {
	svc := newInMemoryService()
	starter := newMockAutomationStarter()

	handler := buildTestHandlerWithManager(svc, starter, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/automation/scripts/run", bytes.NewBufferString(`{"scriptId":"news-query-txt"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if !starter.lastRunRequest.UseScriptSelector || !starter.lastRunRequest.UseScriptParams {
		t.Fatalf("缺省时应回退到脚本默认 selector/params: %+v", starter.lastRunRequest)
	}
	if starter.lastRunRequest.SelectorText != "" || starter.lastRunRequest.ParamsText != "" {
		t.Fatalf("缺省时不应透传 selectorText/paramsText: %+v", starter.lastRunRequest)
	}
}

func TestAutomationScriptRunsEndpointPassesLimit(t *testing.T) {
	svc := newInMemoryService()
	starter := newMockAutomationStarter()
	starter.runs = []automation.ScriptRunRecord{
		{ID: "run-1", ScriptID: "script-a", Status: "success"},
		{ID: "run-2", ScriptID: "script-b", Status: "failed"},
	}

	handler := buildTestHandlerWithManager(svc, starter, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/automation/scripts/runs?limit=1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if starter.lastRunListLimit != 1 {
		t.Fatalf("limit 透传错误: %d", starter.lastRunListLimit)
	}

	var resp struct {
		OK    bool                         `json:"ok"`
		Count int                          `json:"count"`
		Items []automation.ScriptRunRecord `json:"items"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !resp.OK || resp.Count != 1 || len(resp.Items) != 1 {
		t.Fatalf("runs 响应错误: %+v", resp)
	}
}

func TestAutomationScriptAPIUnavailableReturnsServiceUnavailable(t *testing.T) {
	handler := buildTestHandlerWithManager(newInMemoryService(), newMockStarterWithParams(), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/automation/scripts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("期望 503，实际 %d，body=%s", w.Code, w.Body.String())
	}
}

func TestAutomationScriptRunEndpointRejectsInvalidBody(t *testing.T) {
	svc := newInMemoryService()
	starter := newMockAutomationStarter()
	handler := buildTestHandlerWithManager(svc, starter, nil)

	t.Run("invalid-json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/automation/scripts/run", bytes.NewBufferString("{bad json}"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d，body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("selector-must-be-object", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/automation/scripts/run", bytes.NewBufferString(`{
			"scriptId":"news-query-txt",
			"selector":"BUYER_001"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d，body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "selector must be a JSON object") {
			t.Fatalf("错误信息不正确: %s", w.Body.String())
		}
	})
}
