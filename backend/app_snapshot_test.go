package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZipDirAndUnzipTo(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "src")
	dstZip := filepath.Join(root, "archive.zip")
	dstDir := filepath.Join(root, "dst")

	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	if err := zipDir(src, dstZip); err != nil {
		t.Fatalf("zipDir failed: %v", err)
	}
	if err := unzipTo(dstZip, dstDir); err != nil {
		t.Fatalf("unzipTo failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dstDir, "nested", "file.txt"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("extracted content = %q, want hello", string(data))
	}
}

func TestFindSnapshotFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	metaPath := filepath.Join(dir, "snap-1_demo.meta.json")
	zipPath := filepath.Join(dir, "snap-1_demo.zip")

	if err := os.WriteFile(metaPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	if err := os.WriteFile(zipPath, []byte("zip"), 0o644); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	gotMeta, gotZip, err := findSnapshotFiles(dir, "snap-1")
	if err != nil {
		t.Fatalf("findSnapshotFiles failed: %v", err)
	}
	if gotMeta != metaPath {
		t.Fatalf("meta path = %q, want %q", gotMeta, metaPath)
	}
	if gotZip != zipPath {
		t.Fatalf("zip path = %q, want %q", gotZip, zipPath)
	}
}
