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
	if version != 12 {
		t.Fatalf("schema version = %d, want 12", version)
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
