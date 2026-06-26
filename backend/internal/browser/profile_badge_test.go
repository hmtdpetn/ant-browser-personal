package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProfileBadgeLabelUsesNameInitials(t *testing.T) {
	profiles := []*Profile{
		{ProfileId: "p1", ProfileName: "hello", CreatedAt: "2026-01-01T00:00:00Z"},
		{ProfileId: "p2", ProfileName: "9 shop", CreatedAt: "2026-01-02T00:00:00Z"},
		{ProfileId: "p3", ProfileName: "\u6d4b\u8bd5", CreatedAt: "2026-01-03T00:00:00Z"},
	}

	if got := ProfileBadgeLabel(profiles, "p1"); got != "H" {
		t.Fatalf("english initial mismatch: got=%s", got)
	}
	if got := ProfileBadgeLabel(profiles, "p2"); got != "9" {
		t.Fatalf("numeric initial mismatch: got=%s", got)
	}
	if got := ProfileBadgeLabel(profiles, "p3"); got != "C" {
		t.Fatalf("chinese pinyin initial mismatch: got=%s", got)
	}
}

func TestProfileBadgeLabelAddsStableDuplicateSuffixes(t *testing.T) {
	profiles := []*Profile{
		{ProfileId: "newer", ProfileName: "Helen", CreatedAt: "2026-01-02T00:00:00Z"},
		{ProfileId: "older", ProfileName: "Harry", CreatedAt: "2026-01-01T00:00:00Z"},
		{ProfileId: "other", ProfileName: "Bob", CreatedAt: "2026-01-03T00:00:00Z"},
	}

	if got := ProfileBadgeLabel(profiles, "older"); got != "H1" {
		t.Fatalf("older duplicate suffix mismatch: got=%s", got)
	}
	if got := ProfileBadgeLabel(profiles, "newer"); got != "H2" {
		t.Fatalf("newer duplicate suffix mismatch: got=%s", got)
	}
	if got := ProfileBadgeLabel(profiles, "other"); got != "B" {
		t.Fatalf("unique base should not get suffix: got=%s", got)
	}
}

func TestProfileBadgeLabelKeepsSuffixCompactAfterNine(t *testing.T) {
	profiles := make([]*Profile, 0, 11)
	for i := 0; i < 11; i++ {
		profiles = append(profiles, &Profile{
			ProfileId:   string(rune('a' + i)),
			ProfileName: "Hero",
			CreatedAt:   fmt.Sprintf("2026-01-01T00:00:%02dZ", i),
		})
	}

	if got := ProfileBadgeLabel(profiles, "j"); got != "H0" {
		t.Fatalf("tenth compact suffix mismatch: got=%s", got)
	}
	if got := ProfileBadgeLabel(profiles, "k"); got != "HA" {
		t.Fatalf("eleventh compact suffix mismatch: got=%s", got)
	}
}

func TestEnsureProfileBadgeIconWritesIcon(t *testing.T) {
	root := t.TempDir()
	iconPath, err := EnsureProfileBadgeIcon(root, "Harry", "H1")
	if err != nil {
		t.Fatalf("EnsureProfileBadgeIcon returned error: %v", err)
	}
	if iconPath != filepath.Join(root, "Default", ProfileBadgeIconName) {
		t.Fatalf("unexpected icon path: %s", iconPath)
	}
	if info, err := os.Stat(iconPath); err != nil {
		t.Fatalf("expected icon file to exist: %v", err)
	} else if info.Size() == 0 {
		t.Fatalf("icon file should not be empty")
	}
}
