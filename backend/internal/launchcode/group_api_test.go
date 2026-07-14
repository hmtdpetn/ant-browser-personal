package launchcode

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/database"
)

const groupAPITestKey = "group-api-test-key"

func newGroupAPITestHandler(t *testing.T) (http.Handler, *browser.Manager, *database.DB) {
	t.Helper()
	db, err := database.NewDB(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	mgr := browser.NewManager(&config.Config{}, t.TempDir())
	mgr.ProfileDAO = browser.NewSQLiteProfileDAO(db.GetConn())
	mgr.GroupDAO = browser.NewSQLiteGroupDAO(db.GetConn())
	mgr.ProxyDAO = browser.NewSQLiteProxyDAO(db.GetConn())
	mgr.ProxyGroupDAO = browser.NewSQLiteProxyGroupDAO(db.GetConn())

	server := NewLaunchServer(nil, nil, mgr, 0)
	server.SetAPIAuthConfig(APIAuthConfig{Enabled: true, APIKey: groupAPITestKey})
	return NewTestHandler(server), mgr, db
}

func groupAPIRequest(t *testing.T, handler http.Handler, method string, path string, body string, authorized bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authorized {
		req.Header.Set(DefaultAPIKeyHeader, groupAPITestKey)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func decodeGroupAPIJSON(t *testing.T, body *bytes.Buffer, target interface{}) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestInstanceGroupHTTPAPI(t *testing.T) {
	handler, mgr, db := newGroupAPITestHandler(t)
	if res := groupAPIRequest(t, handler, http.MethodGet, "/api/groups", "", false); res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list status = %d, want %d", res.Code, http.StatusUnauthorized)
	}

	if _, err := db.GetConn().Exec(`
		INSERT INTO browser_profiles (profile_id, profile_name, user_data_dir, created_at, updated_at)
		VALUES ('profile-1', 'Profile 1', '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert profile: %v", err)
	}
	mgr.Profiles["profile-1"] = &browser.Profile{ProfileId: "profile-1", ProfileName: "Profile 1"}

	create := groupAPIRequest(t, handler, http.MethodPost, "/api/groups", `{"groupName":"Instance Root","sortOrder":1}`, true)
	if create.Code != http.StatusCreated {
		t.Fatalf("create instance group status = %d: %s", create.Code, create.Body.String())
	}
	var created struct {
		OK    bool          `json:"ok"`
		Group browser.Group `json:"group"`
	}
	decodeGroupAPIJSON(t, create.Body, &created)
	if !created.OK || created.Group.GroupId == "" {
		t.Fatalf("unexpected create payload: %+v", created)
	}

	update := groupAPIRequest(t, handler, http.MethodPut, "/api/groups/"+created.Group.GroupId, `{"groupName":"Instance Root Renamed","sortOrder":2}`, true)
	if update.Code != http.StatusOK {
		t.Fatalf("update instance group status = %d: %s", update.Code, update.Body.String())
	}

	move := groupAPIRequest(t, handler, http.MethodPost, "/api/groups/move-profiles", `{"profileIds":["profile-1","profile-1"],"groupId":"`+created.Group.GroupId+`"}`, true)
	if move.Code != http.StatusOK {
		t.Fatalf("move profile status = %d: %s", move.Code, move.Body.String())
	}
	if got := mgr.Profiles["profile-1"].GroupId; got != created.Group.GroupId {
		t.Fatalf("in-memory profile group = %q, want %q", got, created.Group.GroupId)
	}

	list := groupAPIRequest(t, handler, http.MethodGet, "/api/groups", "", true)
	if list.Code != http.StatusOK {
		t.Fatalf("list instance groups status = %d: %s", list.Code, list.Body.String())
	}
	var listed struct {
		OK    bool                     `json:"ok"`
		Items []browser.GroupWithCount `json:"items"`
	}
	decodeGroupAPIJSON(t, list.Body, &listed)
	if !listed.OK || len(listed.Items) != 1 || listed.Items[0].InstanceCount != 1 {
		t.Fatalf("unexpected group list payload: %+v", listed)
	}

	deleteRes := groupAPIRequest(t, handler, http.MethodDelete, "/api/groups/"+created.Group.GroupId, "", true)
	if deleteRes.Code != http.StatusOK {
		t.Fatalf("delete instance group status = %d: %s", deleteRes.Code, deleteRes.Body.String())
	}
	if got := mgr.Profiles["profile-1"].GroupId; got != "" {
		t.Fatalf("in-memory profile group after delete = %q, want empty", got)
	}
	var persistedGroupID string
	if err := db.GetConn().QueryRow(`SELECT group_id FROM browser_profiles WHERE profile_id = 'profile-1'`).Scan(&persistedGroupID); err != nil {
		t.Fatalf("read persisted profile group: %v", err)
	}
	if persistedGroupID != "" {
		t.Fatalf("persisted profile group after delete = %q, want empty", persistedGroupID)
	}
}

func TestProxyGroupHTTPAPI(t *testing.T) {
	handler, _, db := newGroupAPITestHandler(t)
	proxyDAO := browser.NewSQLiteProxyDAO(db.GetConn())
	if err := proxyDAO.Upsert(browser.Proxy{ProxyId: "proxy-1", ProxyName: "Proxy 1", ProxyConfig: "socks5://127.0.0.1:1080"}); err != nil {
		t.Fatalf("insert proxy: %v", err)
	}

	rootRes := groupAPIRequest(t, handler, http.MethodPost, "/api/proxy-groups", `{"groupName":"Subscriptions","sortOrder":1}`, true)
	if rootRes.Code != http.StatusCreated {
		t.Fatalf("create root proxy group status = %d: %s", rootRes.Code, rootRes.Body.String())
	}
	var rootPayload struct {
		Group browser.ProxyGroup `json:"group"`
	}
	decodeGroupAPIJSON(t, rootRes.Body, &rootPayload)

	childRes := groupAPIRequest(t, handler, http.MethodPost, "/api/proxy-groups", `{"groupName":"Provider A","parentId":"`+rootPayload.Group.GroupId+`","sortOrder":2}`, true)
	if childRes.Code != http.StatusCreated {
		t.Fatalf("create child proxy group status = %d: %s", childRes.Code, childRes.Body.String())
	}
	var childPayload struct {
		Group browser.ProxyGroup `json:"group"`
	}
	decodeGroupAPIJSON(t, childRes.Body, &childPayload)

	update := groupAPIRequest(t, handler, http.MethodPut, "/api/proxy-groups/"+childPayload.Group.GroupId, `{"groupName":"Provider A Updated","parentId":"`+rootPayload.Group.GroupId+`","sortOrder":3}`, true)
	if update.Code != http.StatusOK {
		t.Fatalf("update proxy group status = %d: %s", update.Code, update.Body.String())
	}

	move := groupAPIRequest(t, handler, http.MethodPost, "/api/proxy-groups/move-proxies", `{"proxyIds":["proxy-1"],"groupId":"`+childPayload.Group.GroupId+`"}`, true)
	if move.Code != http.StatusOK {
		t.Fatalf("move proxy status = %d: %s", move.Code, move.Body.String())
	}

	list := groupAPIRequest(t, handler, http.MethodGet, "/api/proxy-groups", "", true)
	if list.Code != http.StatusOK {
		t.Fatalf("list proxy groups status = %d: %s", list.Code, list.Body.String())
	}
	var listed struct {
		Items []browser.ProxyGroupWithCount `json:"items"`
	}
	decodeGroupAPIJSON(t, list.Body, &listed)
	if len(listed.Items) != 2 {
		t.Fatalf("proxy group count = %d, want 2", len(listed.Items))
	}
	for _, item := range listed.Items {
		if item.GroupId == childPayload.Group.GroupId && item.ProxyCount != 1 {
			t.Fatalf("child proxy count = %d, want 1", item.ProxyCount)
		}
	}

	deleteRes := groupAPIRequest(t, handler, http.MethodDelete, "/api/proxy-groups/"+childPayload.Group.GroupId, "", true)
	if deleteRes.Code != http.StatusOK {
		t.Fatalf("delete proxy group status = %d: %s", deleteRes.Code, deleteRes.Body.String())
	}
	proxies, err := proxyDAO.List()
	if err != nil {
		t.Fatalf("list proxies: %v", err)
	}
	if len(proxies) != 1 || proxies[0].GroupId != rootPayload.Group.GroupId {
		t.Fatalf("proxy group after delete = %+v, want parent %q", proxies, rootPayload.Group.GroupId)
	}
}
