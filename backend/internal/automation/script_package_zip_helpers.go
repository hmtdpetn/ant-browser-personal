package automation

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func sanitizedImportedZipPath(destDir string, rawName string) (string, bool, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return "", true, nil
	}

	cleanName := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(filepath.ToSlash(name), "/")))
	if cleanName == "." || cleanName == "" {
		return "", true, nil
	}

	targetPath := filepath.Join(destDir, cleanName)
	cleanDest := filepath.Clean(destDir)
	cleanTarget := filepath.Clean(targetPath)
	if cleanTarget != cleanDest && !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) {
		return "", false, fmt.Errorf("script zip contains invalid path %s", rawName)
	}

	return targetPath, false, nil
}

func shouldSkipImportedZipEntry(name string) bool {
	normalized := filepath.ToSlash(strings.TrimSpace(name))
	if normalized == "" {
		return true
	}

	base := pathBase(normalized)
	if base == ".DS_Store" || strings.HasPrefix(base, "._") {
		return true
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == "__MACOSX" {
			return true
		}
	}
	return false
}

func isImportManifestPath(relativePath string) bool {
	normalized := filepath.ToSlash(strings.TrimSpace(relativePath))
	for _, candidate := range importManifestCandidates {
		if strings.EqualFold(normalized, candidate) {
			return true
		}
	}
	return false
}

func isZipArchiveData(nameHint string, data []byte) bool {
	if strings.EqualFold(strings.TrimSpace(filepath.Ext(nameHint)), ".zip") {
		return true
	}
	return len(data) >= 4 && bytes.Equal(data[:4], []byte("PK\x03\x04"))
}

func pathBase(value string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(value), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}
