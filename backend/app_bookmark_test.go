package backend

import (
	internalbrowser "ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBookmarkSyncToProfilesAppliesCurrentDefaults(t *testing.T) {
	t.Parallel()

	appRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Browser.UserDataRoot = "data"
	cfg.Browser.DefaultBookmarks = []config.BrowserBookmark{
		{Name: "默认书签", URL: "https://default.example/"},
	}

	app := NewApp(appRoot)
	app.config = cfg
	app.browserMgr = internalbrowser.NewManager(cfg, appRoot)
	app.browserMgr.Profiles["profile-1"] = &internalbrowser.Profile{
		ProfileId:   "profile-1",
		ProfileName: "实例 1",
		UserDataDir: "profile-1",
	}

	result := app.BookmarkSyncToProfiles()
	if result.Total != 1 || result.Synced != 1 || result.Skipped != 0 || result.Failed != 0 {
		t.Fatalf("unexpected sync result: %+v", result)
	}

	bookmarksPath := filepath.Join(appRoot, "data", "profile-1", "Default", "Bookmarks")
	data, err := os.ReadFile(bookmarksPath)
	if err != nil {
		t.Fatalf("read bookmarks: %v", err)
	}
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("unmarshal bookmarks: %v", err)
	}
	if countBookmarkURLInRoot(root, "https://default.example/") != 1 {
		t.Fatalf("default bookmark was not synced once: %s", string(data))
	}
}

func TestBookmarkListAlwaysIncludesVerificationBookmarks(t *testing.T) {
	t.Parallel()

	appRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Browser.DefaultBookmarks = []config.BrowserBookmark{
		{Name: "用户默认书签", URL: "https://user.example/"},
	}

	app := NewApp(appRoot)
	app.config = cfg
	app.browserMgr = internalbrowser.NewManager(cfg, appRoot)

	bookmarks := app.BookmarkList()
	for _, url := range []string{"https://ippure.com/", "https://iplark.com/", "https://ping0.cc/"} {
		if countBookmarkItemsByURL(bookmarks, url) != 1 {
			t.Fatalf("expected verification bookmark %s exactly once, got %+v", url, bookmarks)
		}
	}
}

func TestBookmarkSavePersistsVerificationBookmarks(t *testing.T) {
	t.Parallel()

	appRoot := t.TempDir()
	cfg := config.DefaultConfig()
	app := NewApp(appRoot)
	app.config = cfg
	app.browserMgr = internalbrowser.NewManager(cfg, appRoot)

	if err := app.BookmarkSave([]config.BrowserBookmark{{Name: "用户默认书签", URL: "https://user.example/", OpenOnStart: true}}); err != nil {
		t.Fatalf("BookmarkSave returned error: %v", err)
	}
	if item, ok := findBookmarkItemByURL(app.config.Browser.DefaultBookmarks, "https://user.example/"); !ok || !item.OpenOnStart {
		t.Fatalf("expected open_on_start to be preserved, got %+v", app.config.Browser.DefaultBookmarks)
	}
	for _, url := range []string{"https://ippure.com/", "https://iplark.com/", "https://ping0.cc/"} {
		if countBookmarkItemsByURL(app.config.Browser.DefaultBookmarks, url) != 1 {
			t.Fatalf("expected saved verification bookmark %s exactly once, got %+v", url, app.config.Browser.DefaultBookmarks)
		}
	}
}

func TestBrowserDefaultStartURLsIncludesOpenOnStartBookmarks(t *testing.T) {
	t.Parallel()

	appRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Browser.DefaultStartURLs = []string{"https://home.example/"}
	cfg.Browser.DefaultBookmarks = []config.BrowserBookmark{
		{Name: "启动打开", URL: "https://open.example/", OpenOnStart: true},
		{Name: "普通书签", URL: "https://closed.example/"},
		{Name: "重复启动页", URL: "https://home.example/", OpenOnStart: true},
	}

	app := NewApp(appRoot)
	app.config = cfg
	app.browserMgr = internalbrowser.NewManager(cfg, appRoot)

	got := app.browserDefaultStartURLs()
	want := []string{"https://home.example/", "https://open.example/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default start urls mismatch: got=%v want=%v", got, want)
	}
}

func countBookmarkURLInRoot(root map[string]interface{}, url string) int {
	count := 0
	roots, ok := root["roots"].(map[string]interface{})
	if !ok {
		return count
	}
	for _, item := range roots {
		folder, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if children, ok := folder["children"].([]interface{}); ok {
			count += countBookmarkURLInNodes(children, url)
		}
	}
	return count
}

func countBookmarkURLInNodes(nodes []interface{}, url string) int {
	count := 0
	for _, item := range nodes {
		node, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if node["type"] == "url" && node["url"] == url {
			count++
		}
		if children, ok := node["children"].([]interface{}); ok {
			count += countBookmarkURLInNodes(children, url)
		}
	}
	return count
}

func findBookmarkItemByURL(items []config.BrowserBookmark, url string) (config.BrowserBookmark, bool) {
	for _, item := range items {
		if item.URL == url {
			return item, true
		}
	}
	return config.BrowserBookmark{}, false
}

func countBookmarkItemsByURL(items []config.BrowserBookmark, url string) int {
	count := 0
	for _, item := range items {
		if item.URL == url {
			count++
		}
	}
	return count
}
