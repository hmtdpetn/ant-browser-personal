package automation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteScriptPackageDirectoryRoundTripsAdditionalFiles(t *testing.T) {
	exportDir := filepath.Join(t.TempDir(), "demo-package")

	if err := WriteScriptPackageDirectory(exportDir, ImportedBundle{
		Record: ScriptRecord{
			ID:          "dir-roundtrip",
			Name:        "目录导出",
			Description: "包含额外文件",
			Type:        "playwright-cdp",
			Status:      "ready",
			EntryFile:   "scripts/index.cjs",
			ScriptText:  "const helper = require('./helpers/helper.cjs')\nmodule.exports.run = async () => helper.run()",
		},
		Files: []ImportedBundleFile{
			{
				Path:    "scripts/index.cjs",
				Content: []byte("const helper = require('./helpers/helper.cjs')\nmodule.exports.run = async () => helper.run()"),
			},
			{
				Path:    "scripts/helpers/helper.cjs",
				Content: []byte("module.exports.run = async () => ({ ok: true })"),
			},
			{
				Path:    "assets/raw.bin",
				Content: []byte{0x00, 0x01, 0x02, 0xff},
			},
		},
	}); err != nil {
		t.Fatalf("WriteScriptPackageDirectory returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(exportDir, scriptPackageManifestName)); err != nil {
		t.Fatalf("expected manifest to exist: %v", err)
	}

	imported, err := ImportBundleFromDirectory(exportDir, "", "本地目录 "+exportDir)
	if err != nil {
		t.Fatalf("ImportBundleFromDirectory returned error: %v", err)
	}

	if imported.Record.EntryFile != "scripts/index.cjs" {
		t.Fatalf("unexpected entry file: %s", imported.Record.EntryFile)
	}
	if !hasBundleFile(imported.Files, "scripts/helpers/helper.cjs", []byte("module.exports.run = async () => ({ ok: true })")) {
		t.Fatalf("expected helper file to round-trip, got %+v", imported.Files)
	}
	if !hasBundleFile(imported.Files, "assets/raw.bin", []byte{0x00, 0x01, 0x02, 0xff}) {
		t.Fatalf("expected binary file to round-trip, got %+v", imported.Files)
	}
}
