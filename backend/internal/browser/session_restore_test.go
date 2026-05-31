package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearSessionRestoreDataRemovesSessionArtifactsOnly(t *testing.T) {
	t.Parallel()

	userDataDir := t.TempDir()
	profileDir := filepath.Join(userDataDir, "Default")
	sessionsDir := filepath.Join(profileDir, "Sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("创建 Sessions 目录失败: %v", err)
	}

	filesToCreate := []string{
		filepath.Join(sessionsDir, "Session_1"),
		filepath.Join(sessionsDir, "Tabs_1"),
		filepath.Join(profileDir, "Last Session"),
		filepath.Join(profileDir, "Current Tabs"),
		filepath.Join(profileDir, "Preferences"),
	}
	for _, path := range filesToCreate {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if err := os.WriteFile(path, []byte("stub"), 0o644); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}
	}

	if err := ClearSessionRestoreData(userDataDir); err != nil {
		t.Fatalf("ClearSessionRestoreData 返回错误: %v", err)
	}

	if entries, err := os.ReadDir(sessionsDir); err != nil {
		t.Fatalf("读取 Sessions 目录失败: %v", err)
	} else if len(entries) != 0 {
		t.Fatalf("Sessions 目录应为空: got=%d", len(entries))
	}

	for _, name := range []string{"Last Session", "Current Tabs"} {
		if _, err := os.Stat(filepath.Join(profileDir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s 应已删除: err=%v", name, err)
		}
	}

	if _, err := os.Stat(filepath.Join(profileDir, "Preferences")); err != nil {
		t.Fatalf("Preferences 不应被删除: %v", err)
	}
}
