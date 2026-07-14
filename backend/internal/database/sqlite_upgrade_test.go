package database

import (
	"path/filepath"
	"testing"
)

func TestMigrateFromVersionSevenToCurrent(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	if _, err := db.conn.Exec(`
		CREATE TABLE schema_migrations (
			version    INTEGER PRIMARY KEY,
			desc       TEXT NOT NULL DEFAULT '',
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create historical schema_migrations: %v", err)
	}

	for _, migration := range migrations {
		if migration.version > 7 {
			break
		}
		if err := db.applyMigration(migration); err != nil {
			t.Fatalf("apply historical migration v%d: %v", migration.version, err)
		}
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate from v7: %v", err)
	}

	var version int
	if err := db.conn.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != migrations[len(migrations)-1].version {
		t.Fatalf("schema version = %d, want %d", version, migrations[len(migrations)-1].version)
	}

	for _, query := range []string{
		`SELECT icon_data_url FROM browser_extensions LIMIT 1`,
		`SELECT deleted_at FROM browser_profiles LIMIT 1`,
		`SELECT preferred_kernel FROM browser_proxies LIMIT 1`,
	} {
		if _, err := db.conn.Exec(query); err != nil {
			t.Fatalf("post-upgrade schema check %q: %v", query, err)
		}
	}
}

func TestMigrateLegacyProxyGroupNamesToHierarchy(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "legacy-groups.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()

	if _, err := db.conn.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, desc TEXT NOT NULL DEFAULT '', applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	for _, migration := range migrations {
		if migration.version > 13 {
			break
		}
		if err := db.applyMigration(migration); err != nil {
			t.Fatalf("apply migration v%d: %v", migration.version, err)
		}
	}
	if _, err := db.conn.Exec(`INSERT INTO browser_proxies (proxy_id, proxy_name, proxy_config, group_name) VALUES ('legacy-proxy', '旧代理', 'http://127.0.0.1:9000', '旧代理组')`); err != nil {
		t.Fatalf("insert legacy proxy: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate v13 to current: %v", err)
	}

	var proxyGroupID, groupName string
	if err := db.conn.QueryRow(`SELECT group_id, group_name FROM browser_proxies WHERE proxy_id = 'legacy-proxy'`).Scan(&proxyGroupID, &groupName); err != nil {
		t.Fatalf("read migrated proxy: %v", err)
	}
	if proxyGroupID == "" || groupName != "旧代理组" {
		t.Fatalf("migrated proxy group = (%q, %q)", proxyGroupID, groupName)
	}
	var parentID string
	if err := db.conn.QueryRow(`SELECT parent_id FROM browser_proxy_groups WHERE group_id = ? AND group_name = ?`, proxyGroupID, groupName).Scan(&parentID); err != nil {
		t.Fatalf("read migrated proxy group: %v", err)
	}
	if parentID != "" {
		t.Fatalf("legacy proxy group parent = %q, want root", parentID)
	}
}
