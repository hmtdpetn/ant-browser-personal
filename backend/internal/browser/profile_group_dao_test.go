package browser_test

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/database"
	"path/filepath"
	"testing"
)

func newProfileGroupTestDAO(t *testing.T) (*database.DB, *browser.SQLiteProfileDAO, *browser.SQLiteGroupDAO) {
	t.Helper()
	db, err := database.NewDB(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	if err := db.Migrate(); err != nil {
		db.Close()
		t.Fatalf("Migrate: %v", err)
	}
	return db, browser.NewSQLiteProfileDAO(db.GetConn()), browser.NewSQLiteGroupDAO(db.GetConn())
}

func insertProfileForGroupTest(t *testing.T, db *database.DB, profileID string) {
	t.Helper()
	_, err := db.GetConn().Exec(`
		INSERT INTO browser_profiles (profile_id, profile_name, user_data_dir, created_at, updated_at)
		VALUES (?, ?, '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`, profileID, profileID)
	if err != nil {
		t.Fatalf("insert profile %s: %v", profileID, err)
	}
}

func readProfileGroupForTest(t *testing.T, db *database.DB, profileID string) (string, string) {
	t.Helper()
	var groupID, updatedAt string
	if err := db.GetConn().QueryRow(`SELECT group_id, updated_at FROM browser_profiles WHERE profile_id = ?`, profileID).Scan(&groupID, &updatedAt); err != nil {
		t.Fatalf("read profile %s: %v", profileID, err)
	}
	return groupID, updatedAt
}

func TestProfileMoveToGroupAndUngrouped(t *testing.T) {
	db, profileDAO, groupDAO := newProfileGroupTestDAO(t)
	defer db.Close()

	group, err := groupDAO.Create(browser.GroupInput{GroupName: "目标分组"})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	insertProfileForGroupTest(t, db, "profile-1")
	insertProfileForGroupTest(t, db, "profile-2")

	if err := profileDAO.MoveToGroup([]string{"profile-1", "profile-2", "profile-1"}, group.GroupId); err != nil {
		t.Fatalf("move profiles: %v", err)
	}
	for _, profileID := range []string{"profile-1", "profile-2"} {
		groupID, updatedAt := readProfileGroupForTest(t, db, profileID)
		if groupID != group.GroupId {
			t.Fatalf("%s group = %q, want %q", profileID, groupID, group.GroupId)
		}
		if updatedAt == "2026-01-01T00:00:00Z" {
			t.Fatalf("%s updated_at was not refreshed", profileID)
		}
	}

	if err := profileDAO.MoveToGroup([]string{"profile-1", "profile-2"}, ""); err != nil {
		t.Fatalf("move profiles to ungrouped: %v", err)
	}
	for _, profileID := range []string{"profile-1", "profile-2"} {
		groupID, _ := readProfileGroupForTest(t, db, profileID)
		if groupID != "" {
			t.Fatalf("%s group = %q, want ungrouped", profileID, groupID)
		}
	}
}

func TestProfileMoveToGroupRollsBackOnInvalidTargetOrProfile(t *testing.T) {
	db, profileDAO, groupDAO := newProfileGroupTestDAO(t)
	defer db.Close()

	group, err := groupDAO.Create(browser.GroupInput{GroupName: "原分组"})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	insertProfileForGroupTest(t, db, "profile-1")
	if err := profileDAO.MoveToGroup([]string{"profile-1"}, group.GroupId); err != nil {
		t.Fatalf("initial move: %v", err)
	}

	if err := profileDAO.MoveToGroup([]string{"profile-1"}, "missing-group"); err == nil {
		t.Fatal("expected invalid target group to fail")
	}
	if err := profileDAO.MoveToGroup([]string{"profile-1", "missing-profile"}, ""); err == nil {
		t.Fatal("expected missing profile to fail")
	}
	groupID, _ := readProfileGroupForTest(t, db, "profile-1")
	if groupID != group.GroupId {
		t.Fatalf("transaction did not roll back: group = %q, want %q", groupID, group.GroupId)
	}
}
