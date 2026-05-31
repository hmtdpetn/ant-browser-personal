package main

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/database"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func loadExistingProfiles(dbPath string, userDataRoot string, apply bool) ([]existingProfile, *sql.DB, *database.DB, error) {
	dbExists := fileExists(dbPath)
	if !dbExists && !apply {
		return nil, nil, nil, nil
	}

	if apply {
		handle, err := database.NewDB(dbPath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("open database: %w", err)
		}
		list, err := browser.NewSQLiteProfileDAO(handle.GetConn()).List()
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "no such table") {
				return nil, nil, handle, nil
			}
			_ = handle.Close()
			return nil, nil, nil, fmt.Errorf("load existing profiles: %w", err)
		}
		return toExistingProfiles(list, userDataRoot), nil, handle, nil
	}

	db, err := openQueryDB(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open database: %w", err)
	}
	list, err := browser.NewSQLiteProfileDAO(db).List()
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("load existing profiles: %w", err)
	}
	return toExistingProfiles(list, userDataRoot), db, nil, nil
}

func toExistingProfiles(items []*browser.Profile, userDataRoot string) []existingProfile {
	out := make([]existingProfile, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, existingProfile{
			ProfileID:    item.ProfileId,
			ProfileName:  item.ProfileName,
			UserDataDir:  item.UserDataDir,
			ResolvedPath: resolveUserDataPath(userDataRoot, item.UserDataDir),
		})
	}
	return out
}

func resolveUserDataPath(userDataRoot string, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	return filepath.Join(userDataRoot, raw)
}

func openQueryDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func backupDatabaseFiles(dbPath string, now time.Time) (string, error) {
	dataRoot := filepath.Dir(dbPath)
	backupDir := filepath.Join(dataRoot, "recovery-backups", now.Format("20060102-150405"))
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	for _, src := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		if !fileExists(src) {
			continue
		}
		dst := filepath.Join(backupDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			return "", fmt.Errorf("backup %s: %w", src, err)
		}
	}

	return backupDir, nil
}
