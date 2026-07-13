package backend

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
)

func TestProfilePackageImportSkipsMissingUserDataWithWarning(t *testing.T) {
	app, zipPath := newProfilePackageImportTestApp(t, []browser.Profile{{
		ProfileId:     "source-1",
		ProfileName:   "源实例",
		UserDataDir:   "source-1",
		ProxyBindName: "missing-proxy",
	}}, nil)

	result, err := app.importProfilePackageFromPath(zipPath)
	if err != nil {
		t.Fatalf("importProfilePackageFromPath returned error: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("imported count = %d, want 1", result.ImportedCount)
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("warnings = %#v, want proxy and missing user-data warnings", result.Warnings)
	}
	joinedWarnings := strings.Join(result.Warnings, "\n")
	if !strings.Contains(joinedWarnings, "missing-proxy") || !strings.Contains(joinedWarnings, "没有用户数据目录") {
		t.Fatalf("unexpected warnings: %#v", result.Warnings)
	}
	newID := result.ProfileMappings["source-1"]
	if newID == "" {
		t.Fatalf("missing profile mapping: %#v", result.ProfileMappings)
	}
	if _, err := os.Stat(filepath.Join(app.config.Browser.UserDataRoot, newID)); !os.IsNotExist(err) {
		t.Fatalf("missing user-data import should not create final dir, stat err=%v", err)
	}
}

func TestProfilePackageImportCleansFinalDirWhenSaveFails(t *testing.T) {
	app, zipPath := newProfilePackageImportTestApp(t, []browser.Profile{{
		ProfileId:   "source-1",
		ProfileName: "源实例",
		UserDataDir: "source-1",
	}}, map[string]string{"source-1/Default/Preferences": "{}"})

	configPath := filepath.Join(app.appRoot, "config.yaml")
	if err := os.WriteFile(configPath, []byte("blocked"), 0o444); err != nil {
		t.Fatalf("prepare readonly config failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(configPath, 0o644) })

	_, err := app.importProfilePackageFromPath(zipPath)
	if err == nil {
		t.Fatal("expected import to fail when config save fails")
	}
	entries, readErr := os.ReadDir(app.config.Browser.UserDataRoot)
	if readErr != nil {
		t.Fatalf("read user data root failed: %v", readErr)
	}
	for _, entry := range entries {
		if entry.Name() == ".imports" {
			continue
		}
		t.Fatalf("expected final user-data dirs to be rolled back, found %s", entry.Name())
	}
}

func newProfilePackageImportTestApp(t *testing.T, profiles []browser.Profile, userDataFiles map[string]string) (*App, string) {
	t.Helper()
	root := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Browser.UserDataRoot = filepath.Join(root, "user-data")
	app := NewApp(root)
	app.config = cfg
	app.browserMgr = browser.NewManager(cfg, root)
	app.browserMgr.InitData()
	zipPath := filepath.Join(root, "profile-package.zip")
	writeTestProfilePackage(t, zipPath, profiles, userDataFiles)
	return app, zipPath
}

func writeTestProfilePackage(t *testing.T, zipPath string, profiles []browser.Profile, userDataFiles map[string]string) {
	t.Helper()
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip failed: %v", err)
	}
	zipWriter := zip.NewWriter(file)
	writeJSONToZip(t, zipWriter, "manifest.json", ProfilePackageManifest{Format: profilePackageFormat, Version: 1, ProfileCount: len(profiles)})
	writeJSONToZip(t, zipWriter, "profiles.json", profiles)
	for name, content := range userDataFiles {
		writer, err := zipWriter.Create("user-data/" + filepath.ToSlash(name))
		if err != nil {
			t.Fatalf("create zip entry failed: %v", err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry failed: %v", err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip failed: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close file failed: %v", err)
	}
}

func writeJSONToZip(t *testing.T, zipWriter *zip.Writer, name string, value any) {
	t.Helper()
	writer, err := zipWriter.Create(name)
	if err != nil {
		t.Fatalf("create json entry failed: %v", err)
	}
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Fatalf("encode json failed: %v", err)
	}
}
