package automation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func collectImportedBundleFiles(root string) ([]ImportedBundleFile, error) {
	files := make([]ImportedBundleFile, 0, 8)
	totalSize := 0

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if relativePath == "." {
			return nil
		}

		if entry.IsDir() {
			name := strings.ToLower(entry.Name())
			if name == ".git" {
				return filepath.SkipDir
			}
			if name == "node_modules" {
				return fmt.Errorf("script bundle must not include node_modules")
			}
			return nil
		}

		if relativePath == "manifest.json" || relativePath == "automation.script.json" || relativePath == "ant-automation.json" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		totalSize += len(content)
		if totalSize > maxImportedBundleBytes {
			return fmt.Errorf("script bundle is too large")
		}
		if len(files) >= maxImportedBundleFiles {
			return fmt.Errorf("script bundle contains too many files")
		}

		files = append(files, ImportedBundleFile{
			Path:    relativePath,
			Content: content,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect script bundle files failed: %w", err)
	}

	return files, nil
}

func resolveImportManifest(root string) (string, error) {
	for _, candidate := range importManifestCandidates {
		targetPath := filepath.Join(root, candidate)
		if _, err := os.Stat(targetPath); err == nil {
			return targetPath, nil
		}
	}
	return "", nil
}

func resolvePathUnderRoot(root string, target string) (string, error) {
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	if cleanRoot == "" || cleanRoot == "." {
		return "", fmt.Errorf("script root path is required")
	}

	candidate := cleanRoot
	if strings.TrimSpace(target) != "" {
		candidate = filepath.Clean(filepath.Join(cleanRoot, target))
	}

	relativePath, err := filepath.Rel(cleanRoot, candidate)
	if err != nil {
		return "", fmt.Errorf("resolve script path failed: %w", err)
	}
	relativePath = filepath.ToSlash(relativePath)
	if relativePath == ".." || strings.HasPrefix(relativePath, "../") {
		return "", fmt.Errorf("script path escapes package root")
	}

	return candidate, nil
}
