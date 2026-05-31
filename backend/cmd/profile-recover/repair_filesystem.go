package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/google/uuid"
)

var volatileDirNames = map[string]struct{}{
	"browsermetrics":         {},
	"deferredbrowsermetrics": {},
	"graphitedawncache":      {},
	"grshadercache":          {},
	"shadercache":            {},
	"component_crx_cache":    {},
	"extensions_crx_cache":   {},
	"cache":                  {},
	"code cache":             {},
	"gpucache":               {},
}

var volatileFileNames = map[string]struct{}{
	"lock":            {},
	"local state.bad": {},
}

func inspectUserDataDir(dirPath string, currentCoreBinaryPath string) candidateInspection {
	inspection := candidateInspection{}

	markers := make([]string, 0, 4)
	for _, marker := range []string{"Local State", "Default", "Last Browser", "Last Version"} {
		if fileExists(filepath.Join(dirPath, marker)) {
			markers = append(markers, marker)
		}
	}
	inspection.Markers = markers
	inspection.LooksLikeBrowserData = len(markers) > 0
	if !inspection.LooksLikeBrowserData {
		return inspection
	}

	if raw, err := os.ReadFile(filepath.Join(dirPath, "Last Browser")); err == nil {
		inspection.LastBrowser = decodePossiblyUTF16(raw)
	}
	if raw, err := os.ReadFile(filepath.Join(dirPath, "Last Version")); err == nil {
		inspection.LastVersion = strings.TrimSpace(string(raw))
	}

	if fileExists(filepath.Join(dirPath, "Local State.bad")) {
		inspection.Risky = true
		inspection.RiskReasons = append(inspection.RiskReasons, "Local State.bad exists")
	}
	if inspection.LastBrowser != "" && currentCoreBinaryPath != "" {
		if normalizePath(inspection.LastBrowser) != normalizePath(currentCoreBinaryPath) {
			inspection.Risky = true
			inspection.RiskReasons = append(inspection.RiskReasons, fmt.Sprintf("Last Browser points to %s", inspection.LastBrowser))
		}
	}

	return inspection
}

func createRepairCopy(userDataRoot, dirName, sourcePath string) (string, string, error) {
	targetDirName := uniqueRepairDirName(userDataRoot, dirName)
	targetPath := filepath.Join(userDataRoot, targetDirName)
	if err := copyDirFiltered(sourcePath, targetPath); err != nil {
		return "", "", err
	}
	return targetDirName, targetPath, nil
}

func predictedRepairDirName(dirName string, now time.Time) string {
	return fmt.Sprintf("%s__repair_%s", dirName, now.Format("20060102-150405"))
}

func uniqueRepairDirName(userDataRoot, dirName string) string {
	base := predictedRepairDirName(dirName, time.Now())
	target := filepath.Join(userDataRoot, base)
	if !fileExists(target) {
		return base
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s_%02d", base, i)
		if !fileExists(filepath.Join(userDataRoot, candidate)) {
			return candidate
		}
	}
}

func copyDirFiltered(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if shouldSkipRepairPath(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

func shouldSkipRepairPath(rel string, isDir bool) bool {
	clean := filepath.ToSlash(strings.TrimSpace(rel))
	base := strings.ToLower(filepath.Base(clean))

	if strings.HasPrefix(base, "singleton") {
		return true
	}
	if strings.HasSuffix(base, ".tmp") {
		return true
	}
	if _, ok := volatileFileNames[base]; ok {
		return true
	}

	if !isDir {
		return false
	}

	if _, ok := volatileDirNames[base]; ok {
		return true
	}

	parent := strings.ToLower(filepath.Base(filepath.Dir(clean)))
	if parent == "default" {
		if _, ok := volatileDirNames[base]; ok {
			return true
		}
	}

	return false
}

func buildProfileName(prefix, dirName string) string {
	name := strings.TrimSpace(dirName)
	if isUUIDLike(name) {
		name = name[:8]
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return name
	}
	return fmt.Sprintf("%s-%s", prefix, name)
}

func isUUIDLike(value string) bool {
	_, err := uuid.Parse(strings.TrimSpace(value))
	return err == nil
}

func decodePossiblyUTF16(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	if len(raw) >= 2 && len(raw)%2 == 0 {
		zeros := 0
		for i := 1; i < len(raw); i += 2 {
			if raw[i] == 0 {
				zeros++
			}
		}
		if zeros >= len(raw)/4 {
			u16 := make([]uint16, 0, len(raw)/2)
			for i := 0; i+1 < len(raw); i += 2 {
				u16 = append(u16, binary.LittleEndian.Uint16(raw[i:i+2]))
			}
			return strings.TrimSpace(string(utf16.Decode(u16)))
		}
	}
	return strings.TrimSpace(string(raw))
}

func normalizeRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return filepath.Clean(root)
	}
	return filepath.Clean(abs)
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return strings.ToLower(filepath.Clean(p))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
