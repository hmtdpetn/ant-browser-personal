package browser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
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
func TestProfileBadgeImageRendersHorizontalLabel(t *testing.T) {
	img := newProfileBadgeImageAtSize("H1", "Harry", 256)
	if got := img.Bounds().Dx(); got != 256 {
		t.Fatalf("badge width mismatch: got=%d", got)
	}
	if got := img.Bounds().Dy(); got != 256 {
		t.Fatalf("badge height mismatch: got=%d", got)
	}
	if alpha := img.RGBAAt(0, 0).A; alpha != 0 {
		t.Fatalf("rounded icon corner should be transparent: alpha=%d", alpha)
	}
	if alpha := img.RGBAAt(128, 128).A; alpha < 240 {
		t.Fatalf("badge center should be opaque: alpha=%d", alpha)
	}

	brightPixels := func(x0, y0, x1, y1 int) int {
		count := 0
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				pixel := img.RGBAAt(x, y)
				if pixel.A > 220 && pixel.R > 225 && pixel.G > 225 && pixel.B > 225 {
					count++
				}
			}
		}
		return count
	}
	leftLabel := brightPixels(35, 55, 128, 200)
	rightLabel := brightPixels(128, 55, 225, 200)
	lowerRightCorner := brightPixels(190, 195, 240, 240)
	if leftLabel < 250 || rightLabel < 250 {
		t.Fatalf("both horizontal label characters should be clearly rendered: left=%d right=%d", leftLabel, rightLabel)
	}
	if lowerRightCorner > rightLabel/3 {
		t.Fatalf("suffix should be inline, not a lower-right circular badge: corner=%d right=%d", lowerRightCorner, rightLabel)
	}
}

func TestProfileBadgeICOContainsExactSizePNGs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "badge.ico")
	if err := writeProfileBadgeICO(path, "H2", "Helen"); err != nil {
		t.Fatalf("writeProfileBadgeICO returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read icon: %v", err)
	}
	if len(data) < 6 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		t.Fatalf("invalid ICO header: %v", data)
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count != len(profileBadgeIconSizes) {
		t.Fatalf("ICO image count mismatch: got=%d want=%d", count, len(profileBadgeIconSizes))
	}

	found := make(map[int]bool, count)
	for index := 0; index < count; index++ {
		directoryOffset := 6 + index*16
		if directoryOffset+16 > len(data) {
			t.Fatalf("ICO directory entry %d exceeds file length", index)
		}
		width := int(data[directoryOffset])
		height := int(data[directoryOffset+1])
		if width == 0 {
			width = 256
		}
		if height == 0 {
			height = 256
		}
		if width != height {
			t.Fatalf("ICO entry %d is not square: %dx%d", index, width, height)
		}
		imageSize := int(binary.LittleEndian.Uint32(data[directoryOffset+8 : directoryOffset+12]))
		imageOffset := int(binary.LittleEndian.Uint32(data[directoryOffset+12 : directoryOffset+16]))
		if imageSize <= 0 || imageOffset < 6+count*16 || imageOffset+imageSize > len(data) {
			t.Fatalf("invalid ICO image entry %d: offset=%d size=%d", index, imageOffset, imageSize)
		}
		decoded, err := png.Decode(bytes.NewReader(data[imageOffset : imageOffset+imageSize]))
		if err != nil {
			t.Fatalf("decode embedded PNG %d: %v", index, err)
		}
		if decoded.Bounds().Dx() != width || decoded.Bounds().Dy() != height {
			t.Fatalf("embedded PNG dimensions mismatch: entry=%dx%d decoded=%v", width, height, decoded.Bounds())
		}
		found[width] = true
	}

	for _, required := range []int{16, 24, 32, 48, 256} {
		if !found[required] {
			t.Fatalf("required Windows icon size %d is missing", required)
		}
	}
}
