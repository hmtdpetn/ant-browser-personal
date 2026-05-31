package automation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteScriptPackageDirectory(dirPath string, bundle ImportedBundle) error {
	normalizedPath := filepath.Clean(strings.TrimSpace(dirPath))
	if normalizedPath == "" || normalizedPath == "." {
		return fmt.Errorf("script package directory path is required")
	}

	record, files, err := collectScriptPackageExportFiles(bundle)
	if err != nil {
		return err
	}

	manifestData, err := MarshalScriptPackageManifest(record)
	if err != nil {
		return fmt.Errorf("marshal script package manifest failed: %w", err)
	}

	if info, err := os.Stat(normalizedPath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("script package directory already exists")
		}
		return fmt.Errorf("script package path is not a directory")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat script package directory failed: %w", err)
	}

	parentDir := filepath.Dir(normalizedPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create script package parent dir failed: %w", err)
	}

	tempDir, err := os.MkdirTemp(parentDir, filepath.Base(normalizedPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create script package temp dir failed: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := writeFileAtomic(filepath.Join(tempDir, scriptPackageManifestName), manifestData, 0o644); err != nil {
		return fmt.Errorf("write script package manifest failed: %w", err)
	}
	for _, bundleFile := range files {
		targetPath := filepath.Join(tempDir, filepath.FromSlash(bundleFile.Path))
		if err := writeFileAtomic(targetPath, bundleFile.Content, 0o644); err != nil {
			return fmt.Errorf("write script package file %s failed: %w", bundleFile.Path, err)
		}
	}

	if err := os.Rename(tempDir, normalizedPath); err != nil {
		return fmt.Errorf("move script package directory failed: %w", err)
	}
	return nil
}
