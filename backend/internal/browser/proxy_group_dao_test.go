package browser_test

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/database"
	"path/filepath"
	"testing"
)

func newProxyGroupTestDAO(t *testing.T) (*database.DB, *browser.SQLiteProxyGroupDAO) {
	t.Helper()
	db, err := database.NewDB(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.Migrate(); err != nil {
		db.Close()
		t.Fatalf("Migrate: %v", err)
	}
	return db, browser.NewSQLiteProxyGroupDAO(db.GetConn())
}

func TestProxyGroupHierarchyRenameMoveAndDelete(t *testing.T) {
	db, dao := newProxyGroupTestDAO(t)
	defer db.Close()

	root, err := dao.Create(browser.ProxyGroupInput{GroupName: "主分组", SortOrder: 1})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	child, err := dao.Create(browser.ProxyGroupInput{GroupName: "子分组", ParentId: root.GroupId})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	if _, err := dao.Create(browser.ProxyGroupInput{GroupName: "子分组", ParentId: root.GroupId}); err == nil {
		t.Fatal("expected duplicate sibling name to fail")
	}
	if _, err := dao.Update(root.GroupId, browser.ProxyGroupInput{GroupName: root.GroupName, ParentId: child.GroupId}); err == nil {
		t.Fatal("expected circular hierarchy update to fail")
	}

	if _, err := db.GetConn().Exec(`INSERT INTO browser_proxies (proxy_id, proxy_name, proxy_config) VALUES ('proxy-1', '代理一', 'http://127.0.0.1:8080')`); err != nil {
		t.Fatalf("insert proxy: %v", err)
	}
	if err := dao.MoveProxies([]string{"proxy-1"}, child.GroupId); err != nil {
		t.Fatalf("move proxy: %v", err)
	}

	updated, err := dao.Update(child.GroupId, browser.ProxyGroupInput{GroupName: "已重命名", ParentId: root.GroupId, SortOrder: 2})
	if err != nil {
		t.Fatalf("rename child: %v", err)
	}
	if updated.CreatedAt == "" {
		t.Fatal("updated group did not preserve createdAt")
	}
	var groupID, groupName string
	if err := db.GetConn().QueryRow(`SELECT group_id, group_name FROM browser_proxies WHERE proxy_id = 'proxy-1'`).Scan(&groupID, &groupName); err != nil {
		t.Fatalf("read moved proxy: %v", err)
	}
	if groupID != child.GroupId || groupName != "已重命名" {
		t.Fatalf("proxy assignment after rename = (%q, %q)", groupID, groupName)
	}

	if err := dao.Delete(child.GroupId); err != nil {
		t.Fatalf("delete child: %v", err)
	}
	if err := db.GetConn().QueryRow(`SELECT group_id, group_name FROM browser_proxies WHERE proxy_id = 'proxy-1'`).Scan(&groupID, &groupName); err != nil {
		t.Fatalf("read proxy after group delete: %v", err)
	}
	if groupID != root.GroupId || groupName != root.GroupName {
		t.Fatalf("proxy assignment after delete = (%q, %q), want parent", groupID, groupName)
	}
}

func TestProxyGroupResolveLegacyName(t *testing.T) {
	db, dao := newProxyGroupTestDAO(t)
	defer db.Close()

	id, name, err := dao.ResolveAssignment("", "旧分组")
	if err != nil {
		t.Fatalf("resolve legacy group: %v", err)
	}
	if id == "" || name != "旧分组" {
		t.Fatalf("resolved assignment = (%q, %q)", id, name)
	}
	id2, name2, err := dao.ResolveAssignment("", "旧分组")
	if err != nil {
		t.Fatalf("resolve existing legacy group: %v", err)
	}
	if id2 != id || name2 != name {
		t.Fatalf("second resolve = (%q, %q), want (%q, %q)", id2, name2, id, name)
	}
}
