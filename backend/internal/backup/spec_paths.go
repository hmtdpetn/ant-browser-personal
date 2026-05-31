package backup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func resolvePath(appRoot, p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return filepath.Clean(appRoot)
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(appRoot, p))
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func samePath(a, b string) bool {
	return normalizeForCompare(a) == normalizeForCompare(b)
}

func isPathWithin(path, dir string) bool {
	p := normalizeForCompare(path)
	d := normalizeForCompare(dir)
	if p == d {
		return true
	}
	if d == "" || p == "" {
		return false
	}
	if !strings.HasSuffix(d, string(filepath.Separator)) {
		d += string(filepath.Separator)
	}
	return strings.HasPrefix(p, d)
}

func normalizeForCompare(p string) string {
	normalized := filepath.Clean(strings.TrimSpace(p))
	if runtime.GOOS == "windows" {
		normalized = strings.ToLower(normalized)
	}
	return normalized
}
