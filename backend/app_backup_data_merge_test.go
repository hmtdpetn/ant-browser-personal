package backend

import (
	"ant-chrome/backend/internal/database"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestBackupMergeDatabaseSupportsLegacyAndCurrentProxySchemas(t *testing.T) {
	root := t.TempDir()
	destination, err := database.NewDB(filepath.Join(root, "destination.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { _ = destination.Close() })
	if err := destination.Migrate(); err != nil {
		t.Fatalf("Migrate destination: %v", err)
	}
	app := NewApp(root)
	app.db = destination
	stats := &backupMergeStats{}

	legacyPath := filepath.Join(root, "legacy-source.db")
	createBackupMergeSource(t, legacyPath, []string{
		`CREATE TABLE browser_proxies (
			proxy_id TEXT PRIMARY KEY,
			proxy_name TEXT NOT NULL,
			proxy_config TEXT NOT NULL
		)`,
		`INSERT INTO browser_proxies (proxy_id, proxy_name, proxy_config)
		 VALUES ('legacy-1', 'Legacy One', 'http://127.0.0.1:18080')`,
	})
	if err := app.backupMergeDatabaseFromSource(legacyPath, false, stats); err != nil {
		t.Fatalf("merge legacy proxy schema: %v", err)
	}

	var legacyGroupID, legacyUserAgent, legacyKernel string
	var legacyFallback, legacyLatency int
	if err := destination.GetConn().QueryRow(`
		SELECT group_id, source_user_agent, source_user_agent_fallback, preferred_kernel, last_latency_ms
		FROM browser_proxies WHERE proxy_id = 'legacy-1'
	`).Scan(&legacyGroupID, &legacyUserAgent, &legacyFallback, &legacyKernel, &legacyLatency); err != nil {
		t.Fatalf("read merged legacy proxy: %v", err)
	}
	if legacyGroupID != "" || legacyUserAgent != "" || legacyFallback != 1 || legacyKernel != "" || legacyLatency != -1 {
		t.Fatalf("legacy defaults mismatch: group=%q ua=%q fallback=%d kernel=%q latency=%d",
			legacyGroupID, legacyUserAgent, legacyFallback, legacyKernel, legacyLatency)
	}

	currentPath := filepath.Join(root, "current-source.db")
	createBackupMergeSource(t, currentPath, []string{
		`CREATE TABLE browser_proxy_groups (
			group_id TEXT PRIMARY KEY,
			group_name TEXT NOT NULL,
			parent_id TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`INSERT INTO browser_proxy_groups
		 (group_id, group_name, parent_id, sort_order, created_at, updated_at)
		 VALUES ('proxy-group-1', 'Primary', '', 3, '2026-07-13T00:00:00Z', '2026-07-13T00:00:00Z')`,
		`CREATE TABLE browser_proxies (
			proxy_id TEXT PRIMARY KEY,
			proxy_name TEXT NOT NULL,
			proxy_config TEXT NOT NULL,
			preferred_kernel TEXT,
			group_name TEXT,
			group_id TEXT,
			source_user_agent TEXT,
			source_user_agent_fallback INTEGER,
			created_at TEXT
		)`,
		`INSERT INTO browser_proxies
		 (proxy_id, proxy_name, proxy_config, preferred_kernel, group_name, group_id,
		  source_user_agent, source_user_agent_fallback, created_at)
		 VALUES ('current-1', 'Current One', 'socks5://127.0.0.1:19090', 'sing-box',
		         'Primary', 'proxy-group-1', 'FlClash/test', 0, '2026-07-13T00:00:00Z')`,
	})
	if err := app.backupMergeDatabaseFromSource(currentPath, false, stats); err != nil {
		t.Fatalf("merge current proxy schema after legacy merge: %v", err)
	}

	var groupName, parentID string
	if err := destination.GetConn().QueryRow(`
		SELECT group_name, parent_id FROM browser_proxy_groups WHERE group_id = 'proxy-group-1'
	`).Scan(&groupName, &parentID); err != nil {
		t.Fatalf("read merged proxy group: %v", err)
	}
	if groupName != "Primary" || parentID != "" {
		t.Fatalf("merged proxy group mismatch: name=%q parent=%q", groupName, parentID)
	}

	var currentGroupID, currentGroupName, currentUserAgent, currentKernel string
	var currentFallback int
	if err := destination.GetConn().QueryRow(`
		SELECT group_id, group_name, source_user_agent, source_user_agent_fallback, preferred_kernel
		FROM browser_proxies WHERE proxy_id = 'current-1'
	`).Scan(&currentGroupID, &currentGroupName, &currentUserAgent, &currentFallback, &currentKernel); err != nil {
		t.Fatalf("read merged current proxy: %v", err)
	}
	if currentGroupID != "proxy-group-1" || currentGroupName != "Primary" ||
		currentUserAgent != "FlClash/test" || currentFallback != 0 || currentKernel != "sing-box" {
		t.Fatalf("current proxy fields mismatch: groupID=%q group=%q ua=%q fallback=%d kernel=%q",
			currentGroupID, currentGroupName, currentUserAgent, currentFallback, currentKernel)
	}
	if stats.Imported != 3 || stats.Skipped != 0 {
		t.Fatalf("merge stats mismatch: %+v", *stats)
	}
}

func createBackupMergeSource(t *testing.T, path string, statements []string) {
	t.Helper()
	connection, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open source database: %v", err)
	}
	for _, statement := range statements {
		if _, err := connection.Exec(statement); err != nil {
			_ = connection.Close()
			t.Fatalf("prepare source database: %v", err)
		}
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("close source database: %v", err)
	}
}
