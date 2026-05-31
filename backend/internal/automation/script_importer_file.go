package automation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ImportBundleFromFile(path string, sourceLabel string) (ImportedBundle, error) {
	return ImportBundleFromFileWithOptions(path, sourceLabel, ImportOptions{})
}

func ImportBundleFromFileWithOptions(path string, sourceLabel string, options ImportOptions) (ImportedBundle, error) {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return ImportedBundle{}, fmt.Errorf("script file path is required")
	}
	for _, candidate := range importManifestCandidates {
		if strings.EqualFold(filepath.Base(normalizedPath), candidate) {
			return ImportBundleFromDirectoryWithOptions(filepath.Dir(normalizedPath), "", sourceLabel, options)
		}
	}
	if strings.EqualFold(filepath.Ext(normalizedPath), ".zip") {
		return ImportBundleFromZipWithOptions(normalizedPath, sourceLabel, options)
	}

	data, err := os.ReadFile(normalizedPath)
	if err != nil {
		return ImportedBundle{}, fmt.Errorf("read script file failed: %w", err)
	}

	return ImportBundleFromBytesWithOptions(filepath.Base(normalizedPath), data, sourceLabel, options)
}
