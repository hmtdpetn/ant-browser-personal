package browser

import (
	"ant-chrome/backend/internal/config"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDefaultBookmarksOnlyAppendsMissingItems(t *testing.T) {
	t.Parallel()

	userDataDir := t.TempDir()
	profileDir := filepath.Join(userDataDir, "Default")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatalf("create profile dir: %v", err)
	}

	root := newEmptyBookmarkRoot("0")
	roots := root["roots"].(map[string]interface{})
	bar := roots["bookmark_bar"].(map[string]interface{})
	bar["children"] = []interface{}{
		map[string]interface{}{
			"id":   "4",
			"name": "用户自己的书签",
			"type": "url",
			"url":  "https://user.example/",
		},
	}
	other := roots["other"].(map[string]interface{})
	other["children"] = []interface{}{
		map[string]interface{}{
			"id":   "5",
			"name": "其他文件夹已有默认书签",
			"type": "url",
			"url":  "https://existing.example/",
		},
	}
	writeBookmarkRoot(t, profileDir, root)

	err := EnsureDefaultBookmarks(userDataDir, []config.BrowserBookmark{
		{Name: "已存在默认书签", URL: "https://existing.example/"},
		{Name: "新增默认书签", URL: "https://new.example/"},
		{Name: "", URL: "https://ignored-name.example/"},
		{Name: "忽略空 URL", URL: ""},
	})
	if err != nil {
		t.Fatalf("EnsureDefaultBookmarks returned error: %v", err)
	}

	updated := readBookmarkRoot(t, profileDir)
	if got := countBookmarkURL(updated, "https://user.example/"); got != 1 {
		t.Fatalf("用户自己的书签不应被改动: count=%d", got)
	}
	if got := countBookmarkURL(updated, "https://existing.example/"); got != 1 {
		t.Fatalf("已存在 URL 不应跨文件夹重复添加: count=%d", got)
	}
	if got := countBookmarkURL(updated, "https://new.example/"); got != 1 {
		t.Fatalf("新增默认书签应追加一次: count=%d", got)
	}
	if got := countBookmarkURL(updated, "https://ignored-name.example/"); got != 0 {
		t.Fatalf("空名称书签不应写入: count=%d", got)
	}
	if !bookmarkBarHasURL(updated, "https://user.example/") {
		t.Fatalf("用户自己的书签应保留在书签栏")
	}
	if !bookmarkBarHasURL(updated, "https://new.example/") {
		t.Fatalf("新增默认书签应追加到书签栏")
	}
}

func TestEnsureDefaultBookmarksDoesNotRewriteWhenNothingMissing(t *testing.T) {
	t.Parallel()

	userDataDir := t.TempDir()
	profileDir := filepath.Join(userDataDir, "Default")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatalf("create profile dir: %v", err)
	}

	root := newEmptyBookmarkRoot("0")
	roots := root["roots"].(map[string]interface{})
	bar := roots["bookmark_bar"].(map[string]interface{})
	bar["date_modified"] = "unchanged"
	bar["children"] = []interface{}{
		map[string]interface{}{
			"id":   "4",
			"name": "已有默认书签",
			"type": "url",
			"url":  "https://existing.example/",
		},
	}
	writeBookmarkRoot(t, profileDir, root)
	before, err := os.ReadFile(filepath.Join(profileDir, "Bookmarks"))
	if err != nil {
		t.Fatalf("read before: %v", err)
	}

	err = EnsureDefaultBookmarks(userDataDir, []config.BrowserBookmark{
		{Name: "已有默认书签", URL: "https://existing.example/"},
	})
	if err != nil {
		t.Fatalf("EnsureDefaultBookmarks returned error: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(profileDir, "Bookmarks"))
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("没有新增项时不应重写用户书签文件")
	}
}

func writeBookmarkRoot(t *testing.T, profileDir string, root map[string]interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(root, "", "   ")
	if err != nil {
		t.Fatalf("marshal bookmarks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "Bookmarks"), data, 0o644); err != nil {
		t.Fatalf("write bookmarks: %v", err)
	}
}

func readBookmarkRoot(t *testing.T, profileDir string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(profileDir, "Bookmarks"))
	if err != nil {
		t.Fatalf("read bookmarks: %v", err)
	}
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("unmarshal bookmarks: %v", err)
	}
	return root
}

func countBookmarkURL(root map[string]interface{}, url string) int {
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
			count += countURLInNodes(children, url)
		}
	}
	return count
}

func countURLInNodes(nodes []interface{}, url string) int {
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
			count += countURLInNodes(children, url)
		}
	}
	return count
}

func bookmarkBarHasURL(root map[string]interface{}, url string) bool {
	roots, ok := root["roots"].(map[string]interface{})
	if !ok {
		return false
	}
	bar, ok := roots["bookmark_bar"].(map[string]interface{})
	if !ok {
		return false
	}
	children, ok := bar["children"].([]interface{})
	return ok && countURLInNodes(children, url) > 0
}
