package browser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var sessionRestoreLegacyFiles = []string{
	"Current Session",
	"Current Tabs",
	"Last Session",
	"Last Tabs",
}

// ClearSessionRestoreData 删除 Chromium 用于恢复上次标签页的会话文件，
// 保留 cookies、Local Storage 等其他用户数据不变。
func ClearSessionRestoreData(userDataDir string) error {
	rootDir := strings.TrimSpace(userDataDir)
	if rootDir == "" {
		return fmt.Errorf("user data dir is empty")
	}

	profileDir := filepath.Join(rootDir, "Default")
	sessionsDir := filepath.Join(profileDir, "Sessions")
	var errs []error

	if err := os.RemoveAll(sessionsDir); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("remove sessions dir: %w", err))
	} else if err == nil {
		if mkErr := os.MkdirAll(sessionsDir, 0o755); mkErr != nil {
			errs = append(errs, fmt.Errorf("recreate sessions dir: %w", mkErr))
		}
	}

	for _, name := range sessionRestoreLegacyFiles {
		if err := os.Remove(filepath.Join(profileDir, name)); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove %s: %w", name, err))
		}
	}

	return errors.Join(errs...)
}
