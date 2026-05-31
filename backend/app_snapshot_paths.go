package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// snapshotDir 返回指定实例的快照目录路径（存放在 data/snapshots 下）
func (a *App) snapshotDir(profileId string) (string, error) {
	dir := filepath.Join(a.resolveAppPath("data"), "snapshots", profileId)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// findSnapshotFiles 在快照目录中找到指定 snapshotId 的 meta 和 zip 路径
func findSnapshotFiles(snapDir, snapshotId string) (metaPath, zipPath string, err error) {
	entries, err := os.ReadDir(snapDir)
	if err != nil {
		return "", "", err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), snapshotId) && strings.HasSuffix(entry.Name(), ".meta.json") {
			metaPath = filepath.Join(snapDir, entry.Name())
			zipPath = strings.TrimSuffix(metaPath, ".meta.json") + ".zip"
			if _, err := os.Stat(zipPath); err != nil {
				return "", "", fmt.Errorf("快照文件不存在: %s", zipPath)
			}
			return metaPath, zipPath, nil
		}
	}
	return "", "", fmt.Errorf("快照不存在: %s", snapshotId)
}
