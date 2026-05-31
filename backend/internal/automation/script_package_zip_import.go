package automation

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ImportBundleFromZip(path string, sourceLabel string) (ImportedBundle, error) {
	return ImportBundleFromZipWithOptions(path, sourceLabel, ImportOptions{})
}

func ImportBundleFromZipWithOptions(path string, sourceLabel string, options ImportOptions) (ImportedBundle, error) {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return ImportedBundle{}, fmt.Errorf("script zip path is required")
	}

	reader, err := zip.OpenReader(normalizedPath)
	if err != nil {
		return ImportedBundle{}, fmt.Errorf("open script zip failed: %w", err)
	}
	defer reader.Close()

	return importBundleFromZipReader(&reader.Reader, sourceLabel, options)
}

func importBundleFromZipBytes(nameHint string, data []byte, sourceLabel string, options ImportOptions) (ImportedBundle, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ImportedBundle{}, fmt.Errorf("open script zip failed: %w", err)
	}
	return importBundleFromZipReader(reader, sourceLabel, options)
}

func importBundleFromZipReader(reader *zip.Reader, sourceLabel string, options ImportOptions) (ImportedBundle, error) {
	extractRoot, err := os.MkdirTemp("", "ant-automation-zip-*")
	if err != nil {
		return ImportedBundle{}, fmt.Errorf("create script zip temp dir failed: %w", err)
	}
	defer os.RemoveAll(extractRoot)

	if err := extractImportedZip(reader, extractRoot); err != nil {
		return ImportedBundle{}, err
	}

	bundle, err := ImportBundleFromDirectoryWithOptions(extractRoot, "", sourceLabel, options)
	if err == nil {
		return bundle, nil
	}

	nestedRoot, nestedFound, nestedErr := detectSingleImportedZipRoot(extractRoot)
	if nestedErr != nil {
		return ImportedBundle{}, nestedErr
	}
	if nestedFound {
		return ImportBundleFromDirectoryWithOptions(nestedRoot, "", sourceLabel, options)
	}
	return ImportedBundle{}, err
}

func extractImportedZip(reader *zip.Reader, destDir string) error {
	fileCount := 0
	totalBytes := 0

	for _, file := range reader.File {
		if shouldSkipImportedZipEntry(file.Name) {
			continue
		}

		targetPath, skip, err := sanitizedImportedZipPath(destDir, file.Name)
		if err != nil {
			return err
		}
		if skip {
			continue
		}

		mode := file.Mode()
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("create script zip dir failed: %w", err)
			}
			continue
		}
		if !mode.IsRegular() {
			return fmt.Errorf("script zip contains unsupported entry %s", file.Name)
		}

		fileCount++
		if fileCount > maxImportedZipFiles {
			return fmt.Errorf("script zip contains too many files")
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create script zip file dir failed: %w", err)
		}

		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("open script zip entry failed: %w", err)
		}

		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			src.Close()
			return fmt.Errorf("create script zip file failed: %w", err)
		}

		written, copyErr := io.Copy(dst, io.LimitReader(src, int64(maxImportedZipBytes-totalBytes)+1))
		closeErr := dst.Close()
		srcCloseErr := src.Close()
		if copyErr != nil {
			return fmt.Errorf("extract script zip entry failed: %w", copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close extracted script file failed: %w", closeErr)
		}
		if srcCloseErr != nil {
			return fmt.Errorf("close script zip entry failed: %w", srcCloseErr)
		}

		totalBytes += int(written)
		if totalBytes > maxImportedZipBytes {
			return fmt.Errorf("script zip is too large")
		}
	}

	return nil
}

func detectSingleImportedZipRoot(root string) (string, bool, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", false, fmt.Errorf("read script zip temp dir failed: %w", err)
	}

	directories := make([]string, 0, 1)
	for _, entry := range entries {
		if shouldSkipImportedZipEntry(entry.Name()) {
			continue
		}
		if !entry.IsDir() {
			return "", false, nil
		}
		directories = append(directories, filepath.Join(root, entry.Name()))
	}

	if len(directories) != 1 {
		return "", false, nil
	}
	return directories[0], true, nil
}
